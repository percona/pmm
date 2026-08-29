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

func lagSample(serviceID string, age float64) *model.Sample {
	return &model.Sample{
		Metric: model.Metric{"service_id": model.LabelValue(serviceID)},
		Value:  model.SampleValue(age),
	}
}

// TestReducedValueIsDatedByTheOldestSeries pins the pairing between a reduced value and
// the age it is stamped with.
//
// The value query folds every series for a service into one sample -- the largest, for
// replication lag -- and it chooses that winner without reference to which series was
// freshest. So the age has to come from the oldest series, or a stale maximum would be
// reported as a statement about now.
func TestReducedValueIsDatedByTheOldestSeries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	const serviceID = "svc-1"

	vm := stubVM{
		lag: map[string]model.Vector{
			// Two series for one service: one scraped 30s ago, one 4 hours ago.
			metricReplicationLag: {
				lagSample(serviceID, 30),
				lagSample(serviceID, 4*60*60),
			},
		},
		values: map[string]model.Vector{
			// reduceMax keeps the larger lag, which is the one from the stale series.
			metricReplicationLag: {
				&model.Sample{
					Metric: model.Metric{"service_id": model.LabelValue(serviceID)},
					Value:  model.SampleValue(900),
				},
			},
		},
	}

	src := metricsSource{vm: vm, l: logrus.WithField("test", t.Name()), now: now}
	result := src.collect(t.Context(), []*models.Service{{ServiceID: serviceID}})

	var lag *Fact
	for i := range result.Facts {
		if result.Facts[i].Field == fieldReplicationLag {
			lag = &result.Facts[i]
		}
	}
	require.NotNil(t, lag, "expected a replication-lag fact")
	assert.InDelta(t, 900.0, lag.Value, 0.001)
	require.NotNil(t, lag.ObservedAt)

	// Dated by the 4-hour-old series, not the 30-second-old one.
	assert.Equal(t, now.Add(-4*time.Hour), lag.ObservedAt.UTC(),
		"a reduced value must carry the oldest contributing series' age")
}
