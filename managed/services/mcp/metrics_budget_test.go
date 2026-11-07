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

package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var budgetStart = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// mkSeries returns a series of n points that step a minute apart; value gives each one.
func mkSeries(labels map[string]string, n int, value func(i int) float64) metricSeries {
	points := make([]point, n)
	for i := range points {
		points[i] = point{T: budgetStart.Add(time.Duration(i) * time.Minute), V: value(i)}
	}
	return metricSeries{Labels: labels, Points: points}
}

func constant(v float64) func(int) float64 {
	return func(int) float64 { return v }
}

// validateMetricsOutput validates r against the output schema addTool generates for it.
func validateMetricsOutput(t *testing.T, r metricsResult) {
	t.Helper()

	schema, err := jsonschema.For[metricsResult](nil)
	require.NoError(t, err)
	validateOutput(t, &mcp.Tool{OutputSchema: schema}, r)
}

func sizeOf(t *testing.T, r metricsResult) int {
	t.Helper()

	return len(r.render()) + len(mustJSON(t, r))
}

func TestShapeSeries(t *testing.T) {
	t.Parallel()

	t.Run("TopByMax", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 300)
		for i := range series {
			series[i] = mkSeries(map[string]string{"service_name": fmt.Sprintf("svc-%03d", i)}, 3, constant(float64(i)))
		}
		series[7] = mkSeries(map[string]string{"service_name": "huge"}, 3, constant(1e6))

		got := shapeSeries("rate(x[5m])", series, 0)

		require.Len(t, got.Series, defaultSeries)
		assert.Equal(t, "huge", got.Series[0].Labels["service_name"])
		for i := 1; i < len(got.Series); i++ {
			assert.Greater(t, *got.Series[i-1].Max, *got.Series[i].Max)
		}
		assert.Equal(t, "svc-299", got.Series[1].Labels["service_name"])
		assert.Equal(t, 300, got.TotalSeries)
		assert.True(t, got.Truncated)
		assert.Contains(t, got.Hint, "topk(10, rate(x[5m]))")
		assert.Contains(t, got.Hint, "sum by (service_name) (rate(x[5m]))")
		assert.Contains(t, got.Hint, "280 series")
		validateMetricsOutput(t, got)
	})

	t.Run("LimitIsClamped", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 300)
		for i := range series {
			series[i] = mkSeries(map[string]string{"n": strconv.Itoa(i)}, 1, constant(float64(i)))
		}

		got := shapeSeries("x", series, 1000)

		assert.LessOrEqual(t, len(got.Series), maxSeriesShown)
		assert.Greater(t, len(got.Series), defaultSeries)
		assert.Len(t, shapeSeries("x", series, 5).Series, 5)
	})

	t.Run("FewSeriesAreNotTruncated", func(t *testing.T) {
		t.Parallel()

		got := shapeSeries("x", []metricSeries{mkSeries(map[string]string{"a": "1"}, 3, constant(1))}, 0)

		assert.False(t, got.Truncated)
		assert.Empty(t, got.Hint)
		assert.Equal(t, 1, got.TotalSeries)
	})

	t.Run("InternalAndSharedLabels", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 3)
		for i := range series {
			labels := map[string]string{"env": "prod", "service_name": fmt.Sprintf("svc-%d", i)}
			for _, k := range internalLabels {
				labels[k] = "internal-" + strconv.Itoa(i)
			}
			if i < 2 {
				labels["region"] = "eu"
			}
			series[i] = mkSeries(labels, 2, constant(float64(i)))
		}

		got := shapeSeries("x", series, 0)

		assert.Equal(t, map[string]string{"env": "prod"}, got.SharedLabels)
		for _, s := range got.Series {
			for _, k := range internalLabels {
				assert.NotContains(t, s.Labels, k)
			}
			assert.NotContains(t, s.Labels, "env")
			assert.Contains(t, s.Labels, "service_name")
		}
		assert.Contains(t, got.Series[2].Labels, "region")
		assert.Contains(t, got.render(), `Shared labels: {env="prod"}`)
		assert.NotContains(t, got.render(), "internal-")
		assert.NotContains(t, string(mustJSON(t, got)), "internal-")
	})

	t.Run("RawPointsWhenTheyFit", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 3)
		for i := range series {
			series[i] = mkSeries(map[string]string{"n": strconv.Itoa(i)}, 50, constant(float64(i)))
		}

		got := shapeSeries("x", series, 0)

		assert.False(t, got.Summarized)
		for _, s := range got.Series {
			assert.Len(t, s.Points, 50)
		}
		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		assert.NotContains(t, got.render(), "2025-01-01T", "raw points belong in the structured content only")
		validateMetricsOutput(t, got)
	})

	t.Run("SummarizedOverMaxPoints", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 20)
		for i := range series {
			series[i] = mkSeries(map[string]string{"n": strconv.Itoa(i)}, 200, constant(float64(i)))
		}

		got := shapeSeries("x", series, 0)

		assert.True(t, got.Summarized)
		assert.Len(t, got.Series, 20)
		for _, s := range got.Series {
			assert.Empty(t, s.Points)
		}
		assert.NotContains(t, string(mustJSON(t, got)), `"points"`)
		validateMetricsOutput(t, got)
	})

	t.Run("SummarizedWhenPointsExceedBytes", func(t *testing.T) {
		t.Parallel()

		series := []metricSeries{
			mkSeries(map[string]string{"n": "0"}, 1000, constant(1.5)),
			mkSeries(map[string]string{"n": "1"}, 1000, constant(2.5)),
		}

		got := shapeSeries("x", series, 0)

		assert.True(t, got.Summarized, "2,000 points are within maxPoints but far over 16 KiB")
		assert.Len(t, got.Series, 2)
		assert.False(t, got.Truncated)
		assert.Empty(t, got.Series[0].Points)
		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
	})

	t.Run("OverlongSummariesKeepFewerSeries", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 100)
		for i := range series {
			label := fmt.Sprintf("%03d-%s", i, strings.Repeat("x", 400))
			series[i] = mkSeries(map[string]string{"query": label}, 2, constant(float64(i)))
		}

		got := shapeSeries("x", series, maxSeriesShown)

		assert.True(t, got.Summarized)
		assert.Less(t, len(got.Series), 100)
		assert.NotEmpty(t, got.Series)
		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		assert.Equal(t, 100, got.TotalSeries)
		assert.True(t, got.Truncated)
		assert.Contains(t, got.Hint, fmt.Sprintf("%d series are left out", 100-len(got.Series)))
		assert.True(t, strings.HasPrefix(got.Series[0].Labels["query"], "099-"), "the largest series stays")
		assert.True(t, strings.HasPrefix(got.Series[len(got.Series)-1].Labels["query"], fmt.Sprintf("%03d-", 100-len(got.Series))))
		validateMetricsOutput(t, got)
	})

	t.Run("NonFiniteValuesAreNull", func(t *testing.T) {
		t.Parallel()

		values := []float64{1, math.NaN(), math.Inf(1), 3}
		series := []metricSeries{mkSeries(map[string]string{"a": "1"}, len(values), func(i int) float64 { return values[i] })}

		got := shapeSeries("x", series, 0)

		b, err := json.Marshal(got)
		require.NoError(t, err)
		assert.Equal(t, 2, strings.Count(string(b), `"v":null`))
		s := got.Series[0]
		require.NotNil(t, s.Min)
		assert.InDelta(t, 1, *s.Min, 0)
		assert.InDelta(t, 2, *s.Avg, 0)
		assert.InDelta(t, 3, *s.Max, 0)
		assert.InDelta(t, 3, *s.Last, 0)
		assert.Contains(t, got.Note, "2 values")
		assert.Contains(t, got.render(), "2 values")
		validateMetricsOutput(t, got)
	})

	t.Run("NonFiniteLastAndNoFinitePoint", func(t *testing.T) {
		t.Parallel()

		series := []metricSeries{
			mkSeries(map[string]string{"a": "nan"}, 2, constant(math.NaN())),
			mkSeries(map[string]string{"a": "tail"}, 2, func(i int) float64 { return []float64{5, math.Inf(-1)}[i] }),
			mkSeries(map[string]string{"a": "ok"}, 2, constant(1)),
		}

		got := shapeSeries("x", series, 0)

		require.Len(t, got.Series, 3)
		assert.Equal(t, []string{"tail", "ok", "nan"}, []string{got.Series[0].Labels["a"], got.Series[1].Labels["a"], got.Series[2].Labels["a"]})
		assert.Nil(t, got.Series[0].Last)
		assert.InDelta(t, 5, *got.Series[0].Max, 0)
		assert.Nil(t, got.Series[2].Min)
		assert.Nil(t, got.Series[2].Avg)
		assert.Nil(t, got.Series[2].Max)
		assert.Contains(t, got.Note, "3 values")
		_, err := json.Marshal(got)
		require.NoError(t, err)
	})

	t.Run("AllInternalLabels", func(t *testing.T) {
		t.Parallel()

		series := []metricSeries{mkSeries(map[string]string{"agent_id": "a", "job": "j"}, 2, constant(1))}

		got := shapeSeries("x", series, 0)

		require.Len(t, got.Series, 1)
		assert.NotNil(t, got.Series[0].Labels)
		assert.Empty(t, got.Series[0].Labels)
		b := string(mustJSON(t, got))
		assert.Contains(t, b, `"labels":{}`)
		assert.NotContains(t, b, "shared_labels")
		validateMetricsOutput(t, got)
	})

	t.Run("NoSeries", func(t *testing.T) {
		t.Parallel()

		for _, in := range [][]metricSeries{nil, {}} {
			got := shapeSeries("x", in, 0)

			assert.NotNil(t, got.Series)
			assert.Zero(t, got.TotalSeries)
			assert.False(t, got.Truncated)
			assert.Contains(t, string(mustJSON(t, got)), `"series":[]`)
			validateMetricsOutput(t, got)
		}
	})

	t.Run("LongExpressionStaysWithinBudget", func(t *testing.T) {
		t.Parallel()

		expr := "sum(" + strings.Repeat("a", 4000) + ")"
		series := make([]metricSeries, 50)
		for i := range series {
			series[i] = mkSeries(map[string]string{"n": strconv.Itoa(i)}, 2, constant(float64(i)))
		}

		got := shapeSeries(expr, series, 40)

		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		assert.Contains(t, got.Hint, "topk(10, <expr>)")
		assert.NotEmpty(t, got.Series)
	})
}

func TestMetricsResultRender(t *testing.T) {
	t.Parallel()

	got := shapeSeries("up", []metricSeries{
		mkSeries(map[string]string{"service_name": "a\nb", "env": "prod"}, 2, constant(1.23456)),
		mkSeries(map[string]string{"service_name": "c", "env": "prod"}, 2, constant(7)),
	}, 0)
	got.Step = "15s"

	text := got.render()

	assert.Contains(t, text, "Query: up")
	assert.Contains(t, text, "Step: 15s")
	assert.Contains(t, text, `Shared labels: {env="prod"}`)
	assert.Contains(t, text, `{service_name="c"} min=7 avg=7 max=7 last=7`)
	assert.Contains(t, text, `{service_name="a\nb"} min=1.235`)
	assert.Equal(t, 1, strings.Count(text, "a\\nb"), "a newline in a label value stays on one line")
}

func TestChooseStep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		span      time.Duration
		requested time.Duration
		want      time.Duration
	}{
		{"OneHour", time.Hour, 0, 2 * time.Second},
		{"SixDaysRoundsUp", 6 * day, 0, 260 * time.Second},
		{"RequestedWins", time.Hour, 5 * time.Minute, 5 * time.Minute},
		{"FloorIsOneSecond", time.Minute, 0, time.Second},
		{"RequestedRoundsUp", time.Minute, 1500 * time.Millisecond, 2 * time.Second},
		{"InstantHasNoStep", 0, 0, 0},
		{"InstantIgnoresRequested", 0, time.Minute, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, chooseStep(tc.span, tc.requested))
		})
	}
}
