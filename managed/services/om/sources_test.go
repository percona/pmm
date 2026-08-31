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

// stubVM answers the two queries metricsSource issues per signal: lag() for ages and
// last_over_time() for values. Keyed by metric name so one stub can cover a whole pass.
type stubVM struct {
	lag    map[string]model.Vector
	values map[string]model.Vector
}

func (s stubVM) Query(_ context.Context, query string, _ time.Time, _ ...v1.Option) (model.Value, v1.Warnings, error) {
	for metric, vector := range s.lag {
		if strings.HasPrefix(query, "lag("+metric+"{") {
			return vector, nil, nil
		}
	}
	for metric, vector := range s.values {
		if strings.HasPrefix(query, "last_over_time("+metric+"{") {
			return vector, nil, nil
		}
	}
	return model.Vector{}, nil, nil
}

// recordingVM answers nothing and keeps what it was asked, for the tests that are about
// the queries the catalog issues rather than about the samples they return.
type recordingVM struct {
	queries []string
}

func (r *recordingVM) Query(_ context.Context, query string, _ time.Time, _ ...v1.Option) (model.Value, v1.Warnings, error) {
	r.queries = append(r.queries, query)
	return model.Vector{}, nil, nil
}

// testServiceID is the one service the sample helpers below build series for.
const testServiceID = "svc-1"

// seriesLabels builds one series' label set: service_id plus the given name/value pairs,
// which is what distinguishes two series of the same metric under one service.
func seriesLabels(labels ...string) model.Metric {
	m := model.Metric{"service_id": testServiceID}
	for i := 0; i+1 < len(labels); i += 2 {
		m[model.LabelName(labels[i])] = model.LabelValue(labels[i+1])
	}
	return m
}

// lagSample is a sample as the age query returns it: lag() is a rollup, so it carries no
// __name__ -- which is why seriesKey pairs the two queries on everything else.
func lagSample(age float64, labels ...string) *model.Sample {
	return &model.Sample{Metric: seriesLabels(labels...), Value: model.SampleValue(age)}
}

// valueSample is a sample as the value query returns it, __name__ included.
func valueSample(metric string, value float64, labels ...string) *model.Sample {
	m := seriesLabels(labels...)
	m[model.MetricNameLabel] = model.LabelValue(metric)
	return &model.Sample{Metric: m, Value: model.SampleValue(value)}
}

// factFor returns the last fact collected for a field, or nil.
func factFor(result SourceResult, field string) *Fact {
	var found *Fact
	for i := range result.Facts {
		if result.Facts[i].Field == field {
			found = &result.Facts[i]
		}
	}
	return found
}

// TestReducedValueCarriesItsOwnSeriesAge pins the pairing between a reduced value and the
// age it is stamped with.
//
// The value query folds every series for a service into one sample -- the largest, for
// replication lag -- and the fact has to carry the age of the series that sample came
// from. Dating it from the oldest series of the set instead reports an age no contributing
// series ever had, in whichever direction the set happens to be arranged.
func TestReducedValueCarriesItsOwnSeriesAge(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

	// Two series for one service -- one per peer, as replication lag is emitted -- scraped
	// 30s and 4h ago respectively.
	lag := map[string]model.Vector{
		metricReplicationLag: {
			lagSample(30, "member_idx", "peer-a"),
			lagSample(4*60*60, "member_idx", "peer-b"),
		},
	}

	t.Run("the max is fresh", func(t *testing.T) {
		t.Parallel()

		vm := stubVM{lag: lag, values: map[string]model.Vector{
			metricReplicationLag: {
				valueSample(metricReplicationLag, 900, "member_idx", "peer-a"),
				valueSample(metricReplicationLag, 100, "member_idx", "peer-b"),
			},
		}}
		src := metricsSource{vm: vm, l: logrus.WithField("test", t.Name()), now: now}
		fact := factFor(src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}}), fieldReplicationLag)

		require.NotNil(t, fact, "expected a replication-lag fact")
		assert.InDelta(t, 900.0, fact.Value, 0.001)
		require.NotNil(t, fact.ObservedAt)
		assert.Equal(t, now.Add(-30*time.Second), fact.ObservedAt.UTC(),
			"the largest lag came from the 30s-old series, so that is its age")
	})

	t.Run("the max is stale", func(t *testing.T) {
		t.Parallel()

		vm := stubVM{lag: lag, values: map[string]model.Vector{
			metricReplicationLag: {
				valueSample(metricReplicationLag, 100, "member_idx", "peer-a"),
				valueSample(metricReplicationLag, 900, "member_idx", "peer-b"),
			},
		}}
		src := metricsSource{vm: vm, l: logrus.WithField("test", t.Name()), now: now}
		fact := factFor(src.collect(t.Context(), []*models.Service{{ServiceID: testServiceID}}), fieldReplicationLag)

		require.NotNil(t, fact, "expected a replication-lag fact")
		assert.InDelta(t, 900.0, fact.Value, 0.001)
		require.NotNil(t, fact.ObservedAt)

		// Still reported, and still dated 4 hours old, so fieldSet.live drops it rather
		// than the projection reading a stale maximum as a statement about now.
		assert.Equal(t, now.Add(-4*time.Hour), fact.ObservedAt.UTC(),
			"the largest lag came from the 4h-old series, so that is its age")
	})
}

// TestAbandonedSeriesDoesNotStaleTheLiveOne pins the fix for a live service reporting DOWN.
//
// An exporter's label set changes over its own lifetime: mongodb_up is emitted without
// cluster_role until the exporter can determine the role, at which point the series is
// abandoned and another begins under the same service_id. Both stay inside metricsLookback,
// so both come back, and dating exporter_up from the oldest of them made the fact age out of
// volatileMaxAge while the live sample was seconds old. Measured against a 14-service
// sandbox: 6 of them DOWN, stably, every one of them up in PMM's own inventory.
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

// TestEveryQueryNarrowsToTheHighResolutionJob pins the other half of that pairing: the
// series a reduction may see.
//
// A mongodb_exporter is scraped at two resolutions, so a metric it emits on every scrape
// -- mongodb_up -- exists as an hr and an lr series per service. Reduced across both,
// exporter_up is dated by the lr series, whose age exceeds volatileMaxAge for most of every
// minute, and a healthy service reports DOWN half the time it is asked.
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
