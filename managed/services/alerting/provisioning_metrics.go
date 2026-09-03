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

package alerting

import (
	"sync"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

// Provisioned rules are built so that an execution error stays quiet rather than paging a whole
// fleet, and several ways this can fail - an unreachable Grafana database, a reload that was
// refused, a file written but never ingested - are otherwise invisible. These metrics are how
// support answers "did the bundle actually apply?" without shell access to the server.
const (
	provisioningMetricsNamespace = "pmm"
	provisioningMetricsSubsystem = "alerting_provisioning"
)

// Stages a reconcile can fail at, reported as the stage label of the error counter. Resolving the
// datasource is called out separately from the rest of rendering because it is the one stage that
// fails for a reason outside PMM - an unreachable or misconfigured Grafana database - and the
// operator response is different.
const (
	stageDatasource = "datasource"
	stageConflict   = "conflict"
	stageRender     = "render"
	stageValidate   = "validate"
	stageWrite      = "write"
	stageApply      = "apply"
)

// States reported for each bundle.
//
// These describe what PMM has done, not what Grafana holds. PMM writes a provisioning file; Grafana
// decides when to read it, and nothing here asks it whether it did. That is deliberate: proving
// ingestion would mean querying Grafana's own schema at runtime to establish something whose failure
// is already loud, because a Grafana that cannot load its provisioning does not start, and that
// takes the whole interface down with it.
const (
	// Written: PMM rendered this content and it is on disk. The ordinary healthy state.
	stateWritten = "written"
	// Pending: written, but an apply action PMM had to perform itself failed, and a retry is owed.
	// Not used for a follower that deferred to the leader - that is the design working, not a fault.
	statePending = "pending"
	// Disabled: the bundle does not apply here, so its rules are deleted rather than kept.
	stateDisabled = "disabled"
	// Error: the last attempt did not get as far as a file on disk, so what Grafana holds is
	// whatever the previous attempt left. The error counter says which stage failed.
	stateError = "error"
)

// ProvisioningMetrics reports what this node rendered and how far PMM got in applying it.
type ProvisioningMetrics struct {
	mInfo        *prom.Desc
	mLastSuccess *prom.Desc
	mErrors      *prom.Desc
	mConflicts   *prom.Desc

	m sync.RWMutex
	// renderedHash and writtenHash differ exactly while a render has not reached disk.
	renderedHash string
	writtenHash  string
	// applyPending is set when an apply action PMM performs itself failed and is being retried.
	applyPending bool
	bundles      map[string]bool
	lastSuccess  time.Time
	errors       map[string]float64
	// conflicts counts rule UIDs that belong to someone else, so PMM is not provisioning them.
	conflicts float64
	// failed is set when a reconcile could not produce a file, and cleared by the next one that
	// does. It separates "Grafana has the previous rules because nothing changed" from "Grafana has
	// them because the new ones could not be built".
	failed bool
}

func newProvisioningMetrics() *ProvisioningMetrics {
	return &ProvisioningMetrics{
		mInfo: prom.NewDesc(
			prom.BuildFQName(provisioningMetricsNamespace, provisioningMetricsSubsystem, "info"),
			"Reports the built-in alert rule bundles this node provisions, the content it rendered and how far PMM got in applying it.",
			[]string{"bundle", "state", "hash"}, nil,
		),
		mLastSuccess: prom.NewDesc(
			prom.BuildFQName(provisioningMetricsNamespace, provisioningMetricsSubsystem, "last_success_timestamp_seconds"),
			"Unix timestamp of the last time the rendered alert rules were successfully written to disk.",
			nil, nil,
		),
		mErrors: prom.NewDesc(
			prom.BuildFQName(provisioningMetricsNamespace, provisioningMetricsSubsystem, "errors_total"),
			"Number of failures while provisioning built-in alert rules, by the stage that failed.",
			[]string{"stage"}, nil,
		),

		mConflicts: prom.NewDesc(
			prom.BuildFQName(provisioningMetricsNamespace, provisioningMetricsSubsystem, "conflicting_rules"),
			"Number of built-in alert rules PMM is not provisioning because their UID already belongs "+
				"to a rule someone else created. Those rules are absent until the UID is released.",
			nil, nil,
		),

		// Seeded from the catalog so the series exist from process start: a dashboard can then
		// tell "no rules yet" from "this PMM does not know about that bundle", and a failure
		// before the first successful render still has something to report against. The enabled
		// flags are provisional and the first render replaces them.
		bundles: seedBundles(),
		errors:  make(map[string]float64),
	}
}

func seedBundles() map[string]bool {
	bundles := make(map[string]bool, len(builtinBundles))
	for _, bundle := range builtinBundles {
		bundles[bundle.id] = true
	}
	return bundles
}

// setRendered records the content this node has rendered and which bundles it covers.
func (m *ProvisioningMetrics) setRendered(hash string, bundles map[string]bool) {
	m.m.Lock()
	defer m.m.Unlock()

	m.renderedHash = hash
	m.failed = false
	if bundles != nil {
		m.bundles = bundles
	}
}

// setWritten records that the given content is on disk. It says nothing about Grafana having read
// it - see the note on the state constants.
func (m *ProvisioningMetrics) setWritten(hash string) {
	m.m.Lock()
	defer m.m.Unlock()

	m.writtenHash = hash
	m.lastSuccess = time.Now()
}

// setApplyPending records whether an apply action PMM performs itself is owed a retry.
func (m *ProvisioningMetrics) setApplyPending(pending bool) {
	m.m.Lock()
	defer m.m.Unlock()

	m.applyPending = pending
}

// setConflicts records how many rule UIDs belong to someone else.
func (m *ProvisioningMetrics) setConflicts(n int) {
	m.m.Lock()
	defer m.m.Unlock()

	m.conflicts = float64(n)
}

func (m *ProvisioningMetrics) recordError(stage string) {
	m.m.Lock()
	defer m.m.Unlock()

	m.errors[stage]++
	// A failed apply still leaves a good file on disk for the next Grafana start, so only the
	// stages before that leave the bundle in an error state.
	if stage != stageApply {
		m.failed = true
	}
}

// Describe implements prometheus.Collector.
func (m *ProvisioningMetrics) Describe(ch chan<- *prom.Desc) {
	ch <- m.mInfo
	ch <- m.mLastSuccess
	ch <- m.mErrors
	ch <- m.mConflicts
}

// Collect implements prometheus.Collector.
func (m *ProvisioningMetrics) Collect(ch chan<- prom.Metric) {
	m.m.RLock()
	defer m.m.RUnlock()

	for bundle, enabled := range m.bundles {
		var state string
		switch {
		case !enabled:
			state = stateDisabled
		case m.failed:
			state = stateError
		case m.applyPending:
			state = statePending
		case m.renderedHash != "" && m.renderedHash == m.writtenHash:
			state = stateWritten
		default:
			state = statePending
		}

		ch <- prom.MustNewConstMetric(m.mInfo, prom.GaugeValue, 1, bundle, state, m.renderedHash)
	}

	var lastSuccess float64
	if !m.lastSuccess.IsZero() {
		lastSuccess = float64(m.lastSuccess.Unix())
	}
	ch <- prom.MustNewConstMetric(m.mLastSuccess, prom.GaugeValue, lastSuccess)

	// Report every stage, so that a dashboard or alert can rely on the series existing before the
	// first failure happens.
	ch <- prom.MustNewConstMetric(m.mConflicts, prom.GaugeValue, m.conflicts)

	for _, stage := range []string{stageDatasource, stageConflict, stageRender, stageValidate, stageWrite, stageApply} {
		ch <- prom.MustNewConstMetric(m.mErrors, prom.CounterValue, m.errors[stage], stage)
	}
}

// Check interfaces.
var _ prom.Collector = (*ProvisioningMetrics)(nil)
