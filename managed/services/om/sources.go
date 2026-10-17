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
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

type factSource interface {
	key() string
	// collect reports failures in its result rather than as an error, so one failing source
	// does not discard the facts of the others.
	collect(ctx context.Context, services []*models.Service) SourceResult
}

// inventorySource facts carry no ObservedAt: inventory is current by definition.
type inventorySource struct {
	nodes map[string]*models.Node
}

func (inventorySource) key() string { return sourceInventory }

func (s inventorySource) collect(_ context.Context, services []*models.Service) SourceResult {
	result := SourceResult{Source: sourceInventory, Status: SourceOK}
	for _, service := range services {
		node := s.nodes[service.NodeID]
		add := func(field string, value any) {
			if value == nil || value == "" {
				return
			}
			result.Facts = append(result.Facts, Fact{
				Service: service.ServiceID,
				Field:   field,
				Value:   value,
				Source:  sourceInventory,
			})
		}
		add(fieldServiceType, string(service.ServiceType))
		add(fieldCluster, service.Cluster)
		add(fieldReplicationSet, service.ReplicationSet)
		add(fieldEnvironment, service.Environment)
		if node != nil {
			add(fieldHost, node.NodeName)
		}
		add(fieldEndpoint, inventoryEndpoint(service, node))
	}
	result.Detail = map[string]any{"services": len(services), "facts": len(result.Facts)}
	return result
}

// inventoryEndpoint prefers the node's address: the service's is where its local pmm-agent
// reaches it, often 127.0.0.1, which is useless to anyone else.
func inventoryEndpoint(service *models.Service, node *models.Node) string {
	var host string
	if node != nil {
		host = firstNonEmpty(node.Address, node.NodeName)
	}
	if host == "" {
		host = pointer.GetString(service.Address)
	}
	if host == "" || service.Port == nil {
		return ""
	}
	return net.JoinHostPort(host, strconv.Itoa(int(*service.Port)))
}

type metricsSource struct {
	vm  victoriaMetricsClient
	l   *logrus.Entry
	now time.Time
}

func (metricsSource) key() string { return sourceMetrics }

// queryBatch bounds the length of the service_id regex matcher, not the result set.
const queryBatch = 50

type labelSignal struct {
	field string
	label string
}

type metricSignal struct {
	metric        string
	labels        []labelSignal
	valueField    string
	presenceField string
	reduceMax     bool
}

var metricSignals = []metricSignal{
	{
		metric: metricVersionInfo,
		labels: []labelSignal{
			{fieldHost, "node_name"},
			{fieldServiceType, "service_type"},
			{fieldCluster, "cluster"},
			{fieldReplicationSet, "replication_set"},
			{fieldEnvironment, "environment"},
			{fieldVersion, "mongodb"},
			{fieldVendor, "vendor"},
			{fieldEdition, "edition"},
		},
	},
	{
		metric: metricMembersSelf,
		labels: []labelSignal{
			{fieldState, "member_state"},
			// member_idx is the host:port the replica set knows the member by, truer than inventory's.
			{fieldEndpoint, "member_idx"},
			{fieldClusterRole, "cl_role"},
		},
	},
	{metric: metricUp, valueField: fieldExporterUp},
	{metric: metricMongosShards, presenceField: fieldIsMongos},
	// Each member reports lag against every secondary; the max is the worst lag this node saw.
	{metric: metricReplicationLag, valueField: fieldReplicationLag, reduceMax: true},
	{metric: metricOplogHead, valueField: fieldOplogHead},
	{metric: metricOplogTail, valueField: fieldOplogTail},
}

type metricsRun struct {
	src     metricsSource
	result  *SourceResult
	covered map[string]bool
	oldest  float64
	queries int
}

func (s metricsSource) collect(ctx context.Context, services []*models.Service) SourceResult {
	result := SourceResult{Source: sourceMetrics, Status: SourceOK}
	if len(services) == 0 {
		return result
	}

	ids := make([]string, 0, len(services))
	for _, service := range services {
		ids = append(ids, service.ServiceID)
	}

	run := &metricsRun{src: s, result: &result, covered: make(map[string]bool, len(ids))}
	for batch := range slices.Chunk(ids, queryBatch) {
		matcher := fmt.Sprintf(`service_id=~%q,%s`, strings.Join(batch, "|"), highResolutionJob)
		for _, signal := range metricSignals {
			run.signal(ctx, matcher, signal)
		}
		run.expressions(ctx, matcher)
		run.lastUp(ctx, matcher)
	}

	switch {
	case len(result.Errors) > 0 && len(run.covered) == 0:
		result.Status = SourceFailed
	case len(result.Errors) > 0 || len(run.covered) < len(services):
		result.Status = SourcePartial
	}
	result.Detail = map[string]any{
		"services":           len(services),
		"services_covered":   len(run.covered),
		"facts":              len(result.Facts),
		"queries":            run.queries,
		"oldest_age_seconds": int(run.oldest),
	}
	s.l.Infof("metrics source: %d/%d service(s) covered by %d queries, oldest sample %ds",
		len(run.covered), len(services), run.queries, int(run.oldest))
	return result
}

func (r *metricsRun) signal(ctx context.Context, matcher string, signal metricSignal) {
	ages := r.ages(ctx, matcher, signal)
	if signal.valueField == "" {
		return
	}

	// A separate value query, because lag()'s value is the series' age, not the metric's value.
	query := fmt.Sprintf("last_over_time(%s{%s}[%s])", signal.metric, matcher, metricsLookback)
	r.queries++
	values := make(map[string]seriesSample)
	r.src.each(ctx, r.result, query, func(serviceID string, m model.Metric, v float64) {
		age, dated := ages[seriesKey(m)]
		sample := seriesSample{value: v, age: age, dated: dated}
		if held, ok := values[serviceID]; ok && !sample.supersedes(held, signal.reduceMax) {
			return
		}
		values[serviceID] = sample
	})

	for serviceID, sample := range values {
		fact := Fact{Service: serviceID, Field: signal.valueField, Value: sample.value, Source: sourceMetrics}
		if sample.dated {
			fact.ObservedAt = r.observedAt(sample.age)
		}
		r.result.Facts = append(r.result.Facts, fact)
	}
}

type seriesSample struct {
	value float64
	age   float64
	dated bool
}

// supersedes prefers the largest value under reduceMax, otherwise the freshest series: an exporter
// can abandon a series and start another under the same service_id, and the stale one is not current.
func (s seriesSample) supersedes(held seriesSample, reduceMax bool) bool {
	if reduceMax && s.value != held.value {
		return s.value > held.value
	}
	if !s.dated {
		return false
	}
	if !held.dated {
		return true
	}
	return s.age < held.age
}

// seriesKey ignores __name__ so one series matches across lag() and last_over_time(),
// since lag() need not carry the metric name through.
func seriesKey(m model.Metric) model.Fingerprint {
	if _, ok := m[model.MetricNameLabel]; !ok {
		return m.Fingerprint()
	}
	trimmed := make(model.Metric, len(m)-1)
	for name, value := range m {
		if name != model.MetricNameLabel {
			trimmed[name] = value
		}
	}
	return trimmed.Fingerprint()
}

// ages uses lag(), the seconds since each series' last raw sample. A timestamp(last_over_time()) query
// reports the evaluation time instead, so a series scraped days ago would read as fresh.
func (r *metricsRun) ages(ctx context.Context, matcher string, signal metricSignal) map[model.Fingerprint]float64 {
	query := fmt.Sprintf("lag(%s{%s}[%s])", signal.metric, matcher, metricsLookback)
	r.queries++
	ages := make(map[model.Fingerprint]float64)

	r.src.each(ctx, r.result, query, func(serviceID string, m model.Metric, age float64) {
		r.covered[serviceID] = true
		if age > r.oldest {
			r.oldest = age
		}
		ages[seriesKey(m)] = age
		observedAt := r.observedAt(age)

		for _, label := range signal.labels {
			if value := string(m[model.LabelName(label.label)]); value != "" {
				r.result.Facts = append(r.result.Facts, Fact{
					Service: serviceID, Field: label.field, Value: value,
					Source: sourceMetrics, ObservedAt: observedAt,
				})
			}
		}
		if signal.presenceField != "" {
			r.result.Facts = append(r.result.Facts, Fact{
				Service: serviceID, Field: signal.presenceField, Value: true,
				Source: sourceMetrics, ObservedAt: observedAt,
			})
		}
	})
	return ages
}

// expressions facts are dated to the run because lag() cannot date a derived expression; the
// projection already suppresses both for a service that is not up.
func (r *metricsRun) expressions(ctx context.Context, matcher string) {
	for _, expr := range []struct {
		query string
		field string
	}{
		{fmt.Sprintf(queryCPUUsage, matcher, metricsLookback), fieldCPUUsage},
		{fmt.Sprintf(queryConnectionsFree, matcher, metricsLookback), fieldConnectionsFree},
	} {
		r.queries++
		r.src.each(ctx, r.result, expr.query, func(serviceID string, _ model.Metric, v float64) {
			observedAt := r.src.now
			r.result.Facts = append(r.result.Facts, Fact{
				Service: serviceID, Field: expr.field, Value: v,
				Source: sourceMetrics, ObservedAt: &observedAt,
			})
		})
	}
}

// lastUp facts are dated by the last "up" sample itself; unlike the up flag, they stay true however old.
func (r *metricsRun) lastUp(ctx context.Context, matcher string) {
	r.queries++
	query := fmt.Sprintf(queryLastUp, matcher, metricsLookback)
	r.src.each(ctx, r.result, query, func(serviceID string, _ model.Metric, seconds float64) {
		at := time.Unix(0, int64(seconds*float64(time.Second))).UTC()
		r.result.Facts = append(r.result.Facts, Fact{
			Service: serviceID, Field: fieldLastUp, Value: seconds,
			Source: sourceMetrics, ObservedAt: &at,
		})
	})
}

func (r *metricsRun) observedAt(age float64) *time.Time {
	observed := r.src.now.Add(-time.Duration(age * float64(time.Second)))
	return &observed
}

// each skips samples without a service_id label; guessing from the instance label would mix
// exporter generations.
func (s metricsSource) each(ctx context.Context, result *SourceResult, query string, apply func(string, model.Metric, float64)) {
	value, warnings, err := s.vm.Query(ctx, query, s.now)
	if err != nil {
		s.l.Warnf("query '%s' failed: %s", query, err)
		result.Errors = append(result.Errors, RunError{
			Scope: "query", Code: "vm_query_failed",
			Message: fmt.Sprintf("%s: %s", query, err),
		})
		return
	}
	for _, w := range warnings {
		s.l.Warnf("query '%s': %s", query, w)
	}

	vector, ok := value.(model.Vector)
	if !ok {
		result.Errors = append(result.Errors, RunError{
			Scope: "query", Code: "vm_unexpected_result",
			Message: fmt.Sprintf("%s: unexpected result type %T", query, value),
		})
		return
	}
	for _, sample := range vector {
		if serviceID := string(sample.Metric["service_id"]); serviceID != "" {
			apply(serviceID, sample.Metric, float64(sample.Value))
		}
	}
}
