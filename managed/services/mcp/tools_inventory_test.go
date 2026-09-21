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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inventoryRoutes() map[string]fixture {
	return map[string]fixture{
		"GET /v1/server/version":                                        {file: "version.json"},
		"GET /v1/inventory/services":                                    {file: "services.json"},
		"GET /v1/inventory/nodes":                                       {file: "nodes.json"},
		"GET /graph/api/datasources":                                    {file: "datasources.json"},
		"GET /graph/api/datasources/proxy/uid/metrics-uid/api/v1/query": {file: "vm_mysql_version.json"},
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

func TestInventoryTool(t *testing.T) {
	t.Parallel()

	t.Run("Enriched", func(t *testing.T) {
		t.Parallel()

		// The metrics fixture is keyed by path only, so both engines' version
		// queries get the MySQL answer; PostgreSQL then degrades to unknown.
		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.False(t, isError)
		assert.Equal(t, strings.Join([]string{
			"Monitored services (3):",
			"- shop-mysql | mysql 8.0.46-37 | id=svc-1 | node=demo-node | address=mysql:3306",
			"- pmm-server-postgresql | postgresql unknown | id=svc-pmm-pg | node=pmm-server | address=127.0.0.1:5432",
			"- shop-postgres | postgresql unknown | id=svc-2 | node=demo-node | address=postgres:5432",
		}, "\n"), text)
		fake.assertForwardedAuth(t)

		// One version query per engine, through the uid-based datasource proxy.
		queries := fake.requestsTo("/graph/api/datasources/proxy/uid/metrics-uid/api/v1/query")
		require.Len(t, queries, 2)
		promql := make([]string, 0, len(queries))
		for _, q := range queries {
			promql = append(promql, q.query)
		}
		assert.ElementsMatch(t, []string{"query=mysql_version_info", "query=pg_static"}, promql)
	})

	t.Run("EngineAndNodeFilters", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "postgresql", "node": "demo-node"})
		assert.False(t, isError)
		assert.Equal(t, "Monitored services (1):\n- shop-postgres | postgresql unknown | id=svc-2 | node=demo-node | address=postgres:5432", text)

		text, isError = callText(t, session, "pmm_inventory", map[string]any{"engine": "mongodb"})
		assert.False(t, isError)
		assert.Equal(t, "No monitored services found.", text)
	})

	t.Run("InvalidEngine", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "oracle"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Empty(t, fake.requestsTo("/v1/inventory/services"))
	})

	t.Run("MetricsUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := inventoryRoutes()
		routes["GET /graph/api/datasources"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		fake := newFakePMM(t, routes)
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "mysql"})
		assert.False(t, isError)
		assert.Equal(t, "Monitored services (1):\n- shop-mysql | mysql unknown | id=svc-1 | node=demo-node | address=mysql:3306", text)
	})

	t.Run("NodesUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := inventoryRoutes()
		routes["GET /v1/inventory/nodes"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		fake := newFakePMM(t, routes)
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{"engine": "mysql"})
		assert.False(t, isError)
		assert.Equal(t, "Monitored services (1):\n- shop-mysql | mysql 8.0.46-37 | id=svc-1 | address=mysql:3306", text)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()

		routes := inventoryRoutes()
		routes["GET /v1/inventory/services"] = fixture{file: "error_401.json", status: http.StatusUnauthorized}
		fake := newFakePMM(t, routes)
		session := connect(t, newTestService(t, fake))

		text, isError := callText(t, session, "pmm_inventory", map[string]any{})
		assert.True(t, isError)
		assert.Equal(t, "error: unauthorized\nPMM returned 401 on GET /v1/inventory/services: Unauthorized", text)
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
