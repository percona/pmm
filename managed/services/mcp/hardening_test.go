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
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

// This file covers the PMM-15528 hardening changes. Each test names the
// behaviour it pins rather than the function it calls, so a regression reads
// as a broken promise rather than a broken implementation detail.

// TestOrderByRanking pins the QAN ordering each contract value produces.
//
// The qan-api2 getOrderBy does not sort by the metric it is handed: a time
// metric such as query_time becomes m_query_time_avg, while the pseudo-metric
// load becomes m_query_time_sum. Sending "-query_time" for both
// total_query_time and avg_query_time therefore ranked both by the average,
// so the tool advertised four rankings and delivered three.
func TestOrderByRanking(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		orderBy    string
		wantOrder  string
		wantMetric string
		why        string
	}{
		{"load", "-load", "load", "load is sum(query_time) over the window"},
		{"total_query_time", "-load", "query_time", "totals must order by sum, which only -load expresses"},
		{"avg_query_time", "-query_time", "query_time", "a time metric orders by its average"},
		{"count", "-num_queries", "num_queries", "count orders by the call count"},
	} {
		t.Run(tc.orderBy, func(t *testing.T) {
			t.Parallel()

			fake := newFakePMM(t, qanRoutes())
			session := connect(t, newQANService(t, fake, true))

			_, isError := callText(t, session, "pmm_top_queries",
				map[string]any{"service_id": "svc-1", "order_by": tc.orderBy})
			require.False(t, isError)

			reqs := fake.requestsTo("/v1/qan/metrics:getReport")
			require.Len(t, reqs, 1)
			body := unmarshalBody[qan_service.GetReportBody](t, reqs[0])
			assert.Equal(t, tc.wantOrder, body.OrderBy, tc.why)
			assert.Equal(t, tc.wantMetric, body.MainMetric, tc.why)
		})
	}

	// The regression itself: these two must not produce the same ranking.
	t.Run("TotalAndAverageDiffer", func(t *testing.T) {
		t.Parallel()

		order := func(orderBy string) string {
			fake := newFakePMM(t, qanRoutes())
			session := connect(t, newQANService(t, fake, true))
			_, isError := callText(t, session, "pmm_top_queries",
				map[string]any{"service_id": "svc-1", "order_by": orderBy})
			require.False(t, isError)
			return unmarshalBody[qan_service.GetReportBody](t, fake.requestsTo("/v1/qan/metrics:getReport")[0]).OrderBy
		}
		assert.NotEqual(t, order("total_query_time"), order("avg_query_time"),
			"total_query_time and avg_query_time must not rank identically")
	})
}

// TestRelativeWindowOverflow pins that a relative expression which cannot be
// represented is rejected instead of silently wrapping.
//
// A time.Duration is an int64 of nanoseconds, so n*unit wraps for large n and
// parseTime returned a plausible but wrong timestamp.
func TestRelativeWindowOverflow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	t.Run("Rejected", func(t *testing.T) {
		t.Parallel()

		for _, expr := range []string{
			"now-9223372036854775807s",
			"now-9223372036854775807d",
			// Too long for int64 at all: the same out-of-range answer, not a
			// generic parse error that depends on the digit count.
			"now-99999999999999999999s",
			"now-999999999999h",
			"now-3651d",
			// An absolute time is held to the same lookback, and may not reach
			// past what ClickHouse can store either.
			"0001-01-01T00:00:00Z",
			"9999-12-31T00:00:00Z",
			"2026-09-15T13:00:00Z",
		} {
			_, err := parseTime(expr, now)
			require.Error(t, err, "expected '%s' to be rejected", expr)
			assert.Contains(t, err.Error(), "out of range; the maximum lookback is 10 years, and times may be at most 24h ahead", expr)
		}
	})

	t.Run("OrdinaryWindowsStillWork", func(t *testing.T) {
		t.Parallel()

		for expr, want := range map[string]time.Duration{
			"now-45s": 45 * time.Second,
			"now-30m": 30 * time.Minute,
			"now-1h":  time.Hour,
			"now-2d":  2 * day,
		} {
			got, err := parseTime(expr, now)
			require.NoError(t, err, expr)
			assert.Equal(t, now.Add(-want), got, expr)
		}

		got, err := parseTime("2026-09-01T00:00:00Z", now)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), got)

		// Clock skew: a little ahead of now is fine.
		_, err = parseTime("2026-09-15T11:00:00Z", now)
		require.NoError(t, err)
	})

	t.Run("NoWrapAroundPastNow", func(t *testing.T) {
		t.Parallel()

		// The wrap produced a timestamp in the future; a rejected expression
		// can never do that.
		got, err := parseTime("now-9223372036854775807s", now)
		require.Error(t, err)
		assert.True(t, got.IsZero(), "a rejected expression must not return a timestamp")
	})
}

// TestMetricsDatasourceSelection pins that the metrics datasource is looked up
// by name, not chosen by type.
//
// Picking the first Prometheus-typed datasource sent MCP queries to whichever
// one Grafana happened to return first - a customer's own Prometheus, say -
// and then cached that choice for the lifetime of the process.
func TestMetricsDatasourceSelection(t *testing.T) {
	t.Parallel()

	t.Run("LooksUpMetricsByNamePerQuery", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		s := newTestService(t, fake)
		for range 2 {
			_, err := s.queryMetrics(t.Context(), callerAuth{authorization: testToken}, "mysql_version_info", time.Time{})
			require.NoError(t, err)
		}
		// With the caller's credentials every time: nothing is shared between
		// callers, and nothing goes stale when the datasource is recreated.
		assert.Len(t, fake.requestsTo("/graph/api/datasources/name/Metrics"), 2)
		assert.Len(t, fake.requestsTo("/graph/api/datasources/proxy/uid/metrics-uid/api/v1/query"), 2)
		assert.Empty(t, fake.requestsTo("/graph/api/datasources"), "the datasources are never listed")
		fake.assertForwardedAuth(t)
	})

	t.Run("ErrorsWhenNoMetricsDatasource", func(t *testing.T) {
		t.Parallel()

		routes := inventoryRoutes()
		delete(routes, "GET /graph/api/datasources/name/Metrics")
		s := newTestService(t, newFakePMM(t, routes))
		_, err := s.metricsDatasourceUID(t.Context(), callerAuth{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Grafana datasource named 'Metrics' found")
	})

	t.Run("ErrorsWhenMetricsIsNotPrometheus", func(t *testing.T) {
		t.Parallel()

		routes := inventoryRoutes()
		routes["GET /graph/api/datasources/name/Metrics"] = fixture{file: "datasource_not_prometheus.json"}
		s := newTestService(t, newFakePMM(t, routes))
		_, err := s.metricsDatasourceUID(t.Context(), callerAuth{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not Prometheus-typed")
	})
}

// TestConfigAcceptsServiceID pins that pmm_get_config takes the same
// identifier as every other tool.
//
// It was the only tool addressing a service by service_name, so a model had to
// carry both identifiers through a triage and know which tool wanted which.
func TestConfigAcceptsServiceID(t *testing.T) {
	t.Parallel()

	routes := inventoryRoutes()
	routes["GET /graph/api/datasources/proxy/uid/metrics-uid/api/v1/query"] = fixture{file: "vm_variables.json"}

	t.Run("ResolvesIDToName", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1"})
		assert.False(t, isError, text)
		assert.Contains(t, text, "innodb_buffer_pool_size")

		// The PromQL must carry the resolved name, since metrics are labelled
		// by service_name and not by service_id.
		reqs := fake.requestsTo("/graph/api/datasources/proxy/uid/metrics-uid/api/v1/query")
		require.NotEmpty(t, reqs)
		assert.Contains(t, reqs[0].query, "shop-mysql")
	})

	t.Run("ServiceNameStillAccepted", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql"})
		assert.False(t, isError, text)
		assert.Contains(t, text, "innodb_buffer_pool_size")
	})

	t.Run("NeitherIsAnError", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_config", map[string]any{})
		assert.True(t, isError)
		assert.Contains(t, text, "service_id is required")
	})

	t.Run("UnknownServiceIDIsNotFound", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-does-not-exist"})
		assert.True(t, isError)
		assert.Contains(t, text, "not_found")
	})

	t.Run("EngineComesFromTheService", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		_, _ = callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-2"})
		reqs := fake.requestsTo("/graph/api/datasources/proxy/uid/metrics-uid/api/v1/query")
		require.NotEmpty(t, reqs)
		assert.Contains(t, reqs[0].query, "pg_settings_", "a PostgreSQL service must be read with the PostgreSQL prefix")
	})

	t.Run("UnsupportedServiceIsNamedNotBlamedOnEngine", func(t *testing.T) {
		t.Parallel()

		mongoRoutes := inventoryRoutes()
		mongoRoutes["GET /v1/inventory/services"] = fixture{file: "services_with_mongo.json"}
		fake := newFakePMM(t, mongoRoutes)
		session := connect(t, newQANService(t, fake, true))

		// The caller passed no engine, so the error must say what the
		// service is instead of rejecting an argument that was never given.
		text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-mongo"})
		assert.True(t, isError)
		assert.Contains(t, text, "covers MySQL and PostgreSQL services")
		assert.Contains(t, text, "shop-mongo")
		assert.NotContains(t, text, "engine must be")
	})

	t.Run("ExplicitEngineValidatedBeforeAnyAPICall", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1", "engine": "mongodb"})
		assert.True(t, isError)
		assert.Contains(t, text, "engine must be mysql or postgresql")
		assert.Empty(t, fake.requestsTo("/v1/inventory/services"))
	})
}

// TestStoredPlanProbe pins where pmm_get_explain looks for a stored plan.
//
// Only PostgreSQL has stored plans - pg_stat_monitor captures one per digest -
// so the probe runs for PostgreSQL alone. It sends service_id, which qan-api2
// ignores until PMM-15697. These tests pin what the probe does promise.
func TestStoredPlanProbe(t *testing.T) {
	t.Parallel()

	t.Run("UnknownServiceRejectedBeforeAnyPlanLookup", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["GET /v1/qan/query/QID-AAA/plan"] = fixture{file: "plan.json"}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain",
			map[string]any{"service_id": "svc-does-not-exist", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.Contains(t, text, "not_found")
		assert.Empty(t, fake.requestsTo("/v1/qan/query/QID-AAA/plan"),
			"the service must be validated before the plan is fetched, not after")
	})

	t.Run("MySQLNeverProbes", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		_, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		require.False(t, isError)
		assert.Empty(t, fake.requestsTo("/v1/qan/query/QID-AAA/plan"),
			"MySQL has no stored plans, so the lookup is pure cost")
	})

	t.Run("PostgreSQLReturnsTheStoredPlan", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-PG"})
		require.False(t, isError, text)
		assert.Contains(t, text, "Stored plan (pg_stat_monitor, planid PLAN-1)")
		assert.Contains(t, text, "user42@example.com", "raw SQL is on in this test")
		assert.Contains(t, text, "var-service_name=shop-postgres", "the deep link now names the service")
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"), "PostgreSQL has no live EXPLAIN action")
		assert.Empty(t, fake.requestsTo("/v1/inventory/nodes"), "a lookup by service id needs no node names")
		plans := fake.requestsTo("/v1/qan/query/QID-PG/plan")
		require.Len(t, plans, 1)
		assert.Equal(t, "service_id=svc-2", plans[0].query, "the plan is requested for this service")
	})

	t.Run("PostgreSQLProbeErrorIsSurfaced", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["GET /v1/qan/query/QID-PG/plan"] = fixture{file: "error_403.json", status: http.StatusInternalServerError}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		// A failed lookup is a failure, not "no plan": the setup hint would send
		// the operator to enable a setting that may well be on already.
		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-PG"})
		assert.True(t, isError)
		assert.Contains(t, text, "pmm_unavailable")
		assert.NotContains(t, text, "pgsm_enable_query_plan")
	})

	t.Run("PostgreSQLWithoutAPlanGivesTheSetupHint", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["GET /v1/qan/query/QID-PG/plan"] = fixture{file: "plan_empty.json"}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-PG"})
		assert.True(t, isError)
		assert.Contains(t, text, "not_found")
		assert.Contains(t, text, "pgsm_enable_query_plan")
	})

	t.Run("PostgreSQLNeedsAQueryID", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "query": "SELECT 1"})
		assert.True(t, isError)
		assert.Contains(t, text, "invalid_input")
	})
}

// TestExplainMasksLiteralsWhenRawSQLIsOff pins that plan bodies lose their
// literal values by default, while keeping the plan's shape.
func TestExplainMasksLiteralsWhenRawSQLIsOff(t *testing.T) {
	t.Parallel()

	t.Run("MySQLJSON", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, false))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		require.False(t, isError, text)
		assert.NotContains(t, text, "user42@example.com")
		assert.Contains(t, text, `"attached_condition": "(`+"`shop`.`customers`.`email`"+` = ?)"`)
		assert.Contains(t, text, `"access_type": "ALL"`, "the plan's shape survives masking")
		assert.Contains(t, text, `"rows_examined_per_scan": 20000`, "row estimates are not literals")
	})

	t.Run("PostgreSQLStoredPlan", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, false))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-PG"})
		require.False(t, isError, text)
		assert.NotContains(t, text, "user42@example.com")
		assert.Contains(t, text, "Filter: ((email)::text = ?::text)")
		assert.Contains(t, text, "(cost=0.00..458.00 rows=1 width=64)", "cost and row estimates are not literals")
	})

	// Each read of the setting is a settings query, and two reads that
	// disagree would keep the explained statement while skipping the masking.
	t.Run("SettingReadOncePerCall", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		s := newQANService(t, fake, false)
		var reads atomic.Int32
		s.rawSQL = func() bool {
			// Flip on every read: only a single read can yield a consistent answer.
			return reads.Add(1)%2 == 0
		}
		session := connect(t, s)

		for _, format := range []string{formatJSON, formatTraditional} {
			reads.Store(0)
			text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA", "format": format})
			require.False(t, isError, text)
			assert.Equal(t, int32(1), reads.Load(), format)
			assert.NotContains(t, text, "user42@example.com", format)
		}
	})
}

// TestIsLoopbackPeer pins how a request shows it came through nginx.
//
// nginx reaches pmm-managed over loopback whatever interface pmm-managed is
// bound to, so a loopback remote address is the signal. Anything else, an
// empty or unparseable address included, counts as a direct connection.
func TestIsLoopbackPeer(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:51234", true},
		{"127.10.0.1:51234", true},
		{"[::1]:51234", true},
		{"10.0.0.5:51234", false},
		{"192.168.1.10:7772", false},
		{"[2001:db8::1]:443", false},
		{"127.0.0.1", false},
		{"localhost:51234", false},
		{"@", false},
		{"", false},
	} {
		t.Run(tc.addr, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, isLoopbackPeer(tc.addr), "addr '%s'", tc.addr)
		})
	}
}

// TestHandlerOnlyServesLoopbackPeers pins that /mcp is invisible to anything
// that did not come through nginx. PMM_INTERFACE_TO_BIND can bind pmm-managed
// to a routable interface, and then only this check keeps /mcp behind
// auth_request.
func TestHandlerOnlyServesLoopbackPeers(t *testing.T) {
	t.Parallel()

	// The enabled switch is a settings query: a direct peer, rejected anyway,
	// must not cost one.
	var settingsReads atomic.Int32
	s, err := New(Params{API: &mockPmmAPI{}, Enabled: func() bool { settingsReads.Add(1); return true }})
	require.NoError(t, err)

	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	for _, tc := range []struct {
		remote string
		want   int
	}{
		{"203.0.113.7:40000", http.StatusNotFound},
		{"10.0.0.5:40000", http.StatusNotFound},
		{"127.0.0.1:40000", http.StatusOK},
	} {
		t.Run(tc.remote, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", strings.NewReader(initialize))
			req.RemoteAddr = tc.remote
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			rec := httptest.NewRecorder()

			s.Handler().ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code, "peer %s", tc.remote)
		})
	}
	t.Cleanup(func() {
		assert.Equal(t, int32(1), settingsReads.Load(), "only the loopback peer reads the settings")
	})
}

// TestServesRequestsProxiedByNginx pins that a request shaped exactly like
// nginx's is served: it arrives over loopback and carries the upstream name,
// managed-json, as its Host header.
//
// That is the request the SDK's DNS-rebinding guard rejects with 403, so the
// guard has to stay off whatever interface pmm-managed is bound to.
func TestServesRequestsProxiedByNginx(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, inventoryRoutes())
	upstream := httptest.NewServer(newTestService(t, fake).Handler())
	t.Cleanup(upstream.Close)
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)

	nginx := httptest.NewServer(&httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = "managed-json"
		},
	})
	t.Cleanup(nginx.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   nginx.URL + "/mcp",
		HTTPClient: &http.Client{Transport: &headerTransport{next: http.DefaultTransport}},
	}, nil)
	require.NoError(t, err, "a request proxied the way nginx proxies it must be served")
	t.Cleanup(func() { _ = session.Close() })

	text, isError := callText(t, session, "pmm_version", nil)
	assert.False(t, isError)
	assert.Contains(t, text, "PMM Server")
}

// TestDefaultsAreOff pins that both security-relevant switches default to off
// when no callback is supplied, so a wiring mistake cannot expose the endpoint
// or un-redact its output.
func TestDefaultsAreOff(t *testing.T) {
	t.Parallel()

	s, err := New(Params{})
	require.NoError(t, err)

	assert.False(t, s.enabled(), "the endpoint must default to disabled")
	assert.False(t, s.rawSQL(), "raw SQL must default to off")

	// Disabled means invisible, not an error the caller can distinguish.
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/mcp", strings.NewReader("{}"))
	require.NoError(t, err)
	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	assert.Equal(t, http.StatusNotFound, res.StatusCode, "a disabled endpoint must answer 404")
}
