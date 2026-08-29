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

package om

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

// probeSource reads on-host facts from SEP's om_inventory app.
//
// The facts it contributes are the ones nothing on this side can answer: the command
// line a mongod was started with, the config file it read, and the *installed* binary
// version as against the *running* server the metrics report. Their divergence is the
// upgraded-but-not-restarted case, and no metric anywhere carries it.
//
// It pulls; it does not drive. The app sweeps its estate over Nomad on its own
// schedule and serves whatever the last completed sweep stored, so this is a database
// read on the other side rather than a fan-out waiting to happen. That is what keeps
// an OM run in the tenth-of-a-second range with a source attached whose own work
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

// probeAppPath is where SEP mounts the inventory app, under the `/api/apps/<module>`
// convention every SEP app follows. Paths passed around below are relative to it.
const probeAppPath = "api/apps/om_inventory"

// probeServicesPath is the app's estate, flat. /hosts nests the same service rows
// under their host and carries host-level attributes besides, which this source has
// nothing to do with: the document it feeds is keyed by service.
const probeServicesPath = "services"

func (probeSource) key() string { return sourceProbe }

// probeService is one row of the app's GET /services contract.
//
// The app used to serve a flat fact list at GET /facts, assembled per sweep. It now
// keeps an estate -- a row per host and a row per service, upserted -- and the facts
// this source wants are the `observed` document on the service row. Reading the
// estate rather than the last sweep's output is what makes a service that has not
// been probed for a week still carry what it was running a week ago, with a timestamp
// saying so, instead of vanishing when a sweep misses it.
type probeService struct {
	ServiceID string         `json:"service_id"`
	NodeID    string         `json:"node_id"`
	Name      string         `json:"name"`
	Observed  map[string]any `json:"observed"`

	// LastSuccessAt is the age of everything in Observed. Null means this service has
	// never answered a probe, which is different from having answered nothing.
	LastSuccessAt *time.Time `json:"last_success_at"`
	// FailingSince is the *first* failure after the last success, so it says "failing
	// for three days" rather than "failed a minute ago".
	FailingSince        *time.Time `json:"failing_since"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastError           string     `json:"last_error"`
}

// observedCollectedAt is the key the app stamps on every document it stores. It is
// metadata about the document rather than a fact about the service, so it is used for
// the timestamp and never emitted as a field.
const observedCollectedAt = "collected_at"

// probeRequestTimeout bounds the pull.
//
// Generous for a database read on the other side and far short of anything that would
// make an OM run feel slow. The app never probes on this path, so a request that takes
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
			Detail: map[string]any{"endpoint": s.endpoint(probeServicesPath)},
		}
	}

	var (
		covered  = make(map[string]bool)
		unknown  int
		failing  int
		oldest   float64
		failures []string
	)
	now := time.Now()

	for _, service := range answer {
		// A service the app knows and this PMM does not is not an error: the app reads
		// its own inventory, which can be a moment ahead or behind. Counting them is
		// enough -- merging them would put facts in the document for services it has no
		// row for.
		if !known[service.ServiceID] {
			unknown++
			continue
		}

		if service.FailingSince != nil {
			failing++
			if len(failures) < probeFailuresReported {
				failures = append(failures, service.failureSummary())
			}
		}

		observedAt := service.observedAt()
		if observedAt == nil || len(service.Observed) == 0 {
			// Seen but never successfully probed. Not covered, and not a fact.
			continue
		}
		covered[service.ServiceID] = true
		if age := now.Sub(*observedAt).Seconds(); age > oldest {
			oldest = age
		}

		for field, value := range service.Observed {
			if field == observedCollectedAt {
				continue
			}
			at := *observedAt
			result.Facts = append(result.Facts, Fact{
				Service:    service.ServiceID,
				Field:      field,
				Value:      value,
				Source:     sourceProbe,
				ObservedAt: &at,
			})
		}
	}

	// The estate's own condition is part of the answer, not a detail. Every service
	// failing its probe means something is wrong on the probe side, and reporting that
	// as "ok, 0 facts" reads as "there is nothing to probe here" -- the opposite of
	// what happened.
	switch {
	case len(services) == 0:
		// Nothing to cover. Not this source's failure, and not a partial answer.
		result.Status = SourceOK
	case len(covered) == 0 && failing > 0:
		result.Status = SourceFailed
		result.Errors = append(result.Errors, RunError{
			Scope:   "source",
			Code:    "probe_all_failing",
			Message: "no service answered its probe: " + strings.Join(failures, "; "),
		})
	case len(covered) < len(services):
		// The usual steady state, and not an alarm: a service whose node runs no
		// healthy executor is a fact about the estate rather than a failure here.
		result.Status = SourcePartial
	default:
		result.Status = SourceOK
	}

	result.Detail = map[string]any{
		"endpoint":           s.endpoint(probeServicesPath),
		"services":           len(services),
		"services_covered":   len(covered),
		"services_unknown":   unknown,
		"services_failing":   failing,
		"oldest_age_seconds": int(oldest),
		"facts":              len(result.Facts),
	}
	s.l.Infof("probe source: %d/%d service(s) covered, %d failing, oldest %ds",
		len(covered), len(services), failing, int(oldest))
	return result
}

// probeFailuresReported bounds how many failures reach the run receipt.
//
// A whole estate failing the same way produces one message per service, and the
// receipt is meant to be read. The count is reported in full either way.
const probeFailuresReported = 3

// observedAt is when this service's document was collected.
//
// The last_success_at column is authoritative because the app maintains it; the document's own
// collected_at is the fallback for a row written before that column carried a value.
func (p probeService) observedAt() *time.Time {
	if p.LastSuccessAt != nil {
		return p.LastSuccessAt
	}
	stamp, ok := p.Observed[observedCollectedAt].(string)
	if !ok {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return nil
	}
	return &parsed
}

// failureSummary explains one failing service in the run's own receipt, so a reader is
// told the cause here rather than having to go and ask the other service.
func (p probeService) failureSummary() string {
	name := p.Name
	if name == "" {
		name = p.ServiceID
	}
	if p.LastError == "" {
		return fmt.Sprintf("%s failing since %s", name, p.FailingSince.Format(time.RFC3339))
	}
	return fmt.Sprintf("%s: %s", name, p.LastError)
}

// endpoint builds an absolute URL for a path relative to the inventory app.
func (s probeSource) endpoint(path string) string {
	return strings.TrimSuffix(s.sepURL, "/") + "/" + probeAppPath + "/" + strings.TrimPrefix(path, "/")
}

// fetch reads the app's service estate.
func (s probeSource) fetch(ctx context.Context) ([]probeService, error) {
	ctx, cancel := context.WithTimeout(ctx, probeRequestTimeout)
	defer cancel()

	url := s.endpoint(probeServicesPath)
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

	answer := []probeService{}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil { //nolint:noinlineerr
		return nil, fmt.Errorf("GET %s: failed to decode the response: %w", url, err)
	}
	return answer, nil
}
