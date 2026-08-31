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

// factSource is one place collection reads the estate from.
//
// The interface is the extension point: the SEP inventory app's on-host facts arrive as
// another implementation of this and nothing downstream changes, because the merge
// resolves overlap by the precedence table rather than by which source ran.
type factSource interface {
	// key is the source key every fact it produces carries.
	key() string
	// collect reads the estate. It reports rather than returns an error: a source that
	// could not answer is a fact about the run, and losing the sources that did answer
	// because one did not is the opposite of useful.
	collect(ctx context.Context, services []*models.Service) SourceResult
}

// inventorySource restates PMM's inventory as facts.
//
// Not time-bounded -- every fact carries a nil observed_at -- because inventory is
// current by definition. That is the distinction Fact.ObservedAt exists to keep: a
// version read from a metric sample can be nine days old, a cluster label read from the
// services table cannot.
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

// inventoryEndpoint builds the host:port inventory knows the service by.
//
// The node, not the service. A service's address is where its own pmm-agent reaches it,
// which for an agent on the same host is 127.0.0.1 -- true, and useless as an endpoint
// for anyone else. The node's address is the one that means something off the box.
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

// metricsSource reads the signal catalog out of VictoriaMetrics.
type metricsSource struct {
	vm  victoriaMetricsClient
	l   *logrus.Entry
	now time.Time
}

func (metricsSource) key() string { return sourceMetrics }

// queryBatch is how many services are pinned per query.
//
// Each contributes a 37-byte UUID to a regex matcher, so this bounds the query string
// rather than the result set. Without it a few hundred services build a matcher tens of
// kilobytes long on every one of the queries below.
const queryBatch = 50

// labelSignal maps one series label onto one document field.
type labelSignal struct {
	field string
	label string
}

// metricSignals says, per metric, which labels to read off it and whether its sample
// value is wanted too.
//
// The age query carries the labels, so a label-only metric costs one query. A metric
// whose value matters costs two, because lag()'s value is the age rather than the
// metric's own.
type metricSignal struct {
	metric     string
	labels     []labelSignal
	valueField string
	// presenceField is set from the series existing at all, whatever its value.
	presenceField string
	// reduceMax folds several series for one service into the largest.
	reduceMax bool
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
			// member_idx is the host:port the replica set itself knows the member by,
			// which is a truer endpoint than inventory's address.
			{fieldEndpoint, "member_idx"},
			{fieldClusterRole, "cl_role"},
		},
	},
	{metric: metricUp, valueField: fieldExporterUp},
	{metric: metricMongosShards, presenceField: fieldIsMongos},
	// Each member reports lag against every secondary, so the worst of them is "the worst
	// lag this node saw". Double-counting is harmless under max.
	{metric: metricReplicationLag, valueField: fieldReplicationLag, reduceMax: true},
	{metric: metricOplogHead, valueField: fieldOplogHead},
	{metric: metricOplogTail, valueField: fieldOplogTail},
}

// metricsRun is the mutable state of one pass over the catalog, kept out of collect so
// each stage stays readable on its own.
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

// signal reads one catalog entry: its age query, and its value query when the metric's
// own value is wanted.
func (r *metricsRun) signal(ctx context.Context, matcher string, signal metricSignal) {
	ages := r.ages(ctx, matcher, signal)
	if signal.valueField == "" {
		return
	}

	// The value query, issued only when the metric's own value is wanted, because lag()'s
	// value is the age rather than the metric's.
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
		// Dated by its own series' age, so a value read over a day-long window still knows
		// how old it is -- and the fact that survives the reduction carries the age of the
		// series it actually came from.
		if sample.dated {
			fact.ObservedAt = r.observedAt(sample.age)
		}
		r.result.Facts = append(r.result.Facts, fact)
	}
}

// seriesSample is one series' contribution to a valued signal: the sample the value query
// returned, and the age of that same series' last raw sample.
type seriesSample struct {
	value float64
	age   float64
	// dated is false when the age query returned no matching series, which leaves the
	// fact undatable rather than pretending to an age of zero.
	dated bool
}

// supersedes reports whether this sample should displace the one already held for the
// service.
//
// Several series of one metric under one service_id is the ordinary case rather than a
// pathology, and the two kinds need different reducers:
//
// Under reduceMax the largest sample wins, which is the point of the reducer --
// mongodb_mongod_replset_member_replication_lag emits one series per peer and the worst of
// them is what the node saw. The winner is then dated by its own series, so a stale maximum
// reads as stale and fieldSet.live drops it, rather than being reported as a statement about
// now.
//
// Otherwise the freshest series wins, because the series are successive generations of one
// reading rather than peers: an exporter that learns cluster_role only after startup
// abandons its first mongodb_up series and begins another under the same service_id, and the
// abandoned one is not evidence about the present. Dating the reduction from the oldest of
// them -- which is what this used to do -- made every such service report DOWN with its live
// sample seconds old.
//
// An undated sample never displaces a dated one, and never displaces its own kind: with no
// age to compare, first seen stays.
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

// seriesKey identifies one series across the two queries a valued signal issues.
//
// Every label except __name__, because the age query wraps the metric in lag() and a
// rollup need not carry the metric name through, while last_over_time() does. Keying on
// the rest pairs each sample with its own series' age, which is what a service carrying
// several series of one metric needs and what a per-service key cannot express.
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

// ages runs the age query, harvesting the labels it carries, and returns each series' age
// so the value query can date its own facts.
//
// Keyed per series rather than per service, because a service can carry several series of
// one metric and they do not share an age. Reducing them here -- to the oldest, which this
// used to do -- loses the pairing the value query needs: the sample that survives its own
// reduction then gets stamped with an age no series it came from ever reported. See
// seriesSample.supersedes for what each reducer does with the pairing.
//
// The label and presence facts below were always unaffected: each is dated with its own
// series' age inside the callback, because each comes from exactly one series.
//
// MetricsQL's lag() answers the seconds since the series' last raw sample and preserves
// every label, so one query yields both the labels the catalog reads and the staleness
// the run records.
//
// Wrapping timestamp() around a last_over_time() looks like it should do the same and
// does not: it reports the evaluation time rounded to the step, which reads as "fresh"
// for a series last scraped days ago -- the single most dangerous way to be wrong here.
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

// expressions reads the two derived percentages.
//
// Neither exists as a series, so lag() has nothing to report on and their facts are dated
// to the run. That is safe because both are gauges the projection already suppresses for
// a service that is not up.
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

// observedAt turns an age in seconds into the moment the sample was taken.
func (r *metricsRun) observedAt(age float64) *time.Time {
	observed := r.src.now.Add(-time.Duration(age * float64(time.Second)))
	return &observed
}

// each runs one instant query and applies apply to every sample, keyed by service_id.
//
// A sample carrying no service_id label is skipped: it cannot be attributed to a service,
// and guessing from the instance label is how generations get mixed.
func (s metricsSource) each(ctx context.Context, result *SourceResult, query string, apply func(string, model.Metric, float64)) {
	value, warnings, err := s.vm.Query(ctx, query, s.now)
	if err != nil {
		s.l.Warnf("query %q failed: %s", query, err)
		result.Errors = append(result.Errors, RunError{
			Scope: "query", Code: "vm_query_failed",
			Message: fmt.Sprintf("%s: %s", query, err),
		})
		return
	}
	for _, w := range warnings {
		s.l.Warnf("query %q: %s", query, w)
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
