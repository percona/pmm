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

type stubVM struct {
	lag    map[string]model.Vector
	values map[string]model.Vector
}

func (s stubVM) Query(_ context.Context, query string, _ time.Time, _ ...v1.Option) (model.Value, v1.Warnings, v1.Infos, error) {
	for metric, vector := range s.lag {
		if strings.HasPrefix(query, "lag("+metric+"{") {
			return vector, nil, nil, nil
		}
	}
	for metric, vector := range s.values {
		if strings.HasPrefix(query, "last_over_time("+metric+"{") {
			return vector, nil, nil, nil
		}
	}
	return model.Vector{}, nil, nil, nil
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

func TestReducedValueCarriesItsOwnSeriesAge(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

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

		// Dated 4h old so fieldSet.live drops it instead of reporting a stale maximum as current.
		assert.Equal(t, now.Add(-4*time.Hour), fact.ObservedAt.UTC(),
			"the largest lag came from the 4h-old series, so that is its age")
	})
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
