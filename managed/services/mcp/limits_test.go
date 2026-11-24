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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLimiter(t *testing.T) {
	t.Parallel()

	l := newLimiter("test", 2, 3)
	relA1, hit1 := l.acquire("a")
	relA2, hit2 := l.acquire("a")
	_, hit3 := l.acquire("a")
	require.Equal(t, limitNone, hit1)
	require.Equal(t, limitNone, hit2)
	require.Equal(t, limitPerCaller, hit3)
	relB, hitB := l.acquire("b")
	_, hitC := l.acquire("c")
	require.Equal(t, limitNone, hitB)
	require.Equal(t, limitPerReplica, hitC)
	// Both caps are full: the per-caller cap is reported.
	_, hitA := l.acquire("a")
	require.Equal(t, limitPerCaller, hitA)
	relA1()
	relA1()
	relA3, hitA3 := l.acquire("a")
	require.Equal(t, limitNone, hitA3)
	relA2()
	relA3()
	relB()
	require.Empty(t, l.byKey)
	require.Zero(t, l.total)
}

func TestLimiterPerService(t *testing.T) {
	t.Parallel()

	l := newLimiter("live-inspection", maxLiveCallsPerService, maxLiveCallsPerReplica)
	rel1, hit := l.acquire("svc-1")
	require.Equal(t, limitNone, hit)
	_, hit = l.acquire("svc-1")
	assert.Equal(t, limitPerCaller, hit)
	rel2, hit := l.acquire("svc-2")
	require.Equal(t, limitNone, hit)

	rel1()
	rel3, hit := l.acquire("svc-1")
	assert.Equal(t, limitNone, hit)
	rel2()
	rel3()
	assert.Empty(t, l.byKey)
}

func TestCallerKey(t *testing.T) {
	t.Parallel()

	basic := func(userInfo string) string { return "Basic " + base64.StdEncoding.EncodeToString([]byte(userInfo)) }

	t.Run("SameCaller", func(t *testing.T) {
		t.Parallel()

		bearer := keyOf(callerAuth{authorization: "Bearer T"})
		assert.Equal(t, bearer, keyOf(callerAuth{authorization: basic("service_token:T")}))
		assert.Equal(t, bearer, keyOf(callerAuth{authorization: "Bearer T", cookie: "x=1"}))
		assert.Equal(t, bearer, keyOf(callerAuth{authorization: "Bearer T", cookie: "pmm_session=s"}))

		alice := keyOf(callerAuth{authorization: basic("alice:pw")})
		assert.Equal(t, alice, keyOf(callerAuth{authorization: basic("ALICE:pw")}))

		both := keyOf(callerAuth{authorization: basic("alice:pw"), cookie: "pmm_session=s"})
		assert.Equal(t, both, keyOf(callerAuth{authorization: basic("bob:x"), cookie: "pmm_session=t"}))
		assert.NotEqual(t, both, alice)
		assert.NotEqual(t, both, keyOf(callerAuth{cookie: "pmm_session=s"}))
		assert.NotEqual(t, both, keyOf(callerAuth{}))

		assert.Equal(t, keyOf(callerAuth{cookie: "pmm_session=abc"}), keyOf(callerAuth{cookie: "pmm_session=%61bc"}))

		session := keyOf(callerAuth{cookie: "pmm_session=s1"})
		assert.Equal(t, session, keyOf(callerAuth{cookie: "pmm_session=s1; other=1"}))
	})

	t.Run("DifferentCallers", func(t *testing.T) {
		t.Parallel()

		keys := []string{
			keyOf(callerAuth{authorization: "Bearer T1"}),
			keyOf(callerAuth{authorization: "Bearer T2"}),
			keyOf(callerAuth{authorization: basic("alice:pw")}),
			keyOf(callerAuth{authorization: basic("bob:pw")}),
			keyOf(callerAuth{cookie: "pmm_session=s1"}),
			keyOf(callerAuth{cookie: "pmm_session=s2"}),
			keyOf(callerAuth{}),
		}
		seen := map[string]bool{}
		for _, k := range keys {
			assert.False(t, seen[k], "duplicate key %s", k)
			seen[k] = true
		}
	})

	t.Run("IsSHA256", func(t *testing.T) {
		t.Parallel()

		sum := sha256.Sum256([]byte("token\x00T"))
		assert.Equal(t, hex.EncodeToString(sum[:]), keyOf(callerAuth{authorization: "Bearer T"}))
	})

	t.Run("NoSecretInKey", func(t *testing.T) {
		t.Parallel()

		for _, a := range []callerAuth{
			{authorization: "Bearer secrettoken"},
			{authorization: basic("aliceuser:pw")},
			{cookie: "pmm_session=sessionvalue"},
		} {
			key := keyOf(a)
			for _, secret := range []string{"secrettoken", "aliceuser", "sessionvalue"} {
				assert.NotContains(t, key, secret)
			}
		}
	})
}

func TestRateLimiter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := newRateLimiter(60, 3)

	for range 3 {
		_, ok := r.take("a", now)
		require.True(t, ok)
	}
	wait, ok := r.take("a", now)
	assert.False(t, ok)
	assert.Equal(t, time.Second, wait)

	wait, ok = r.take("a", now.Add(400*time.Millisecond))
	assert.False(t, ok)
	assert.Equal(t, 600*time.Millisecond, wait)

	_, ok = r.take("a", now.Add(time.Second))
	assert.True(t, ok, "a token comes back after the wait")
	_, ok = r.take("a", now.Add(time.Second))
	assert.False(t, ok)

	for range 3 {
		_, ok = r.take("b", now)
		assert.True(t, ok, "another key has its own bucket")
	}
	assert.Len(t, r.buckets, 2)

	_, ok = r.take("c", now.Add(time.Hour))
	require.True(t, ok)
	assert.Len(t, r.buckets, 1, "buckets refilled to the burst are deleted")
	assert.Contains(t, r.buckets, "c")

	t.Run("EarlierTimeRefillsNothingTwice", func(t *testing.T) {
		t.Parallel()

		// takeToken reads the time before it takes the lock, so a call can arrive with an earlier time than the last.
		r := newRateLimiter(60, 1)
		_, ok := r.take("a", now)
		require.True(t, ok)
		_, ok = r.take("a", now.Add(time.Second))
		require.True(t, ok)
		_, ok = r.take("a", now.Add(500*time.Millisecond))
		require.False(t, ok)
		wait, ok := r.take("a", now.Add(1500*time.Millisecond))
		assert.False(t, ok, "the half second up to the newest time was refilled once already")
		assert.Equal(t, 500*time.Millisecond, wait)
	})
}

func TestRateLimitedRetryAfter(t *testing.T) {
	const questions = `rate(mysql_global_status_questions{service_name="db1"}[5m])`
	t.Parallel()

	retryAfter := func(t *testing.T, res *mcp.CallToolResult) (int, string) {
		t.Helper()

		text := textOf(t, res)
		require.True(t, res.IsError, text)
		require.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		meta, ok := res.Meta["com.percona/error"].(map[string]any)
		require.True(t, ok, "%v", res.Meta)
		assert.Equal(t, "rate_limited", meta["code"])
		assert.Equal(t, true, meta["retryable"])
		n, ok := meta["retry_after"].(float64)
		require.True(t, ok, "%v", meta)
		return int(n), text
	}
	other := http.Header{"Authorization": {"Bearer glsa_other_token"}}

	t.Run("Live", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		s := newQANService(t, fake, true)
		session := connect(t, s)
		args := map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}}

		for range liveCallsBurst {
			text, isError := callText(t, session, "pmm_get_schema", args)
			require.False(t, isError, text)
		}
		n, text := retryAfter(t, callTool(t, session, "pmm_get_schema", args))
		assert.Equal(t, 6, n)
		assert.Contains(t, text, "too many calls for these credentials; retry in 6 s")

		text, isError := callText(t, connectWithHeaders(t, s, other), "pmm_get_schema", args)
		assert.False(t, isError, text)
	})

	t.Run("Metric", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		session := connect(t, s)
		args := map[string]any{"query": questions}

		for range metricCallsBurst {
			text, isError := callText(t, session, "pmm_query_metrics", args)
			require.False(t, isError, text)
		}
		before := fake.requestCount()
		n, text := retryAfter(t, callTool(t, session, "pmm_query_metrics", args))
		assert.Equal(t, 1, n)
		assert.Contains(t, text, "too many calls for these credentials; retry in 1 s")
		assert.Equal(t, before, fake.requestCount(), "a refused call reaches no backend")

		n, _ = retryAfter(t, callTool(t, session, "pmm_list_metrics", map[string]any{}))
		assert.Equal(t, 1, n, "metric tools share one bucket")

		text, isError := callText(t, connectWithHeaders(t, s, other), "pmm_query_metrics", args)
		assert.False(t, isError, text)
	})

	t.Run("RefusedInputTakesNoToken", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, metricsRoutes(nil))
		s := newQANService(t, fake, false)
		session := connect(t, s)

		for range 2 * metricCallsBurst {
			text, isError := callText(t, session, "pmm_query_metrics", map[string]any{"query": "up", "period_from": "now-30d"})
			require.True(t, isError)
			require.True(t, strings.HasPrefix(text, "error: invalid_input"), text)
		}
		s.metricRate.mu.Lock()
		defer s.metricRate.mu.Unlock()
		assert.Empty(t, s.metricRate.buckets)
	})
}

func TestRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, retryAfterSeconds(time.Millisecond))
	assert.Equal(t, 1, retryAfterSeconds(time.Second))
	assert.Equal(t, 2, retryAfterSeconds(1001*time.Millisecond))

	res := errorResult(&toolError{code: codeRateLimited, retryAfter: 1500 * time.Millisecond})
	meta, ok := res.Meta["com.percona/error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 2, meta["retry_after"])
	assert.Equal(t, true, meta["retryable"])

	res = errorResult(&toolError{code: codeRateLimited})
	meta, ok = res.Meta["com.percona/error"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, meta, "retry_after")
}

func TestQANWindowCap(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, qanRoutes())
	s := newQANService(t, fake, true)
	session := connect(t, s)
	detail := map[string]any{"queryid": "QID-AAA", "service_id": "svc-1"}

	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"pmm_top_queries", map[string]any{"period_from": "now-31d"}},
		{"pmm_query_detail", map[string]any{"queryid": "QID-AAA", "period_from": "now-31d"}},
	} {
		text, isError := callText(t, session, tc.name, tc.args)
		assert.True(t, isError, tc.name)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Contains(t, text, "30 days", tc.name)
	}
	assert.Zero(t, fake.requestCount(), "a refused window reaches no backend")
	s.metrics.mu.Lock()
	assert.Empty(t, s.metrics.byKey)
	s.metrics.mu.Unlock()

	detail["period_from"] = "now-30d"
	text, isError := callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-30d"})
	assert.False(t, isError, text)
	text, isError = callText(t, session, "pmm_query_detail", detail)
	assert.False(t, isError, text)
}

func TestQANCallsShareMetricSlots(t *testing.T) {
	t.Parallel()

	key := keyOf(callerAuth{authorization: testToken})
	args := map[string]map[string]any{
		"pmm_top_queries":  {"period_from": "now-1h"},
		"pmm_query_detail": {"queryid": "QID-AAA", "period_from": "now-1h"},
	}

	t.Run("PerCaller", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		s := newQANService(t, fake, true)
		session := connect(t, s)
		rels := make([]func(), 0, maxMetricCallsPerCaller)
		for range maxMetricCallsPerCaller {
			rel, hit := s.metrics.acquire(key)
			require.Equal(t, limitNone, hit)
			t.Cleanup(rel)
			rels = append(rels, rel)
		}

		for name, a := range args {
			text, isError := callText(t, session, name, a)
			assert.True(t, isError, name)
			assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
			assert.Contains(t, text, "at most 4 metric calls can be in flight for these credentials", name)
		}
		assert.Zero(t, fake.requestCount())

		rels[0]()
		for name, a := range args {
			text, isError := callText(t, session, name, a)
			assert.False(t, isError, "%s: %s", name, text)
		}
		s.metricRate.mu.Lock()
		assert.Empty(t, s.metricRate.buckets, "QAN calls take no token")
		s.metricRate.mu.Unlock()
	})

	t.Run("PerReplica", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		s := newQANService(t, fake, true)
		session := connect(t, s)
		for i := range maxMetricCallsPerReplica {
			rel, hit := s.metrics.acquire("other-" + strconv.Itoa(i))
			require.Equal(t, limitNone, hit)
			t.Cleanup(rel)
		}
		for name, a := range args {
			text, isError := callText(t, session, name, a)
			assert.True(t, isError, name)
			assert.Contains(t, text, "this replica already has 8 metric calls in flight", name)
		}
	})

	t.Run("DetailTakesOneSlot", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		s := newQANService(t, fake, true)
		var mu sync.Mutex
		var held []int
		next := fake.server.Config.Handler
		fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			s.metrics.mu.Lock()
			n := s.metrics.byKey[key]
			s.metrics.mu.Unlock()
			mu.Lock()
			held = append(held, n)
			mu.Unlock()
			next.ServeHTTP(rw, r)
		})

		text, isError := callText(t, connect(t, s), "pmm_query_detail", args["pmm_query_detail"])
		require.False(t, isError, text)

		mu.Lock()
		defer mu.Unlock()
		require.GreaterOrEqual(t, len(held), 2, "the call makes several backend requests")
		for _, n := range held {
			assert.Equal(t, 1, n, fmt.Sprint(held))
		}
	})
}

// TestLiveLookupSharesMetricCaps pins that pmm_get_explain and pmm_get_schema look a query up under the 30-day window
// and one metric-call slot, with no metric token, and release that slot before the live part.
func TestLiveLookupSharesMetricCaps(t *testing.T) {
	t.Parallel()

	key := keyOf(callerAuth{authorization: testToken})
	tools := []string{"pmm_get_explain", "pmm_get_schema"}

	t.Run("WindowCap", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		s := newQANService(t, fake, true)
		session := connect(t, s)
		old := map[string]any{"queryid": "QID-AAA", "period_from": testNow.Add(-31 * day).Format(time.RFC3339), "period_to": testNow.Format(time.RFC3339)}
		for _, tool := range tools {
			for _, args := range []map[string]any{
				{"queryid": "QID-AAA", "period_from": "now-365d"},
				{"query_ref": old},
			} {
				text, isError := callText(t, session, tool, args)
				assert.True(t, isError, tool)
				assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
				assert.Contains(t, text, "the window is longer than the cap of 30 days", tool)
			}
		}
		assert.Zero(t, fake.requestCount(), "a refused window reaches no backend")
		s.metricRate.mu.Lock()
		assert.Empty(t, s.metricRate.buckets, "a refused window takes no token")
		s.metricRate.mu.Unlock()
	})

	t.Run("RateLimited", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		s := newQANService(t, fake, true)
		session := connect(t, s)
		for range maxMetricCallsPerCaller {
			rel, hit := s.metrics.acquire(key)
			require.Equal(t, limitNone, hit)
			t.Cleanup(rel)
		}
		for _, tool := range tools {
			text, isError := callText(t, session, tool, map[string]any{"queryid": "QID-AAA"})
			assert.True(t, isError, tool)
			assert.Contains(t, text, "at most 4 metric calls can be in flight for these credentials", tool)
		}
		assert.Zero(t, fake.requestCount())
	})

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			t.Parallel()

			fake, _ := newActionStub(t, 0)
			if tool == "pmm_get_schema" {
				postgresQuery(fake)
			}
			s := newQANService(t, fake, true)
			var mu sync.Mutex
			held := map[string][]int{}
			next := fake.server.Config.Handler
			fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				s.metrics.mu.Lock()
				n := s.metrics.byKey[key]
				s.metrics.mu.Unlock()
				kind := "lookup"
				if strings.HasPrefix(r.URL.Path, "/v1/actions") {
					kind = "action"
				}
				mu.Lock()
				held[kind] = append(held[kind], n)
				mu.Unlock()
				next.ServeHTTP(rw, r)
			})

			text, isError := callText(t, connect(t, s), tool, map[string]any{"queryid": "QID-AAA"})
			require.False(t, isError, text)

			mu.Lock()
			defer mu.Unlock()
			require.NotEmpty(t, held["lookup"])
			require.NotEmpty(t, held["action"])
			for _, n := range held["lookup"] {
				assert.Equal(t, 1, n, "the lookup holds one metric slot: %v", held)
			}
			for _, n := range held["action"] {
				assert.Zero(t, n, "the live part holds no metric slot: %v", held)
			}
			s.metrics.mu.Lock()
			assert.Empty(t, s.metrics.byKey)
			s.metrics.mu.Unlock()
			s.metricRate.mu.Lock()
			assert.Empty(t, s.metricRate.buckets, "the lookup takes no metric token, like the QAN tools")
			s.metricRate.mu.Unlock()
		})
	}
}

// TestInventoryAndConfigShareMetricCaps pins that pmm_inventory and pmm_get_config, which read VictoriaMetrics,
// take a metric-call slot and token, after their input is checked.
func TestInventoryAndConfigShareMetricCaps(t *testing.T) {
	t.Parallel()

	key := keyOf(callerAuth{authorization: testToken})
	args := map[string]map[string]any{
		"pmm_inventory":  {},
		"pmm_get_config": {"service_id": "svc-1"},
	}

	fake := newFakePMM(t, withRoutes(inventoryRoutes(), map[string]fixture{vmQuery + " ?mysql_global_variables_": {file: "vm_variables.json"}}))
	s := newQANService(t, fake, true)
	session := connect(t, s)

	for name, a := range map[string]map[string]any{
		"pmm_inventory":  {"engine": "oracle"},
		"pmm_get_config": {"service_id": "svc-1", "at": "yesterday"},
	} {
		text, isError := callText(t, session, name, a)
		assert.True(t, isError, name)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
	}
	s.metricRate.mu.Lock()
	assert.Empty(t, s.metricRate.buckets, "refused input takes no token")
	s.metricRate.mu.Unlock()

	rels := make([]func(), 0, maxMetricCallsPerCaller)
	for range maxMetricCallsPerCaller {
		rel, hit := s.metrics.acquire(key)
		require.Equal(t, limitNone, hit)
		t.Cleanup(rel)
		rels = append(rels, rel)
	}
	for name, a := range args {
		text, isError := callText(t, session, name, a)
		assert.True(t, isError, name)
		assert.Contains(t, text, "at most 4 metric calls can be in flight for these credentials", name)
	}
	assert.Zero(t, fake.requestCount(), "a refused call reaches no backend")

	rels[0]()
	for name, a := range args {
		text, isError := callText(t, session, name, a)
		assert.False(t, isError, "%s: %s", name, text)
	}
	s.metrics.mu.Lock()
	assert.Equal(t, maxMetricCallsPerCaller-1, s.metrics.byKey[key], "the calls gave their slot back")
	s.metrics.mu.Unlock()
	s.metricRate.mu.Lock()
	assert.Len(t, s.metricRate.buckets, 1, "the calls took metric tokens")
	s.metricRate.mu.Unlock()
}

// TestSchemaByNameLookupSharesMetricCaps pins that pmm_get_schema with service_id, database and tables looks the
// service up under a metric-call token and slot, and releases the slot before the live part.
func TestSchemaByNameLookupSharesMetricCaps(t *testing.T) {
	t.Parallel()

	key := keyOf(callerAuth{authorization: testToken})
	args := map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}}

	t.Run("RateLimited", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		s := newQANService(t, fake, true)
		for range metricCallsBurst {
			_, ok := s.metricRate.take(key, testNow)
			require.True(t, ok)
		}

		text, isError := callText(t, connect(t, s), "pmm_get_schema", map[string]any{"service_id": "svc-unknown", "database": "shop", "tables": []string{"customers"}})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Zero(t, listQueries(fake), "a refused call reaches no service-list query")
	})

	t.Run("SlotReleasedBeforeLive", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		s := newQANService(t, fake, true)
		var mu sync.Mutex
		held := map[string][]int{}
		next := fake.server.Config.Handler
		fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			s.metrics.mu.Lock()
			n := s.metrics.byKey[key]
			s.metrics.mu.Unlock()
			kind := "lookup"
			if strings.HasPrefix(r.URL.Path, "/v1/actions") {
				kind = "action"
			}
			mu.Lock()
			held[kind] = append(held[kind], n)
			mu.Unlock()
			next.ServeHTTP(rw, r)
		})

		text, isError := callText(t, connect(t, s), "pmm_get_schema", args)
		require.False(t, isError, text)

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, held["lookup"])
		require.NotEmpty(t, held["action"])
		for _, n := range held["lookup"] {
			assert.Equal(t, 1, n, "the lookup holds one metric slot: %v", held)
		}
		for _, n := range held["action"] {
			assert.Zero(t, n, "the live part holds no metric slot: %v", held)
		}
	})
}

// TestLookupPanicReleasesSlot pins that a panic in the query lookup of pmm_get_explain, or in the service lookup of
// pmm_get_schema with named tables, gives the lookup's metric-call slot back.
func TestLookupPanicReleasesSlot(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		tool string
		args map[string]any
	}{
		"QueryLookup":   {tool: "pmm_get_explain", args: map[string]any{"queryid": "QID-AAA"}},
		"ServiceLookup": {tool: "pmm_get_schema", args: map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			api := newMockPmmAPI(t)
			api.On("GetDatasourceByName", mock.Anything, mock.Anything, mock.Anything).Run(func(mock.Arguments) { panic("boom in lookup") }).Return(nil, nil)
			s, err := New(Params{API: api, Enabled: func() bool { return true }})
			require.NoError(t, err)
			logger, _ := test.NewNullLogger()
			s.l = logrus.NewEntry(logger)
			session := connect(t, s)

			for range maxMetricCallsPerCaller + 1 {
				text, isError := callText(t, session, tc.tool, tc.args)
				assert.True(t, isError)
				require.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
			}
			assertLimitersEmpty(t, s)
		})
	}
}
