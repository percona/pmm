// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package annotator turns Grafana unified-alerting notifications into Grafana annotations.
// It receives the standard Alertmanager webhook payload and, for each alert, creates a
// service/node-tagged annotation on firing and closes it into a region on resolve. The tags
// match the built-in "PMM Annotations" dashboard layer, so the annotation shows across all
// panels of the affected service.
package annotator

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const fingerprintTagPrefix = "pmm_alert_fingerprint:"

// grafanaClient is the subset of the Grafana client used by the annotator.
type grafanaClient interface {
	CreateAlertAnnotation(ctx context.Context, tags []string, start time.Time, text string) (int, error)
	SetAlertAnnotationEnd(ctx context.Context, id int, end time.Time) error
	FindAlertAnnotationID(ctx context.Context, tags []string, from, to time.Time) (int, error)
}

// Service handles alert notification webhooks and creates Grafana annotations.
type Service struct {
	grafanaClient grafanaClient
	l             *logrus.Entry
}

// New creates a new annotator Service.
func New(grafanaClient grafanaClient) *Service {
	return &Service{
		grafanaClient: grafanaClient,
		l:             logrus.WithField("component", "alerting/annotator"),
	}
}

// webhookPayload is the subset of the Alertmanager/Grafana webhook payload we consume.
type webhookPayload struct {
	Alerts []webhookAlert `json:"alerts"`
}

type webhookAlert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Fingerprint string            `json:"fingerprint"`
}

// ServeHTTP receives a Grafana webhook notification and annotates the affected panels. A
// well-formed payload always returns 200 (per-alert errors are logged); a malformed body, 400.
func (s *Service) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload webhookPayload
	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		s.l.Warnf("Failed to decode webhook payload: %s", err)
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}

	for _, alert := range payload.Alerts {
		err := s.processAlert(req.Context(), alert)
		if err != nil {
			s.l.Errorf("Failed to process alert %q: %s", alert.Fingerprint, err)
		}
	}

	rw.WriteHeader(http.StatusOK)
}

func (s *Service) processAlert(ctx context.Context, a webhookAlert) error {
	fpTag := fingerprintTagPrefix + a.Fingerprint
	tags := buildTags(a, fpTag)
	text := annotationText(a)

	// Generous window around this firing episode to correlate the firing and resolved
	// notifications via the per-fingerprint tag.
	from := a.StartsAt.Add(-time.Minute)
	to := time.Now().Add(time.Minute)

	id, err := s.grafanaClient.FindAlertAnnotationID(ctx, []string{fpTag}, from, to)
	if err != nil {
		return err
	}

	if a.Status == "resolved" {
		if id == 0 {
			// Firing notification was missed; create the start point so we can close it.
			id, err = s.grafanaClient.CreateAlertAnnotation(ctx, tags, a.StartsAt, text)
			if err != nil {
				return err
			}
		}
		return s.grafanaClient.SetAlertAnnotationEnd(ctx, id, a.EndsAt)
	}

	// firing
	if id != 0 {
		// Already annotated this firing episode (Grafana resends notifications periodically).
		return nil
	}
	_, err = s.grafanaClient.CreateAlertAnnotation(ctx, tags, a.StartsAt, text)
	return err
}

// buildTags derives annotation tags. service_name/node_name are added as bare tags so the
// built-in "PMM Annotations" layer scopes the annotation to that service; without them the
// global "pmm_annotation" tag is used.
func buildTags(a webhookAlert, fpTag string) []string {
	tags := []string{"pmm_alert", fpTag}

	scoped := false
	if sn := a.Labels["service_name"]; sn != "" {
		tags = append(tags, sn)
		scoped = true
	}
	if nn := a.Labels["node_name"]; nn != "" {
		tags = append(tags, nn)
		scoped = true
	}
	if !scoped {
		tags = append(tags, "pmm_annotation")
	}

	if an := a.Labels["alertname"]; an != "" {
		tags = append(tags, "alertname:"+an)
	}
	if sev := a.Labels["severity"]; sev != "" {
		tags = append(tags, "severity:"+sev)
	}
	return tags
}

func annotationText(a webhookAlert) string {
	if s := a.Annotations["summary"]; s != "" {
		return s
	}
	if d := a.Annotations["description"]; d != "" {
		return d
	}
	if an := a.Labels["alertname"]; an != "" {
		return an
	}
	return "Alert"
}
