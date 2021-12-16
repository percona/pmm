// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package checks

import (
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/prometheus/common/model"

	"github.com/percona/pmm-managed/services"
)

// registry stores alerts and delay information by IDs.
type registry struct {
	rw sync.RWMutex
	// Results stored grouped by interval and by check name. It allows us to remove results for specific group.
	checkResults map[check.Interval]map[string][]services.STTCheckResult

	alertTTL time.Duration
	nowF     func() time.Time // for tests
}

// newRegistry creates a new registry.
func newRegistry(alertTTL time.Duration) *registry {
	return &registry{
		checkResults: make(map[check.Interval]map[string][]services.STTCheckResult),
		alertTTL:     alertTTL,
		nowF:         time.Now,
	}
}

// set adds check results.
func (r *registry) set(checkResults []services.STTCheckResult) {
	r.rw.Lock()
	defer r.rw.Unlock()

	for _, result := range checkResults {
		// Empty interval means standard.
		if result.Interval == "" {
			result.Interval = check.Standard
		}

		if _, ok := r.checkResults[result.Interval]; !ok {
			r.checkResults[result.Interval] = make(map[string][]services.STTCheckResult)
		}

		r.checkResults[result.Interval][result.CheckName] = append(r.checkResults[result.Interval][result.CheckName], result)
	}
}

// deleteByName removes results for specified checks.
func (r *registry) deleteByName(checkNames []string) {
	r.rw.Lock()
	defer r.rw.Unlock()
	for _, intervalGroup := range r.checkResults {
		for _, name := range checkNames {
			delete(intervalGroup, name)
		}
	}
}

// deleteByInterval removes results for specified interval.
func (r *registry) deleteByInterval(interval check.Interval) {
	r.rw.Lock()
	defer r.rw.Unlock()

	delete(r.checkResults, interval)
}

// cleanup removes all stt results form registry.
func (r *registry) cleanup() {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.checkResults = make(map[check.Interval]map[string][]services.STTCheckResult)
}

// collect returns a slice of alerts created from the stored check results.
func (r *registry) collect() ammodels.PostableAlerts {
	r.rw.RLock()
	defer r.rw.RUnlock()

	var alerts ammodels.PostableAlerts
	for _, intervalGroup := range r.checkResults {
		for _, checkNameGroup := range intervalGroup {
			for _, checkResult := range checkNameGroup {
				alerts = append(alerts, r.createAlert(checkResult.CheckName, &checkResult.Target, &checkResult.Result, r.alertTTL))
			}
		}
	}
	return alerts
}

func (r *registry) getCheckResults() []services.STTCheckResult {
	r.rw.RLock()
	defer r.rw.RUnlock()

	var results []services.STTCheckResult
	for _, intervalGroup := range r.checkResults {
		for _, checkNameGroup := range intervalGroup {
			results = append(results, checkNameGroup...)
		}
	}

	return results
}

func (r *registry) createAlert(name string, target *services.Target, result *check.Result, alertTTL time.Duration) *ammodels.PostableAlert {
	labels := make(map[string]string, len(target.Labels)+len(result.Labels)+4)
	annotations := make(map[string]string, 2)
	for k, v := range result.Labels {
		labels[k] = v
	}
	for k, v := range target.Labels {
		labels[k] = v
	}

	labels[model.AlertNameLabel] = name
	labels["severity"] = result.Severity.String()
	labels["stt_check"] = "1"
	labels["alert_id"] = makeID(target, result)

	annotations["summary"] = result.Summary
	annotations["description"] = result.Description
	annotations["read_more_url"] = result.ReadMoreURL

	endsAt := r.nowF().Add(alertTTL).UTC().Round(0) // strip a monotonic clock reading
	return &ammodels.PostableAlert{
		Alert: ammodels.Alert{
			// GeneratorURL: "TODO",
			Labels: labels,
		},
		EndsAt:      strfmt.DateTime(endsAt),
		Annotations: annotations,
	}
}

// makeID creates an ID for STT check alert.
func makeID(target *services.Target, result *check.Result) string {
	s := sha1.New() //nolint:gosec
	fmt.Fprintf(s, "%s\n", target.AgentID)
	fmt.Fprintf(s, "%s\n", target.ServiceID)
	fmt.Fprintf(s, "%s\n", result.Summary)
	fmt.Fprintf(s, "%s\n", result.Description)
	fmt.Fprintf(s, "%s\n", result.ReadMoreURL)
	fmt.Fprintf(s, "%v\n", result.Severity)
	return alertsPrefix + hex.EncodeToString(s.Sum(nil))
}
