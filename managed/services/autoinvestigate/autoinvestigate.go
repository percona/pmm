// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package autoinvestigate drives authoritative, idempotent auto-investigations from Grafana alert
// data (webhook or reconciliation poll). It replaces the legacy Slack-message FIRING scraping: each
// firing alert episode produces at most one investigation, run through the standard investigations
// pipeline, with its summary posted to the configured Slack output channels.
package autoinvestigate

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// SlackRef ties an alert to the Slack message it was scraped from, so the investigation's notices can
// be posted back as replies in that alert's thread. Nil for webhook/poll alerts (no Slack origin).
type SlackRef struct {
	TeamID   string
	Channel  string
	ThreadTS string
}

// Alert is an authoritative alert from a Grafana webhook, the Alertmanager poll, or a scraped Slack
// alert message.
type Alert struct {
	Fingerprint string
	Status      string // "firing" or "resolved"
	Labels      map[string]string
	Annotations map[string]string
	// Slack is set only for alerts scraped from a Slack message; it carries the thread to post into.
	Slack *SlackRef
}

func (a Alert) firing() bool   { return strings.EqualFold(strings.TrimSpace(a.Status), "firing") }
func (a Alert) resolved() bool { return strings.EqualFold(strings.TrimSpace(a.Status), "resolved") }

// Runner starts a background investigation run. Implemented by investigations.Handlers.
type Runner interface {
	StartRun(ctx context.Context, investigationID string) error
}

// Notifier posts auto-investigate output to Slack. Implemented by a slackbot-backed adapter. It must
// be safe to call with an empty channel list (no-op) and must never block for long.
type Notifier interface {
	PostAutoInvestigateStarted(ctx context.Context, channels []string, inv *models.Investigation)
}

// severityRank orders the standard alert severities for the min-severity floor. Unknown severities
// rank 0 (treated as below any configured floor).
var severityRank = map[string]int{"info": 1, "warning": 2, "critical": 3}

// maxEpisodes bounds the in-memory episode map so it can't grow without limit (e.g. on an HA
// follower that receives webhooks but never runs the pruning poll). Eviction only costs a redundant
// DB claim that the partial unique index rejects — the DB is the authoritative dedup, the map is an
// optimization.
const maxEpisodes = 10000

// Service coordinates idempotent auto-investigations. The in-memory episode map prevents
// re-investigating an alert that is still firing after its investigation has already completed (the
// DB partial unique index only coalesces while the investigation is active); the DB claim is the
// authoritative guard against concurrent duplicates and the DB-backed hourly count is the global cap.
type Service struct {
	db       *reform.DB
	runner   Runner
	notifier Notifier
	l        *logrus.Entry

	mu             sync.Mutex
	episodes       map[string]struct{} // fingerprint -> already investigated this firing episode
	warnedNoOutput bool                // emit the "auto-investigate on but no output channels" warning once
}

// New creates an auto-investigate Service.
func New(db *reform.DB, runner Runner, notifier Notifier, l *logrus.Entry) *Service {
	return &Service{
		db:       db,
		runner:   runner,
		notifier: notifier,
		l:        l.WithField("component", "auto-investigate"),
		episodes: make(map[string]struct{}),
	}
}

// ProcessAlerts evaluates a batch of authoritative alerts: firing alerts that pass the selection and
// cost guards are claimed and run (once per episode); resolved alerts re-arm their episode. It loads
// settings once and is a no-op when auto-investigate is disabled.
func (s *Service) ProcessAlerts(ctx context.Context, alerts []Alert) {
	if len(alerts) == 0 {
		return
	}
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("GetSettings: %v", err)
		return
	}
	if !settings.Adre.SlackAutoInvestigate || !settings.IsAdreEnabled() || settings.GetAdreURL() == "" {
		return
	}
	if len(settings.Adre.SlackAutoInvestigateChannels) == 0 {
		s.mu.Lock()
		if !s.warnedNoOutput {
			s.warnedNoOutput = true
			s.l.Warn("auto-investigate is enabled but no Slack output channels are configured; investigations will run with no Slack notification")
		}
		s.mu.Unlock()
	}
	for _, a := range alerts {
		switch {
		case a.resolved():
			s.rearm(a.Fingerprint)
		case a.firing():
			s.processFiring(ctx, settings, a)
		}
	}
}

// rearm clears the in-memory episode marker so the next firing of this alert is investigated afresh.
func (s *Service) rearm(fingerprint string) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return
	}
	s.mu.Lock()
	delete(s.episodes, fingerprint)
	s.mu.Unlock()
}

func (s *Service) processFiring(ctx context.Context, settings *models.Settings, a Alert) {
	fp := strings.TrimSpace(a.Fingerprint)
	if fp == "" {
		return
	}
	if !passesSelection(settings, a) {
		return
	}

	s.mu.Lock()
	if _, seen := s.episodes[fp]; seen {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	// Global, HA-safe, restart-safe hourly cap: count investigations actually created in the window
	// rather than in-memory reservations (so duplicates/failures never burn the budget). This is a
	// soft cap — concurrent claimers can each read the count before the other commits, so it may be
	// exceeded by up to (concurrent processors − 1) within a window.
	if cap := settings.Adre.AutoInvestigateHourlyCap; cap > 0 {
		n, err := models.CountAutoInvestigationsSince(s.db, time.Now().Add(-time.Hour))
		if err != nil {
			s.l.Errorf("CountAutoInvestigationsSince: %v", err)
			return
		}
		if n >= cap {
			s.l.Warnf("auto-investigate hourly cap (%d) reached; skipping alert %s", cap, fp)
			return
		}
	}

	inv, claimed, err := models.ClaimInvestigationForAlert(s.db, buildAlertInvestigation(a))
	if err != nil {
		s.l.Errorf("ClaimInvestigationForAlert(%s): %v", fp, err)
		return // not marked seen → retried next cycle
	}
	if !claimed {
		// An active investigation already exists for this episode; mark seen and don't run.
		s.markEpisode(fp)
		return
	}

	if err := s.runner.StartRun(ctx, inv.ID); err != nil {
		// The claimed investigation never started; delete the empty record so it doesn't block the
		// active-dedup index, and don't mark the episode so the next cycle retries.
		s.l.Warnf("StartRun(%s) for alert %s: %v; deleting orphan and retrying next cycle", inv.ID, fp, err)
		if dErr := models.DeleteInvestigation(s.db, inv.ID); dErr != nil {
			s.l.Errorf("DeleteInvestigation(%s) after StartRun failure: %v", inv.ID, dErr)
		}
		return
	}

	s.markEpisode(fp)
	s.notifier.PostAutoInvestigateStarted(ctx, settings.Adre.SlackAutoInvestigateChannels, inv)
	s.l.Infof("auto-investigation started: investigation=%s alert=%s", inv.ID, fp)
}

// markEpisode records that an alert was handled this firing episode, bounding the map size.
func (s *Service) markEpisode(fp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.episodes) >= maxEpisodes {
		for k := range s.episodes { // randomized iteration ⇒ arbitrary eviction (only costs a redundant DB claim)
			delete(s.episodes, k)
			break
		}
	}
	s.episodes[fp] = struct{}{}
}

// passesSelection applies the configured min-severity floor and label matchers.
func passesSelection(settings *models.Settings, a Alert) bool {
	if !passesSeverityFloor(settings.Adre.AutoInvestigateMinSeverity, a.Labels["severity"]) {
		return false
	}
	return passesLabelMatchers(settings.Adre.AutoInvestigateLabelMatchers, a.Labels)
}

// passesSeverityFloor enforces the min-severity floor only for alerts that declare a *known*
// severity. An alert with no severity label (or an unrankable value) is NOT silently dropped — it
// passes the floor and can still be filtered by the label matchers. This avoids excluding the many
// PMM/Grafana rules that don't emit a standard severity.
func passesSeverityFloor(floor, alertSeverity string) bool {
	floor = strings.ToLower(strings.TrimSpace(floor))
	if floor == "" {
		return true
	}
	alertRank, known := severityRank[strings.ToLower(strings.TrimSpace(alertSeverity))]
	if !known {
		return true
	}
	return alertRank >= severityRank[floor]
}

func passesLabelMatchers(matchers []string, labels map[string]string) bool {
	for _, m := range matchers {
		k, v, ok := strings.Cut(strings.TrimSpace(m), "=")
		if !ok {
			continue
		}
		if labels[strings.TrimSpace(k)] != strings.TrimSpace(v) {
			return false
		}
	}
	return true
}

// alertSnapshotEntry mirrors the alert shape Holmes/investigation context expects in config.alert_snapshot.
type alertSnapshotEntry struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Fingerprint string            `json:"fingerprint"`
	Status      string            `json:"status"`
}

// buildAlertInvestigation constructs an "open" investigation record for a firing alert, ready to be
// claimed and run. The alert snapshot is stored as a JSON string in config (matching the manual
// "Investigate alert" path) so the run pipeline builds the same Holmes context.
func buildAlertInvestigation(a Alert) *models.Investigation {
	name := strings.TrimSpace(a.Labels["alertname"])
	if name == "" {
		name = "alert"
	}
	cfg := map[string]any{}
	if v := a.Labels["node_name"]; v != "" {
		cfg["node_name"] = v
	}
	if v := a.Labels["service_name"]; v != "" {
		cfg["service_name"] = v
	}
	if v := a.Labels["cluster"]; v != "" {
		cfg["cluster_name"] = v
	}
	snapshot, _ := json.Marshal([]alertSnapshotEntry{{
		Labels:      a.Labels,
		Annotations: a.Annotations,
		Fingerprint: a.Fingerprint,
		Status:      a.Status,
	}})
	cfg["alert_snapshot"] = string(snapshot)

	// Carry the Slack thread origin (scrape path) so started/report notices post as in-thread replies.
	if a.Slack != nil && a.Slack.Channel != "" && a.Slack.ThreadTS != "" {
		cfg["slack_team_id"] = a.Slack.TeamID
		cfg["slack_channel"] = a.Slack.Channel
		cfg["slack_thread_ts"] = a.Slack.ThreadTS
	}

	config := []byte("{}")
	if b, err := json.Marshal(cfg); err == nil {
		config = b
	}

	now := time.Now().UTC()
	return &models.Investigation{
		ID:               models.NewInvestigationID(),
		Title:            "Alert: " + name,
		Status:           "open",
		Severity:         a.Labels["severity"],
		TimeFrom:         now.Add(-time.Hour),
		TimeTo:           now,
		SourceType:       "alert",
		SourceRef:        a.Fingerprint,
		AlertFingerprint: a.Fingerprint,
		CreatedBy:        models.AutoInvestigateCreatedBy,
		Config:           config,
	}
}
