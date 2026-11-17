// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package grafana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEngineIntent(t *testing.T) {
	t.Parallel()

	engine, err := ParseEngine("mysql")
	require.NoError(t, err)
	assert.Equal(t, EngineMySQL, engine)

	intent, err := ParseIntent("wal")
	require.NoError(t, err)
	assert.Equal(t, IntentWAL, intent)
}

func TestLookupRouteValkey(t *testing.T) {
	t.Parallel()

	route, err := LookupRoute(EngineValkey, IntentMemory)
	require.NoError(t, err)
	assert.Equal(t, "valkey-memory", route.DashboardUID)
	assert.Contains(t, route.PanelIDs, 31)
}

func TestExtractPanelQueries(t *testing.T) {
	t.Parallel()

	d := dashboardInner{
		Panels: []dashboardPanel{
			{
				ID:    8,
				Type:  "graph",
				Title: "Handlers",
				Targets: []panelTarget{
					{Expr: "rate(mysql_global_status_handlers_total[5m])", LegendFormat: "{{handler}}"},
				},
			},
			{Type: "row", Panels: []dashboardPanel{
				{
					ID:    92,
					Type:  "graph",
					Title: "Connections",
					Targets: []panelTarget{
						{Expr: "mysql_global_status_threads_connected"},
					},
				},
			}},
		},
	}
	panels := extractPanelQueries(d, panelIDSet([]int{8, 92}))
	require.Len(t, panels, 2)
	assert.Equal(t, "rate(mysql_global_status_handlers_total[5m])", panels[0].Expr)
}

func TestMergePanelTargetsUsesFirstExpr(t *testing.T) {
	t.Parallel()

	expr, legend := mergePanelTargets([]panelTarget{
		{Expr: "sum(rate(a[5m]))", LegendFormat: "a"},
		{Expr: "sum(rate(b[5m]))", LegendFormat: "b"},
	})
	assert.Equal(t, "sum(rate(a[5m]))", expr)
	assert.Equal(t, "a", legend)
}

func TestObservabilityMapHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	h := NewObservabilityMapHandler(NewClient("127.0.0.1:1"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/grafana/observability-map?engine=mysql&intent=workload", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestObservabilityMapHandler_RouteWithoutPanelQueries(t *testing.T) {
	t.Parallel()

	h := NewObservabilityMapHandler(NewClient("127.0.0.1:1"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/grafana/observability-map?engine=mysql&intent=workload&include_panel_queries=false", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp observabilityMapResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "mysql-instance-summary", resp.Primary.DashboardUID)
	assert.Empty(t, resp.Panels)
}

func TestObservabilityMapHandler_OmitsEmptyExprPanels(t *testing.T) {
	dashboardJSON := []byte(`{
		"dashboard": {
			"panels": [
				{"id": 8, "type": "graph", "title": "WithExpr", "targets": [{"expr": "up"}]},
				{"id": 9, "type": "text", "title": "NoExpr"}
			]
		}
	}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/graph/api/dashboards/uid/mysql-instance-summary") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(dashboardJSON)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://")
	h := NewObservabilityMapHandler(NewClient(host))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/grafana/observability-map?engine=mysql&intent=workload&panel_ids=8,9", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp observabilityMapResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Panels, 1)
	assert.Equal(t, 8, resp.Panels[0].ID)
	assert.NotEmpty(t, resp.Warnings)
}
