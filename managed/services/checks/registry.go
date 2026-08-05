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

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/services"
)

// registry keeps a snapshot of the current check results and exposes it as the insights metric.
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
			Help:      "Number of advisor insights per service type, service name, advisor, check name and severity",
		}, []string{"service_type", "service_name", "advisor", "check_name", "severity"}),
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

// deleteByNameAndService removes results for the specified checks, but only those
// produced for the specified services, leaving other services' results in place.
func (r *registry) deleteByNameAndService(checkNames, serviceIDs []string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	wanted := make(map[string]struct{}, len(serviceIDs))
	for _, id := range serviceIDs {
		wanted[id] = struct{}{}
	}

	for _, intervalGroup := range r.checkResults {
		for _, name := range checkNames {
			results, ok := intervalGroup[name]
			if !ok {
				continue
			}

			kept := make([]services.CheckResult, 0, len(results))
			for _, result := range results {
				_, drop := wanted[result.Target.ServiceID]
				if !drop {
					kept = append(kept, result)
				}
			}

			if len(kept) == 0 {
				delete(intervalGroup, name)
				continue
			}
			intervalGroup[name] = kept
		}
	}
}

// deleteByInterval removes results for specified interval.
func (r *registry) deleteByInterval(interval check.Interval) {
	r.rw.Lock()
	defer r.rw.Unlock()

	delete(r.checkResults, interval)
}

// cleanup removes all check results from the registry.
func (r *registry) cleanup() {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.checkResults = make(map[check.Interval]map[string][]services.CheckResult)
}

// getCheckResults returns checks results for all services.
func (r *registry) getCheckResults() []services.CheckResult {
	r.rw.RLock()
	defer r.rw.RUnlock()

	var results []services.CheckResult
	for _, intervalGroup := range r.checkResults {
		for _, checkNameGroup := range intervalGroup {
			results = append(results, checkNameGroup...)
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
	res := r.getCheckResults()
	for _, re := range res {
		r.mInsights.WithLabelValues(string(re.Target.ServiceType), re.Target.ServiceName, re.Subcategory, re.CheckName, re.Result.Severity.String()).Inc()
	}
	r.mInsights.Collect(ch)
}
