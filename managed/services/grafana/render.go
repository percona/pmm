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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultRenderCacheDir = "/srv/pmm/grafana_render_cache"

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

	// Grafana render API expects panelId like "panel-92"; normalize if we got numeric "92".
	grafanaPanelID := panelID
	if grafanaPanelID != "" && !strings.HasPrefix(grafanaPanelID, "panel-") {
		grafanaPanelID = "panel-" + grafanaPanelID
	}
	// Build query string for Grafana render (and for image_url): panelId, from, to, width, height, scale, tz, orgId, and var-*
	renderParams := url.Values{}
	renderParams.Set("panelId", grafanaPanelID)
	renderParams.Set("orgId", "1")
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
	// Copy our own params for image_url (excluding format=json and cache)
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
	if q.Get("cache") == "1" {
		imageURLParams.Set("cache", "1")
	}

	wantJSON := q.Get("format") == "json" || strings.Contains(r.Header.Get("Accept"), "application/json")
	useCache := q.Get("cache") == "1"

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

	// Optional disk cache: only when cache=1
	if useCache {
		cacheKey := renderCacheKey(imageURLParams)
		if cached, err := h.readRenderCache(cacheKey); err == nil {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(cached)
			return
		}
	}

	// Call Grafana render API. Path must include /graph prefix (serve_from_sub_path = true in grafana.ini).
	rawQuery := renderParams.Encode()
	headers := make(http.Header)
	if auth := r.Header.Get("Authorization"); auth != "" {
		headers.Set("Authorization", auth)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		headers.Set("Cookie", cookie)
	}

	path := "/graph/render/d-solo/" + dashboardUID + "/"
	body, contentType, err := h.client.DoRaw(r.Context(), http.MethodGet, path, rawQuery, headers, nil)
	if err != nil {
		h.l.Warnf("Grafana render: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to render panel"})
		return
	}
	if useCache && len(body) > 0 {
		_ = h.writeRenderCache(renderCacheKey(imageURLParams), body)
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = w.Write(body)
}

// renderCacheKey returns a stable hash key from image params (excluding format and cache); same params => same key.
func renderCacheKey(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "format" || k == "cache" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		for _, v := range params[k] {
			_, _ = h.Write([]byte(k))
			_, _ = h.Write([]byte("\x00"))
			_, _ = h.Write([]byte(v))
			_, _ = h.Write([]byte("\x00"))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (h *RenderHandler) readRenderCache(key string) ([]byte, error) {
	dir := defaultRenderCacheDir
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(dir, key))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (h *RenderHandler) writeRenderCache(key string, body []byte) error {
	dir := defaultRenderCacheDir
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, key), body, 0o644)
}
