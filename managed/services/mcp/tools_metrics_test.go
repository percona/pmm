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
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const vmAPI = "/graph/api/datasources/proxy/uid/metrics-uid/api/v1/"

const (
	vmQuery      = "GET " + vmAPI + "query"
	vmQueryRange = "GET " + vmAPI + "query_range"
	vmSeries     = "GET " + vmAPI + "series"
)

// metricsRoutes serves the Metrics datasource and the three VictoriaMetrics endpoints; extra adds or replaces routes.
func metricsRoutes(extra map[string]fixture) map[string]fixture {
	return withRoutes(map[string]fixture{
		"GET /graph/api/datasources/name/Metrics": {file: "datasource_metrics.json"},
		vmSeries:     {body: seriesBody(3)},
		vmQuery:      {body: vectorBody(2)},
		vmQueryRange: {body: matrixBody(2, 3, "db")},
	}, extra)
}

func seriesBody(n int) []byte {
	entries := make([]string, n)
	for i := range entries {
		entries[i] = fmt.Sprintf(`{"__name__":"m","id":"%d"}`, i)
	}
	return []byte(`{"status":"success","data":[` + strings.Join(entries, ",") + `]}`)
}

func vectorBody(n int) []byte {
	entries := make([]string, n)
	for i := range entries {
		entries[i] = fmt.Sprintf(`{"metric":{"service_name":"db%d","instance":"i"},"value":[1700000000,"%d"]}`, i, i)
	}
	return []byte(`{"status":"success","data":{"resultType":"vector","result":[` + strings.Join(entries, ",") + `]}}`)
}

// matrixBody returns n series of the given number of points each, whose service_name starts with prefix.
func matrixBody(n, points int, prefix string) []byte {
	entries := make([]string, n)
	for i := range entries {
		values := make([]string, points)
		for j := range values {
			values[j] = fmt.Sprintf(`[%d,"%d"]`, 1700000000+60*j, i+j)
		}
		entries[i] = fmt.Sprintf(`{"metric":{"service_name":"%s%d","instance":"i"},"values":[%s]}`, prefix, i, strings.Join(values, ","))
	}
	return []byte(`{"status":"success","data":{"resultType":"matrix","result":[` + strings.Join(entries, ",") + `]}}`)
}

func unixOf(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }

// requireInvalid checks that s refuses a call of the tool name as invalid input, with a message that holds want, before
// any request to fake.
func requireInvalid(t *testing.T, s *Service, fake *fakePMM, name string, args map[string]any, want string) {
	t.Helper()

	text, isError := callText(t, connect(t, s), name, args)
	require.True(t, isError, "%v: %s", args, text)
	assertToolError(t, text, codeInvalidInput, "")
	assert.Contains(t, text, want)
	assert.Zero(t, fake.requestCount(), "%v sent a request", args)
}

func TestQueryMetricsTool(t *testing.T) {
	t.Parallel()

	const questions = `rate(mysql_global_status_questions{service_name="db1"}[5m])`

	t.Run("Instant", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions})
		require.False(t, isError, text)

		assert.Equal(t, questions, res.Query)
		assert.Nil(t, res.Window)
		assert.Empty(t, res.Step)
		assert.Equal(t, 2, res.TotalSeries)
		require.Len(t, res.Series, 2)
		assert.Equal(t, "db1", res.Series[0].Labels["service_name"], "the largest maximum first")
		assert.NotContains(t, res.Series[0].Labels, "instance")

		reqs := fake.requestsTo(vmAPI + "query")
		require.Len(t, reqs, 1)
		p := queryOf(t, reqs[0])
		assert.Equal(t, questions, p.Get("query"))
		assert.Equal(t, unixOf(testNow), p.Get("time"))
		assert.Equal(t, "25s", p.Get("timeout"))
		assert.Equal(t, "300", p.Get("step"))
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"))
		fake.assertForwardedAuth(t)

		// The selector is counted over the span read: one step of 5m back for an instant query, plus the 5m window.
		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 1)
		sp := queryOf(t, series[0])
		assert.Equal(t, []string{`{__name__="mysql_global_status_questions", service_name="db1"}`}, sp["match[]"])
		assert.Equal(t, unixOf(testNow.Add(-5*time.Minute)), sp.Get("start"))
		assert.Equal(t, unixOf(testNow), sp.Get("end"))
		assert.Equal(t, "2001", sp.Get("limit"))
	})

	t.Run("InstantAtTime", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "time": "now-2h"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "query")
		require.Len(t, reqs, 1)
		assert.Equal(t, unixOf(testNow.Add(-2*time.Hour)), queryOf(t, reqs[0]).Get("time"))
	})

	t.Run("Range", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-2h"})
		require.False(t, isError, text)

		from := testNow.Add(-2 * time.Hour)
		require.NotNil(t, res.Window)
		assert.Equal(t, window{From: from, To: testNow}, *res.Window)
		assert.Equal(t, "4s", res.Step, "2h in at most 2000 points")
		require.Len(t, res.Series, 2)
		assert.Len(t, res.Series[0].Points, 3)

		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		p := queryOf(t, reqs[0])
		assert.Equal(t, unixOf(from), p.Get("start"))
		assert.Equal(t, unixOf(testNow), p.Get("end"))
		assert.Equal(t, "4", p.Get("step"))
		assert.Equal(t, "25s", p.Get("timeout"))
		assert.Empty(t, fake.requestsTo(vmAPI+"query"))

		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 1)
		sp := queryOf(t, series[0])
		assert.Equal(t, unixOf(from.Add(-5*time.Minute)), sp.Get("start"), "the span read includes the 5m window")
		assert.Equal(t, unixOf(testNow), sp.Get("end"))
	})

	t.Run("RangeWithStep", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-2h", "step": "10m"})
		require.False(t, isError, text)
		assert.Equal(t, "10m", res.Step)
		assert.Equal(t, "600", queryOf(t, fake.requestsTo(vmAPI + "query_range")[0]).Get("step"))
	})

	t.Run("RefusedExpressionSendsNothing", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics", map[string]any{"query": `{__name__=~"mysql_global_.*"}`}, "service_id, service_name")
		// A negative offset reads after the window the series are counted over.
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics",
			map[string]any{"query": `sum(rate(node_cpu_seconds_total[5m] offset -6d))`, "time": "now-6d"}, "a negative offset is not accepted")
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics",
			map[string]any{"query": `rate(node_cpu_seconds_total{node_name="n1"}[5m] offset -6d)`, "period_from": "now-145h", "period_to": "now-6d"},
			"a negative offset is not accepted")
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics",
			map[string]any{"query": questions, "period_from": "now-7d"}, "the cap is 7d")
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics", map[string]any{"query": "rate("}, "cannot parse")
	})

	t.Run("InstantRangeIsRefused", func(t *testing.T) {
		t.Parallel()

		ranges := []string{
			`mysql_up{service_name="db1"}[5m]`,
			`(mysql_up{service_name="db1"}[5m])`,
			`mysql_up{service_name="db1"}[5m] offset 1h`,
			`rate(mysql_global_status_questions{service_name="db1"}[5m])[1h:1m]`,
			`rate(mysql_global_status_questions{service_name="db1"}[5m])[1h:]`,
			`rate(mysql_global_status_questions{service_name="db1"}[5m])[1h:1m] offset 1h`,
			`rate(mysql_global_status_questions{service_name="db1"}[5m])[1h]`,
			`mysql_up{service_name="db1"}[:1m]`,
			`mysql_up{service_name="db1"}[:]`,
			`(mysql_up{service_name="db1"}[:1m]) offset 5m`,
		}
		for _, q := range ranges {
			fake := newFakePMM(t, metricsRoutes(nil))
			s := newQANService(t, fake, false)
			requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": q, "time": "now-1h"}, "wrap it in a rollup such as last_over_time() or rate(), or send a range query")

			s.metricRate.mu.Lock()
			assert.Empty(t, s.metricRate.buckets, "%s took a token", q)
			s.metricRate.mu.Unlock()
		}

		for _, q := range []string{
			`last_over_time(mysql_up{service_name="db1"}[5m])`,
			`sum(mysql_up{service_name="db1"}[5m])`,
			`mysql_up{service_name="db1"} offset 1h`,
			`max_over_time(rate(mysql_global_status_questions{service_name="db1"}[5m])[1h:1m])`,
		} {
			fake := newFakePMM(t, metricsRoutes(nil))
			_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": q})
			require.False(t, isError, "%s: %s", q, text)
			assert.Len(t, fake.requestsTo(vmAPI+"query"), 1, q)
		}

		for _, q := range ranges {
			fake := newFakePMM(t, metricsRoutes(nil))
			_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": q, "period_from": "now-1h", "step": "1m"})
			require.False(t, isError, "a range query of %s: %s", q, text)
			assert.Len(t, fake.requestsTo(vmAPI+"query_range"), 1, q)
		}
	})

	t.Run("CutToTwentySeries", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQuery: {body: vectorBody(300)}}))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions})
		require.False(t, isError, text)
		assert.Len(t, res.Series, 20)
		assert.Equal(t, 300, res.TotalSeries)
		assert.True(t, res.Truncated)
		assert.Contains(t, res.Hint, "280 series are left out; 20 are shown")
		assert.Contains(t, text, "280 series are left out")
	})

	t.Run("Limit", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQuery: {body: vectorBody(300)}}))
		s := newQANService(t, fake, false)
		for _, limit := range []int{101, -1} {
			requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "limit": limit}, "limit must be between 1 and 100")
		}
		res, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": questions, "limit": 50})
		require.False(t, isError, text)
		assert.Len(t, res.Series, 50)
		res, _, _ = callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": questions, "limit": 3})
		assert.Len(t, res.Series, 3)
	})

	t.Run("BadArguments", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "time": "now", "period_from": "now-1h"}, "use either time, or period_from and period_to")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "time": "now", "period_to": "now"}, "use either time")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h", "step": "1d"}, "step '1d' is not a duration")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h", "step": "5i"}, "step '5i' is not a duration")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h", "step": "1500ms"}, "whole number of seconds")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h", "step": "0s"}, "positive whole number")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "step": "30s"}, "range queries only")
		requireInvalid(t, s, fake, "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-6h", "step": "1s"},
			"at step 1s, the window of 6h is 21601 points per series; the cap is 2000: raise step to at least 11s, or shorten the window")
	})

	t.Run("PeriodToAloneReadsTheHourBefore", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		res, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": questions, "period_to": "now-10m"})
		require.False(t, isError, text)

		end := testNow.Add(-10 * time.Minute)
		require.NotNil(t, res.Window)
		assert.Equal(t, window{From: end.Add(-time.Hour), To: end}, *res.Window)
		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		p := queryOf(t, reqs[0])
		assert.Equal(t, unixOf(end.Add(-time.Hour)), p.Get("start"))
		assert.Equal(t, unixOf(end), p.Get("end"))
	})

	t.Run("RateLimited", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		args := map[string]any{"query": questions}
		key := keyOf(callerAuth{authorization: testToken})

		releases := make([]func(), 0, maxMetricCallsPerCaller)
		for range maxMetricCallsPerCaller {
			release, hit := s.metrics.acquire(key)
			require.Equal(t, limitNone, hit)
			releases = append(releases, release)
		}
		_, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", args)
		require.True(t, isError, text)
		assertToolError(t, text, codeRateLimited, "")
		assert.Contains(t, text, "at most 4 metric calls")
		assert.Zero(t, fake.requestCount())

		// A refused expression is reported before the limit.
		_, text, _ = callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": "rate("})
		assertToolError(t, text, codeInvalidInput, "")

		for _, release := range releases {
			release()
		}
		for i := range maxMetricCallsPerReplica {
			release, hit := s.metrics.acquire(fmt.Sprintf("other-%d", i))
			require.Equal(t, limitNone, hit)
			t.Cleanup(release)
		}
		_, text, isError = callAs[metricsResult](t, s, "pmm_query_metrics", args)
		require.True(t, isError, text)
		assertToolError(t, text, codeRateLimited, "")
		assert.Contains(t, text, "this replica already has 8 metric calls")
		assert.Zero(t, fake.requestCount())
	})

	t.Run("SlotIsReleased", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		// This test counts slots, not tokens: its calls outnumber the burst on a frozen clock.
		s.metricRate = newRateLimiter(60000, 1000)
		for range 2 * maxMetricCallsPerCaller {
			_, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": questions})
			require.False(t, isError, text)
		}
		// A failed call releases its slot too.
		bad := newFakePMM(t, nil)
		s.api = newTestService(t, bad).api
		for range 2 * maxMetricCallsPerCaller {
			_, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": questions})
			require.True(t, isError)
			assertToolError(t, text, codePMMUnavailable, "")
		}
	})

	t.Run("SeriesCap", func(t *testing.T) {
		t.Parallel()

		const hint = "this selector matches more than 2000 series over the span; narrow it with a service_name or node_name matcher, and aggregate"
		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(2001)}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h"})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, hint)
		assert.Len(t, fake.requestsTo(vmAPI+"series"), 1)
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"), "the query is refused before it runs")
		assert.Empty(t, fake.requestsTo(vmAPI+"query"))

		fake = newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(2000)}}))
		_, text, isError = callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h"})
		require.False(t, isError, text)
		assert.Len(t, fake.requestsTo(vmAPI+"query_range"), 1)
	})

	t.Run("SeriesCountPerSelector", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `mysql_up{service_name="db1"} / on() mysql_global_status_uptime{service_name="db1"}`,
		})
		require.False(t, isError, text)
		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 2)
		assert.Equal(t, []string{`{__name__="mysql_up", service_name="db1"}`}, queryOf(t, series[0])["match[]"])
		assert.Equal(t, []string{`{__name__="mysql_global_status_uptime", service_name="db1"}`}, queryOf(t, series[1])["match[]"])
	})

	t.Run("StepRisesForAnImplicitSubquery", func(t *testing.T) {
		t.Parallel()

		const nested = `max_over_time(sum by (service_name) (rate(mysql_global_status_questions{service_name="db1"}[5m]))[1h])`
		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)

		res, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": nested, "period_from": "now-15m"})
		require.False(t, isError, text)
		assert.Equal(t, "2s", res.Step, "1s would be 4800 points; the result says the step used")
		assert.Contains(t, res.Note, "The step was raised from 1s to 2s so that a rollup over an expression stays within 4000 points per series")
		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		assert.Equal(t, "2", queryOf(t, reqs[0]).Get("step"))

		// A step the caller gave is never raised.
		fake2 := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake2, false), fake2, "pmm_query_metrics",
			map[string]any{"query": nested, "period_from": "now-15m", "step": "1s"}, "raise step to at least 2s")

		res, text, isError = callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": nested, "period_from": "now-15m", "step": "3s"})
		require.False(t, isError, text)
		assert.Equal(t, "3s", res.Step)
	})

	t.Run("StepRisesForASubqueryWithoutAStep", func(t *testing.T) {
		t.Parallel()

		const peak = `max_over_time(rate(mysql_global_status_questions{service_name="db1"}[5m])[1h:])`
		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)

		res, text, isError := callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{"query": peak, "period_from": "now-1h"})
		require.False(t, isError, text)
		assert.Equal(t, "4s", res.Step, "[1h:] runs at the query's step, so it rises with it")
		assert.Contains(t, res.Note, "The step was raised from 2s to 4s so that a subquery's step is at least the span read ÷ 2000")
		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		assert.Equal(t, "4", queryOf(t, reqs[0]).Get("step"))

		// A rollup window inside [5m:] is read at the query's step too.
		res, text, isError = callAs[metricsResult](t, s, "pmm_query_metrics", map[string]any{
			"query":       `max_over_time(avg_over_time(mysql_global_status_questions{service_name="db1"}[90m])[5m:])`,
			"period_from": "now-24h",
		})
		require.False(t, isError, text)
		assert.Equal(t, "1m6s", res.Step)
		assert.Contains(t, res.Note, "The step was raised from 44s to 1m6s")
		assert.Contains(t, res.Note, "rollup windows read at most 2000 hours of samples per series")

		// A step the caller gave is never raised.
		fake2 := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake2, false), fake2, "pmm_query_metrics",
			map[string]any{"query": peak, "period_from": "now-1h", "step": "2s"}, "the subquery step 2s is too small")
	})

	t.Run("StepStaysWhenNothingNeedsRaising", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-15m"})
		require.False(t, isError, text)
		assert.Equal(t, "1s", res.Step)
	})

	t.Run("StepRisesForLongWindows", func(t *testing.T) {
		t.Parallel()

		const m = `mysql_global_status_questions{service_name="db1"}`
		const note = "so that rollup windows read at most 2000 hours of samples per series"
		for _, tc := range []struct {
			query, from, step, note string
		}{
			{query: `avg_over_time(` + m + `[90m])`, from: "now-24h", step: "1m5s", note: "The step was raised from 44s to 1m5s " + note},
			{query: `max_over_time(` + m + `[1d])`, from: "now-6d", step: "1h43m41s", note: "The step was raised from 4m20s to 1h43m41s " + note},
			{query: `increase(` + m + `[1d])`, from: "now-6d", step: "1h43m41s", note: "The step was raised from 4m20s to 1h43m41s " + note},
			{query: `avg_over_time(` + m + `[2h])`, from: "now-1h", step: "4s", note: "The step was raised from 2s to 4s " + note},
			{query: `quantile_over_time(0.99, ` + m + `[6d23h])`, from: "now-1h", step: "5m1s", note: "The step was raised from 2s to 5m1s " + note},
		} {
			fake := newFakePMM(t, metricsRoutes(nil))
			res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": tc.query, "period_from": tc.from})
			require.False(t, isError, "%s: %s", tc.query, text)
			assert.Equal(t, tc.step, res.Step, tc.query)
			assert.Contains(t, res.Note, tc.note, tc.query)
			assert.Contains(t, text, tc.note, tc.query)
			reqs := fake.requestsTo(vmAPI + "query_range")
			require.Len(t, reqs, 1, tc.query)
			step, err := time.ParseDuration(tc.step)
			require.NoError(t, err)
			assert.Equal(t, strconv.Itoa(int(step.Seconds())), queryOf(t, reqs[0]).Get("step"), tc.query)
		}
	})

	t.Run("RaisedStepMeetsTheOtherRules", func(t *testing.T) {
		t.Parallel()

		// At the raised step, the implicit subquery's lead steps take the read past 7 days.
		fake := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics", map[string]any{
			"query":       `max_over_time(quantile_over_time(0.99, mysql_global_status_questions{service_name="db1"}[3d])[1h])`,
			"period_from": "now-5640m",
		}, "the step rises to 3h41m19s so that rollup windows read at most 2000 hours of samples per series; at that step: the query reads")
	})

	t.Run("LongWindowWithAGivenStepIsRefused", func(t *testing.T) {
		t.Parallel()

		const long = `quantile_over_time(0.99, mysql_global_status_questions{service_name="db1"}[6d23h])`
		fake := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics",
			map[string]any{"query": long, "period_from": "now-1h", "step": "2s"}, "raise step to at least 5m1s, shorten the window, or send an instant query")

		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": long})
		require.False(t, isError, text)
		assert.Len(t, res.Series, 2, "an instant query reads the window once")

		fake2 := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake2, false), fake2, "pmm_query_metrics",
			map[string]any{"query": `max_over_time(quantile_over_time(0.99, mysql_global_status_questions{service_name="db1"}[3d])[3d])`},
			"give the subquery a step of at least 2h47m47s")
	})

	t.Run("WideRangeRaisesTheStep", func(t *testing.T) {
		t.Parallel()

		const broad = `{__name__=~"mysql_global_status_.*", service_name="db1"}`
		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": broad, "period_from": "now-6h"})
		require.False(t, isError, text)

		// 500 series at 11s would be about 39 MB; 1m59s keeps them at 182 points each.
		assert.Equal(t, "1m59s", res.Step)
		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		assert.Equal(t, "119", queryOf(t, reqs[0]).Get("step"))
		assert.Contains(t, res.Note, "The step was raised from 11s to 1m59s so that the answer for up to 500 series fits")
		assert.Contains(t, text, "The step was raised from 11s to 1m59s")
	})

	t.Run("RollupNamesMultiplyTheSeries", func(t *testing.T) {
		t.Parallel()

		// 500 matched series fit at 1m59s, but three named functions return 1500, which fit at 6m7s.
		const broad = `{__name__=~"mysql_global_status_.*", service_name="db1"}[5m]`
		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `aggr_over_time(("min_over_time", "max_over_time", "avg_over_time"), ` + broad + `)`, "period_from": "now-6h",
		})
		require.False(t, isError, text)
		assert.Contains(t, res.Note, "so that the answer for up to 1500 series fits")
		assert.Equal(t, "6m7s", res.Step)
	})

	t.Run("RaisedStepCountsTheWiderSpan", func(t *testing.T) {
		t.Parallel()

		// A selector without a window reads one step back: 11s before the raise, 1m59s after it.
		const broad = `{__name__=~"mysql_global_status_.*", service_name="db1"}`
		args := map[string]any{"query": broad, "period_from": "now-6h"}
		from := testNow.Add(-6 * time.Hour)
		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", args)
		require.False(t, isError, text)
		assert.Equal(t, "1m59s", res.Step)
		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 2)
		assert.Equal(t, unixOf(from.Add(-11*time.Second)), queryOf(t, series[0]).Get("start"))
		assert.Equal(t, unixOf(from.Add(-119*time.Second)), queryOf(t, series[1]).Get("start"))
		assert.Equal(t, unixOf(testNow), queryOf(t, series[1]).Get("end"))

		// Over the cap on the wider span, the query is refused before it runs.
		fake = newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		wider := unixOf(from.Add(-119 * time.Second))
		fake.handle(func(rw http.ResponseWriter, req *http.Request, _ []byte) bool {
			if req.URL.Path != vmAPI+"series" || req.URL.Query().Get("start") != wider {
				return false
			}
			fake.reply(rw, req, fixture{body: seriesBody(2001)})
			return true
		})
		_, text, isError = callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", args)
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "this selector matches more than 2000 series over the span")
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"), "the query is refused before it runs")
	})

	t.Run("RaisedStepFitsTheWiderCount", func(t *testing.T) {
		t.Parallel()

		// 500 series over the hour and 11s fit at 1m59s; 1500 over the wider span read at 1m59s fit only at 6m7s.
		from := testNow.Add(-6 * time.Hour)
		narrow := unixOf(from.Add(-11 * time.Second))
		fake := newFakePMM(t, metricsRoutes(nil))
		fake.handle(func(rw http.ResponseWriter, req *http.Request, _ []byte) bool {
			if req.URL.Path != vmAPI+"series" {
				return false
			}
			n := 1500
			if req.URL.Query().Get("start") == narrow {
				n = 500
			}
			fake.reply(rw, req, fixture{body: seriesBody(n)})
			return true
		})
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `{__name__=~"mysql_global_status_.*", service_name="db1"}`, "period_from": "now-6h",
		})
		require.False(t, isError, text)
		assert.Equal(t, "6m7s", res.Step)
		assert.Contains(t, res.Note, "The step was raised from 11s to 6m7s so that the answer for up to 1500 series fits")
		requests := fake.requestsTo(vmAPI + "series")
		starts := make([]string, 0, len(requests))
		for _, r := range requests {
			starts = append(starts, queryOf(t, r).Get("start"))
		}
		assert.Equal(t, []string{narrow, unixOf(from.Add(-119 * time.Second)), unixOf(from.Add(-367 * time.Second))}, starts)
	})

	t.Run("StepThatKeepsRisingIsRefused", func(t *testing.T) {
		t.Parallel()

		// Each count over the wider span a raised step reads finds more series, which raise the step again.
		counts := []int{200, 400, 800, 1600, 2000}
		fake := newFakePMM(t, metricsRoutes(nil))
		fake.handle(func(rw http.ResponseWriter, req *http.Request, _ []byte) bool {
			if req.URL.Path != vmAPI+"series" {
				return false
			}
			n := counts[min(len(fake.requestsTo(vmAPI+"series")), len(counts))-1]
			fake.reply(rw, req, fixture{body: seriesBody(n)})
			return true
		})
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `{__name__=~"mysql_global_status_.*", service_name="db1"}`, "period_from": "now-6h",
		})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "the step that fits the answer in the 4 MiB that MCP reads keeps rising, since each rise makes the query read further back, "+
			"where its selectors match more series: aggregate, as in sum by (service_name) ({__name__=~\"mysql_global_status_.*\", service_name=\"db1\"})")
		assert.Len(t, fake.requestsTo(vmAPI+"series"), maxStepRaises+1)
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"), "the query is refused before it runs over a span it did not count")
	})

	t.Run("UnraisedStepCountsOnce", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `sum by (service_name) (rate({__name__=~"mysql_global_status_.*", service_name="db1"}[5m]))`, "period_from": "now-6h",
		})
		require.False(t, isError, text)
		assert.Len(t, fake.requestsTo(vmAPI+"series"), 1)
	})

	t.Run("WideRangeWithAnAggregationKeepsTheStep", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		res, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `sum by (service_name) (rate({__name__=~"mysql_global_status_.*", service_name="db1"}[5m]))`, "period_from": "now-6h",
		})
		require.False(t, isError, text)
		assert.Equal(t, "11s", res.Step)
		assert.Empty(t, res.Note)
	})

	t.Run("WideRangeWithAGivenStepIsRefused", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `{__name__=~"mysql_global_status_.*", service_name="db1"}`, "period_from": "now-6h", "step": "11s",
		})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "raise step to at least 1m59s, or aggregate")
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"), "the query is refused before it runs")
	})

	t.Run("WideRangeOverTheBudgetAtTheRaisedStep", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(500)}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{
			"query": `{__name__=~"mysql_global_status_.*", service_name="db1"}`, "period_from": "now-10070m",
		})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "the answer for up to 500 series fits")
		assert.Contains(t, text, "aggregate, as in sum by (service_name)")
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"))
	})

	t.Run("NotOnePointFits", func(t *testing.T) {
		t.Parallel()

		// 2000 series with 1.8 KB of labels each are over the 3.5 MiB estimate before any point.
		entries := make([]string, 2000)
		for i := range entries {
			entries[i] = fmt.Sprintf(`{"__name__":"m","id":"%d","pad":"%s"}`, i, strings.Repeat("x", 1800))
		}
		body := []byte(`{"status":"success","data":[` + strings.Join(entries, ",") + `]}`)
		for _, step := range []string{"", "1h"} {
			fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: body}}))
			args := map[string]any{"query": `{__name__=~"mysql_global_status_.*", service_name="db1"}`, "period_from": "now-6h"}
			if step != "" {
				args["step"] = step
			}
			_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", args)
			require.True(t, isError, text)
			assertToolError(t, text, codeInvalidInput, "")
			assert.Contains(t, text, "the answer for up to 2000 series is over the 4 MiB that MCP reads at any step")
			assert.Contains(t, text, "aggregate, as in sum by (service_name)")
			assert.Empty(t, fake.requestsTo(vmAPI+"query_range"), "the query is refused before it runs")
		}
	})

	t.Run("OtherRefusalsAreNotRetried", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake, false), fake, "pmm_query_metrics",
			map[string]any{"query": `max_over_time(mysql_up[7d])`, "period_from": "now-1d"}, "the cap is 7d")
	})

	t.Run("ResponseOverTheBound", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQueryRange: {body: []byte(strings.Repeat(" ", 5<<20))}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h"})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "narrow the query")
		assert.Contains(t, text, "sum by (service_name)")
	})

	t.Run("SeriesResponseOverTheBound", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: []byte(strings.Repeat(" ", 5<<20))}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Empty(t, fake.requestsTo(vmAPI+"query"))
	})

	t.Run("VictoriaMetricsRefusal", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQueryRange: {body: []byte(`{"status":"error","error":"bad query"}`), status: 422}}))
		_, text, isError := callAs[metricsResult](t, newQANService(t, fake, false), "pmm_query_metrics", map[string]any{"query": questions, "period_from": "now-1h"})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "bad query")
	})

	t.Run("TruncatedRangeFitsTheBudget", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQueryRange: {body: matrixBody(100, 20, strings.Repeat("service-", 12))}}))
		r := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_query_metrics",
			map[string]any{"query": questions, "period_from": "now-1h", "limit": 100})
		require.False(t, r.IsError, textOf(t, r))
		res := structured[metricsResult](t, r)

		assert.True(t, res.Truncated, "the fixture is meant to be cut")
		assert.True(t, res.Summarized)
		assert.Equal(t, 100, res.TotalSeries)
		assert.Less(t, len(res.Series), 100)
		assert.Contains(t, res.Hint, "series are left out")
		size := len(textOf(t, r)) + len(mustJSON(t, r.StructuredContent))
		assert.LessOrEqual(t, size, 16<<10, "text plus structured content")
		assert.Greater(t, size, 12<<10, "the cut keeps as many series as fit")
	})

	t.Run("EscapedQueryFitsTheBudget", func(t *testing.T) {
		t.Parallel()

		const prefix, suffix = `mysql_up{service_name="`, `"}`
		query := prefix + strings.Repeat("<", maxExprBytes-len(prefix)-len(suffix)) + suffix
		require.Len(t, query, 4096)

		fake := newFakePMM(t, metricsRoutes(nil))
		r := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_query_metrics", map[string]any{"query": query})
		require.False(t, r.IsError, textOf(t, r))
		res := structured[metricsResult](t, r)

		size := len(textOf(t, r)) + len(mustJSON(t, r.StructuredContent))
		assert.LessOrEqual(t, size, 16<<10, "text plus structured content")
		assert.Len(t, res.Series, 2, "the cut query leaves room for the series")
		assert.Equal(t, echo(query), res.Query)
		assert.Contains(t, res.Query, "…")
	})

	t.Run("Description", func(t *testing.T) {
		t.Parallel()

		desc := listedTools(t)["pmm_query_metrics"].Description
		assert.Contains(t, desc, "MetricsQL")
		assert.Contains(t, strings.ToLower(desc), "aggregate first")
		assert.Equal(t, 1, strings.Count(desc, "sum by (service_name)"))
	})
}

func TestSeriesPerMatched(t *testing.T) {
	t.Parallel()

	for expr, want := range map[string]int{
		`mysql_up`:                               1,
		`rate(m[5m]) / mysql_up`:                 1,
		`aggr_over_time("max_over_time", m[5m])`: 1,
		`aggr_over_time(("min_over_time", "max_over_time"), m[5m])`:       2,
		`label_del(aggr_over_time(("a", "b", "c"), m[5m]), "rollup") + n`: 3,
		`rollup(m[5m])`:                                    3,
		`rollup(m[5m], "max")`:                             1,
		`rollup_candlestick(m[5m])`:                        4,
		`ROLLUP_RATE(m[5m])`:                               3,
		`rollup(rate(m[5m])[1h:1m])`:                       3,
		`rollup(aggr_over_time(("a", "b"), m[5m])[1h:1m])`: 6,
		// 3^40 series per series is over the largest int; the product stops one past the cap.
		strings.Repeat("rollup(", 40) + "m[5m])" + strings.Repeat("[1h:])", 39): maxRollupNames + 1,
		// An aggregation can return fewer series.
		`sum(aggr_over_time(("a", "b"), m[5m]))`: 0,
	} {
		e, err := metricsql.Parse(expr)
		require.NoError(t, err, expr)
		assert.Equal(t, want, seriesPerMatched(e), expr)
	}
}

const (
	vmNames  = "GET " + vmAPI + "label/__name__/values"
	vmLabels = "GET " + vmAPI + "labels"
	vmValues = "GET " + vmAPI + "label/service_name/values"
)

func stringsBody(items []string) []byte {
	b, err := json.Marshal(map[string]any{"status": "success", "data": items})
	if err != nil {
		panic(err)
	}
	return b
}

func numbered(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%04d", prefix, i)
	}
	return out
}

func listRoutes(extra map[string]fixture) map[string]fixture {
	return metricsRoutes(withRoutes(map[string]fixture{
		vmNames:  {body: stringsBody([]string{"mysql_up"})},
		vmLabels: {body: stringsBody([]string{"job", "service_name"})},
		vmValues: {body: stringsBody([]string{"db1"})},
	}, extra))
}

// requireListInvalid is requireInvalid for pmm_list_metrics on a new fake PMM.
func requireListInvalid(t *testing.T, args map[string]any, want string) {
	t.Helper()

	fake := newFakePMM(t, listRoutes(nil))
	requireInvalid(t, newQANService(t, fake, false), fake, "pmm_list_metrics", args, want)
}

func TestListMetricsTool(t *testing.T) {
	t.Parallel()

	t.Run("NameSearchEscapesTheRegexAndTheString", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"search": ".*"})
		require.False(t, isError, text)

		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{`{__name__=~".*\\.\\*.*"}`}, queryOf(t, reqs[0])["match[]"])
		fake.assertForwardedAuth(t)
	})

	t.Run("ServiceNarrowsTheSearch", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"search": "mongodb", "service_name": "db1"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{`{__name__=~".*mongodb.*",service_name="db1"}`}, queryOf(t, reqs[0])["match[]"])

		fake = newFakePMM(t, listRoutes(nil))
		_, text, isError = callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"service_id": `/service_id/a"b`})
		require.False(t, isError, text)
		reqs = fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{`{service_id="/service_id/a\"b"}`}, queryOf(t, reqs[0])["match[]"])
	})

	t.Run("NoSearchNoMatch", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		res, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", nil)
		require.False(t, isError, text)
		assert.Equal(t, "names", res.Kind)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.False(t, queryOf(t, reqs[0]).Has("match[]"))
	})

	t.Run("EveryRequestIsBounded", func(t *testing.T) {
		t.Parallel()

		argsOf := map[string]map[string]any{
			"label/__name__/values":     {"kind": "names"},
			"labels":                    {"kind": "labels", "match": []string{"mysql_up"}},
			"label/service_name/values": {"kind": "values", "label": "service_name"},
			"series":                    {"kind": "series", "match": []string{"mysql_up"}},
		}
		for endpoint, args := range argsOf {
			fake := newFakePMM(t, listRoutes(nil))
			_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", args)
			require.False(t, isError, "%s: %s", endpoint, text)
			reqs := fake.requestsTo(vmAPI + endpoint)
			require.Len(t, reqs, 1, endpoint)
			p := queryOf(t, reqs[0])
			assert.Equal(t, "1001", p.Get("limit"), endpoint)
			assert.Equal(t, unixOf(testNow), p.Get("end"), endpoint)
			assert.Equal(t, unixOf(testNow.Add(-time.Hour)), p.Get("start"), endpoint)
		}
	})

	t.Run("PeriodFromSetsTheStart", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"period_from": "now-6h"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, unixOf(testNow.Add(-6*time.Hour)), queryOf(t, reqs[0]).Get("start"))
	})

	t.Run("NamesOverTheLimitAreGrouped", func(t *testing.T) {
		t.Parallel()

		names := slices.Concat(numbered("mysql_global_status_", 800), numbered("node_cpu_seconds_total_", 700))
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: stringsBody(names)}}))
		res, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", nil)
		require.False(t, isError, text)

		assert.Empty(t, res.Names)
		assert.Equal(t, []nameGroup{{"mysql_global_status_*", 800}, {"node_cpu_seconds_*", 700}}, res.Groups)
		assert.Contains(t, text, `"mysql_global_status_*" (800)`)
		assert.True(t, res.Page.Truncated, "more than 1000 came back")
		assert.Contains(t, res.Hint, "more than 1000")
		assert.NotContains(t, text, "pass offset", "every group is on this page")
	})

	t.Run("ThousandAndOneIsTruncated", func(t *testing.T) {
		t.Parallel()

		// Three hex digits keep a page of 1000 values within 16 KiB.
		short := func(n int) []string {
			out := make([]string, n)
			for i := range out {
				out[i] = fmt.Sprintf("%03x", i)
			}
			return out
		}
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmValues: {body: stringsBody(short(1001))}}))
		res, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"kind": "values", "label": "service_name", "limit": 1000, "offset": 1000})
		require.False(t, isError, text)

		assert.Equal(t, 1001, res.Page.Total)
		assert.Equal(t, []string{"3e8"}, res.Values)
		assert.True(t, res.Page.Truncated, "the last page is still truncated: VictoriaMetrics had more")
		assert.Contains(t, res.Hint, "more than 1000")
		assert.Contains(t, res.Hint, "narrow it with match")
		assert.Contains(t, text, res.Hint)
		assert.NotContains(t, text, "pass offset", "no entry follows the last page")

		_, text, isError = callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"kind": "values", "label": "service_name", "limit": 1000})
		require.False(t, isError, text)
		assert.Contains(t, text, "showing 1–1000 of 1001; pass offset=1000 for more")

		fake = newFakePMM(t, listRoutes(map[string]fixture{vmValues: {body: stringsBody(short(1000))}}))
		res, text, isError = callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"kind": "values", "label": "service_name", "limit": 1000})
		require.False(t, isError, text)
		assert.False(t, res.Page.Truncated)
		assert.Empty(t, res.Hint)
	})

	t.Run("Labels", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		res, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"kind": "labels", "match": []string{"mysql_up"}})
		require.False(t, isError, text)

		assert.Equal(t, []string{"job", "service_name"}, res.Labels)
		reqs := fake.requestsTo(vmAPI + "labels")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{"mysql_up"}, queryOf(t, reqs[0])["match[]"])
	})

	t.Run("RefusedInputs", func(t *testing.T) {
		t.Parallel()

		requireListInvalid(t, map[string]any{"kind": "values"}, "needs a label")
		requireListInvalid(t, map[string]any{"kind": "series"}, "needs match")
		requireListInvalid(t, map[string]any{"kind": "labels", "search": "mysql"}, "kind labels does not take search")
		requireListInvalid(t, map[string]any{"kind": "bogus"}, "kind 'bogus'")
		requireListInvalid(t, map[string]any{"label": "service_name"}, "kind names does not take label")
		requireListInvalid(t, map[string]any{"match": []string{"mysql_up"}}, "kind names does not take match")
		requireListInvalid(t, map[string]any{"kind": "series", "match": []string{"up"}, "service_name": "db1"}, "does not take service_name")
		requireListInvalid(t, map[string]any{"service_id": "a", "service_name": "b"}, "not both")
		requireListInvalid(t, map[string]any{"limit": 1001}, "limit must be between 1 and 1000")
		requireListInvalid(t, map[string]any{"offset": -1}, "offset must not be negative")
		requireListInvalid(t, map[string]any{"kind": "labels", "match": make([]string, 11)}, "the cap is 10")
	})

	t.Run("OffsetPagesValues", func(t *testing.T) {
		t.Parallel()

		values := numbered("db", 250)
		reversed := slices.Clone(values)
		slices.Reverse(reversed)
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmValues: {body: stringsBody(reversed)}}))
		res, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"kind": "values", "label": "service_name", "offset": 100})
		require.False(t, isError, text)

		assert.Equal(t, values[100:200], res.Values, "sorted, 101 to 200")
		assert.Equal(t, page{Total: 250, Offset: 100, Limit: 100, Truncated: true}, res.Page)
		assert.Contains(t, text, "showing 101–200 of 250; pass offset=200 for more")
	})

	t.Run("SeriesStayWithinTheBudget", func(t *testing.T) {
		t.Parallel()

		entries := make([]string, 500)
		for i := range entries {
			entries[i] = fmt.Sprintf(`{"__name__":"mysql_up","service_name":"%s%04d","node_name":"node-%04d"}`, strings.Repeat("s", 40), i, i)
		}
		body := []byte(`{"status":"success","data":[` + strings.Join(entries, ",") + `]}`)
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmSeries: {body: body}}))
		r := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_list_metrics",
			map[string]any{"kind": "series", "match": []string{"mysql_up"}, "limit": 500})
		require.False(t, r.IsError, textOf(t, r))
		res := structured[listMetricsResult](t, r)

		size := len(textOf(t, r)) + len(mustJSON(t, r.StructuredContent))
		assert.LessOrEqual(t, size, 16<<10, "text plus structured content")
		assert.Greater(t, size, 12<<10, "as many series as fit are kept")
		assert.NotEmpty(t, res.Series)
		assert.Less(t, len(res.Series), 500)
		assert.True(t, res.Page.Truncated)
		assert.Equal(t, 500, res.Page.Total)
		assert.Equal(t, len(res.Series), res.Page.Limit)
		assert.Contains(t, res.Hint, "narrow match")
		assert.Contains(t, textOf(t, r), fmt.Sprintf("pass offset=%d for more", len(res.Series)))
		for _, s := range res.Series {
			assert.NotNil(t, s)
		}
	})

	t.Run("EveryKindStaysWithinTheBudget", func(t *testing.T) {
		t.Parallel()

		long := numbered(strings.Repeat("v", 200), 1000)
		for _, tc := range []struct {
			args   map[string]any
			routes map[string]fixture
			advice string
			shown  func(listMetricsResult) int
		}{
			{
				args:   map[string]any{"kind": "names", "limit": 1000},
				routes: map[string]fixture{vmNames: {body: stringsBody(long)}},
				advice: "narrow the search",
				shown:  func(r listMetricsResult) int { return len(r.Names) },
			},
			{
				args:   map[string]any{"kind": "labels", "limit": 1000},
				routes: map[string]fixture{vmLabels: {body: stringsBody(long)}},
				advice: "narrow it with match",
				shown:  func(r listMetricsResult) int { return len(r.Labels) },
			},
			{
				args:   map[string]any{"kind": "values", "label": "service_name", "limit": 1000},
				routes: map[string]fixture{vmValues: {body: stringsBody(long)}},
				advice: "narrow it with match",
				shown:  func(r listMetricsResult) int { return len(r.Values) },
			},
		} {
			fake := newFakePMM(t, listRoutes(tc.routes))
			r := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_list_metrics", tc.args)
			require.False(t, r.IsError, textOf(t, r))
			res := structured[listMetricsResult](t, r)

			kind := tc.args["kind"]
			size := len(textOf(t, r)) + len(mustJSON(t, r.StructuredContent))
			assert.LessOrEqual(t, size, 16<<10, "%s: text plus structured content", kind)
			assert.Greater(t, size, 12<<10, "%s: as many entries as fit are kept", kind)
			shown := tc.shown(res)
			assert.Positive(t, shown, kind)
			assert.Less(t, shown, 1000, kind)
			assert.True(t, res.Page.Truncated, kind)
			assert.Equal(t, 1000, res.Page.Total, kind)
			assert.Equal(t, shown, res.Page.Limit, kind)
			assert.Contains(t, res.Hint, "do not fit in 16 KiB", kind)
			assert.Contains(t, res.Hint, tc.advice, kind)
			assert.Contains(t, textOf(t, r), fmt.Sprintf("pass offset=%d for more", shown), kind)
		}
	})

	t.Run("SeriesTooLargeAloneCanBeSkipped", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"status":"success","data":[{"__name__":"mysql_up","a":"` + strings.Repeat("x", 20<<10) + `"},{"__name__":"mysql_up","b":"y"}]}`)
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmSeries: {body: body}}))
		s := newQANService(t, fake, false)
		args := map[string]any{"kind": "series", "match": []string{"mysql_up"}}
		res, text, isError := callAs[listMetricsResult](t, s, "pmm_list_metrics", args)
		require.False(t, isError, text)
		assert.Empty(t, res.Series)
		assert.True(t, res.Page.Truncated)
		assert.Contains(t, res.Hint, "pass offset=1 to skip it")

		args["offset"] = 1
		res, text, isError = callAs[listMetricsResult](t, s, "pmm_list_metrics", args)
		require.False(t, isError, text)
		assert.Equal(t, []map[string]string{{"__name__": "mysql_up", "b": "y"}}, res.Series)
	})

	t.Run("SeriesMapsAreNeverNil", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"status":"success","data":[null,{"__name__":"up"}]}`)
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmSeries: {body: body}}))
		r := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_list_metrics", map[string]any{"kind": "series", "match": []string{"up"}})
		require.False(t, r.IsError, textOf(t, r))
		res := structured[listMetricsResult](t, r)
		assert.Equal(t, []map[string]string{{"__name__": "up"}, {}}, res.Series)
	})

	t.Run("CatalogueMeaning", func(t *testing.T) {
		t.Parallel()

		info, ok := metricsCatalogIndex()["mysql_global_status_threads_connected"]
		require.True(t, ok)
		body := stringsBody([]string{"zz_not_in_catalogue", "mysql_global_status_threads_connected"})
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: body}}))
		res, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", nil)
		require.False(t, isError, text)

		require.Len(t, res.Names, 2)
		assert.Equal(t, metricName{Name: "mysql_global_status_threads_connected", Meaning: info.Meaning, Type: info.Type}, res.Names[0])
		assert.Equal(t, metricName{Name: "zz_not_in_catalogue"}, res.Names[1])
		assert.NotEmpty(t, info.Meaning)
		assert.Contains(t, text, `"mysql_global_status_threads_connected" [`+info.Type+`]: `+info.Meaning)
		assert.True(t, strings.HasSuffix(text, "\n\"zz_not_in_catalogue\""), text)
	})

	t.Run("NamesAreQuoted", func(t *testing.T) {
		t.Parallel()

		evil := "a\nKind: series\nTotal: 0"
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: stringsBody([]string{evil})}}))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", nil)
		require.False(t, isError, text)
		assert.Equal(t, "Kind: names\nTotal: 1\n"+strconv.Quote(evil), text)

		fake = newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: stringsBody(numbered("a\nb_c_d_", 150))}}))
		_, text, isError = callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", nil)
		require.False(t, isError, text)
		assert.Contains(t, text, "\n"+strconv.Quote("a\nb_c_d_*")+" (150)")
		assert.NotContains(t, text, "a\nb")
	})

	t.Run("MatchIsChecked", func(t *testing.T) {
		t.Parallel()

		requireListInvalid(t, map[string]any{"kind": "labels", "match": []string{`{__name__=~".+"}`}}, "metric")
		requireListInvalid(t, map[string]any{"kind": "series", "match": []string{"up", "rate("}}, "cannot parse")
	})

	t.Run("MatchIsASelector", func(t *testing.T) {
		t.Parallel()

		for _, sel := range []string{`sum(mysql_up)`, `rate(mysql_up[5m])`, `mysql_up offset 1h`, `mysql_up[5m]`, `mysql_up or mysql_down`} {
			requireListInvalid(t, map[string]any{"kind": "series", "match": []string{"mysql_up", sel}}, "is not a series selector")
		}

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{
			"kind": "series", "match": []string{`mysql_up`, `{__name__="mysql_up", service_name="db1"}`, `{__name__="mysql_up", a="x" or b="y"}`},
		})
		require.False(t, isError, text)
	})

	t.Run("SearchIsCapped", func(t *testing.T) {
		t.Parallel()

		requireListInvalid(t, map[string]any{"search": strings.Repeat("é", 201)}, "search must be at most 200 characters")

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"search": strings.Repeat("é", 200)})
		require.False(t, isError, text)
	})

	t.Run("PeriodFromInTheFuture", func(t *testing.T) {
		t.Parallel()

		from := testNow.Add(time.Hour).Format(time.RFC3339)
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, newFakePMM(t, listRoutes(nil)), false), "pmm_list_metrics", map[string]any{"period_from": from})
		require.True(t, isError, text)
		assert.Equal(t, "error: invalid_input\nperiod_from ("+from+") must be before now", text)
	})

	t.Run("LabelIsRefused", func(t *testing.T) {
		t.Parallel()

		for _, label := range []string{"../query", "a/b", "1a", "a b"} {
			requireListInvalid(t, map[string]any{"kind": "values", "label": label}, "is not a label name")
		}
	})

	t.Run("LookbackIsAtMostSevenDays", func(t *testing.T) {
		t.Parallel()

		for _, args := range []map[string]any{
			{"kind": "names"},
			{"kind": "labels"},
			{"kind": "values", "label": "service_name"},
			{"kind": "series", "match": []string{"up"}},
		} {
			args["period_from"] = "now-8d"
			requireListInvalid(t, args, "the cap is 7d")
		}

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"period_from": "now-7d"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, unixOf(testNow.Add(-7*24*time.Hour)), queryOf(t, reqs[0]).Get("start"))

		requireListInvalid(t, map[string]any{"period_from": "2026-09-01T00:00:00Z"}, "the cap is 7d")
	})

	t.Run("MatchWithSevenDays", func(t *testing.T) {
		t.Parallel()

		for _, args := range []map[string]any{
			{"kind": "labels", "match": []string{"mysql_up"}},
			{"kind": "values", "label": "service_name", "match": []string{"mysql_up"}},
			{"kind": "series", "match": []string{"mysql_up"}},
		} {
			args["period_from"] = "now-7d"
			fake := newFakePMM(t, listRoutes(nil))
			_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", args)
			require.False(t, isError, "%v: %s", args, text)
			assert.Equal(t, 1, fake.requestCount()-len(fake.requestsTo("/graph/api/datasources/name/Metrics")), "%v", args)

			args["period_from"] = "now-8d"
			requireListInvalid(t, args, "the cap is 7d")
		}
	})

	t.Run("VictoriaMetricsRefusal", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: []byte(`{"status":"error","error":"bad matcher"}`), status: 422}}))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", map[string]any{"search": "mysql"})
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "bad matcher")
	})

	t.Run("TooLargeAnswerAsksToNarrow", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"status":"success","data":[` + strings.Repeat(`"`+strings.Repeat("x", 1000)+`",`, 5000) + `"x"]}`)
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: body}}))
		_, text, isError := callAs[listMetricsResult](t, newQANService(t, fake, false), "pmm_list_metrics", nil)
		require.True(t, isError, text)
		assertToolError(t, text, codeInvalidInput, "")
		assert.Contains(t, text, "larger than 4 MiB: narrow the request")
	})

	t.Run("OneSlotPerCall", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		s := newQANService(t, fake, false)
		key := keyOf(callerAuth{authorization: testToken})
		releases := make([]func(), 0, maxMetricCallsPerCaller)
		for range maxMetricCallsPerCaller - 1 {
			release, hit := s.metrics.acquire(key)
			require.Equal(t, limitNone, hit)
			releases = append(releases, release)
		}

		for _, args := range []map[string]any{
			{"kind": "names"},
			{"kind": "labels"},
			{"kind": "values", "label": "service_name"},
			{"kind": "series", "match": []string{"up"}},
		} {
			_, text, isError := callAs[listMetricsResult](t, s, "pmm_list_metrics", args)
			require.False(t, isError, "%v takes more than one slot: %s", args, text)
		}

		release, hit := s.metrics.acquire(key)
		require.Equal(t, limitNone, hit, "a finished call released its slot")
		releases = append(releases, release)
		before := fake.requestCount()
		_, text, isError := callAs[listMetricsResult](t, s, "pmm_list_metrics", nil)
		require.True(t, isError, text)
		assertToolError(t, text, codeRateLimited, "")
		assert.Equal(t, before, fake.requestCount(), "no request is sent without a slot")
		for _, release := range releases {
			release()
		}
	})
}

func TestNameSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                         string
		search, serviceID, serviceNm string
		want                         string
	}{
		{"Nothing", "", "", "", ""},
		{"Search", "mysql", "", "", `{__name__=~".*mysql.*"}`},
		{"RegexIsQuoted", "a.b", "", "", `{__name__=~".*a\\.b.*"}`},
		{"QuoteIsEscaped", `a"b`, "", "", `{__name__=~".*a\"b.*"}`},
		{"BackslashIsEscapedTwice", `a\b`, "", "", `{__name__=~".*a\\\\b.*"}`},
		{"ServiceAlone", "", "", "db1", `{service_name="db1"}`},
		{"ServiceIDAlone", "", "/service_id/1", "", `{service_id="/service_id/1"}`},
		{"ServiceValueIsEscaped", "", "", `d"b`, `{service_name="d\"b"}`},
		{"SearchAndService", "up", "", "db1", `{__name__=~".*up.*",service_name="db1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, nameSelector(tc.search, tc.serviceID, tc.serviceNm))
		})
	}
}

func TestGroupNames(t *testing.T) {
	t.Parallel()

	t.Run("UnderTheLimitStaysAsNames", func(t *testing.T) {
		t.Parallel()

		names, groups := groupNames([]string{"a_b_c_d", "x"}, 2)
		assert.Equal(t, []string{"a_b_c_d", "x"}, names)
		assert.Nil(t, groups)
	})

	t.Run("ByThreeParts", func(t *testing.T) {
		t.Parallel()

		names, groups := groupNames([]string{"mysql_global_status_a", "mysql_global_status_b", "mysql_global_variables_a", "up"}, 3)
		assert.Nil(t, names)
		assert.Equal(t, []nameGroup{{"mysql_global_status_*", 2}, {"mysql_global_variables_*", 1}, {"up", 1}}, groups)
	})

	t.Run("ByTwoPartsWhenThreeIsStillOver", func(t *testing.T) {
		t.Parallel()

		names, groups := groupNames([]string{"mysql_global_status_a", "mysql_global_status_b", "mysql_global_variables_a", "mysql_info_schema_x"}, 2)
		assert.Nil(t, names)
		assert.Equal(t, []nameGroup{{"mysql_global_*", 3}, {"mysql_info_*", 1}}, groups)
	})

	t.Run("ShortNamesKeepTheirName", func(t *testing.T) {
		t.Parallel()

		_, groups := groupNames([]string{"a_b_c", "a_b", "a"}, 2)
		assert.Equal(t, []nameGroup{{"a", 1}, {"a_b", 1}, {"a_b_*", 1}}, groups)
	})
}
