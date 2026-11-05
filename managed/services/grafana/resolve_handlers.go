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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultMinRenderPNGBytes rejects tiny PNG bodies that usually indicate an empty or error panel.
	defaultMinRenderPNGBytes int64 = 2048
	// MaxCachedRenderBlobBytes caps internal reads (e.g. Slack uploads) from the Tier-1 PNG cache.
	maxCachedRenderBlobBytes int64 = 15 << 20
)

var (
	safeUIDRe     = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	safePanelIDRe = regexp.MustCompile(`^[0-9]+$`)
	hexSHA256Re   = regexp.MustCompile(`^[a-f0-9]{64}$`)

	grafanaResolveRequestsTotal = prom.NewCounter(prom.CounterOpts{
		Name: "pmm_grafana_resolve_requests_total",
		Help: "POST /v1/grafana/render/resolve requests completed (any outcome).",
	})
	grafanaResolveCacheHitsTotal = prom.NewCounter(prom.CounterOpts{
		Name: "pmm_grafana_resolve_cache_hits_total",
		Help: "Resolve requests that returned a cached PNG without calling Grafana render.",
	})
	grafanaResolveRenderSeconds = prom.NewHistogram(prom.HistogramOpts{
		Name:    "pmm_grafana_resolve_render_latency_seconds",
		Help:    "Time spent waiting on Grafana image renderer when cache missed.",
		Buckets: prom.ExponentialBuckets(0.05, 2, 12), //nolint:mnd
	})
	grafanaResolveErrorsTotal = prom.NewCounterVec(prom.CounterOpts{
		Name: "pmm_grafana_resolve_errors_total",
		Help: "Resolve failures by reason label.",
	}, []string{"reason"})
)

func init() {
	prom.MustRegister(
		grafanaResolveRequestsTotal,
		grafanaResolveCacheHitsTotal,
		grafanaResolveRenderSeconds,
		grafanaResolveErrorsTotal,
	)
}

// ResolveHandler serves POST /v1/grafana/render/resolve.
type ResolveHandler struct {
	client *Client
	cache  *dashboardJSONCache
	l      *logrus.Entry
}

// NewResolveHandler returns a handler that resolves dashboard variables server-side and caches PNG blobs.
func NewResolveHandler(client *Client) *ResolveHandler {
	return &ResolveHandler{
		client: client,
		cache:  newDashboardJSONCache(defaultDashboardCacheTTL),
		l:      logrus.WithField("component", "grafana/resolve"),
	}
}

// resolveRequest is the JSON body for POST /v1/grafana/render/resolve.
type resolveRequest struct {
	DashboardUID     string            `json:"dashboard_uid"`
	PanelID          json.RawMessage   `json:"panel_id"`
	From             string            `json:"from"`
	To               string            `json:"to"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	Scale            int               `json:"scale"`
	OrgID            int               `json:"org_id"`
	TZ               string            `json:"tz"`
	Theme            string            `json:"theme"`
	Overrides        map[string]string `json:"overrides"`
	RefreshDashboard bool              `json:"refresh_dashboard"`
}

// resolveResponse is the JSON body for a successful resolve.
type resolveResponse struct {
	ContentHash  string            `json:"content_hash"`
	ImageURL     string            `json:"image_url"`
	DashboardURL string            `json:"dashboard_url"`
	ResolvedVars map[string]string `json:"resolved_vars"`
	CacheHit     bool              `json:"cache_hit"`
	RenderMs     int64             `json:"render_ms"`
}

func parsePanelIDField(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", errors.New("panel_id is required")
	}
	var n json.Number
	err := json.Unmarshal(raw, &n)
	if err == nil {
		panel := strings.TrimSpace(n.String())
		if !safePanelIDRe.MatchString(panel) {
			return "", errors.New("panel_id must be an integer")
		}
		return panel, nil
	}
	var s string
	err = json.Unmarshal(raw, &s)
	if err == nil {
		return NormalizePanelID(s), nil
	}
	return "", errors.New("invalid panel_id")
}

func buildDashboardViewerURL(dashboardUID, panelID, from, to string, mergedVars map[string]string) string {
	fromParam, toParam := from, to
	if fromMs, toMs, ok := isoToEpochMs(from, to); ok {
		fromParam = strconv.FormatInt(fromMs, 10)
		toParam = strconv.FormatInt(toMs, 10)
	}
	q := url.Values{}
	q.Set("viewPanel", panelID)
	q.Set("from", fromParam)
	q.Set("to", toParam)
	for k, v := range mergedVars {
		if strings.HasPrefix(k, "var-") && v != "" {
			q.Set(k, v)
		}
	}
	return "/graph/d/" + dashboardUID + "?" + q.Encode()
}

func requestBaseURL(r *http.Request) string {
	proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return ""
	}
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	return proto + "://" + host
}

func absoluteURL(base, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if base == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(base, "/") + path
}

// ServeHTTP implements POST /v1/grafana/render/resolve.
func (h *ResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}
	grafanaResolveRequestsTotal.Inc()

	var req resolveRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&req)
	if err != nil {
		grafanaResolveErrorsTotal.WithLabelValues("bad_json").Inc()
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body", err.Error())
		return
	}

	dashboardUID := strings.TrimSpace(req.DashboardUID)
	if dashboardUID == "" || !safeUIDRe.MatchString(dashboardUID) {
		grafanaResolveErrorsTotal.WithLabelValues("bad_dashboard_uid").Inc()
		writeJSONError(w, http.StatusBadRequest, "invalid or missing dashboard_uid", "")
		return
	}

	panelID, err := parsePanelIDField(req.PanelID)
	if err != nil || panelID == "" || !safePanelIDRe.MatchString(panelID) {
		grafanaResolveErrorsTotal.WithLabelValues("bad_panel_id").Inc()
		writeJSONError(w, http.StatusBadRequest, "invalid or missing panel_id", "")
		return
	}

	from := strings.TrimSpace(req.From)
	to := strings.TrimSpace(req.To)
	if from == "" || to == "" {
		grafanaResolveErrorsTotal.WithLabelValues("bad_time_range").Inc()
		writeJSONError(w, http.StatusBadRequest, "from and to are required", "")
		return
	}

	width, height, scale := req.Width, req.Height, req.Scale
	if width <= 0 {
		width = 1000
	}
	if height <= 0 {
		height = 500
	}
	if scale <= 0 {
		scale = 1
	}
	orgID := req.OrgID
	if orgID <= 0 {
		orgID = 1
	}
	tz := strings.TrimSpace(req.TZ)
	if tz == "" {
		tz = "browser"
	}
	theme := strings.TrimSpace(req.Theme)

	if req.RefreshDashboard {
		h.cache.Invalidate(orgID, dashboardUID)
	}

	headers := forwardAuthHeaders(r)

	loader := wrapFetchDashboard(h.client, dashboardUID, headers)
	env, err := h.cache.fetchOrLoad(r.Context(), h.client, orgID, dashboardUID, headers, loader)
	if err != nil {
		h.l.Warnf("dashboard fetch: %v", err)
		grafanaResolveErrorsTotal.WithLabelValues("dashboard_fetch").Inc()
		writeJSONError(w, http.StatusBadGateway, "failed to load dashboard JSON", err.Error())
		return
	}

	d := env.Dashboard
	if !panelExistsInDashboard(d, panelID) {
		grafanaResolveErrorsTotal.WithLabelValues("unknown_panel").Inc()
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("panel_id %s not found on dashboard", panelID), "")
		return
	}

	overrides := req.Overrides
	if overrides == nil {
		overrides = map[string]string{}
	}
	merged, err := MergeDashboardVars(d, overrides)
	if err != nil {
		grafanaResolveErrorsTotal.WithLabelValues("merge_vars").Inc()
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	fromN, toN := normalizeFromToForCanonical(from, to)

	hash := ContentHashFromRenderParams(RenderCanonicalParams{
		DashboardUID: dashboardUID,
		PanelID:      panelID,
		From:         fromN,
		To:           toN,
		OrgID:        orgID,
		Width:        width,
		Height:       height,
		Scale:        scale,
		TZ:           tz,
		Theme:        theme,
		Vars:         merged,
	})

	imageURL := "/v1/grafana/render/blob/" + hash + ".png"
	dashboardURL := buildDashboardViewerURL(dashboardUID, panelID, fromN, toN, merged)
	baseURL := requestBaseURL(r)
	imageURLAbs := absoluteURL(baseURL, imageURL)
	dashboardURLAbs := absoluteURL(baseURL, dashboardURL)

	cached, cacheErr := readBlobPNG(hash)
	if cacheErr == nil && len(cached) > 0 {
		grafanaResolveCacheHitsTotal.Inc()
		h.l.WithFields(logrus.Fields{
			"dashboard_uid": dashboardUID,
			"panel_id":      panelID,
			"content_hash":  hash,
			"cache_hit":     true,
		}).Debug("resolve cache hit")
		writeResolveJSON(w, http.StatusOK, resolveResponse{
			ContentHash:  hash,
			ImageURL:     imageURLAbs,
			DashboardURL: dashboardURLAbs,
			ResolvedVars: merged,
			CacheHit:     true,
			RenderMs:     0,
		})
		return
	}

	fromRender, toRender := fromToForGrafanaImageRendererQuery(fromN, toN)
	renderVals := buildGrafanaRenderQueryValues(panelID, fromRender, toRender, orgID, width, height, scale, tz, theme, merged)
	rawQuery := renderVals.Encode()

	t0 := time.Now()
	body, contentType, err := h.client.DoRaw(r.Context(), http.MethodGet, "/graph/render/d-solo/"+dashboardUID+"/", rawQuery, headers, nil)
	renderMs := time.Since(t0).Milliseconds()
	grafanaResolveRenderSeconds.Observe(time.Since(t0).Seconds())

	if err != nil {
		h.l.Warnf("Grafana render error (dashboard=%s panel=%s): %v", dashboardUID, panelID, err)
		grafanaResolveErrorsTotal.WithLabelValues("render_transport").Inc()
		writeJSONError(w, http.StatusBadGateway, "failed to render panel", err.Error())
		return
	}
	if !strings.HasPrefix(contentType, "image/") {
		h.l.Warnf("Grafana render returned non-image content-type %q (dashboard=%s panel=%s): %s", contentType, dashboardUID, panelID, truncateBodySnippet(body))
		grafanaResolveErrorsTotal.WithLabelValues("non_image").Inc()
		writeJSONError(w, http.StatusBadGateway, "Grafana returned non-image response", contentType)
		return
	}
	if int64(len(body)) < defaultMinRenderPNGBytes {
		grafanaResolveErrorsTotal.WithLabelValues("likely_empty_panel").Inc()
		writeJSONError(w, http.StatusBadGateway, "likely_empty_panel", fmt.Sprintf("PNG smaller than %d bytes", defaultMinRenderPNGBytes))
		return
	}

	err = writeBlobPNG(hash, body)
	if err != nil {
		h.l.Warnf("cache write: %v", err)
		grafanaResolveErrorsTotal.WithLabelValues("cache_write").Inc()
		writeJSONError(w, http.StatusInternalServerError, "failed to persist rendered PNG", err.Error())
		return
	}

	h.l.WithFields(logrus.Fields{
		"dashboard_uid": dashboardUID,
		"panel_id":      panelID,
		"content_hash":  hash,
		"cache_hit":     false,
		"render_ms":     renderMs,
	}).Info("resolve rendered and cached")

	writeResolveJSON(w, http.StatusOK, resolveResponse{
		ContentHash:  hash,
		ImageURL:     imageURLAbs,
		DashboardURL: dashboardURLAbs,
		ResolvedVars: merged,
		CacheHit:     false,
		RenderMs:     renderMs,
	})
}

func forwardAuthHeaders(r *http.Request) http.Header {
	h := make(http.Header)
	if auth := r.Header.Get("Authorization"); auth != "" {
		h.Set("Authorization", auth)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		h.Set("Cookie", cookie)
	}
	return h
}

func truncateBodySnippet(body []byte) string {
	if len(body) <= 1024 { //nolint:mnd
		return string(body)
	}
	return string(body[:1024]) + "…"
}

func writeResolveJSON(w http.ResponseWriter, status int, resp resolveResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson // response already committed
}

func writeJSONError(w http.ResponseWriter, status int, msg, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	m := map[string]string{"error": msg} //nolint:goconst
	if detail != "" {
		m["detail"] = detail
	}
	_ = json.NewEncoder(w).Encode(m) //nolint:errchkjson // response already committed
}

// BlobHandler serves GET /v1/grafana/render/blob/{sha256}.png from disk.
type BlobHandler struct {
	l *logrus.Entry
}

// NewBlobHandler returns a handler for immutable cached PNG blobs.
func NewBlobHandler() *BlobHandler {
	return &BlobHandler{l: logrus.WithField("component", "grafana/blob")}
}

// ServeHTTP expects path /v1/grafana/render/blob/<64-hex>.png .
func (h *BlobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}
	base := "/v1/grafana/render/blob/"
	if !strings.HasPrefix(r.URL.Path, base) {
		writeJSONError(w, http.StatusNotFound, "not found", "")
		return
	}
	suffix := strings.TrimPrefix(r.URL.Path, base)
	suffix = strings.TrimSpace(suffix)
	if !strings.HasSuffix(suffix, ".png") {
		writeBlobNotFound(w, suffix)
		return
	}
	hash := strings.TrimSuffix(suffix, ".png")
	if !hexSHA256Re.MatchString(hash) {
		writeJSONError(w, http.StatusBadRequest, "invalid content hash", "")
		return
	}
	data, err := readBlobPNG(hash)
	if err != nil {
		h.l.Debugf("blob miss %s: %v", hash, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{ //nolint:errchkjson // response already committed
			"error":        "snapshot not found",
			"content_hash": hash,
		})
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) //nolint:gosec // Content-Type=image/png; data is binary PNG addressed by sha256 (validated above)
}

func writeBlobNotFound(w http.ResponseWriter, suffix string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "snapshot not found", "detail": suffix}) //nolint:errchkjson // response already committed
}

// LegacyGETRenderGoneHandler responds to legacy GET /v1/grafana/render with 410 and migration hint.
type LegacyGETRenderGoneHandler struct{}

// NewLegacyGETRenderGoneHandler returns a handler that replaces the query-string render proxy.
func NewLegacyGETRenderGoneHandler() *LegacyGETRenderGoneHandler {
	return &LegacyGETRenderGoneHandler{}
}

// ServeHTTP implements GET-only 410 Gone with JSON body pointing at POST /resolve.
func (h *LegacyGETRenderGoneHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(map[string]string{ //nolint:errchkjson // response already committed
		"error":      "This endpoint was removed. Use POST /v1/grafana/render/resolve with a JSON body, then embed image_url (blob path).",
		"migration":  "POST /v1/grafana/render/resolve",
		"blob_fetch": "GET /v1/grafana/render/blob/{content_hash}.png",
	})
}

func readBlobPNG(contentHash string) ([]byte, error) {
	return os.ReadFile(blobPNGPath(contentHash))
}

// ReadCachedRenderBlob returns PNG bytes from the Tier-1 render disk cache for a valid SHA-256 content hash
// (64 lowercase hex characters). Rejects invalid hashes and blobs larger than maxCachedRenderBlobBytes.
// Callers that serve arbitrary HTTP clients should use BlobHandler; this API is for trusted in-process use (e.g. Slack uploads).
func ReadCachedRenderBlob(contentHash string) ([]byte, error) {
	if !hexSHA256Re.MatchString(contentHash) {
		return nil, errors.New("invalid content hash")
	}
	f, err := os.Open(blobPNGPath(contentHash))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxCachedRenderBlobBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxCachedRenderBlobBytes {
		return nil, errors.New("cached blob exceeds max size")
	}
	return data, nil
}

func writeBlobPNG(contentHash string, body []byte) error {
	dir := renderCacheDir()
	err := os.MkdirAll(dir, 0o750) //nolint:mnd
	if err != nil {
		return err
	}
	return os.WriteFile(blobPNGPath(contentHash), body, 0o600) //nolint:mnd
}

func blobPNGPath(contentHash string) string {
	return filepath.Join(renderCacheDir(), contentHash+".png")
}

// renderCacheDir returns the on-disk directory for Tier-1 PNG blobs (override in tests).
func renderCacheDir() string {
	if s := grafanaRenderCacheDirForTest; s != "" {
		return s
	}
	return defaultRenderCacheDir
}

// grafanaRenderCacheDirForTest, when non-empty, overrides defaultRenderCacheDir (tests only).
var grafanaRenderCacheDirForTest string
