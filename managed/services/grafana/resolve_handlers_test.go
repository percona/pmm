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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyGETRenderGone(t *testing.T) {
	h := NewLegacyGETRenderGoneHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/grafana/render", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusGone, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body["error"], "POST /v1/grafana/render/resolve")
}

func TestResolveRejectsNonIntegerPanelID(t *testing.T) {
	h := NewResolveHandler(NewClient("127.0.0.1:1"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/grafana/render/resolve", strings.NewReader(`{"dashboard_uid":"test-dash","panel_id":12.5,"from":"2026-01-01T00:00:00Z","to":"2026-01-01T01:00:00Z"}`))
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestResolveSecondRequestUsesDiskCache(t *testing.T) {
	grafanaRenderCacheDirForTest = t.TempDir()
	t.Cleanup(func() { grafanaRenderCacheDirForTest = "" })

	dashboardJSON, err := os.ReadFile(filepath.Join("testdata", "dashboard_merge.json"))
	require.NoError(t, err)

	var renderCalls atomic.Int32
	pngBody := bytes.Repeat([]byte{0x0d}, 2500)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/graph/api/dashboards/uid/test-dash"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(dashboardJSON)
		case strings.HasPrefix(r.URL.Path, "/graph/render/d-solo/test-dash"):
			renderCalls.Add(1)
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBody)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://")
	client := NewClient(host)
	h := NewResolveHandler(client)

	body := `{"dashboard_uid":"test-dash","panel_id":12,"from":"2026-01-01T00:00:00Z","to":"2026-01-01T01:00:00Z","overrides":{"service_name":"svc-default"}}`
	do := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/grafana/render/resolve", strings.NewReader(body))
		h.ServeHTTP(rec, req)
		return rec
	}

	rec1 := do()
	require.Equal(t, http.StatusOK, rec1.Code, rec1.Body.String())
	var out1 resolveResponse
	require.NoError(t, json.NewDecoder(rec1.Body).Decode(&out1))
	assert.False(t, out1.CacheHit)
	assert.Equal(t, 1, int(renderCalls.Load()))

	rec2 := do()
	require.Equal(t, http.StatusOK, rec2.Code)
	var out2 resolveResponse
	require.NoError(t, json.NewDecoder(rec2.Body).Decode(&out2))
	assert.True(t, out2.CacheHit)
	assert.Equal(t, out1.ContentHash, out2.ContentHash)
	assert.Equal(t, 1, int(renderCalls.Load()))
}

func TestResolveReturnsAbsoluteURLs(t *testing.T) {
	grafanaRenderCacheDirForTest = t.TempDir()
	t.Cleanup(func() { grafanaRenderCacheDirForTest = "" })

	dashboardJSON, err := os.ReadFile(filepath.Join("testdata", "dashboard_merge.json"))
	require.NoError(t, err)

	pngBody := bytes.Repeat([]byte{0x0d}, 2500)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/graph/api/dashboards/uid/test-dash"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(dashboardJSON)
		case strings.HasPrefix(r.URL.Path, "/graph/render/d-solo/test-dash"):
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBody)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://")
	h := NewResolveHandler(NewClient(host))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/grafana/render/resolve", strings.NewReader(`{"dashboard_uid":"test-dash","panel_id":12,"from":"2026-01-01T00:00:00Z","to":"2026-01-01T01:00:00Z"}`))
	req.Host = "pmm.example:8443"
	req.Header.Set("X-Forwarded-Proto", "https")
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out resolveResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.True(t, strings.HasPrefix(out.ImageURL, "https://pmm.example:8443/v1/grafana/render/blob/"))
	assert.True(t, strings.HasPrefix(out.DashboardURL, "https://pmm.example:8443/graph/d/test-dash?"))
}

func TestReadCachedRenderBlob(t *testing.T) {
	t.Run("invalid_hash", func(t *testing.T) {
		_, err := ReadCachedRenderBlob("not-a-hash")
		require.Error(t, err)
	})
	t.Run("miss", func(t *testing.T) {
		grafanaRenderCacheDirForTest = t.TempDir()
		t.Cleanup(func() { grafanaRenderCacheDirForTest = "" })
		hash := strings.Repeat("a", 64)
		_, err := ReadCachedRenderBlob(hash)
		require.Error(t, err)
	})
	t.Run("hit", func(t *testing.T) {
		grafanaRenderCacheDirForTest = t.TempDir()
		t.Cleanup(func() { grafanaRenderCacheDirForTest = "" })
		hash := strings.Repeat("b", 64)
		path := filepath.Join(grafanaRenderCacheDirForTest, hash+".png")
		payload := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
		require.NoError(t, os.WriteFile(path, payload, 0o644))
		got, err := ReadCachedRenderBlob(hash)
		require.NoError(t, err)
		assert.Equal(t, payload, got)
	})
}
