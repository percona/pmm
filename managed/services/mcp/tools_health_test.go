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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// healthMatrix is one series of a range query whose values end at end, a minute apart; a value may be "NaN".
func healthMatrix(end time.Time, values ...string) []byte {
	points := make([]string, len(values))
	for i, v := range values {
		points[i] = fmt.Sprintf(`[%d,"%s"]`, end.Add(-time.Duration(len(values)-1-i)*time.Minute).Unix(), v)
	}
	return []byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[` + strings.Join(points, ",") + `]}]}}`)
}

var healthEmpty = []byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`)

var healthBadExpr = fixture{
	status: http.StatusUnprocessableEntity,
	body:   []byte(`{"status":"error","errorType":"bad_data","error":"cannot evaluate the expression"}`),
}

// healthRoutes serves the inventory, the Metrics datasource and a range query that answers every signal with 1, 2, 3 and 6 ending now.
func healthRoutes(extra map[string]fixture) map[string]fixture {
	return withRoutes(withRoutes(inventoryRoutes(), map[string]fixture{
		vmQueryRange: {body: healthMatrix(testNow, "1", "2", "3", "6")},
	}), extra)
}

// onQuery routes the range queries whose text contains text.
func onQuery(text string) string { return vmQueryRange + " ?" + text }

func healthCall(t *testing.T, s *Service, args map[string]any) (healthResult, string, bool) {
	t.Helper()

	r := callTool(t, connect(t, s), "pmm_service_health", args)
	text := textOf(t, r)
	if r.IsError {
		return healthResult{}, text, true
	}
	return structured[healthResult](t, r), text, false
}

func signalByName(t *testing.T, res healthResult, name string) signalResult {
	t.Helper()

	i := slices.IndexFunc(res.Signals, func(s signalResult) bool { return s.Name == name })
	require.GreaterOrEqual(t, i, 0, "no signal '%s'", name)
	return res.Signals[i]
}

func rangeQueries(fake *fakePMM) []string {
	var out []string
	for _, r := range fake.requestsTo(vmAPI + "query_range") {
		v, err := url.ParseQuery(r.query)
		if err != nil {
			continue
		}
		out = append(out, v.Get("query"))
	}
	return out
}

func TestServiceHealthTool(t *testing.T) {
	t.Parallel()

	t.Run("MySQLSignals", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(nil))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		assert.Equal(t, "svc-1", res.ServiceID)
		assert.Equal(t, "shop-mysql", res.ServiceName)
		assert.Equal(t, engineMySQL, res.Engine)
		assert.Equal(t, window{From: testNow.Add(-time.Hour), To: testNow}, res.Window)
		assert.False(t, res.Partial)
		assert.Empty(t, res.Note)
		assert.Equal(t, "https://pmm.example.com/graph/d/mysql-instance-summary?var-service_name=shop-mysql", res.PMMURL)

		require.Len(t, res.Signals, len(healthSignals[engineMySQL])+len(healthSignals[engineNode]))
		for _, sig := range res.Signals {
			require.NotNil(t, sig.Min, sig.Name)
			require.NotNil(t, sig.Avg, sig.Name)
			require.NotNil(t, sig.Max, sig.Name)
			require.NotNil(t, sig.Last, sig.Name)
			require.NotNil(t, sig.LastAt, sig.Name)
			assert.InDelta(t, 1, *sig.Min, 1e-9, sig.Name)
			assert.InDelta(t, 3, *sig.Avg, 1e-9, sig.Name)
			assert.InDelta(t, 6, *sig.Max, 1e-9, sig.Name)
			assert.InDelta(t, 6, *sig.Last, 1e-9, sig.Name)
			assert.Equal(t, testNow, *sig.LastAt, sig.Name)
			assert.False(t, sig.NoData, sig.Name)
			assert.Nil(t, sig.Error, sig.Name)
			assert.NotEmpty(t, sig.Unit, sig.Name)
			assert.NotEmpty(t, sig.Source, sig.Name)
		}
		qps := signalByName(t, res, "qps")
		assert.Equal(t, "short", qps.Unit)
		assert.Equal(t, "mysql_global_status_queries", qps.Source)

		assert.Contains(t, text, "qps [short]: min=1 avg=3 max=6 last=6 at 2026-09-14T12:00:00Z")
		assert.Contains(t, text, "[View in PMM ↗](")

		want := make([]string, 0, len(res.Signals))
		for _, sig := range slices.Concat(healthSignals[engineMySQL], healthSignals[engineNode]) {
			selector := `service_name="shop-mysql"`
			if sig.Node {
				selector = `node_name="demo-node"`
			}
			want = append(want, fmt.Sprintf(sig.Query, selector))
		}
		assert.ElementsMatch(t, want, rangeQueries(fake), "each signal's query once, with its selector")

		for _, r := range fake.requestsTo(vmAPI + "query_range") {
			p := queryParams(t, r)
			assert.Equal(t, unixOf(testNow.Add(-time.Hour)), p.Get("start"))
			assert.Equal(t, unixOf(testNow), p.Get("end"))
			assert.Equal(t, "2", p.Get("step"))
			assert.Equal(t, "25s", p.Get("timeout"))
		}
		fake.assertForwardedAuth(t)
	})

	t.Run("NodeSignalsCarryTheNodeName", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(nil))
		_, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		nodeQueries, serviceQueries := 0, 0
		for _, q := range rangeQueries(fake) {
			switch {
			case strings.Contains(q, `node_name="demo-node"`):
				nodeQueries++
				assert.NotContains(t, q, "service_name", q)
			case strings.Contains(q, `service_name="shop-mysql"`):
				serviceQueries++
				assert.NotContains(t, q, "node_name", q)
			default:
				assert.Fail(t, "a query with neither selector", q)
			}
		}
		assert.Equal(t, len(healthSignals[engineNode]), nodeQueries)
		assert.Equal(t, len(healthSignals[engineMySQL]), serviceQueries)
	})

	t.Run("OtherEngines", func(t *testing.T) {
		t.Parallel()

		for id, engine := range map[string]string{"svc-2": enginePostgreSQL, "svc-mongo": engineMongoDB} {
			fake := newFakePMM(t, healthRoutes(nil))
			res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": id})
			require.False(t, isError, text)
			assert.Equal(t, engine, res.Engine)
			assert.Len(t, res.Signals, len(healthSignals[engine])+len(healthSignals[engineNode]), engine)
			assert.Len(t, rangeQueries(fake), len(res.Signals), engine)
		}
	})

	t.Run("LastAtIsTheNewestSample", func(t *testing.T) {
		t.Parallel()

		old := testNow.Add(-20 * time.Minute)
		fake := newFakePMM(t, healthRoutes(map[string]fixture{
			onQuery("mysql_global_status_threads_running"): {body: healthMatrix(old, "4", "5")},
		}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		sig := signalByName(t, res, "threads_running")
		require.NotNil(t, sig.LastAt)
		assert.Equal(t, old, *sig.LastAt)
		assert.InDelta(t, 5.0, *sig.Last, 1e-9)
		assert.False(t, sig.NoData)
		assert.Nil(t, sig.Error)
		assert.False(t, res.Partial)
		assert.Empty(t, res.Note)
		assert.Equal(t, testNow, *signalByName(t, res, "qps").LastAt)
	})

	t.Run("EmptyNodeSignalIsNoData", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{onQuery("node_load1"): {body: healthEmpty}}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		sig := signalByName(t, res, "load1")
		assert.True(t, sig.NoData)
		assert.Nil(t, sig.Error)
		assert.Nil(t, sig.Min)
		assert.Nil(t, sig.Last)
		assert.Nil(t, sig.LastAt)
		assert.Contains(t, res.Note, nodeHiddenNote)
		assert.NotContains(t, res.Note, allNoDataNote, "other signals have data")
		assert.Contains(t, text, "load1: no_data")
		assert.Contains(t, text, "Access Role")
		assert.False(t, res.Partial, "no data is not a failure")
		assert.NotNil(t, signalByName(t, res, "cpu_busy").Last)
	})

	t.Run("EverySignalIsNoData", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{vmQueryRange: {body: healthEmpty}}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		require.Len(t, res.Signals, len(healthSignals[engineMySQL])+len(healthSignals[engineNode]))
		for _, sig := range res.Signals {
			assert.True(t, sig.NoData, sig.Name)
		}
		assert.Equal(t, allNoDataNote, res.Note, "the note covers the node signals, so nodeHiddenNote is left out")
		assert.NotContains(t, res.Note, "hides the series of the service")
		assert.Contains(t, text, allNoDataNote)
		assert.False(t, res.Partial)
	})

	t.Run("EmptyServiceSignalIsNoDataWithoutTheNote", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{onQuery("mysql_global_status_threads_running"): {body: healthEmpty}}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		assert.True(t, signalByName(t, res, "threads_running").NoData)
		assert.NotContains(t, res.Note, "Access Role")
	})

	t.Run("OneSignalFails", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{onQuery("mysql_global_status_slow_queries"): healthBadExpr}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		assert.True(t, res.Partial)
		failedSignal := signalByName(t, res, "slow_queries")
		require.NotNil(t, failedSignal.Error)
		assert.Equal(t, codeInvalidInput, failedSignal.Error.Code)
		assert.Contains(t, failedSignal.Error.Message, "cannot evaluate the expression")
		assert.Nil(t, failedSignal.Last)
		assert.False(t, failedSignal.NoData)
		assert.Contains(t, text, "slow_queries: error invalid_input")

		ok := 0
		for _, sig := range res.Signals {
			if sig.Name != "slow_queries" {
				require.Nil(t, sig.Error, sig.Name)
				require.NotNil(t, sig.Last, sig.Name)
				ok++
			}
		}
		assert.Equal(t, len(res.Signals)-1, ok)
	})

	t.Run("EverySignalFails", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{vmQueryRange: healthBadExpr}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.True(t, isError, "%+v", res)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Contains(t, text, "cannot evaluate the expression")
	})

	t.Run("NaNIsNull", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{
			onQuery("mysql_global_status_innodb_buffer_pool_reads"): {body: healthMatrix(testNow, "0.5", "NaN")},
			onQuery("mysql_global_status_threads_running"):          {body: healthMatrix(testNow, "NaN")},
		}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		ratio := signalByName(t, res, "buffer_pool_miss_ratio")
		assert.Nil(t, ratio.Last)
		require.NotNil(t, ratio.Min)
		assert.InDelta(t, 0.5, *ratio.Min, 1e-9)
		assert.InDelta(t, 0.5, *ratio.Max, 1e-9)
		require.NotNil(t, ratio.LastAt)
		assert.Equal(t, testNow, *ratio.LastAt)
		assert.Nil(t, ratio.Error)

		onlyNaN := signalByName(t, res, "threads_running")
		assert.Nil(t, onlyNaN.Min)
		assert.Nil(t, onlyNaN.Avg)
		assert.Nil(t, onlyNaN.Max)
		assert.Nil(t, onlyNaN.Last)
		assert.False(t, onlyNaN.NoData)

		assert.Contains(t, res.Note, "NaN")
		assert.False(t, res.Partial)
		assert.Contains(t, text, "last=null")
		assert.Contains(t, text, "NaN")
	})

	t.Run("ZeroLagIsAValue", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{
			onQuery("mongodb_mongod_replset_member_replication_lag"): {body: healthMatrix(testNow, "0", "0")},
		}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-mongo"})
		require.False(t, isError, text)

		lag := signalByName(t, res, "replication_lag")
		assert.False(t, lag.NoData)
		require.NotNil(t, lag.Max)
		require.NotNil(t, lag.Last)
		assert.InDelta(t, 0, *lag.Max, 1e-9)
		assert.InDelta(t, 0, *lag.Last, 1e-9)
		assert.NotContains(t, signalLagQuery(t), "> 0")
	})

	t.Run("SeveralSeriesAreSaid", func(t *testing.T) {
		t.Parallel()

		two := `{"status":"success","data":{"resultType":"matrix","result":[` +
			`{"metric":{"a":"1"},"values":[[` + unixOf(testNow) + `,"2"]]},{"metric":{"a":"2"},"values":[[` + unixOf(testNow) + `,"9"]]}]}}`
		fake := newFakePMM(t, healthRoutes(map[string]fixture{onQuery("mysql_global_status_threads_running"): {body: []byte(two)}}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		sig := signalByName(t, res, "threads_running")
		require.NotNil(t, sig.Max)
		assert.InDelta(t, 9, *sig.Max, 1e-9)
		assert.Contains(t, res.Note, "threads_running: the largest of 2 series.")
		assert.Contains(t, text, "threads_running: the largest of 2 series.")
		assert.NotContains(t, strings.ReplaceAll(res.Note, "threads_running: the largest of 2 series.", ""), "series", "one series says nothing")
	})

	t.Run("NotInTheServiceList", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(nil))
		_, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-other-secret"})
		require.True(t, isError)
		assert.Equal(t, "error: not_found\n"+errNotInList(window{From: testNow.Add(-time.Hour), To: testNow}, testNow, "; use pmm_inventory to list service ids").message, text)
		assert.NotContains(t, text, "svc-other-secret")
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"))
	})

	t.Run("OtherServiceType", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(map[string]fixture{serviceListKey: {file: "vm_up_list.json"}}))
		_, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-px"})
		require.True(t, isError)
		assert.Equal(t, "error: invalid_input\npmm_service_health covers MySQL, PostgreSQL and MongoDB services; service 'proxy-1' is proxysql; use pmm_query_metrics for its metrics", text)
		assert.Empty(t, fake.requestsTo(vmAPI+"query_range"))
	})

	t.Run("WindowTooLong", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, healthRoutes(nil))
		s := newQANService(t, fake, false)
		for _, from := range []string{"now-7d", "now-30d", testNow.AddDate(-9, 0, 0).Format(time.RFC3339)} {
			_, text, isError := healthCall(t, s, map[string]any{"service_id": "svc-1", "period_from": from})
			require.True(t, isError, from)
			assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
			assert.Contains(t, text, "the cap is 7d, counting the 5m3s the signals read before the window: pass a window of at most 6d23h54m57s")
		}
		assert.Zero(t, fake.requestCount(), "refused before the inventory and any query")
		s.metricRate.mu.Lock()
		assert.Empty(t, s.metricRate.buckets, "a refused window takes no token")
		s.metricRate.mu.Unlock()

		largest := testNow.Add(-(7*24*time.Hour - 5*time.Minute - 3*time.Second)).Format(time.RFC3339)
		_, text, isError := healthCall(t, s, map[string]any{"service_id": "svc-1", "period_from": largest})
		require.False(t, isError, "the largest window the hint names: %s", text)

		_, text, isError = healthCall(t, s, map[string]any{"service_id": "svc-1", "period_from": "now-6d"})
		require.False(t, isError, text)
		assert.NotEmpty(t, fake.requestsTo(vmAPI+"query_range"))
	})

	t.Run("NoNodeName", func(t *testing.T) {
		t.Parallel()

		noNode := `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"service_id":"svc-1","service_name":"shop-mysql","service_type":"mysql"},"value":[1789387200,"1789387140"]}]}}`
		fake := newFakePMM(t, healthRoutes(map[string]fixture{serviceListKey: {body: []byte(noNode)}}))
		res, text, isError := healthCall(t, newQANService(t, fake, false), map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)

		assert.Len(t, rangeQueries(fake), len(healthSignals[engineMySQL]))
		for _, q := range rangeQueries(fake) {
			assert.NotContains(t, q, "node_name", "a node signal never runs with an empty node name")
		}
		assert.True(t, signalByName(t, res, "load1").NoData)
		assert.Contains(t, res.Note, "node name is unknown")
		assert.NotContains(t, res.Note, "Access Role")
	})

	t.Run("NoVerdictAndNoStale", func(t *testing.T) {
		t.Parallel()

		assert.ElementsMatch(t,
			[]string{"service_id", "service_name", "engine", "window", "signals", "partial", "note", "pmm_url"},
			jsonTags(reflect.TypeFor[healthResult]()))
		assert.ElementsMatch(t,
			[]string{"name", "unit", "source", "min", "avg", "max", "last", "last_at", "no_data", "error"},
			jsonTags(reflect.TypeFor[signalResult]()))

		fake := newFakePMM(t, healthRoutes(map[string]fixture{onQuery("node_load1"): {body: healthEmpty}, onQuery("mysql_global_status_slow_queries"): healthBadExpr}))
		r := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_service_health", map[string]any{"service_id": "svc-1"})
		var keys []string
		collectKeys(mustJSONValue(t, r.StructuredContent), &keys)
		for _, banned := range []string{"stale", "healthy", "unhealthy", "status", "verdict", "state", "severity", "score"} {
			assert.NotContains(t, keys, banned)
		}
		assert.NotContains(t, strings.ToLower(textOf(t, r)), "stale")
	})
}

func jsonTags(typ reflect.Type) []string {
	var out []string
	for f := range typ.Fields() {
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		out = append(out, name)
	}
	return out
}

func mustJSONValue(t *testing.T, v any) any {
	t.Helper()

	var out any
	require.NoError(t, json.Unmarshal(mustJSON(t, v), &out))
	return out
}

func collectKeys(v any, keys *[]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			*keys = append(*keys, k)
			collectKeys(child, keys)
		}
	case []any:
		for _, child := range x {
			collectKeys(child, keys)
		}
	}
}

func TestServiceHealthDeadline(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, healthRoutes(map[string]fixture{onQuery("mysql_global_status_threads_running"): {block: true}}))
	s := newQANService(t, fake, false)
	req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{Header: http.Header{"Authorization": {testToken}}}}

	ctx, cancel := context.WithTimeout(t.Context(), 11*time.Second)
	defer cancel()
	began := time.Now()
	res, err := s.serviceHealth(ctx, req, serviceHealthInput{ServiceID: "svc-1"})
	elapsed := time.Since(began)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "the signal ran until the shared deadline, 10 s before the call's")
	assert.Less(t, elapsed, 5*time.Second)
	assert.True(t, res.Partial)
	slow := signalByName(t, res, "threads_running")
	require.NotNil(t, slow.Error)
	assert.Equal(t, codeTimeout, slow.Error.Code)
	for _, sig := range res.Signals {
		if sig.Name != "threads_running" {
			assert.Nil(t, sig.Error, sig.Name)
			assert.NotNil(t, sig.Last, sig.Name)
		}
	}
	assert.NoError(t, ctx.Err(), "the result was built before the call's own deadline")
}

func TestServiceHealthEverySignalTimesOut(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, healthRoutes(map[string]fixture{vmQueryRange: {block: true}}))
	s := newQANService(t, fake, false)
	req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{Header: http.Header{"Authorization": {testToken}}}}

	ctx, cancel := context.WithTimeout(t.Context(), 11*time.Second)
	defer cancel()
	_, err := s.serviceHealth(ctx, req, serviceHealthInput{ServiceID: "svc-1"})
	require.Error(t, err)
	assert.Equal(t, codeTimeout, mapError(err).code)
}

func signalLagQuery(t *testing.T) string {
	t.Helper()

	i := slices.IndexFunc(healthSignals[engineMongoDB], func(s signal) bool { return s.Name == "replication_lag" })
	require.GreaterOrEqual(t, i, 0)
	return healthSignals[engineMongoDB][i].Query
}

func TestServiceHealthLimits(t *testing.T) {
	t.Parallel()

	type observed struct {
		mu           sync.Mutex
		inFlight     int
		maxInFlight  int
		maxSlots     int
		callSlots    map[int]bool
		callerSlots  map[int]bool
		minSlotsSeen int
	}

	run := func(t *testing.T, vmSlots int) *observed {
		t.Helper()

		fake := newFakePMM(t, healthRoutes(nil))
		s := newQANService(t, fake, false)
		c, ok := s.api.(*client)
		require.True(t, ok)
		c.vmSlots = make(chan struct{}, vmSlots)

		obs := &observed{callSlots: map[int]bool{}, callerSlots: map[int]bool{}, minSlotsSeen: 1 << 30}
		key := callerKey(callerAuth{authorization: testToken})
		next := fake.server.Config.Handler
		fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if r.URL.Path != vmAPI+"query_range" {
				next.ServeHTTP(rw, r)
				return
			}
			s.metrics.mu.Lock()
			total, perCaller := s.metrics.total, s.metrics.byKey[key]
			s.metrics.mu.Unlock()

			obs.mu.Lock()
			obs.inFlight++
			obs.maxInFlight = max(obs.maxInFlight, obs.inFlight)
			obs.maxSlots = max(obs.maxSlots, len(c.vmSlots))
			obs.minSlotsSeen = min(obs.minSlotsSeen, len(c.vmSlots)-obs.inFlight)
			obs.callSlots[total] = true
			obs.callerSlots[perCaller] = true
			obs.mu.Unlock()

			time.Sleep(20 * time.Millisecond)

			obs.mu.Lock()
			obs.inFlight--
			obs.mu.Unlock()
			next.ServeHTTP(rw, r)
		})

		res, text, isError := healthCall(t, s, map[string]any{"service_id": "svc-1"})
		require.False(t, isError, text)
		assert.False(t, res.Partial)
		assert.Greater(t, len(res.Signals), 8, "more signals than the request slots of any run")

		s.metrics.mu.Lock()
		assert.Zero(t, s.metrics.total, "the call released its slot")
		assert.Empty(t, s.metrics.byKey)
		s.metrics.mu.Unlock()
		return obs
	}

	t.Run("OneCallSlotAndFourQueriesAtATime", func(t *testing.T) {
		t.Parallel()

		obs := run(t, 8)
		assert.Equal(t, map[int]bool{1: true}, obs.callSlots, "one slot on the metric-call limiter, whatever the number of signals")
		assert.Equal(t, map[int]bool{1: true}, obs.callerSlots)
		assert.LessOrEqual(t, obs.maxInFlight, 4, "at most four signal queries at a time")
		assert.Greater(t, obs.maxInFlight, 1, "the queries run concurrently")
		assert.GreaterOrEqual(t, obs.minSlotsSeen, 0, "every query in flight holds a request slot")
	})

	t.Run("EachQueryHoldsARequestSlot", func(t *testing.T) {
		t.Parallel()

		obs := run(t, 2)
		assert.LessOrEqual(t, obs.maxInFlight, 2, "no more queries in flight than request slots")
		assert.Equal(t, 2, obs.maxSlots)
		assert.GreaterOrEqual(t, obs.minSlotsSeen, 0)
	})
}

// optionalMetric says why a signal's query may name a metric the exporter of its fixture does not send, or "".
func optionalMetric(name string) string {
	switch {
	case strings.HasPrefix(name, "azure_"), name == "node_cpu_average":
		return "sent for Azure database nodes only"
	case strings.HasPrefix(name, "aws_rds_"), strings.HasPrefix(name, "rdsosmetrics_"):
		return "sent for AWS RDS nodes only"
	case name == "mysql_slave_status_seconds_behind_source":
		return "sent by MySQL versions that rename seconds_behind_master; the 8.0 fixture has the old name"
	}
	return ""
}

var firstMetric = regexp.MustCompile(`([a-zA-Z_:][a-zA-Z0-9_:]*)\{`)

func metricNames(t *testing.T, query string) []string {
	t.Helper()

	e, err := metricsql.Parse(query)
	require.NoError(t, err, query)
	var names []string
	metricsql.VisitAll(e, func(e metricsql.Expr) {
		me, ok := e.(*metricsql.MetricExpr)
		if !ok {
			return
		}
		for _, lf := range me.LabelFilterss[0] {
			if lf.Label == "__name__" && !lf.IsRegexp && !lf.IsNegative {
				names = append(names, lf.Value)
			}
		}
	})
	return names
}

func TestHealthSignalsPass(t *testing.T) {
	t.Parallel()

	fixtures := map[string]map[string]*exporterFamily{}
	for _, engine := range metricsEngines {
		fixtures[engine] = readExporterFixture(t, engine)
	}

	windows := []time.Duration{time.Minute, 10 * time.Minute, time.Hour, 6 * day}
	count := 0
	for engine, signals := range healthSignals {
		require.Contains(t, fixtures, engine)
		seen := map[string]bool{}
		for _, sig := range signals {
			count++
			assert.False(t, seen[sig.Name], "%s: %s listed twice", engine, sig.Name)
			seen[sig.Name] = true
			assert.Equal(t, engine == engineNode, sig.Node, "%s: %s", engine, sig.Name)
			assert.NotEmpty(t, sig.Unit, sig.Name)
			assert.Contains(t, sig.Query, "%[1]s", sig.Name)
			assert.NotContains(t, sig.Query, "$", sig.Name)

			selector := `service_name="db1"`
			family := fixtures[engine]
			if sig.Node {
				selector = `node_name="host1"`
			}
			query := fmt.Sprintf(sig.Query, selector)
			assert.NotContains(t, query, "%!", sig.Name)

			for _, span := range windows {
				_, err := checkExpr(query, span, chooseStep(span, 0))
				require.NoError(t, err, "%s/%s over %s: %s", engine, sig.Name, span, query)
			}

			assert.Equal(t, sig.Source, firstMetric.FindStringSubmatch(query)[1], "%s: Source is the first metric the query names", sig.Name)
			names := metricNames(t, query)
			assert.Contains(t, names, sig.Source, sig.Name)
			for _, name := range names {
				_, ok := family[name]
				reason := optionalMetric(name)
				switch {
				case reason != "":
					assert.False(t, ok, "%s: %s is in the fixture now; it is not optional (%s)", sig.Name, name, reason)
					assert.NotEqual(t, sig.Source, name, "%s: the source cannot be optional", sig.Name)
				default:
					assert.True(t, ok, "%s/%s: metric %s is not in the %s exporter output", engine, sig.Name, name, engine)
				}
			}
		}
	}
	assert.Equal(t, 31, count)
	assert.Len(t, healthSignals[engineNode], 6)
	assert.Len(t, healthSignals, 4)
}
