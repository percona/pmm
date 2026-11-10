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
	"maps"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inventoryRoutes() map[string]fixture {
	return map[string]fixture{
		"GET /v1/server/version":                  {file: "version.json"},
		"GET /graph/api/datasources/name/Metrics": {file: "datasource_metrics.json"},
		serviceListKey:                            {file: "vm_up_services.json"},
		vmQuery + " ?mysql_version_info":          {file: "vm_mysql_version.json"},
		vmQuery + " ?pg_static":                   {file: "vm_pg_version.json"},
		vmQuery + " ?mongodb_version_info":        {file: "vm_mongodb_version.json"},
	}
}

func TestVersionTool(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, inventoryRoutes())
	session := connect(t, newTestService(t, fake))

	text, isError := callText(t, session, "pmm_version", nil)
	assert.False(t, isError)
	assert.Equal(t, "PMM Server 3.8.1 (pmm-managed 3.8.1-269-gfad5f58)", text)
	fake.assertForwardedAuth(t)
}

const (
	seenAt   = "last_seen=2026-09-14T11:59:00Z"
	mongoRow = "- shop-mongo | mongodb 7.0.14 | id=svc-mongo | node=demo-node | " + seenAt + " | [View in PMM ↗](graph/d/mongodb-instance-summary?var-service_name=shop-mongo)"
	mysqlRow = "- shop-mysql | mysql 8.0.46-37 | id=svc-1 | node=demo-node | " + seenAt + " | [View in PMM ↗](graph/d/mysql-instance-summary?var-service_name=shop-mysql)"
	pmmPGRow = "- pmm-server-postgresql | postgresql 14.23.0 | id=svc-pmm-pg | node=pmm-server | " + seenAt + " | [View in PMM ↗](graph/d/postgresql-instance-summary?var-service_name=pmm-server-postgresql)"
	pgRow    = "- shop-postgres | postgresql 17.6.0 | id=svc-2 | node=demo-node | " + seenAt + " | [View in PMM ↗](graph/d/postgresql-instance-summary?var-service_name=shop-postgres)"
)

func TestInventoryTool(t *testing.T) {
	t.Parallel()

	t.Run("Enriched", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.False(t, isError)
		assert.Equal(t, strings.Join([]string{"Monitored services (4):", mongoRow, mysqlRow, pmmPGRow, pgRow}, "\n"), text)
		assert.NotContains(t, text, "address=")
		fake.assertForwardedAuth(t)

		queries := fake.requestsTo(vmAPI + "query")
		require.Len(t, queries, 4)
		promql := make([]string, 0, len(queries))
		for _, q := range queries {
			promql = append(promql, queryOf(t, q).Get("query"))
		}
		assert.ElementsMatch(t, []string{
			fmt.Sprintf(serviceListQuery, 3600), "mysql_version_info", "pg_static", "mongodb_version_info",
		}, promql)
	})

	t.Run("Structured", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		res := callTool(t, session, "pmm_inventory", map[string]any{})
		require.False(t, res.IsError)
		var got inventoryResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &got))
		assert.Equal(t, page{Total: 4, Offset: 0, Limit: 50}, got.Page)
		require.Len(t, got.Services, 4)
		assert.Equal(t, serviceRow{
			ServiceID: "svc-mongo", ServiceName: "shop-mongo", Engine: "mongodb", Version: "7.0.14",
			NodeID: "node-1", NodeName: "demo-node", LastSeen: time.Unix(1789387140, 0).UTC(),
			PMMURL: "graph/d/mongodb-instance-summary?var-service_name=shop-mongo",
		}, got.Services[0])
		assert.Equal(t, "svc-pmm-pg", got.Services[2].ServiceID)
		assert.Equal(t, "14.23.0", got.Services[2].Version)

		raw := string(mustJSON(t, res.StructuredContent))
		assert.NotContains(t, raw, `"address"`)
		assert.NotContains(t, raw, `"port"`)
		assert.Contains(t, raw, `"last_seen":"2026-09-14T11:59:00Z"`)

		text := res.Content[0].(*mcp.TextContent).Text
		assert.NotContains(t, text, "showing ")
	})

	t.Run("EnvironmentClusterAndProxySQL", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(inventoryRoutes(), map[string]fixture{serviceListKey: {file: "vm_up_list.json"}}))
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.False(t, isError)
		assert.Contains(t, text, "| id=svc-1 | node=demo-node | environment=prod | cluster=c1 | last_seen=2026-09-14T11:59:00Z |")
		assert.Contains(t, text, "- shop-postgres | postgresql 17.6.0 | id=svc-2 | node=demo-node | last_seen=2026-09-14T11:59:00Z |")
		assert.NotContains(t, text, "shop-postgres | postgresql 17.6.0 | id=svc-2 | node=demo-node | environment")
		assert.NotContains(t, text, "bare-host")

		text, isError = callText(t, session, "pmm_inventory", map[string]any{"engine": "proxysql"})
		assert.False(t, isError)
		assert.Equal(t, "Monitored services (1):\n- proxy-1 | proxysql unknown | id=svc-px | node=demo-node | "+seenAt+
			" | [View in PMM ↗](graph/d/proxysql-instance-summary?var-proxysql=demo-node)", text)
	})

	t.Run("Paging", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		res := callTool(t, session, "pmm_inventory", map[string]any{"limit": 3})
		require.False(t, res.IsError)
		var got inventoryResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &got))
		assert.Equal(t, page{Total: 4, Offset: 0, Limit: 3, Truncated: true}, got.Page)
		require.Len(t, got.Services, 3)
		text := res.Content[0].(*mcp.TextContent).Text
		assert.True(t, strings.HasSuffix(text, "\nshowing 1–3 of 4; pass offset=3 for more"), text)
		assert.Contains(t, text, "Monitored services (4):")

		res = callTool(t, session, "pmm_inventory", map[string]any{"limit": 3, "offset": 3})
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &got))
		assert.Equal(t, page{Total: 4, Offset: 3, Limit: 3}, got.Page)
		require.Len(t, got.Services, 1)
		assert.Equal(t, "svc-2", got.Services[0].ServiceID)

		res = callTool(t, session, "pmm_inventory", map[string]any{"limit": 3, "offset": 5})
		got = inventoryResult{}
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &got))
		assert.Equal(t, page{Total: 4, Offset: 5, Limit: 3}, got.Page)
		assert.Empty(t, got.Services)
		assert.Equal(t, "No rows at offset 5; the service list has 4.", res.Content[0].(*mcp.TextContent).Text)

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"limit": 300})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
	})

	t.Run("InvalidPagingBeforeAnyCall", func(t *testing.T) {
		t.Parallel()

		// No routes: a backing call would answer not_found instead.
		fake := newFakePMM(t, nil)
		session := connect(t, newTestService(t, fake))
		for want, args := range map[string]map[string]any{
			"limit must be between 1 and 200; got 201": {"limit": 201},
			"offset must not be negative; got -1":      {"offset": -1},
		} {
			text, isError := callText(t, session, "pmm_inventory", args)
			assert.True(t, isError, "%v", args)
			assert.Equal(t, "error: invalid_input\n"+want, text)
		}
		assert.Zero(t, fake.requestCount())
	})

	t.Run("EngineAndNodeFilters", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "postgresql", "node": "demo-node"})
		assert.False(t, isError)
		assert.Equal(t, "Monitored services (1):\n"+pgRow, text)

		text, isError = callText(t, session, "pmm_inventory", map[string]any{"engine": "valkey"})
		assert.False(t, isError)
		assert.Equal(t, "No services found in your service list (services that sent metrics in the last hour).", text)
	})

	t.Run("EmptyList", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(inventoryRoutes(), map[string]fixture{serviceListKey: {file: "vm_empty.json"}}))
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.False(t, isError)
		assert.Equal(t, "No services found in your service list (services that sent metrics in the last hour).", text)
	})

	t.Run("InvalidEngine", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "oracle"})
		assert.True(t, isError)
		assert.Equal(t, "error: invalid_input\nengine must be one of mysql, postgresql, mongodb, proxysql, haproxy, valkey, external; got 'oracle'", text)
		assert.Zero(t, fake.requestCount())
	})

	t.Run("VersionsUnavailable", func(t *testing.T) {
		t.Parallel()

		failed := fixture{file: "error_403.json", status: http.StatusForbidden}
		fake := newFakePMM(t, withRoutes(inventoryRoutes(), map[string]fixture{
			vmQuery + " ?mysql_version_info":   failed,
			vmQuery + " ?pg_static":            failed,
			vmQuery + " ?mongodb_version_info": failed,
		}))
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "mysql"})
		assert.False(t, isError)
		assert.Equal(t, "Monitored services (1):\n"+strings.Replace(mysqlRow, "mysql 8.0.46-37", "mysql unknown", 1), text)
	})

	t.Run("DatasourceUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := inventoryRoutes()
		routes["GET /graph/api/datasources/name/Metrics"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		fake := newFakePMM(t, routes)
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: unauthorized\n"), text)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(inventoryRoutes(), map[string]fixture{
			serviceListKey: {file: "error_401.json", status: http.StatusUnauthorized},
		}))
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.True(t, isError)
		assert.Equal(t, "error: unauthorized\nPMM returned 401 on GET "+vmAPI+"query: Unauthorized", text)
	})

	t.Run("Description", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		assert.Contains(t, listTools(t, session)["pmm_inventory"].Description, "sent metrics in the last hour")
	})

	t.Run("Unreachable", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		s := newTestService(t, fake)
		fake.server.Close()
		session := connect(t, s)

		text, isError := callText(t, session, "pmm_version", nil)
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\ncannot reach the PMM API:"), text)
	})
}

func TestMapError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		status int
		code   errorCode
	}{
		{http.StatusUnauthorized, codeUnauthorized},
		{http.StatusForbidden, codeUnauthorized},
		{http.StatusNotFound, codeNotFound},
		{http.StatusBadRequest, codeInvalidInput},
		{http.StatusBadGateway, codePMMUnavailable},
		{http.StatusServiceUnavailable, codePMMUnavailable},
	} {
		te := mapError(&statusError{status: tc.status, method: "GET", path: "/x", message: "m"})
		assert.Equal(t, tc.code, te.code, "status %d", tc.status)
	}

	assert.Equal(t, "Access denied", messageFromBody([]byte(`{"code":7,"error":"Access denied","message":"Access denied"}`)))
	assert.Equal(t, "Unauthorized", messageFromBody([]byte(`{"code":16,"error":"Unauthorized"}`)))
	assert.Equal(t, "<html>maintenance</html>", messageFromBody([]byte(" <html>maintenance</html>\n")))
}

func TestServiceList(t *testing.T) {
	t.Parallel()

	listRoutes := func() map[string]fixture {
		return map[string]fixture{
			"GET /graph/api/datasources/name/Metrics": {file: "datasource_metrics.json"},
			serviceListKey: {file: "vm_up_list.json"},
		}
	}

	t.Run("Services", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes())
		s := newQANService(t, fake, false)

		got, err := s.listServices(t.Context(), testAuth, s.lastHour())
		require.NoError(t, err)

		datasourceLookups := fake.requestsTo("/graph/api/datasources/name/Metrics")
		require.Len(t, datasourceLookups, 1)
		queries := fake.requestsTo(vmAPI + "query")
		require.Len(t, queries, 1)
		assert.Equal(t, 2, fake.requestCount())
		q := queryOf(t, queries[0])
		assert.Equal(t, fmt.Sprintf(serviceListQuery, 3600), q.Get("query"))
		assert.Equal(t, "1789387200", q.Get("time"))
		assert.Equal(t, "25s", q.Get("timeout"))
		assert.Equal(t, testToken, queries[0].header.Get("Authorization"))

		at := func(unix int64) time.Time { return time.Unix(unix, 0).UTC() }
		assert.Equal(t, []serviceInfo{
			{
				ServiceID: "svc-1", ServiceName: "shop-mysql", Engine: "mysql", NodeID: "node-1", NodeName: "demo-node",
				Environment: "prod", Cluster: "c1", LastSeen: at(1789387140),
			},
			{ServiceID: "svc-re-old", ServiceName: "shop-re", Engine: "mysql", NodeID: "node-1", NodeName: "demo-node", LastSeen: at(1789384200)},
			{ServiceID: "svc-2", ServiceName: "shop-postgres", Engine: "postgresql", NodeID: "node-1", NodeName: "demo-node", LastSeen: at(1789387140)},
			{ServiceID: "svc-re-new", ServiceName: "shop-re", Engine: "postgresql", NodeID: "node-1", NodeName: "demo-node", LastSeen: at(1789387140)},
			{ServiceID: "svc-px", ServiceName: "proxy-1", Engine: "proxysql", NodeID: "node-1", NodeName: "demo-node", LastSeen: at(1789387140)},
		}, got)

		svc, ok := serviceNamed(got, "shop-re")
		require.True(t, ok)
		assert.Equal(t, "svc-re-new", svc.ServiceID)
		_, ok = serviceNamed(got, "unknown")
		assert.False(t, ok)
	})

	t.Run("Nodes", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, listRoutes())
		s := newQANService(t, fake, false)

		services, nodes, err := s.listServicesAndNodes(t.Context(), testAuth, s.lastHour())
		require.NoError(t, err)
		assert.Len(t, services, 5)
		assert.Equal(t, []nodeInfo{{NodeID: "node-1", NodeName: "demo-node"}, {NodeID: "node-2", NodeName: "bare-host"}}, nodes,
			"node-1 as its newest row names it, and node-2 from a row without a service")
		assert.Equal(t, 2, fake.requestCount())

		listed, err := s.listServices(t.Context(), testAuth, s.lastHour())
		require.NoError(t, err)
		assert.Equal(t, services, listed)
	})

	t.Run("NodesByNameWithoutAnID", func(t *testing.T) {
		t.Parallel()

		row := func(labels, value string) string {
			return `{"metric":{` + labels + `},"value":[1789387200,"` + value + `"]}`
		}
		body := `{"status":"success","data":{"resultType":"vector","result":[` + strings.Join([]string{
			row(`"node_name":"named-only"`, "1789387140"),
			row(`"node_id":"n-x","node_name":"x-new"`, "1789387140"),
			row(`"node_id":"n-x","node_name":"x-old"`, "1789384200"),
			row(`"node_id":"n-nan","node_name":"nan-host"`, "NaN"),
			row(`"service_id":"svc-1","service_name":"shop-mysql","service_type":"mysql"`, "1789387140"),
		}, ",") + `]}}`
		fake := newFakePMM(t, withRoutes(listRoutes(), map[string]fixture{serviceListKey: {body: []byte(body)}}))
		s := newQANService(t, fake, false)

		services, nodes, err := s.listServicesAndNodes(t.Context(), testAuth, s.lastHour())
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Equal(t, []nodeInfo{{NodeName: "named-only"}, {NodeID: "n-x", NodeName: "x-new"}}, nodes)
	})

	t.Run("QueryHasNoComparisonOnUp", func(t *testing.T) {
		t.Parallel()

		query := fmt.Sprintf(serviceListQuery, 3600)
		assert.Contains(t, query, "tlast_over_time(up[3600s])")
		assert.NotContains(t, query, "up==")
		assert.NotContains(t, query, "up >")
		assert.NotContains(t, query, "up!=")
	})

	t.Run("Span", func(t *testing.T) {
		t.Parallel()

		hour := window{From: testNow.Add(-time.Hour), To: testNow}
		week := window{From: testNow.Add(-7 * day), To: testNow}
		for _, w := range []window{hour, week} {
			got, cut := listSpan(w, testNow)
			assert.Equal(t, w, got)
			assert.False(t, cut)
		}

		got, cut := listSpan(window{From: testNow.Add(-30 * day), To: testNow}, testNow)
		assert.True(t, cut)
		assert.Equal(t, window{
			From: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC),
		}, got)
	})

	t.Run("SpanEndsNowAndLastsFiveMinutesAtLeast", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name string
			w    window
			want window
			cut  bool
		}{
			{
				name: "a future end reads the last 7 days up to now",
				w:    window{From: testNow.Add(-20 * day), To: testNow.Add(day)},
				want: window{From: testNow.Add(-7 * day), To: testNow},
				cut:  true,
			},
			{
				name: "a future end within 7 days ends now",
				w:    window{From: testNow.Add(-time.Hour), To: testNow.Add(time.Hour)},
				want: window{From: testNow.Add(-time.Hour), To: testNow},
			},
			{
				name: "a window shorter than a scrape interval reads 5 minutes",
				w:    window{From: testNow.Add(-30 * time.Second), To: testNow},
				want: window{From: testNow.Add(-5 * time.Minute), To: testNow},
			},
			{
				name: "a short past window reads the 5 minutes before its end",
				w:    window{From: testNow.Add(-2*time.Hour - time.Second), To: testNow.Add(-2 * time.Hour)},
				want: window{From: testNow.Add(-2*time.Hour - 5*time.Minute), To: testNow.Add(-2 * time.Hour)},
			},
			{
				name: "a window wholly in the future reads the last 5 minutes",
				w:    window{From: testNow.Add(time.Hour), To: testNow.Add(2 * time.Hour)},
				want: window{From: testNow.Add(-5 * time.Minute), To: testNow},
			},
		} {
			got, cut := listSpan(tc.w, testNow)
			assert.Equal(t, tc.want, got, tc.name)
			assert.Equal(t, tc.cut, cut, tc.name)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(listRoutes(), map[string]fixture{
			serviceListKey: {file: "error_401.json", status: http.StatusUnauthorized},
		}))
		s := newQANService(t, fake, false)

		got, err := s.listServices(t.Context(), testAuth, s.lastHour())
		require.Error(t, err)
		assert.Empty(t, got)
		assert.Equal(t, codeUnauthorized, mapError(err).code)
	})

	t.Run("NewestRowWinsInEitherOrder", func(t *testing.T) {
		t.Parallel()

		row := func(node, value string) string {
			return `{"metric":{"service_id":"svc-1","service_name":"shop-mysql","service_type":"mysql","node_name":"` + node + `"},"value":[1789387200,"` + value + `"]}`
		}
		for name, rows := range map[string][]string{
			"older first": {row("old-node", "1789384200"), row("demo-node", "1789387140")},
			"newer first": {row("demo-node", "1789387140"), row("old-node", "1789384200")},
		} {
			body := `{"status":"success","data":{"resultType":"vector","result":[` + strings.Join(rows, ",") + `]}}`
			fake := newFakePMM(t, withRoutes(listRoutes(), map[string]fixture{serviceListKey: {body: []byte(body)}}))
			s := newQANService(t, fake, false)

			got, err := s.listServices(t.Context(), testAuth, s.lastHour())
			require.NoError(t, err, name)
			require.Len(t, got, 1, name)
			assert.Equal(t, "demo-node", got[0].NodeName, name)
		}
	})

	t.Run("NonFiniteValuesGiveNoService", func(t *testing.T) {
		t.Parallel()

		row := func(id, value string) string {
			return `{"metric":{"service_id":"` + id + `","service_name":"n-` + id + `","service_type":"mysql"},"value":[1789387200,"` + value + `"]}`
		}
		body := `{"status":"success","data":{"resultType":"vector","result":[` +
			strings.Join([]string{row("nan", "NaN"), row("inf", "+Inf"), row("ninf", "-Inf"), row("ok", "1789387140")}, ",") + `]}}`
		fake := newFakePMM(t, withRoutes(listRoutes(), map[string]fixture{serviceListKey: {body: []byte(body)}}))
		s := newQANService(t, fake, false)

		got, err := s.listServices(t.Context(), testAuth, s.lastHour())
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "ok", got[0].ServiceID)
	})

	t.Run("TooLarge", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(listRoutes(), map[string]fixture{
			serviceListKey: {body: []byte(strings.Repeat(" ", 5<<20))},
		}))
		s := newQANService(t, fake, false)

		_, err := s.listServices(t.Context(), testAuth, s.lastHour())
		require.ErrorIs(t, err, errResponseTooLarge)
		assert.Equal(t, codePMMUnavailable, mapError(err).code)
	})
}

func TestServiceListQueryInBudget(t *testing.T) {
	t.Parallel()

	read, err := checkExpr(fmt.Sprintf(serviceListQuery, int(maxServiceListLookback/time.Second)), 0, 0)
	require.NoError(t, err)
	assert.Equal(t, maxServiceListLookback, read)
	assert.LessOrEqual(t, maxServiceListLookback, maxSpan)
}

func TestListFollowsTheWindow(t *testing.T) {
	t.Parallel()

	const (
		fourHourKey = vmQuery + " ?tlast_over_time(up[14400s]"
		hint        = "; use pmm_inventory to list service ids"
	)
	newFake := func(t *testing.T) *fakePMM {
		t.Helper()

		return newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{
			vmQueryRange: {body: healthMatrix(testNow, "1", "2", "3", "6")},
			fourHourKey:  {file: "vm_up_window.json"},
		}))
	}
	listTimes := func(t *testing.T, fake *fakePMM, want string) {
		t.Helper()

		list := queriesWith(fake, "tlast_over_time(up[")
		require.Len(t, list, 1)
		assert.Contains(t, queryOf(t, list[0]).Get("query"), want)
	}
	window6hTo2h := map[string]any{"period_from": "now-6h", "period_to": "now-2h"}
	with := func(args map[string]any, kv ...any) map[string]any {
		out := map[string]any{}
		maps.Copy(out, args)
		for i := 0; i < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}

	t.Run("HealthWithoutAWindowSearchesTheLastHour", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, false))

		text, isError := callText(t, session, "pmm_service_health", map[string]any{"service_id": "svc-gone"})
		assert.True(t, isError)
		assert.True(t, strings.HasSuffix(text, "; searched 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z"+hint), text)
		assert.NotContains(t, text, "svc-gone")
		assert.NotContains(t, text, "shop-gone")
	})

	t.Run("HealthOverAnEarlierWindowFindsAServiceThatStopped", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, false))

		text, isError := callText(t, session, "pmm_service_health", with(window6hTo2h, "service_id", "svc-gone"))
		assert.False(t, isError, text)
		listTimes(t, fake, "up[14400s]")
		list := queriesWith(fake, "tlast_over_time(up[")
		assert.Equal(t, "1789380000", queryOf(t, list[0]).Get("time"))
	})

	t.Run("QueryDetailAndExplainGetPastTheList", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, true))

		callText(t, session, "pmm_query_detail", with(window6hTo2h, "queryid", "QID-AAA", "service_id", "svc-gone"))
		metrics := fake.requestsTo("/v1/qan:getMetrics")
		require.Len(t, metrics, 1)
		assert.Contains(t, string(metrics[0].body), "svc-gone")

		callText(t, session, "pmm_get_explain", with(window6hTo2h, "queryid", "QID-AAA", "service_id", "svc-gone"))
		reports := fake.requestsTo(reportPath)
		require.NotEmpty(t, reports)
		assert.Contains(t, string(reports[len(reports)-1].body), "svc-gone")
	})

	t.Run("TopQueriesPinsAServiceByNameInTheWindow", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_top_queries", with(window6hTo2h, "service_name", "shop-gone", "group_by", "queryid"))
		require.False(t, res.IsError, textOf(t, res))
		got := structured[topQueriesResult](t, res)
		require.NotEmpty(t, got.Rows)
		for _, row := range got.Rows {
			require.NotNil(t, row.QueryRef)
			assert.Equal(t, "svc-gone", row.QueryRef.ServiceID)
		}
	})

	t.Run("TopQueriesEngineLookupUsesTheWindow", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{
			fourHourKey:          {file: "vm_up_window.json"},
			"POST " + reportPath: {file: "report_quoted_identifier.json"},
		}))
		session := connect(t, newQANService(t, fake, false))

		res := callTool(t, session, "pmm_top_queries", with(window6hTo2h, "service_name", "shop-gone", "group_by", "queryid"))
		require.False(t, res.IsError, textOf(t, res))
		lists := queriesWith(fake, "tlast_over_time(up[")
		require.Len(t, lists, 1, "the engine and the query_ref lookups share one list")
		assert.Contains(t, queryOf(t, lists[0]).Get("query"), "up[14400s]")
		assert.Equal(t, "1789380000", queryOf(t, lists[0]).Get("time"))

		got := structured[topQueriesResult](t, res)
		require.NotEmpty(t, got.Rows)
		require.NotNil(t, got.Rows[0].QueryRef)
		assert.Equal(t, "svc-gone", got.Rows[0].QueryRef.ServiceID, "the query_ref lookup read the list")
		assert.Len(t, fake.requestsTo(reportPath), 1, "the engine lookup found mysql in the list, so no per-row engine report ran")
		assert.Empty(t, fake.requestsTo("/v1/qan/metrics:getFilters"))
	})

	t.Run("ALongWindowIsCutToItsLastSevenDays", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-secret", "period_from": "now-30d"})
		assert.True(t, isError)
		assert.True(t, strings.HasSuffix(text, "; searched 2026-09-07T12:00:00Z – 2026-09-14T12:00:00Z, the last 7 days of the call's window"), text)
		assert.NotContains(t, text, "svc-secret")
		listTimes(t, fake, "up[604800s]")
		list := queriesWith(fake, "tlast_over_time(up[")
		assert.Equal(t, "1789387200", queryOf(t, list[0]).Get("time"))
	})

	t.Run("AFutureEndReadsTheLastSevenDaysUpToNow", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, true))

		to := testNow.Add(day).Format(time.RFC3339)
		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-secret", "period_from": "now-20d", "period_to": to})
		assert.True(t, isError)
		assert.True(t, strings.HasSuffix(text, "; searched 2026-09-07T12:00:00Z – 2026-09-14T12:00:00Z, the last 7 days of the call's window up to now"), text)
		listTimes(t, fake, "up[604800s]")
		list := queriesWith(fake, "tlast_over_time(up[")
		assert.Equal(t, "1789387200", queryOf(t, list[0]).Get("time"))
	})

	t.Run("AShortWindowReadsFiveMinutes", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-1", "period_from": "now-30s"})
		assert.False(t, isError, text)
		listTimes(t, fake, "up[300s]")
		list := queriesWith(fake, "tlast_over_time(up[")
		assert.Equal(t, "1789387200", queryOf(t, list[0]).Get("time"))
	})

	t.Run("ConfigListsTheHourBeforeAt", func(t *testing.T) {
		t.Parallel()

		fake := newFake(t)
		session := connect(t, newQANService(t, fake, true))

		_, _ = callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1", "at": "now-3h"})
		listTimes(t, fake, "up[3600s]")
		list := queriesWith(fake, "tlast_over_time(up[")
		assert.Equal(t, "1789376400", queryOf(t, list[0]).Get("time"))
	})
}
