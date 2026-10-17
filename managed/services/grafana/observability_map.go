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
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// ObservabilityMapHandler serves GET /v1/grafana/observability-map.
type ObservabilityMapHandler struct {
	client *Client
	cache  *dashboardJSONCache
	l      *logrus.Entry
}

// NewObservabilityMapHandler returns a handler for intent-based dashboard/panel routing.
func NewObservabilityMapHandler(client *Client) *ObservabilityMapHandler {
	return &ObservabilityMapHandler{
		client: client,
		cache:  newDashboardJSONCache(defaultDashboardCacheTTL),
		l:      logrus.WithField("component", "grafana/observability-map"),
	}
}

type observabilityMapResponse struct {
	Engine  Engine `json:"engine"`
	Intent  Intent `json:"intent"`
	Primary struct {
		DashboardUID string `json:"dashboard_uid"`
		Title        string `json:"title"`
		UseWhen      string `json:"use_when"`
	} `json:"primary"`
	Panels    []PanelQuery     `json:"panels,omitempty"`
	Secondary []secondaryRoute `json:"secondary,omitempty"`
	Fallback  struct {
		MetricPrefix      string `json:"metric_prefix"`
		ScopedSeriesMatch string `json:"scoped_series_match"`
	} `json:"fallback"`
	Warnings []string `json:"warnings,omitempty"`
}

// ServeHTTP implements http.Handler.
func (h *ObservabilityMapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { //nolint:gocognit
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	engine, err := ParseEngine(r.URL.Query().Get("engine"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	intent, err := ParseIntent(r.URL.Query().Get("intent"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	route, err := LookupRoute(engine, intent)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error(), "")
		return
	}

	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	panelIDs := route.PanelIDs
	if raw := strings.TrimSpace(r.URL.Query().Get("panel_ids")); raw != "" {
		panelIDs = nil
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, convErr := strconv.Atoi(part)
			if convErr != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid panel_ids", convErr.Error())
				return
			}
			panelIDs = append(panelIDs, id)
		}
	}

	includeQueries := true
	if v := strings.TrimSpace(r.URL.Query().Get("include_panel_queries")); v != "" {
		includeQueries, err = strconv.ParseBool(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid include_panel_queries", err.Error())
			return
		}
	}

	resp := observabilityMapResponse{
		Engine: engine,
		Intent: intent,
	}
	resp.Primary.DashboardUID = route.DashboardUID
	resp.Primary.Title = route.Title
	resp.Primary.UseWhen = route.UseWhen
	resp.Secondary = route.Secondary
	resp.Fallback.MetricPrefix = route.FallbackPrefix
	resp.Fallback.ScopedSeriesMatch = scopedSeriesMatch(serviceID, route.FallbackPrefix)

	if includeQueries {
		orgID := 1
		if v := strings.TrimSpace(r.URL.Query().Get("org_id")); v != "" {
			orgID, err = strconv.Atoi(v)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid org_id", err.Error())
				return
			}
		}
		headers := forwardAuthHeaders(r)
		env, fetchErr := h.cache.fetchOrLoad(r.Context(), h.client, orgID, route.DashboardUID, headers,
			wrapFetchDashboard(h.client, route.DashboardUID, headers))
		if fetchErr != nil {
			h.l.Errorf("fetch dashboard %s: %v", route.DashboardUID, fetchErr)
			writeJSONError(w, http.StatusBadGateway, "failed to load dashboard", fetchErr.Error())
			return
		}
		panels := extractPanelQueries(env.Dashboard, panelIDSet(panelIDs))
		found := make(map[int]struct{}, len(panels))
		filtered := make([]PanelQuery, 0, len(panels))
		for _, p := range panels {
			found[p.ID] = struct{}{}
			if strings.TrimSpace(p.Expr) == "" {
				resp.Warnings = append(resp.Warnings,
					"panel_id "+strconv.Itoa(p.ID)+" ("+p.Title+") has no PromQL expr")
				continue
			}
			filtered = append(filtered, p)
		}
		for _, id := range panelIDs {
			if _, ok := found[id]; !ok {
				resp.Warnings = append(resp.Warnings, "panel_id "+strconv.Itoa(id)+" not found on dashboard "+route.DashboardUID)
			}
		}
		resp.Panels = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}
