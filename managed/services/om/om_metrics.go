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
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "om"
)

// MetricsCollector exposes OM's own collection health to Prometheus -- otherwise a
// degraded pass is visible only in the run history, which prunes at runHistory, and in a
// log line at Info.
//
// The following metrics are exposed:
//
//   - pmm_managed_om_runs_total{status="success|partial|failed"} -- completed collection
//     runs so far, by outcome. A rising "partial" or "failed" rate says collection itself
//     is unhealthy, before any caller notices the document is stale.
//
//   - pmm_managed_om_document_age_seconds -- how old the newest observation in the
//     document PMM currently serves is. This is the same age Snapshot.Stale is computed
//     from, exposed as a number so it can be alerted on rather than only rendered. Unset
//     until the estate has been observed at least once.
//
//   - pmm_managed_om_services{state="total|up|down"} -- how much of the estate the last
//     collection pass saw, and how much of it answered.
type MetricsCollector struct {
	svc *Service

	mRunsTotal   *prom.Desc
	mDocumentAge *prom.Desc
	mServices    *prom.Desc
}

// NewMetricsCollector creates a new MetricsCollector backed by the given OM service.
func NewMetricsCollector(svc *Service) *MetricsCollector {
	return &MetricsCollector{
		svc: svc,
		mRunsTotal: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "runs_total"),
			"Total number of OM topology collection runs, by outcome status.",
			[]string{"status"},
			nil,
		),
		mDocumentAge: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "document_age_seconds"),
			"Age, in seconds, of the newest observation in the topology document PMM currently serves.",
			nil,
			nil,
		),
		mServices: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
			"Services in the topology document PMM currently serves, by state.",
			[]string{"state"},
			nil,
		),
	}
}

// Describe implements prom.Collector.
func (c *MetricsCollector) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(c, ch)
}

// Collect implements prom.Collector.
func (c *MetricsCollector) Collect(ch chan<- prom.Metric) {
	ch <- prom.MustNewConstMetric(c.mRunsTotal, prom.CounterValue, float64(c.svc.runsSuccess.Load()), runStatusSuccess)
	ch <- prom.MustNewConstMetric(c.mRunsTotal, prom.CounterValue, float64(c.svc.runsPartial.Load()), runStatusPartial)
	ch <- prom.MustNewConstMetric(c.mRunsTotal, prom.CounterValue, float64(c.svc.runsFailed.Load()), runStatusFailed)

	snap := c.svc.snapshot()
	if snap.GetSnapshot().GetObservedAt() == nil {
		// Nothing has been observed yet -- a fresh estate whose leader has not completed
		// its first pass. Reporting age 0 would read as "current" rather than "no data".
		return
	}
	age := time.Since(snap.Snapshot.ObservedAt.AsTime()).Seconds()
	ch <- prom.MustNewConstMetric(c.mDocumentAge, prom.GaugeValue, age)

	summary := snap.Summary
	ch <- prom.MustNewConstMetric(c.mServices, prom.GaugeValue, float64(summary.GetTotalServices()), "total")
	ch <- prom.MustNewConstMetric(c.mServices, prom.GaugeValue, float64(summary.GetUpServices()), "up")
	ch <- prom.MustNewConstMetric(c.mServices, prom.GaugeValue, float64(summary.GetDownServices()), "down")
}

var _ prom.Collector = (*MetricsCollector)(nil)
