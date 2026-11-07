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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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

func queryParams(t *testing.T, r recordedRequest) url.Values {
	t.Helper()

	v, err := url.ParseQuery(r.query)
	require.NoError(t, err)
	return v
}

func unixOf(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }

// metricsCall calls pmm_query_metrics and returns the result.
func metricsCall(t *testing.T, s *Service, args map[string]any) (res metricsResult, text string, isError bool) {
	t.Helper()

	r := callTool(t, connect(t, s), "pmm_query_metrics", args)
	text = textOf(t, r)
	if r.IsError {
		return metricsResult{}, text, true
	}
	return structured[metricsResult](t, r), text, false
}

func requireInvalid(t *testing.T, s *Service, fake *fakePMM, args map[string]any, want string) {
	t.Helper()

	_, text, isError := metricsCall(t, s, args)
	require.True(t, isError, "%v: %s", args, text)
	assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
	assert.Contains(t, text, want)
	assert.Zero(t, fake.requestCount(), "%v sent a request", args)
}

func TestQueryMetricsTool(t *testing.T) {
	t.Parallel()

	const questions = `rate(mysql_global_status_questions{service_name="db1"}[5m])`

	t.Run("Instant", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions})
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
		p := queryParams(t, reqs[0])
		assert.Equal(t, questions, p.Get("query"))
		assert.Equal(t, unixOf(testNow), p.Get("time"))
		assert.Equal(t, "25s", p.Get("timeout"))
		assert.False(t, p.Has("step"))
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"))
		fake.assertForwardedAuth(t)

		// The selector is counted over the span read: one step of 5m back for an instant query, plus the 5m window.
		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 1)
		sp := queryParams(t, series[0])
		assert.Equal(t, []string{`{__name__="mysql_global_status_questions", service_name="db1"}`}, sp["match[]"])
		assert.Equal(t, unixOf(testNow.Add(-5*time.Minute)), sp.Get("start"))
		assert.Equal(t, unixOf(testNow), sp.Get("end"))
		assert.Equal(t, "2001", sp.Get("limit"))
	})

	t.Run("InstantAtTime", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		_, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "time": "now-2h"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "query")
		require.Len(t, reqs, 1)
		assert.Equal(t, unixOf(testNow.Add(-2*time.Hour)), queryParams(t, reqs[0]).Get("time"))
	})

	t.Run("Range", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-2h"})
		require.False(t, isError, text)

		from := testNow.Add(-2 * time.Hour)
		require.NotNil(t, res.Window)
		assert.Equal(t, window{From: from, To: testNow}, *res.Window)
		assert.Equal(t, "4s", res.Step, "2h in at most 2000 points")
		require.Len(t, res.Series, 2)
		assert.Len(t, res.Series[0].Points, 3)

		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		p := queryParams(t, reqs[0])
		assert.Equal(t, unixOf(from), p.Get("start"))
		assert.Equal(t, unixOf(testNow), p.Get("end"))
		assert.Equal(t, "4", p.Get("step"))
		assert.Equal(t, "25s", p.Get("timeout"))
		assert.Empty(t, fake.requestsTo(vmAPI+"query"))

		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 1)
		sp := queryParams(t, series[0])
		assert.Equal(t, unixOf(from.Add(-5*time.Minute)), sp.Get("start"), "the span read includes the 5m window")
		assert.Equal(t, unixOf(testNow), sp.Get("end"))
	})

	t.Run("RangeWithStep", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-2h", "step": "10m"})
		require.False(t, isError, text)
		assert.Equal(t, "10m", res.Step)
		assert.Equal(t, "600", queryParams(t, fake.requestsTo(vmAPI + "query_range")[0]).Get("step"))
	})

	t.Run("RefusedExpressionSendsNothing", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake, false), fake, map[string]any{"query": `{__name__=~"mysql_global_.*"}`}, "service_id, service_name")
		requireInvalid(t, newQANService(t, fake, false), fake,
			map[string]any{"query": questions, "period_from": "now-7d"}, "the cap is 7d")
		requireInvalid(t, newQANService(t, fake, false), fake, map[string]any{"query": "rate("}, "cannot parse")
	})

	t.Run("CutToTwentySeries", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQuery: {body: vectorBody(300)}}))
		res, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions})
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
			requireInvalid(t, s, fake, map[string]any{"query": questions, "limit": limit}, "limit must be between 1 and 100")
		}
		res, text, isError := metricsCall(t, s, map[string]any{"query": questions, "limit": 50})
		require.False(t, isError, text)
		assert.Len(t, res.Series, 50)
		res, _, _ = metricsCall(t, s, map[string]any{"query": questions, "limit": 3})
		assert.Len(t, res.Series, 3)
	})

	t.Run("BadArguments", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		requireInvalid(t, s, fake, map[string]any{"query": questions, "time": "now", "period_from": "now-1h"}, "use either time, or period_from and period_to")
		requireInvalid(t, s, fake, map[string]any{"query": questions, "time": "now", "period_to": "now"}, "use either time")
		requireInvalid(t, s, fake, map[string]any{"query": questions, "period_from": "now-1h", "step": "1d"}, "step '1d' is not a duration")
		requireInvalid(t, s, fake, map[string]any{"query": questions, "period_from": "now-1h", "step": "5i"}, "step '5i' is not a duration")
		requireInvalid(t, s, fake, map[string]any{"query": questions, "period_from": "now-1h", "step": "1500ms"}, "whole number of seconds")
		requireInvalid(t, s, fake, map[string]any{"query": questions, "period_from": "now-1h", "step": "0s"}, "positive whole number")
		requireInvalid(t, s, fake, map[string]any{"query": questions, "step": "30s"}, "range queries only")
	})

	t.Run("PeriodToAloneReadsTheHourBefore", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		res, text, isError := metricsCall(t, s, map[string]any{"query": questions, "period_to": "now-10m"})
		require.False(t, isError, text)

		end := testNow.Add(-10 * time.Minute)
		require.NotNil(t, res.Window)
		assert.Equal(t, window{From: testNow.Add(-time.Hour), To: end}, *res.Window)
		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		p := queryParams(t, reqs[0])
		assert.Equal(t, unixOf(testNow.Add(-time.Hour)), p.Get("start"))
		assert.Equal(t, unixOf(end), p.Get("end"))

		fake2 := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake2, false), fake2, map[string]any{"query": questions, "period_to": "now-2h"}, "must be after period_from")
	})

	t.Run("RateLimited", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		args := map[string]any{"query": questions}
		key := callerKey(callerAuth{authorization: testToken})

		releases := make([]func(), 0, maxMetricCallsPerCaller)
		for range maxMetricCallsPerCaller {
			release, hit := s.metrics.acquire(key)
			require.Equal(t, limitNone, hit)
			releases = append(releases, release)
		}
		_, text, isError := metricsCall(t, s, args)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Contains(t, text, "at most 4 metric calls")
		assert.Zero(t, fake.requestCount())

		// A refused expression is reported before the limit.
		_, text, _ = metricsCall(t, s, map[string]any{"query": "rate("})
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)

		for _, release := range releases {
			release()
		}
		for i := range maxMetricCallsPerReplica {
			_, hit := s.metrics.acquire(fmt.Sprintf("other-%d", i))
			require.Equal(t, limitNone, hit)
		}
		_, text, isError = metricsCall(t, s, args)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Contains(t, text, "this replica already has 8 metric calls")
		assert.Zero(t, fake.requestCount())
	})

	t.Run("SlotIsReleased", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		for range 2 * maxMetricCallsPerCaller {
			_, text, isError := metricsCall(t, s, map[string]any{"query": questions})
			require.False(t, isError, text)
		}
		// A failed call releases its slot too.
		bad := newFakePMM(t, nil)
		s.api = newTestService(t, bad).api
		for range 2 * maxMetricCallsPerCaller {
			_, text, isError := metricsCall(t, s, map[string]any{"query": questions})
			require.True(t, isError)
			assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
		}
	})

	t.Run("SeriesCap", func(t *testing.T) {
		t.Parallel()

		const hint = "this selector matches more than 2000 series over the span; narrow it with a service_name or node_name matcher, and aggregate"
		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(2001)}}))
		_, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-1h"})
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Contains(t, text, hint)
		assert.Len(t, fake.requestsTo(vmAPI+"series"), 1)
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"), "the query is refused before it runs")
		assert.Empty(t, fake.requestsTo(vmAPI+"query"))

		fake = newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: seriesBody(2000)}}))
		_, text, isError = metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-1h"})
		require.False(t, isError, text)
		assert.Len(t, fake.requestsTo(vmAPI+"query_range"), 1)
	})

	t.Run("SeriesCountPerSelector", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		_, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{
			"query": `mysql_up{service_name="db1"} / on() mysql_global_status_uptime{service_name="db1"}`,
		})
		require.False(t, isError, text)
		series := fake.requestsTo(vmAPI + "series")
		require.Len(t, series, 2)
		assert.Equal(t, []string{`{__name__="mysql_up", service_name="db1"}`}, queryParams(t, series[0])["match[]"])
		assert.Equal(t, []string{`{__name__="mysql_global_status_uptime", service_name="db1"}`}, queryParams(t, series[1])["match[]"])
	})

	t.Run("StepRisesForAnImplicitSubquery", func(t *testing.T) {
		t.Parallel()

		const nested = `max_over_time(sum by (service_name) (rate(mysql_global_status_questions{service_name="db1"}[5m]))[1h])`
		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)

		res, text, isError := metricsCall(t, s, map[string]any{"query": nested, "period_from": "now-15m"})
		require.False(t, isError, text)
		assert.Equal(t, "2s", res.Step, "1s would be 4800 points; the result says the step used")
		reqs := fake.requestsTo(vmAPI + "query_range")
		require.Len(t, reqs, 1)
		assert.Equal(t, "2", queryParams(t, reqs[0]).Get("step"))

		// A step the caller gave is never raised.
		fake2 := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake2, false), fake2,
			map[string]any{"query": nested, "period_from": "now-15m", "step": "1s"}, "raise step to at least 2s")

		res, text, isError = metricsCall(t, s, map[string]any{"query": nested, "period_from": "now-15m", "step": "3s"})
		require.False(t, isError, text)
		assert.Equal(t, "3s", res.Step)
	})

	t.Run("StepStaysWhenNothingNeedsRaising", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		res, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-15m"})
		require.False(t, isError, text)
		assert.Equal(t, "1s", res.Step)
	})

	t.Run("OtherRefusalsAreNotRetried", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		requireInvalid(t, newQANService(t, fake, false), fake,
			map[string]any{"query": `max_over_time(mysql_up[7d])`, "period_from": "now-1d"}, "the cap is 7d")
	})

	t.Run("ResponseOverTheBound", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQueryRange: {body: []byte(strings.Repeat(" ", 5<<20))}}))
		_, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-1h"})
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Contains(t, text, "narrow the query")
		assert.Contains(t, text, "sum by (service_name)")
	})

	t.Run("SeriesResponseOverTheBound", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmSeries: {body: []byte(strings.Repeat(" ", 5<<20))}}))
		_, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions})
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Empty(t, fake.requestsTo(vmAPI+"query"))
	})

	t.Run("VictoriaMetricsRefusal", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(map[string]fixture{vmQueryRange: {body: []byte(`{"status":"error","error":"bad query"}`), status: 422}}))
		_, text, isError := metricsCall(t, newQANService(t, fake, false), map[string]any{"query": questions, "period_from": "now-1h"})
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
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

	t.Run("Description", func(t *testing.T) {
		t.Parallel()

		desc := listedTools(t)["pmm_query_metrics"].Description
		assert.Contains(t, desc, "MetricsQL")
		assert.Contains(t, strings.ToLower(desc), "aggregate first")
		assert.Equal(t, 1, strings.Count(desc, "sum by (service_name)"))
	})
}

func TestSelectorsOf(t *testing.T) {
	t.Parallel()

	got, err := selectorsOf(`rate(mysql_global_status_questions{service_name="db1"}[5m]) / mysql_up`, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{`{__name__="mysql_global_status_questions", service_name="db1"}`, `{__name__="mysql_up"}`}, got)

	got, err = selectorsOf(`{__name__=~"mysql_global_status_.*", service_name="db1" or service_name="db2"}`, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, []string{
		`{__name__=~"mysql_global_status_.*", service_name="db1"}`,
		`{service_name="db2"}`,
	}, got, "one per OR group")

	for _, expr := range []string{`rate(`, `mysql_up @ 100`} {
		_, err = selectorsOf(expr, time.Minute)
		te, ok := errors.AsType[*toolError](err)
		require.True(t, ok, expr)
		assert.Equal(t, codeInvalidInput, te.code, expr)
	}
}
