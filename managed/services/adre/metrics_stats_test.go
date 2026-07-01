// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeSeriesStats(t *testing.T) {
	t.Parallel()

	stats := computeSeriesStats([]float64{10, 20, 30, 40, 100})
	assert.Equal(t, 10.0, stats.Min)  //nolint:testifylint
	assert.Equal(t, 100.0, stats.Max) //nolint:testifylint
	assert.InDelta(t, 40.0, stats.Mean, 0.001)
	assert.Equal(t, 30.0, stats.Median) //nolint:testifylint
	assert.Equal(t, 20.0, stats.P25)    //nolint:testifylint
	assert.Equal(t, 40.0, stats.P75)    //nolint:testifylint
}

func TestFindChangePoints(t *testing.T) {
	t.Parallel()

	ts := []time.Time{
		time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 24, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
	}
	vals := []float64{10, 12, 100}
	cps := findChangePoints(ts, vals, 1)
	require.Len(t, cps, 1)
	assert.Equal(t, "2026-05-24T12:00:00Z", cps[0].Bucket)
	assert.InDelta(t, 88, cps[0].Delta, 0.001)
}

func TestFindAnomalies(t *testing.T) {
	t.Parallel()

	ts := make([]time.Time, 20)
	vals := make([]float64, 20)
	for i := range vals {
		ts[i] = time.Date(2026, 5, 24, 10, i, 0, 0, time.UTC)
		vals[i] = 10
	}
	vals[19] = 1000
	anomalies := findAnomalies(ts, vals, defaultZScoreThreshold)
	require.NotEmpty(t, anomalies)
	assert.Greater(t, anomalies[0].ZScore, defaultZScoreThreshold)
}
