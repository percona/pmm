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

package checks

import (
	"sync"

	"github.com/percona/saas/pkg/check"
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/percona/pmm/managed/services"
)

// registry stores alerts and delay information by IDs.
type registry struct {
	rw sync.RWMutex
	// Results stored grouped by interval and by check name. It allows us to remove results for specific group.
	checkResults map[check.Interval]map[string][]services.CheckResult
	mInsights    *prom.GaugeVec
}

// newRegistry creates a new registry.
func newRegistry() *registry {
	return &registry{
		checkResults: make(map[check.Interval]map[string][]services.CheckResult),
		mInsights: prom.NewGaugeVec(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "check_insights",
			Help:      "Number of advisor insights per service type, advisor and check name",
		}, []string{"service_type", "advisor", "check_name"}),
	}
}

// set adds check results.
func (r *registry) set(checkResults []services.CheckResult) {
	r.rw.Lock()
	defer r.rw.Unlock()

	for _, result := range checkResults {
		// Empty interval means standard.
		if result.Interval == "" {
			result.Interval = check.Standard
		}

		if _, ok := r.checkResults[result.Interval]; !ok {
			r.checkResults[result.Interval] = make(map[string][]services.CheckResult)
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

// cleanup removes all advisors results form registry.
func (r *registry) cleanup() {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.checkResults = make(map[check.Interval]map[string][]services.CheckResult)
}

// getCheckResults returns checks results for the given service. If serviceID is empty it returns results for all services.
func (r *registry) getCheckResults(serviceID string) []services.CheckResult {
	r.rw.RLock()
	defer r.rw.RUnlock()

	var results []services.CheckResult
	for _, intervalGroup := range r.checkResults {
		for _, checkNameGroup := range intervalGroup {
			for _, checkResult := range checkNameGroup {
				if serviceID == "" || checkResult.Target.ServiceID == serviceID {
					results = append(results, checkResult)
				}
			}
		}
	}

	return results
}

// Describe implements prom.Collector.
func (r *registry) Describe(ch chan<- *prom.Desc) {
	r.mInsights.Describe(ch)
}

// Collect implements prom.Collector.
func (r *registry) Collect(ch chan<- prom.Metric) {
	r.mInsights.Reset()
	res := r.getCheckResults("")
	for _, re := range res {
		r.mInsights.WithLabelValues(string(re.Target.ServiceType), re.AdvisorName, re.CheckName).Inc()
	}
	r.mInsights.Collect(ch)
}
