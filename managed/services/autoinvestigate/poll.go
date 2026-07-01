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

package autoinvestigate

import (
	"context"
	"encoding/json"
	"time"
)

// defaultPollInterval is the reconciliation cadence when none is configured.
const defaultPollInterval = 90 * time.Second

// AlertFetcher returns the currently-firing alerts from Grafana's Alertmanager. Implemented by an
// adapter over the Grafana client (server-side, admin-authenticated).
type AlertFetcher interface {
	FetchFiringAlerts(ctx context.Context) ([]Alert, error)
}

// AlertFetcherFunc adapts a plain function to AlertFetcher (used for wiring).
type AlertFetcherFunc func(ctx context.Context) ([]Alert, error)

// FetchFiringAlerts implements AlertFetcher.
func (f AlertFetcherFunc) FetchFiringAlerts(ctx context.Context) ([]Alert, error) { return f(ctx) }

// RunPoll periodically reconciles the firing-alert set. It is the correctness safety-net behind the
// (instant) webhook: it drives auto-investigations for firing alerts the webhook missed and re-arms
// episodes for alerts that are no longer firing. It works without any Grafana provisioning. The loop
// processes only while isLeader() reports true (nil ⇒ always). It returns when ctx is cancelled.
func (s *Service) RunPoll(ctx context.Context, fetcher AlertFetcher, interval time.Duration, isLeader func() bool) {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if isLeader != nil && !isLeader() {
				continue
			}
			s.pollOnce(ctx, fetcher)
		}
	}
}

func (s *Service) pollOnce(ctx context.Context, fetcher AlertFetcher) {
	alerts, err := fetcher.FetchFiringAlerts(ctx)
	if err != nil {
		s.l.Warnf("reconciliation poll: fetch alerts: %v", err)
		return
	}
	// Re-arm episodes whose alert is no longer firing: any fingerprint in the episode map but absent
	// from the current firing set has resolved (covers RESOLVED webhooks that were missed).
	firing := make(map[string]struct{}, len(alerts))
	for _, a := range alerts {
		if a.firing() {
			firing[a.Fingerprint] = struct{}{}
		}
	}
	s.mu.Lock()
	for fp := range s.episodes {
		if _, ok := firing[fp]; !ok {
			delete(s.episodes, fp)
		}
	}
	s.mu.Unlock()

	s.ProcessAlerts(ctx, alerts)
}

// ParseAlertmanagerAlerts converts a Grafana Alertmanager v2 GET /alerts response (queried with
// active=true) into firing Alerts.
func ParseAlertmanagerAlerts(raw []byte) []Alert {
	if len(raw) == 0 {
		return nil
	}
	var items []struct {
		Fingerprint string            `json:"fingerprint"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		Status      struct {
			State string `json:"state"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	out := make([]Alert, 0, len(items))
	for _, it := range items {
		if it.Fingerprint == "" {
			continue
		}
		// The Alertmanager v2 query keeps silenced/inhibited at their defaults, so it also returns
		// "suppressed" alerts. Don't auto-investigate those — only genuinely active ones.
		if it.Status.State != "" && it.Status.State != "active" {
			continue
		}
		out = append(out, Alert{
			Fingerprint: it.Fingerprint,
			Status:      "firing",
			Labels:      it.Labels,
			Annotations: it.Annotations,
		})
	}
	return out
}

// ProcessWebhook parses a Grafana alerting webhook payload and processes its alerts. It is the
// instant-path entry point (the reconciliation poll is the safety-net), and lets callers depend on a
// raw-bytes interface rather than the Alert type.
func (s *Service) ProcessWebhook(ctx context.Context, raw []byte) {
	s.ProcessAlerts(ctx, ParseGrafanaWebhook(raw))
}

// ParseGrafanaWebhook converts a Grafana alerting webhook payload into Alerts (preserving each
// alert's firing/resolved status so resolved alerts re-arm their episode).
func ParseGrafanaWebhook(raw []byte) []Alert {
	if len(raw) == 0 {
		return nil
	}
	var payload struct {
		Alerts []struct {
			Status      string            `json:"status"`
			Fingerprint string            `json:"fingerprint"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	out := make([]Alert, 0, len(payload.Alerts))
	for _, a := range payload.Alerts {
		if a.Fingerprint == "" {
			continue
		}
		out = append(out, Alert{
			Fingerprint: a.Fingerprint,
			Status:      a.Status,
			Labels:      a.Labels,
			Annotations: a.Annotations,
		})
	}
	return out
}
