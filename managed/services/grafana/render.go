// Copyright (C) 2025 Percona LLC
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

package grafana

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// safeUIDRe allows only dashboard UID and panel ID safe characters (alphanumeric, dash, underscore, dot).
var (
	safeUIDRe     = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	safePanelIDRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

// isoToEpochMs parses from/to as ISO 8601 and returns epoch milliseconds for Grafana dashboard URL. If either parse fails, returns false.
func isoToEpochMs(from, to string) (fromMs, toMs int64, ok bool) {
	parse := func(s string) (int64, bool) {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, s)
		}
		if err != nil {
			return 0, false
		}
		return t.UnixMilli(), true
	}
	fms, okFrom := parse(from)
	tms, okTo := parse(to)
	if !okFrom || !okTo {
		return 0, 0, false
	}
	return fms, tms, true
}

// RenderHandler serves GET /v1/grafana/render: proxies Grafana panel render API
// or returns JSON with image_url and dashboard_url when format=json or Accept: application/json.
type RenderHandler struct {
	client *Client
	l      *logrus.Entry
}

// NewRenderHandler returns a new RenderHandler.
func NewRenderHandler(client *Client) *RenderHandler {
	return &RenderHandler{
		client: client,
		l:      logrus.WithField("component", "grafana/render"),
	}
}

// renderResponse is returned when the client requests JSON (format=json or Accept: application/json).
type renderResponse struct {
	ImageURL     string `json:"image_url"`
	DashboardURL string `json:"dashboard_url"`
}

// ServeHTTP handles GET /v1/grafana/render.
func (h *RenderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	q := r.URL.Query()
	dashboardUID := q.Get("dashboard_uid")
	panelID := q.Get("panel_id")
	from := q.Get("from")
	to := q.Get("to")
	if dashboardUID == "" || panelID == "" || from == "" || to == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "missing required query parameters: dashboard_uid, panel_id, from, to",
		})
		return
	}
	if !safeUIDRe.MatchString(dashboardUID) || !safePanelIDRe.MatchString(panelID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid dashboard_uid or panel_id",
		})
		return
	}

	width := q.Get("width")
	if width == "" {
		width = "1000"
	}
	height := q.Get("height")
	if height == "" {
		height = "500"
	}

	// Build query string for Grafana render (and for image_url): panelId, from, to, width, height, scale, tz, and var-*
	renderParams := url.Values{}
	renderParams.Set("panelId", panelID)
	renderParams.Set("from", from)
	renderParams.Set("to", to)
	renderParams.Set("width", width)
	renderParams.Set("height", height)
	renderParams.Set("scale", "1")
	renderParams.Set("tz", "browser")
	for k, v := range q {
		if strings.HasPrefix(k, "var-") && len(v) > 0 {
			renderParams.Set(k, v[0])
		}
	}
	// Copy our own params for image_url (excluding format=json)
	imageURLParams := url.Values{}
	imageURLParams.Set("dashboard_uid", dashboardUID)
	imageURLParams.Set("panel_id", panelID)
	imageURLParams.Set("from", from)
	imageURLParams.Set("to", to)
	imageURLParams.Set("width", width)
	imageURLParams.Set("height", height)
	for k, v := range q {
		if strings.HasPrefix(k, "var-") && len(v) > 0 {
			imageURLParams.Set(k, v[0])
		}
	}

	wantJSON := q.Get("format") == "json" || strings.Contains(r.Header.Get("Accept"), "application/json")

	if wantJSON {
		imageURL := "/v1/grafana/render?" + imageURLParams.Encode()
		// Grafana dashboard URL accepts from/to in epoch ms or relative (e.g. now-12h); ISO 8601 can show wrong range (1970). Use epoch ms for dashboard_url.
		fromParam, toParam := from, to
		if fromMs, toMs, ok := isoToEpochMs(from, to); ok {
			fromParam = strconv.FormatInt(fromMs, 10)
			toParam = strconv.FormatInt(toMs, 10)
		}
		dashboardURL := "/graph/d/" + dashboardUID + "?viewPanel=" + panelID + "&from=" + url.QueryEscape(fromParam) + "&to=" + url.QueryEscape(toParam)
		for k, v := range imageURLParams {
			if strings.HasPrefix(k, "var-") && len(v) > 0 {
				dashboardURL += "&" + url.QueryEscape(k) + "=" + url.QueryEscape(v[0])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(renderResponse{
			ImageURL:     imageURL,
			DashboardURL: dashboardURL,
		})
		return
	}

	// Proxy to Grafana render API
	path := "/render/d-solo/" + dashboardUID + "/"
	rawQuery := renderParams.Encode()
	headers := make(http.Header)
	if auth := r.Header.Get("Authorization"); auth != "" {
		headers.Set("Authorization", auth)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		headers.Set("Cookie", cookie)
	}

	body, contentType, err := h.client.DoRaw(r.Context(), http.MethodGet, path, rawQuery, headers, nil)
	if err != nil {
		h.l.Warnf("Grafana render: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to render panel"})
		return
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = w.Write(body)
}
