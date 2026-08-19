// Copyright (C) 2026 Percona LLC
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

package pom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

// probeSource reads on-host facts from SEP's pom_discovery app.
//
// The facts it contributes are the ones nothing on this side can answer: the command
// line a mongod was started with, the config file it read, and the *installed* binary
// version as against the *running* server the metrics report. Their divergence is the
// upgraded-but-not-restarted case, and no metric anywhere carries it.
//
// It pulls; it does not drive. The app sweeps its estate over Nomad on its own
// schedule and serves whatever the last completed sweep stored, so this is a database
// read on the other side rather than a fan-out waiting to happen. That is what keeps
// a POM run in the tenth-of-a-second range with a source attached whose own work
// takes tens of seconds.
type probeSource struct {
	// sepURL is where SEP is, e.g. http://127.0.0.1:8000 -- not where the app hangs
	// off it. Which app to ask is this source's business, not the operator's, and the
	// next source will point at a different app on the same SEP. Empty means the
	// source is not configured, which is a normal state and reported as disabled
	// rather than failed.
	sepURL string
	token  string
	client *http.Client
	l      *logrus.Entry
}

// probeAppPath is where SEP mounts the discovery app, under the `/api/apps/<module>`
// convention every SEP app follows. Paths passed around below are relative to it.
const probeAppPath = "api/apps/pom_discovery"

func (probeSource) key() string { return sourceProbe }

// probeFactsResponse is the app's GET /facts contract.
type probeFactsResponse struct {
	RunID      string  `json:"run_id"`
	Status     string  `json:"status"`
	ObservedAt *string `json:"observed_at"`
	AgeSeconds float64 `json:"age_seconds"`
	Stale      bool    `json:"stale"`
	Error      string  `json:"error"`
	Facts      []struct {
		ServiceID  string    `json:"service_id"`
		Field      string    `json:"field"`
		Value      any       `json:"value"`
		ObservedAt time.Time `json:"observed_at"`
	} `json:"facts"`
}

// The sweep statuses the app reports. Its own conclusion about the work, which is a
// different question from whether this source could read it.
const (
	sweepSuccess = "success"
	sweepPartial = "partial"
	sweepFailed  = "failed"
)

// probeRequestTimeout bounds the pull.
//
// Generous for a database read on the other side and far short of anything that would
// make a POM run feel slow. The app never probes on this path, so a request that takes
// longer than this is a sign the app is unwell rather than busy -- and the run reports
// the source as failed and carries on with the sources that answered.
const probeRequestTimeout = 10 * time.Second

func (s probeSource) collect(ctx context.Context, services []*models.Service) SourceResult {
	result := SourceResult{Source: sourceProbe, Status: SourceDisabled}
	if s.sepURL == "" {
		result.Detail = map[string]any{"reason": "no SEP endpoint configured"}
		return result
	}

	known := make(map[string]bool, len(services))
	for _, service := range services {
		known[service.ServiceID] = true
	}

	answer, err := s.fetch(ctx)
	if err != nil {
		s.l.Warnf("probe source: %s", err)
		return SourceResult{
			Source: sourceProbe,
			Status: SourceFailed,
			Errors: []RunError{{Scope: "source", Code: "probe_fetch_failed", Message: err.Error()}},
			Detail: map[string]any{"endpoint": s.endpoint("facts")},
		}
	}

	covered := make(map[string]bool)
	unknown := 0
	for _, fact := range answer.Facts {
		// A service the app knows and this PMM does not is not an error: the app reads
		// its own inventory, which can be a moment ahead or behind. Counting them is
		// enough -- merging them would put facts in the document for services it has no
		// row for.
		if !known[fact.ServiceID] {
			unknown++
			continue
		}
		observedAt := fact.ObservedAt
		covered[fact.ServiceID] = true
		result.Facts = append(result.Facts, Fact{
			Service:    fact.ServiceID,
			Field:      fact.Field,
			Value:      fact.Value,
			Source:     sourceProbe,
			ObservedAt: &observedAt,
		})
	}

	// The app's own conclusion is part of the answer, not a detail. A reachable app
	// whose last sweep failed has told us something went wrong on the probe side, and
	// reporting that as "ok, 0 facts" reads as "there is nothing to probe here" --
	// which is the opposite of what happened.
	switch strings.ToLower(answer.Status) {
	case "":
		// No sweep has ever completed. The app is installed and has not run yet, which
		// is a normal state and not anybody's failure.
		result.Status = SourceOK
	case sweepFailed:
		result.Status = SourceFailed
		result.Errors = append(result.Errors, RunError{
			Scope:   "source",
			Code:    "probe_sweep_failed",
			Message: sweepFailureMessage(answer),
		})
	case sweepPartial:
		result.Status = SourcePartial
	case sweepSuccess:
		// The sweep is happy with itself; whether it covered this PMM's estate is a
		// separate question, and one only this side can answer.
		if len(covered) < len(services) {
			result.Status = SourcePartial
		} else {
			result.Status = SourceOK
		}
	default:
		// A status this side does not know. Not worth claiming ok over -- the app has
		// concluded something we cannot interpret.
		s.l.Warnf("probe source: unrecognised sweep status %q", answer.Status)
		result.Status = SourcePartial
	}

	result.Detail = map[string]any{
		"endpoint":         s.endpoint("facts"),
		"sweep_error":      answer.Error,
		"run_id":           answer.RunID,
		"run_status":       answer.Status,
		"age_seconds":      int(answer.AgeSeconds),
		"stale":            answer.Stale,
		"services":         len(services),
		"services_covered": len(covered),
		"services_unknown": unknown,
		"facts":            len(result.Facts),
	}
	s.l.Infof("probe source: %d/%d service(s) covered by run %s (%s, %ds old)",
		len(covered), len(services), answer.RunID, answer.Status, int(answer.AgeSeconds))
	return result
}

// sweepFailureMessage explains a failed sweep in the run's own receipt.
//
// The app's error is included when it sent one, so a reader is told the cause here
// rather than having to go and ask the other service what went wrong.
func sweepFailureMessage(answer *probeFactsResponse) string {
	if answer.Error != "" {
		return fmt.Sprintf("sweep %s failed: %s", answer.RunID, answer.Error)
	}
	return fmt.Sprintf("sweep %s failed with no reason recorded", answer.RunID)
}

// endpoint builds an absolute URL for a path relative to the discovery app.
func (s probeSource) endpoint(path string) string {
	return strings.TrimSuffix(s.sepURL, "/") + "/" + probeAppPath + "/" + strings.TrimPrefix(path, "/")
}

// fetch reads the app's stored facts.
func (s probeSource) fetch(ctx context.Context) (*probeFactsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, probeRequestTimeout)
	defer cancel()

	url := s.endpoint("facts")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build the request: %w", err)
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	answer := &probeFactsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(answer); err != nil { //nolint:noinlineerr
		return nil, fmt.Errorf("GET %s: failed to decode the response: %w", url, err)
	}
	return answer, nil
}
