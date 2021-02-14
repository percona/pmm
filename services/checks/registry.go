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
)

// registry stores alerts and delay information by IDs.
type registry struct {
	rw           sync.RWMutex
	checkResults []sttCheckResult
	alertTTL     time.Duration
	nowF         func() time.Time // for tests
}

// newRegistry creates a new registry.
func newRegistry(alertTTL time.Duration) *registry {
	return &registry{
		alertTTL: alertTTL,
		nowF:     time.Now,
	}
}

// set replaces stored checkResults with a copy of given ones.
func (r *registry) set(checkResults []sttCheckResult) {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.checkResults = make([]sttCheckResult, len(checkResults))
	copy(r.checkResults, checkResults)
}

// collect returns a slice of alerts created from the stored check results.
func (r *registry) collect() ammodels.PostableAlerts {
	r.rw.RLock()
	defer r.rw.RUnlock()

	alerts := make(ammodels.PostableAlerts, len(r.checkResults))
	for i, checkResult := range r.checkResults {
		alerts[i] = r.createAlert(checkResult.checkName, &checkResult.target, &checkResult.result, r.alertTTL)
	}
	return alerts
}

func (r *registry) getCheckResults() []sttCheckResult {
	r.rw.RLock()
	defer r.rw.RUnlock()

	checkResults := make([]sttCheckResult, 0, len(r.checkResults))
	checkResults = append(checkResults, r.checkResults...)

	return checkResults
}

func (r *registry) createAlert(name string, target *target, result *check.Result, alertTTL time.Duration) *ammodels.PostableAlert {
	labels := make(map[string]string, len(target.labels)+len(result.Labels)+4)
	annotations := make(map[string]string, 2)
	for k, v := range target.labels {
		labels[k] = v
	}
	for k, v := range result.Labels {
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
func makeID(target *target, result *check.Result) string {
	s := sha1.New() //nolint:gosec
	fmt.Fprintf(s, "%s\n", target.agentID)
	fmt.Fprintf(s, "%s\n", target.serviceID)
	fmt.Fprintf(s, "%s\n", result.Summary)
	fmt.Fprintf(s, "%s\n", result.Description)
	fmt.Fprintf(s, "%s\n", result.ReadMoreURL)
	fmt.Fprintf(s, "%v\n", result.Severity)
	return alertsPrefix + hex.EncodeToString(s.Sum(nil))
}
