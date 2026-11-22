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

		got := shapeSeries("rate(x[5m])", series, 0, nil, "", "")

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

		got := shapeSeries("x", series, 1000, nil, "", "")

		assert.LessOrEqual(t, len(got.Series), maxSeriesShown)
		assert.Greater(t, len(got.Series), defaultSeries)
		assert.Len(t, shapeSeries("x", series, 5, nil, "", "").Series, 5)
	})

	t.Run("FewSeriesAreNotTruncated", func(t *testing.T) {
		t.Parallel()

		got := shapeSeries("x", []metricSeries{mkSeries(map[string]string{"a": "1"}, 3, constant(1))}, 0, nil, "", "")

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

		got := shapeSeries("x", series, 0, nil, "", "")

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

	t.Run("InternalLabelsThatTellSeriesApart", func(t *testing.T) {
		t.Parallel()

		// sum by (node_id) (rate(node_cpu_seconds_total[5m])) over three nodes.
		byNode := make([]metricSeries, 3)
		for i := range byNode {
			byNode[i] = mkSeries(map[string]string{"node_id": "node-" + strconv.Itoa(i)}, 2, constant(float64(i)))
		}

		got := shapeSeries("x", byNode, 0, nil, "", "")

		assert.Empty(t, got.SharedLabels)
		require.Len(t, got.Series, 3)
		for i, s := range got.Series {
			assert.Equal(t, map[string]string{"node_id": "node-" + strconv.Itoa(2-i)}, s.Labels)
		}
		assert.Contains(t, got.render(), `{node_id="node-2"} min=2`)

		// Two series that only node_id and instance tell apart: those two are kept on every series, the equal job is not.
		mixed := []metricSeries{
			mkSeries(map[string]string{"service_name": "a", "node_id": "n1", "instance": "i1", "job": "j"}, 2, constant(3)),
			mkSeries(map[string]string{"service_name": "a", "node_id": "n2", "instance": "i2", "job": "j"}, 2, constant(2)),
			mkSeries(map[string]string{"service_name": "b", "node_id": "n3", "instance": "i3", "job": "j"}, 2, constant(1)),
		}

		got = shapeSeries("x", mixed, 0, nil, "", "")

		assert.Empty(t, got.SharedLabels)
		require.Len(t, got.Series, 3)
		assert.Equal(t, map[string]string{"service_name": "a", "node_id": "n1", "instance": "i1"}, got.Series[0].Labels)
		assert.Equal(t, map[string]string{"service_name": "a", "node_id": "n2", "instance": "i2"}, got.Series[1].Labels)
		assert.Equal(t, map[string]string{"service_name": "b", "node_id": "n3", "instance": "i3"}, got.Series[2].Labels)
	})

	t.Run("RawPointsWhenTheyFit", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 3)
		for i := range series {
			series[i] = mkSeries(map[string]string{"n": strconv.Itoa(i)}, 50, constant(float64(i)))
		}

		got := shapeSeries("x", series, 0, nil, "", "")

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

		got := shapeSeries("x", series, 0, nil, "", "")

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

		got := shapeSeries("x", series, 0, nil, "", "")

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

		got := shapeSeries("x", series, maxSeriesShown, nil, "", "")

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

		got := shapeSeries("x", series, 0, nil, "", "")

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
		assert.Contains(t, got.Note, "null in points")
		assert.Contains(t, got.render(), "2 values")
		validateMetricsOutput(t, got)
	})

	t.Run("NonFiniteSummarizedNote", func(t *testing.T) {
		t.Parallel()

		series := make([]metricSeries, 3)
		for i := range series {
			series[i] = mkSeries(map[string]string{"n": strconv.Itoa(i)}, 1000, func(j int) float64 {
				if j == 999 && i == 0 {
					return math.NaN()
				}
				return 1
			})
		}

		got := shapeSeries("x", series, 0, nil, "", "")

		require.True(t, got.Summarized)
		assert.Contains(t, got.Note, "1 non-finite values")
		assert.NotContains(t, got.Note, "in points")
		assert.Nil(t, got.Series[0].Last)
	})

	t.Run("OversizedSeriesLeavesNone", func(t *testing.T) {
		t.Parallel()

		series := []metricSeries{mkSeries(map[string]string{"query": strings.Repeat("x", 20<<10)}, 2, constant(1))}

		got := shapeSeries("x", series, 0, nil, "", "")

		assert.Empty(t, got.Series)
		assert.NotNil(t, got.Series)
		assert.True(t, got.Truncated)
		assert.Equal(t, 1, got.TotalSeries)
		assert.Contains(t, got.Hint, "labels are too large to show")
		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		validateMetricsOutput(t, got)
	})

	t.Run("NonFiniteLastAndNoFinitePoint", func(t *testing.T) {
		t.Parallel()

		series := []metricSeries{
			mkSeries(map[string]string{"a": "nan"}, 2, constant(math.NaN())),
			mkSeries(map[string]string{"a": "tail"}, 2, func(i int) float64 { return []float64{5, math.Inf(-1)}[i] }),
			mkSeries(map[string]string{"a": "ok"}, 2, constant(1)),
		}

		got := shapeSeries("x", series, 0, nil, "", "")

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

		got := shapeSeries("x", series, 0, nil, "", "")

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
			got := shapeSeries("x", in, 0, nil, "", "")

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

		got := shapeSeries(expr, series, 40, nil, "", "")

		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		assert.Contains(t, got.Hint, "topk(10, <expr>)")
		assert.NotEmpty(t, got.Series)
	})
}

func TestShapeSeriesBudgetIncludesWindowAndStep(t *testing.T) {
	t.Parallel()

	series := make([]metricSeries, 100)
	for i := range series {
		label := fmt.Sprintf("%03d-%s", i, strings.Repeat("x", 400))
		series[i] = mkSeries(map[string]string{"query": label}, 2, constant(float64(i)))
	}
	w := &window{From: budgetStart, To: budgetStart.Add(time.Hour)}

	// Find the series count that fits without the window and step, then check the real one is smaller or equal and fits.
	bare := shapeSeries("x", series, maxSeriesShown, nil, "", "")
	got := shapeSeries("x", series, maxSeriesShown, w, "2s", "")

	assert.True(t, got.Truncated)
	assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
	assert.Contains(t, got.render(), "Window:")
	assert.Contains(t, got.render(), "Step: 2s")
	assert.Equal(t, w, got.Window)
	assert.LessOrEqual(t, len(got.Series), len(bare.Series))
}

func TestShapeSeriesBudgetBoundary(t *testing.T) {
	t.Parallel()

	// Pad labels so that the bare result is just under 16 KiB: adding the window and step must then push out one more series.
	w := &window{From: budgetStart, To: budgetStart.Add(time.Hour)}
	for pad := 300; pad < 600; pad++ {
		series := make([]metricSeries, 100)
		for i := range series {
			series[i] = mkSeries(map[string]string{"q": fmt.Sprintf("%03d-%s", i, strings.Repeat("x", pad))}, 2, constant(float64(i)))
		}
		bare := shapeSeries("x", series, maxSeriesShown, nil, "", "")
		if sizeOf(t, bare) < maxMetricsBytes-100 {
			continue
		}
		got := shapeSeries("x", series, maxSeriesShown, w, "2s", "")
		assert.Less(t, len(got.Series), len(bare.Series), "pad %d", pad)
		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		return
	}
	t.Fatal("no padding put the bare result within 100 bytes of the budget")
}

func TestShapeSeriesKeepsTheMostThatFit(t *testing.T) {
	t.Parallel()

	for pad := 0; pad < 2000; pad += 197 {
		series := make([]metricSeries, maxSeriesShown)
		for i := range series {
			series[i] = mkSeries(map[string]string{"q": fmt.Sprintf("%03d-%s", i, strings.Repeat("x", pad))}, 2, constant(float64(i)))
		}
		got := shapeSeries("x", series, maxSeriesShown, nil, "", "")
		assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes, "pad %d", pad)
		if len(got.Series) == maxSeriesShown {
			continue
		}

		kept := make([]rankedSeries, len(got.Series)+1)
		for i := range kept {
			kept[i] = rankSeries(series[maxSeriesShown-1-i])
		}
		head := metricsResult{Query: "x"}
		assert.Greater(t, sizeOf(t, buildMetricsResult(head, kept, maxSeriesShown, false)), maxMetricsBytes, "pad %d: one more series does not fit", pad)
	}
}

func TestShapeSeriesLargeLabelsTakeBoundedTime(t *testing.T) {
	t.Parallel()

	big := strings.Repeat("x", 26<<10)
	for _, tc := range []struct {
		name   string
		labels func(i int) map[string]string
		fit    int
	}{
		// Every series has a label of its own that is over 16 KiB alone.
		{name: "LabelsOfTheirOwn", labels: func(i int) map[string]string { return map[string]string{"q": fmt.Sprintf("%03d-%s", i, big)} }},
		// Every other series has the same 4 KiB label, which no two series shown can share.
		{name: "RepeatedLabels", fit: 2, labels: func(i int) map[string]string {
			if i%2 == 0 {
				return map[string]string{"n": strconv.Itoa(i)}
			}
			return map[string]string{"n": strconv.Itoa(i), "q": big[:4<<10]}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			series := make([]metricSeries, maxSeriesShown)
			for i := range series {
				series[i] = mkSeries(tc.labels(i), 2, constant(float64(i)))
			}

			began := time.Now()
			got := shapeSeries("x", series, maxSeriesShown, nil, "", "")
			elapsed := time.Since(began)

			assert.Less(t, elapsed, time.Second)
			assert.Len(t, got.Series, tc.fit)
			assert.True(t, got.Truncated)
			assert.LessOrEqual(t, sizeOf(t, got), maxMetricsBytes)
		})
	}
}

func TestMetricsResultRender(t *testing.T) {
	t.Parallel()

	got := shapeSeries("up", []metricSeries{
		mkSeries(map[string]string{"service_name": "a\nb", "env": "prod"}, 2, constant(1.23456)),
		mkSeries(map[string]string{"service_name": "c", "env": "prod"}, 2, constant(7)),
	}, 0, nil, "15s", "")

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
		{"JustUnder2000Seconds", 1999 * time.Second, 0, time.Second},
		{"Exactly2000Seconds", 2000 * time.Second, 0, 2 * time.Second},
		{"FiveHours", 5 * time.Hour, 0, 10 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := chooseStep(tc.span, tc.requested)
			assert.Equal(t, tc.want, step)
			if tc.span > 0 && tc.requested == 0 {
				assert.LessOrEqual(t, int(tc.span/step)+1, maxPoints, "the range and both of its ends")
			}
		})
	}
}

func TestRankSeriesAverageOfLargeValues(t *testing.T) {
	t.Parallel()

	start := time.Unix(1700000000, 0).UTC()
	s := metricSeries{Labels: map[string]string{"service_name": "db1"}, Points: []point{
		{T: start, V: 1e308},
		{T: start.Add(time.Second), V: 1e308},
		{T: start.Add(2 * time.Second), V: math.NaN()},
	}}
	r := rankSeries(s)
	require.NotNil(t, r.summary.Avg)
	assert.InEpsilon(t, 1e308, *r.summary.Avg, 1e-9, "the sum overflows; the average does not")

	res := shapeSeries("vector(1e308)", []metricSeries{s}, 0, &window{From: start, To: start.Add(time.Hour)}, "1s", "")
	_, err := json.Marshal(res)
	require.NoError(t, err)

	// Each third of the largest float64 rounds up, so even their sum overflows.
	s.Points = []point{
		{T: start, V: math.MaxFloat64},
		{T: start.Add(time.Second), V: math.MaxFloat64},
		{T: start.Add(2 * time.Second), V: math.MaxFloat64},
	}
	r = rankSeries(s)
	require.NotNil(t, r.summary.Avg)
	assert.InEpsilon(t, math.MaxFloat64, *r.summary.Avg, 1e-9, "three overflowing samples still average finite")

	res = shapeSeries("vector(1e308)", []metricSeries{s}, 0, &window{From: start, To: start.Add(time.Hour)}, "1s", "")
	_, err = json.Marshal(res)
	require.NoError(t, err)
}

func TestFmtNum3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{1.23456, "1.235"},
		{0.0004, "0"},
		{-0.0004, "0"},
		{2.5, "2.5"},
		{1.2e17, "120000000000000000"},
		{9.3e15, "9300000000000000"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, fmtNum3(tc.in), "%v", tc.in)
	}
}

func TestAnswerStep(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		matched matchedSeries
		span    time.Duration
		step    time.Duration
		raised  bool
	}{
		{name: "AFewSeriesFit", matched: matchedSeries{count: 10, labelBytes: 3000}, span: time.Hour, step: 2 * time.Second},
		{name: "ManySeriesRaiseTheStep", matched: matchedSeries{count: 500, labelBytes: 150000}, span: 6 * time.Hour, step: 11 * time.Second, raised: true},
		{name: "TheCapAtTheDefaultStep", matched: matchedSeries{count: 2000, labelBytes: 600000}, span: 6 * day, step: 260 * time.Second, raised: true},
		{name: "ALargeStepFits", matched: matchedSeries{count: 500, labelBytes: 150000}, span: 6 * time.Hour, step: 10 * time.Minute},
		{name: "NoSeries", span: 6 * time.Hour, step: 11 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := answerStep(tc.matched, tc.span, tc.step)
			require.True(t, ok)
			if !tc.raised {
				assert.Equal(t, tc.step, got)
				return
			}
			assert.Greater(t, answerBytes(tc.matched, tc.span, tc.step), maxRangeAnswerBytes)
			assert.Equal(t, got, ceilSecond(got), "a whole number of seconds")
			assert.LessOrEqual(t, answerBytes(tc.matched, tc.span, got), maxRangeAnswerBytes)
			assert.Greater(t, answerBytes(tc.matched, tc.span, got-time.Second), maxRangeAnswerBytes, "the smallest step that fits")
		})
	}

	t.Run("LabelsAloneOverTheCap", func(t *testing.T) {
		t.Parallel()

		_, ok := answerStep(matchedSeries{count: 2000, labelBytes: 4 << 20}, 6*time.Hour, 11*time.Second)
		assert.False(t, ok)
	})
}

func TestAnswerStepFitsTheReadCap(t *testing.T) {
	t.Parallel()

	// VictoriaMetrics writes each value in Go's shortest float format, up to 17 significant digits, as in 0.0033333333333333335.
	const value = `"0.0033333333333333335"`
	require.Len(t, value, 23)

	for _, tc := range []struct {
		name  string
		count int
		span  time.Duration
	}{
		{name: "FiveHundredOverSixHours", count: 500, span: 6 * time.Hour},
		{name: "TwoThousandOverSixDays", count: 2000, span: 6 * day},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var matched matchedSeries
			metrics := make([]string, tc.count)
			for i := range metrics {
				labels := map[string]string{"__name__": "mysql_global_status_questions", "service_name": fmt.Sprintf("db%d", i)}
				matched.count++
				for k, v := range labels {
					matched.labelBytes += len(k) + len(v) + answerLabelBytes
				}
				metrics[i] = string(mustJSON(t, labels))
			}
			step, ok := answerStep(matched, tc.span, chooseStep(tc.span, 0))
			require.True(t, ok)

			start := time.Unix(1700000000, 0)
			points := make([]string, int(tc.span/step)+1)
			for i := range points {
				points[i] = "[" + strconv.FormatInt(start.Add(time.Duration(i)*step).Unix(), 10) + "," + value + "]"
			}
			values := strings.Join(points, ",")
			result := make([]string, tc.count)
			for i, m := range metrics {
				result[i] = `{"metric":` + m + `,"values":[` + values + `]}`
			}
			answer := `{"status":"success","data":{"resultType":"matrix","result":[` + strings.Join(result, ",") + `]}}`

			assert.LessOrEqual(t, len(answer), maxVMResponseBytes, "at step %s", step)
		})
	}
}
