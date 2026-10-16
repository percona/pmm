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
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

// probeSource reads on-host facts, such as the installed binary version, from PMM Extensions' om_inventory app.
// It reads what the app's last sweep stored and never triggers a probe, so a run stays fast.
type probeSource struct {
	app extensionsApp
	l   *logrus.Entry
}

const probeAppModule = "om_inventory"

const probeServicesPath = "services"

func (probeSource) key() string { return sourceProbe }

type probeService struct {
	ServiceID string         `json:"service_id"`
	NodeID    string         `json:"node_id"`
	Name      string         `json:"name"`
	Observed  map[string]any `json:"observed"`

	// LastSuccessAt dates everything in Observed; nil means the service has never answered a probe.
	LastSuccessAt *time.Time `json:"last_success_at"`
	// FailingSince is the first failure after the last success, not the latest one.
	FailingSince        *time.Time `json:"failing_since"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastError           string     `json:"last_error"`
}

const observedCollectedAt = "collected_at"

const probeRequestTimeout = 10 * time.Second

func (s probeSource) collect(ctx context.Context, services []*models.Service) SourceResult {
	result := SourceResult{Source: sourceProbe, Status: SourceDisabled}
	if s.app.client == nil {
		result.Detail = map[string]any{"reason": "no PMM Extensions endpoint configured"}
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
			Detail: map[string]any{"endpoint": s.app.endpoint(probeServicesPath)},
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
		// Not an error: the app reads its own inventory, which can be a moment ahead of or behind this one.
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

	// Every service failing its probe is a source failure: "ok, 0 facts" would read as nothing to probe.
	switch {
	case len(services) == 0:
		result.Status = SourceOK
	case len(covered) == 0 && failing > 0:
		result.Status = SourceFailed
		result.Errors = append(result.Errors, RunError{
			Scope:   "source",
			Code:    "probe_all_failing",
			Message: "no service answered its probe: " + strings.Join(failures, "; "),
		})
	case len(covered) < len(services):
		// The usual steady state: a service on a node with no healthy executor is not this source's failure.
		result.Status = SourcePartial
	default:
		result.Status = SourceOK
	}

	result.Detail = map[string]any{
		"endpoint":           s.app.endpoint(probeServicesPath),
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

// probeFailuresReported caps failure messages in the run receipt; the failing count is reported in full.
const probeFailuresReported = 3

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

func (s probeSource) fetch(ctx context.Context) ([]probeService, error) {
	ctx, cancel := context.WithTimeout(ctx, probeRequestTimeout)
	defer cancel()

	return fetchAllPages(func(offset, limit int) (extensionsPage[probeService], error) {
		query := url.Values{
			"offset": {strconv.Itoa(offset)},
			"limit":  {strconv.Itoa(limit)},
		}
		endpoint := s.app.endpoint(probeServicesPath) + "?" + query.Encode()
		req, err := s.app.request(ctx, http.MethodGet, probeServicesPath, query, nil, false)
		if err != nil {
			return extensionsPage[probeService]{}, fmt.Errorf("failed to build the request: %w", err)
		}

		resp, err := s.app.client.http.Do(req)
		if err != nil {
			// Returned bare: Do fails with a *url.Error, whose message already names
			// the verb and the full URL, so a prefix here prints both of them twice.
			return extensionsPage[probeService]{}, err //nolint:wrapcheck
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			return extensionsPage[probeService]{}, fmt.Errorf("GET %s: unexpected status %s", endpoint, resp.Status)
		}

		page := extensionsPage[probeService]{}
		err = json.NewDecoder(resp.Body).Decode(&page)
		if err != nil {
			return extensionsPage[probeService]{}, fmt.Errorf("GET %s: failed to decode the response: %w", endpoint, err)
		}
		return page, nil
	})
}
