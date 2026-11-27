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
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

type stubVM struct {
	lag    map[string]model.Vector
	values map[string]model.Vector
}

func (s stubVM) Query(_ context.Context, query string, _ time.Time, _ ...v1.Option) (model.Value, v1.Warnings, v1.Infos, error) {
	for metric, vector := range s.lag {
		if strings.HasPrefix(query, "lag("+metric+"{") {
			return matching(vector, query), nil, nil, nil
		}
	}
	for metric, vector := range s.values {
		if strings.HasPrefix(query, "last_over_time("+metric+"{") || strings.HasPrefix(query, "irate("+metric+"{") {
			return matching(vector, query), nil, nil, nil
		}
	}
	return model.Vector{}, nil, nil, nil
}

var equalityMatcher = regexp.MustCompile(`(\w+)="([^"]*)"`)

// matching applies the query's label="value" matchers as VictoriaMetrics would; regex matchers are ignored.
func matching(vector model.Vector, query string) model.Vector {
	matchers := equalityMatcher.FindAllStringSubmatch(query, -1)
	kept := model.Vector{}
	for _, sample := range vector {
		if !slices.ContainsFunc(matchers, func(m []string) bool {
			return string(sample.Metric[model.LabelName(m[1])]) != m[2]
		}) {
			kept = append(kept, sample)
		}
	}
	return kept
}

// testSeries is one series as both lag() and last_over_time() return it.
type testSeries struct {
	age, value float64
	labels     []string
}

func collectSeries(t *testing.T, now time.Time, metric string, series ...testSeries) SourceResult {
	t.Helper()

	vm := stubVM{lag: map[string]model.Vector{}, values: map[string]model.Vector{}}
	for _, s := range series {
		vm.lag[metric] = append(vm.lag[metric], lagSample(s.age, s.labels...))
		vm.values[metric] = append(vm.values[metric], valueSample(metric, s.value, s.labels...))
	}
	src := metricsSource{vm: vm, l: logrus.WithField("test", t.Name()), now: now, maxAge: projectionMaxAge}
	return src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}})
}

type recordingVM struct {
	queries []string
}

func (r *recordingVM) Query(_ context.Context, query string, _ time.Time, _ ...v1.Option) (model.Value, v1.Warnings, v1.Infos, error) {
	r.queries = append(r.queries, query)
	return model.Vector{}, nil, nil, nil
}

const testServiceID = "svc-1"

func seriesLabels(labels ...string) model.Metric {
	m := model.Metric{"service_id": testServiceID}
	for i := 0; i+1 < len(labels); i += 2 {
		m[model.LabelName(labels[i])] = model.LabelValue(labels[i+1])
	}
	return m
}

// lagSample carries no __name__ because lag() is a rollup; seriesKey pairs the two queries on the other labels.
func lagSample(age float64, labels ...string) *model.Sample {
	return &model.Sample{Metric: seriesLabels(labels...), Value: model.SampleValue(age)}
}

func valueSample(metric string, value float64, labels ...string) *model.Sample {
	m := seriesLabels(labels...)
	m[model.MetricNameLabel] = model.LabelValue(metric)
	return &model.Sample{Metric: m, Value: model.SampleValue(value)}
}

func factFor(result SourceResult, field string) *Fact {
	var found *Fact
	for i := range result.Facts {
		if result.Facts[i].Field == field {
			found = &result.Facts[i]
		}
	}
	return found
}

// A primary's exporter reports its secondaries' lag, which must not read as its own.
func TestReplicationLagIsTheMembersOwn(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	t.Run("a primary has none", func(t *testing.T) {
		t.Parallel()

		result := collectSeries(t, now, metricReplicationLag,
			testSeries{5, 166, []string{"name", "node01", "self", "0"}},
			testSeries{5, 60, []string{"name", "node02", "self", "0"}},
		)
		assert.Nil(t, factFor(result, fieldReplicationLag))
	})

	t.Run("a secondary reads its own, not its peer's", func(t *testing.T) {
		t.Parallel()

		result := collectSeries(t, now, metricReplicationLag,
			testSeries{5, 166, []string{"name", "node01", "self", "0"}},
			testSeries{5, 60, []string{"name", "node02", "self", "1"}},
		)
		fact := factFor(result, fieldReplicationLag)
		require.NotNil(t, fact, "expected a replication-lag fact")
		assert.InDelta(t, 60.0, fact.Value, 0.001)
	})

	t.Run("the freshest of its own series wins", func(t *testing.T) {
		t.Parallel()

		result := collectSeries(t, now, metricReplicationLag,
			testSeries{4 * 60 * 60, 900, []string{"name", "node02", "state", "SECONDARY", "self", "1"}},
			testSeries{10, 30, []string{"name", "node02", "state", "RECOVERING", "self", "1"}},
		)
		fact := factFor(result, fieldReplicationLag)
		require.NotNil(t, fact, "expected a replication-lag fact")
		assert.InDelta(t, 30.0, fact.Value, 0.001)
		require.NotNil(t, fact.ObservedAt)
		assert.Equal(t, now.Add(-10*time.Second), fact.ObservedAt.UTC())
	})
}

// After a role switch the old rs_state series stays in the lookback, and mixing its counts with the live
// series' misreports the ratio.
func TestConnectionsFreeReadsTheLiveSeries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	result := collectSeries(t, now, metricConnections,
		testSeries{1213, 36, []string{"rs_state", "1", "state", "current"}},
		testSeries{1213, 373, []string{"rs_state", "1", "state", "available"}},
		testSeries{13, 245, []string{"rs_state", "2", "state", "current"}},
		testSeries{13, 164, []string{"rs_state", "2", "state", "available"}},
	)
	merged := mergeFacts([]SourceResult{result}, defaultPrecedence)
	free := connectionsFree(fieldSet{fields: merged[testServiceID], now: now, maxAge: projectionMaxAge})

	require.NotNil(t, free)
	assert.InDelta(t, 100*164.0/409, *free, 0.001)
}

// mongodb_up lacks cluster_role until the exporter learns its role, then continues as a new series.
// Both stay within metricsLookback, so dating exporter_up by the older one reports a live service DOWN.
func TestAbandonedSeriesDoesNotStaleTheLiveOne(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

	vm := stubVM{
		lag: map[string]model.Vector{
			metricUp: {
				lagSample(2, "cluster_role", "mongod"),
				lagSample(540),
			},
		},
		values: map[string]model.Vector{
			metricUp: {
				valueSample(metricUp, 1, "cluster_role", "mongod"),
				valueSample(metricUp, 1),
			},
		},
	}

	src := metricsSource{vm: vm, l: logrus.WithField("test", t.Name()), now: now}
	fact := factFor(src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}}), fieldExporterUp)

	require.NotNil(t, fact, "expected an exporter-up fact")
	assert.InDelta(t, 1.0, fact.Value, 0.001)
	require.NotNil(t, fact.ObservedAt)
	assert.Equal(t, now.Add(-2*time.Second), fact.ObservedAt.UTC(),
		"exporter_up must carry the live series' age, not the abandoned one's")
}

// A stopped exporter target starts a new mongodb_up series without cluster_role, so the arbiter
// label survives only on the abandoned series and must carry that series' age.
func TestNodeTypeComesFromTheUpSeries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	vm := stubVM{lag: map[string]model.Vector{metricUp: {
		lagSample(600, "cluster_role", "arbiter"),
		lagSample(2),
	}}}
	src := metricsSource{vm: vm, l: logrus.WithField("test", t.Name()), now: now}
	fact := factFor(src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}}), fieldNodeType)

	require.NotNil(t, fact, "expected a node-type fact")
	assert.Equal(t, nodeTypeArbiter, fact.Value)
	require.NotNil(t, fact.ObservedAt)
	assert.Equal(t, now.Add(-10*time.Minute), fact.ObservedAt.UTC())
}

// mongodb_exporter is scraped at hr and lr resolutions; the lr series' age exceeds volatileMaxAge
// for most of every minute, so reducing across both reports a healthy service DOWN.
func TestEveryQueryNarrowsToTheHighResolutionJob(t *testing.T) {
	t.Parallel()

	vm := &recordingVM{}
	src := metricsSource{
		vm:  vm,
		l:   logrus.WithField("test", t.Name()),
		now: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
	}
	src.collect(t.Context(), []*models.Service{{ServiceID: "svc-1"}})

	require.NotEmpty(t, vm.queries, "expected the catalog to issue queries")
	for _, query := range vm.queries {
		assert.Contains(t, query, highResolutionJob,
			"every query must select the high-resolution job")
	}
}

// irate needs two samples in its window, so the window scales with the scrape interval through the max age.
func TestCPUIdleRateWindowIsTheMaxAge(t *testing.T) {
	t.Parallel()

	vm := &recordingVM{}
	src := metricsSource{
		vm:     vm,
		l:      logrus.WithField("test", t.Name()),
		now:    time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC),
		maxAge: 4 * time.Minute,
	}
	src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}})

	i := slices.IndexFunc(vm.queries, func(query string) bool { return strings.HasPrefix(query, "irate("+metricCPUIdle+"{") })
	require.GreaterOrEqual(t, i, 0, "expected an idle-rate query")
	assert.Contains(t, vm.queries[i], "[4m]")
}

// After a role switch the old rs_state series keeps an idle rate inside the irate window; CPU reads the live
// series, not the larger idle of the two.
func TestCPUUsageReadsTheLiveSeries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	idle := collectSeries(t, now, metricCPUIdle,
		testSeries{25, 950, []string{"rs_state", "2"}},
		testSeries{5, 200, []string{"rs_state", "1"}},
	)
	cores := collectSeries(t, now, metricCPUCores, testSeries{5, 1, []string{"rs_state", "1"}})
	merged := mergeFacts([]SourceResult{idle, cores}, defaultPrecedence)
	usage := cpuUsage(fieldSet{fields: merged[testServiceID], now: now, maxAge: projectionMaxAge})

	require.NotNil(t, usage)
	assert.InDelta(t, 80.0, *usage, 0.001)
}

type lastUpVM struct {
	seconds float64
}

func (v lastUpVM) Query(_ context.Context, query string, _ time.Time, _ ...v1.Option) (model.Value, v1.Warnings, v1.Infos, error) {
	if !strings.Contains(query, "tmax_over_time("+metricUp+"{") {
		return model.Vector{}, nil, nil, nil
	}
	return model.Vector{{Metric: seriesLabels(), Value: model.SampleValue(v.seconds)}}, nil, nil, nil
}

// Dating the last-up fact by the run would make a row that has been down for hours look freshly observed.
func TestLastUpIsDatedByTheMomentItNames(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	lastUp := now.Add(-22 * time.Hour)
	src := metricsSource{
		vm:  lastUpVM{seconds: float64(lastUp.Unix())},
		l:   logrus.WithField("test", t.Name()),
		now: now,
	}

	result := src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}})

	fact := factFor(result, fieldLastUp)
	require.NotNil(t, fact)
	assert.InDelta(t, float64(lastUp.Unix()), fact.Value, 0)
	require.NotNil(t, fact.ObservedAt)
	assert.True(t, lastUp.Equal(*fact.ObservedAt), "got %s", *fact.ObservedAt)
}
