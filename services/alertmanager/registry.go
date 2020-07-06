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

package alertmanager

import (
	"strings"
	"sync"
	"time"

	"github.com/percona/pmm/api/alertmanager/ammodels"
)

// for tests
var now = time.Now

// Registry stores alerts and delay information by IDs.
type Registry struct {
	rw     sync.RWMutex
	alerts map[string]*ammodels.PostableAlert
	times  map[string]time.Time
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		alerts: make(map[string]*ammodels.PostableAlert),
		times:  make(map[string]time.Time),
	}
}

// CreateAlert creates alert from given AlertParams and adds or replaces alert with given ID in registry.
// If that ID wasn't present before, alert is added in the pending state. It we be transitioned to the firing
// state after delayFor interval. This is similar to `for` field of Prometheus alerting rule:
// https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
func (r *Registry) CreateAlert(id string, labels, annotations map[string]string, delayFor time.Duration) {
	alert := &ammodels.PostableAlert{
		Alert: ammodels.Alert{
			// GeneratorURL: "TODO",
			Labels: labels,
		},

		// StartsAt and EndAt can't be added there without changes in Registry
		Annotations: annotations,
	}

	r.rw.Lock()
	defer r.rw.Unlock()

	r.alerts[id] = alert
	if r.times[id].IsZero() {
		r.times[id] = now().Add(delayFor)
	}
}

// RemovePrefix removes all alerts with given ID prefix except a given list of IDs.
func (r *Registry) RemovePrefix(prefix string, keepIDs map[string]struct{}) {
	r.rw.Lock()
	defer r.rw.Unlock()

	for id := range r.alerts {
		if _, ok := keepIDs[id]; ok {
			continue
		}
		if strings.HasPrefix(id, prefix) {
			delete(r.alerts, id)
			delete(r.times, id)
		}
	}
}

// collect returns all firing alerts.
func (r *Registry) collect() ammodels.PostableAlerts {
	r.rw.RLock()
	defer r.rw.RUnlock()

	var res ammodels.PostableAlerts
	now := now()
	for id, t := range r.times {
		if t.Before(now) {
			res = append(res, r.alerts[id])
		}
	}
	return res
}
