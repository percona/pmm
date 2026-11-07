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
	"errors"
	"fmt"
	"net/url"
	"slices"
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

func listCall(t *testing.T, s *Service, args map[string]any) (res listMetricsResult, text string, isError bool) {
	t.Helper()

	r := callTool(t, connect(t, s), "pmm_list_metrics", args)
	text = textOf(t, r)
	if r.IsError {
		return listMetricsResult{}, text, true
	}
	return structured[listMetricsResult](t, r), text, false
}

func requireListInvalid(t *testing.T, args map[string]any, want string) {
	t.Helper()

	fake := newFakePMM(t, listRoutes(nil))
	_, text, isError := listCall(t, newQANService(t, fake, false), args)
	require.True(t, isError, "%v: %s", args, text)
	assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
	assert.Contains(t, text, want)
	assert.Zero(t, fake.requestCount(), "%v sent a request", args)
}

func TestListMetricsTool(t *testing.T) {
	t.Parallel()

	t.Run("NameSearchEscapesTheRegexAndTheString", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"search": ".*"})
		require.False(t, isError, text)

		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{`{__name__=~".*\\.\\*.*"}`}, queryParams(t, reqs[0])["match[]"])
		fake.assertForwardedAuth(t)
	})

	t.Run("ServiceNarrowsTheSearch", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"search": "mongodb", "service_name": "db1"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{`{__name__=~".*mongodb.*",service_name="db1"}`}, queryParams(t, reqs[0])["match[]"])

		fake = newFakePMM(t, listRoutes(nil))
		_, text, isError = listCall(t, newQANService(t, fake, false), map[string]any{"service_id": `/service_id/a"b`})
		require.False(t, isError, text)
		reqs = fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{`{service_id="/service_id/a\"b"}`}, queryParams(t, reqs[0])["match[]"])
	})

	t.Run("NoSearchNoMatch", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		res, text, isError := listCall(t, newQANService(t, fake, false), nil)
		require.False(t, isError, text)
		assert.Equal(t, "names", res.Kind)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.False(t, queryParams(t, reqs[0]).Has("match[]"))
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
			_, text, isError := listCall(t, newQANService(t, fake, false), args)
			require.False(t, isError, "%s: %s", endpoint, text)
			reqs := fake.requestsTo(vmAPI + endpoint)
			require.Len(t, reqs, 1, endpoint)
			p := queryParams(t, reqs[0])
			assert.Equal(t, "1001", p.Get("limit"), endpoint)
			assert.Equal(t, unixOf(testNow), p.Get("end"), endpoint)
			assert.Equal(t, unixOf(testNow.Add(-time.Hour)), p.Get("start"), endpoint)
		}
	})

	t.Run("PeriodFromSetsTheStart", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		_, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"period_from": "now-6h"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, unixOf(testNow.Add(-6*time.Hour)), queryParams(t, reqs[0]).Get("start"))
	})

	t.Run("NamesOverTheLimitAreGrouped", func(t *testing.T) {
		t.Parallel()

		names := slices.Concat(numbered("mysql_global_status_", 800), numbered("node_cpu_seconds_total_", 700))
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: stringsBody(names)}}))
		res, text, isError := listCall(t, newQANService(t, fake, false), nil)
		require.False(t, isError, text)

		assert.Empty(t, res.Names)
		assert.Equal(t, []nameGroup{{"mysql_global_status_*", 800}, {"node_cpu_seconds_*", 700}}, res.Groups)
		assert.Contains(t, text, "mysql_global_status_* (800)")
		assert.True(t, res.Page.Truncated, "more than 1000 came back")
		assert.Contains(t, res.Hint, "more than 1000")
	})

	t.Run("ThousandAndOneIsTruncated", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(map[string]fixture{vmValues: {body: stringsBody(numbered("db", 1001))}}))
		res, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"kind": "values", "label": "service_name", "limit": 1000, "offset": 1000})
		require.False(t, isError, text)

		assert.Equal(t, 1001, res.Page.Total)
		assert.Equal(t, []string{"db1000"}, res.Values)
		assert.True(t, res.Page.Truncated, "the last page is still truncated: VictoriaMetrics had more")
		assert.Contains(t, res.Hint, "more than 1000")
		assert.Contains(t, res.Hint, "narrow it with match")
		assert.Contains(t, text, res.Hint)

		fake = newFakePMM(t, listRoutes(map[string]fixture{vmValues: {body: stringsBody(numbered("db", 1000))}}))
		res, text, isError = listCall(t, newQANService(t, fake, false), map[string]any{"kind": "values", "label": "service_name", "limit": 1000})
		require.False(t, isError, text)
		assert.False(t, res.Page.Truncated)
		assert.Empty(t, res.Hint)
	})

	t.Run("Labels", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		res, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"kind": "labels", "match": []string{"mysql_up"}})
		require.False(t, isError, text)

		assert.Equal(t, []string{"job", "service_name"}, res.Labels)
		reqs := fake.requestsTo(vmAPI + "labels")
		require.Len(t, reqs, 1)
		assert.Equal(t, []string{"mysql_up"}, queryParams(t, reqs[0])["match[]"])
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
		res, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"kind": "values", "label": "service_name", "offset": 100})
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

		info, ok := metricInfoByName("mysql_global_status_threads_connected")
		require.True(t, ok)
		body := stringsBody([]string{"zz_not_in_catalogue", "mysql_global_status_threads_connected"})
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: body}}))
		res, text, isError := listCall(t, newQANService(t, fake, false), nil)
		require.False(t, isError, text)

		require.Len(t, res.Names, 2)
		assert.Equal(t, metricName{Name: "mysql_global_status_threads_connected", Meaning: info.Meaning, Type: info.Type}, res.Names[0])
		assert.Equal(t, metricName{Name: "zz_not_in_catalogue"}, res.Names[1])
		assert.NotEmpty(t, info.Meaning)
		assert.Contains(t, text, info.Meaning)
	})

	t.Run("MatchIsChecked", func(t *testing.T) {
		t.Parallel()

		requireListInvalid(t, map[string]any{"kind": "labels", "match": []string{`{__name__=~".+"}`}}, "metric")
		requireListInvalid(t, map[string]any{"kind": "series", "match": []string{"up", "rate("}}, "cannot parse")
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
		_, text, isError := listCall(t, newQANService(t, fake, false), map[string]any{"period_from": "now-7d"})
		require.False(t, isError, text)
		reqs := fake.requestsTo(vmAPI + "label/__name__/values")
		require.Len(t, reqs, 1)
		assert.Equal(t, unixOf(testNow.Add(-7*24*time.Hour)), queryParams(t, reqs[0]).Get("start"))

		requireListInvalid(t, map[string]any{"period_from": "2026-09-01T00:00:00Z"}, "the cap is 7d")
	})

	t.Run("TooLargeAnswerAsksToNarrow", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"status":"success","data":[` + strings.Repeat(`"`+strings.Repeat("x", 1000)+`",`, 5000) + `"x"]}`)
		fake := newFakePMM(t, listRoutes(map[string]fixture{vmNames: {body: body}}))
		_, text, isError := listCall(t, newQANService(t, fake, false), nil)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Contains(t, text, "larger than 4 MiB: narrow the request")
	})

	t.Run("OneSlotPerCall", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes(nil))
		s := newQANService(t, fake, false)
		key := callerKey(callerAuth{authorization: testToken})
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
			_, text, isError := listCall(t, s, args)
			require.False(t, isError, "%v takes more than one slot: %s", args, text)
		}

		release, hit := s.metrics.acquire(key)
		require.Equal(t, limitNone, hit, "a finished call released its slot")
		releases = append(releases, release)
		before := fake.requestCount()
		_, text, isError := listCall(t, s, nil)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
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
