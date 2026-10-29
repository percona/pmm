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
	"maps"
	"slices"
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/percona/pmm/managed/services"
)

// registry keeps a snapshot of the current check results and exposes it as the insights metric.
type registry struct {
	rw sync.RWMutex
	// Results stored by check name.
	checkResults map[string][]services.CheckResult
	mInsights    *prom.GaugeVec
}

// newRegistry creates a new registry.
func newRegistry() *registry {
	return &registry{
		checkResults: make(map[string][]services.CheckResult),
		mInsights: prom.NewGaugeVec(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "check_insights",
			Help:      "Number of advisor insights per service type, service name, category, check name and severity",
		}, []string{"service_type", "service_name", "category", "check_name", "severity"}),
	}
}

// set adds check results.
func (r *registry) set(checkResults []services.CheckResult) {
	r.rw.Lock()
	defer r.rw.Unlock()

	for _, result := range checkResults {
		r.checkResults[result.CheckName] = append(r.checkResults[result.CheckName], result)
	}
}

// deleteByName removes results for specified checks.
func (r *registry) deleteByName(checkNames []string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	for _, name := range checkNames {
		delete(r.checkResults, name)
	}
}

// deleteByNameAndService removes the results the specified checks, or all checks
// when none are specified, produced for the specified services, leaving other
// services' results in place.
func (r *registry) deleteByNameAndService(checkNames, serviceIDs []string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	if len(checkNames) == 0 {
		checkNames = slices.Collect(maps.Keys(r.checkResults))
	}

	for _, name := range checkNames {
		results, ok := r.checkResults[name]
		if !ok {
			continue
		}

		kept := slices.DeleteFunc(slices.Clone(results), func(result services.CheckResult) bool {
			return slices.Contains(serviceIDs, result.Target.ServiceID)
		})
		if len(kept) == 0 {
			delete(r.checkResults, name)
			continue
		}
		r.checkResults[name] = kept
	}
}

// cleanup removes all check results from the registry.
func (r *registry) cleanup() {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.checkResults = make(map[string][]services.CheckResult)
}

// getCheckResults returns checks results for all services.
func (r *registry) getCheckResults() []services.CheckResult {
	r.rw.RLock()
	defer r.rw.RUnlock()

	var results []services.CheckResult
	for _, checkNameGroup := range r.checkResults {
		results = append(results, checkNameGroup...)
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
		r.mInsights.WithLabelValues(string(re.Target.ServiceType), re.Target.ServiceName, re.Category, re.CheckName, re.Result.Severity.String()).Inc()
	}
	r.mInsights.Collect(ch)
}
