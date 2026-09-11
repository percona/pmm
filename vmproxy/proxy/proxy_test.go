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

package proxy

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	headerName = "X-Test-Header"
	targetURL  = "http://127.0.0.1"
	// Requests must carry a path the proxy allows through, or the read-only
	// allow-list refuses them before anything else under test runs.
	requestURL = "http://127.0.0.1/api/v1/query"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, filters []string, headers map[string]string) http.HandlerFunc {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if filters != nil {
				assert.Equal(t, url.Values{"extra_filters[]": filters}.Encode(), r.URL.RawQuery)
			}
			for k, v := range headers {
				assert.Equal(t, v, r.Header.Get(k))
			}
		}))
		t.Cleanup(func() {
			server.Close()
		})

		testURL, err := url.Parse(server.URL)
		require.NoError(t, err)

		handler := getHandler(Config{
			HeaderName: headerName,
			TargetURL:  testURL,
		})

		return handler
	}

	t.Run("shall proxy request", func(t *testing.T) {
		t.Parallel()
		handler := setup(t, nil, nil)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, requestURL, nil)
		uri, err := url.Parse(targetURL)
		require.NoError(t, err)

		prepareRequest(req, uri, headerName)

		handler.ServeHTTP(rec, req)
		resp := rec.Result()
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("shall properly handle filters", func(t *testing.T) {
		t.Parallel()

		type testParams struct {
			expectedFilters []string
			expectedStatus  int
			expectedHeader  map[string]string
			headerContent   string
			name            string
			requestURL      string
			targetURL       string
		}

		testCases := []testParams{
			{
				name:            "shall process filters properly",
				expectedFilters: []string{"abc", "def"},
				expectedStatus:  http.StatusOK,
				headerContent:   base64.StdEncoding.EncodeToString([]byte(`["abc", "def"]`)),
			},
			{
				name:            "shall process PromQL strings properly",
				expectedFilters: []string{`{region="east", env="prod"}`, `{region="west", env="dev"}`},
				expectedStatus:  http.StatusOK,
				headerContent:   base64.StdEncoding.EncodeToString([]byte(`["{region=\"east\", env=\"prod\"}", "{region=\"west\", env=\"dev\"}"]`)),
			},
			{
				name:            "shall replace existing extra_filters",
				expectedFilters: []string{"abc", "def"},
				expectedStatus:  http.StatusOK,
				headerContent:   base64.StdEncoding.EncodeToString([]byte(`["abc", "def"]`)),
				requestURL:      "http://127.0.0.1/api/v1/query?extra_filters[]=a&extra_filters[]=b",
			},
			{
				name:            "shall support empty JSON array with no filters",
				expectedFilters: []string{},
				expectedStatus:  http.StatusOK,
				headerContent:   base64.StdEncoding.EncodeToString([]byte(`[]`)),
			},
			{
				name:            "shall not fail on invalid base64 string",
				expectedFilters: []string{},
				expectedStatus:  http.StatusPreconditionFailed,
				headerContent:   "invalid",
			},
			{
				name:            "shall not fail on invalid JSON",
				expectedFilters: nil,
				expectedStatus:  http.StatusPreconditionFailed,
				headerContent:   base64.StdEncoding.EncodeToString([]byte(`"abc, "def"]`)),
			},
			{
				name:            "shall add authorization header",
				expectedFilters: []string{"abc", "def"},
				expectedHeader: map[string]string{
					"Authorization": "Basic dm1hZG1pbjp2bXBhc3M=",
				},
				expectedStatus: http.StatusOK,
				headerContent:  base64.StdEncoding.EncodeToString([]byte(`["abc", "def"]`)),
				targetURL:      "http://vmadmin:vmpass@127.0.0.1",
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				testTargetURL := targetURL
				if tc.targetURL != "" {
					testTargetURL = tc.targetURL
				}

				testRequestURL := requestURL
				if tc.requestURL != "" {
					testRequestURL = tc.requestURL
				}

				handler := setup(t, tc.expectedFilters, tc.expectedHeader)

				rec := httptest.NewRecorder()
				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testRequestURL, nil)

				uri, err := url.Parse(testTargetURL)
				require.NoError(t, err)
				prepareRequest(req, uri, headerName)
				req.Header.Set(headerName, tc.headerContent)

				handler.ServeHTTP(rec, req)
				resp := rec.Result()
				t.Cleanup(func() {
					assert.NoError(t, resp.Body.Close())
				})

				require.Equal(t, tc.expectedStatus, resp.StatusCode)
			})
		}
	})

	t.Run("prepareRequest: set targetURL host as Host header value", func(t *testing.T) {
		t.Parallel()

		headerName := "Host"

		type testParams struct {
			name      string
			targetURL string
		}

		testCases := []testParams{
			{
				name:      "targetURL for external VM",
				targetURL: "https://my-external-vm.example.org:8443/",
			},
			{
				name:      "targetURL for local VM by IP",
				targetURL: "http://127.0.0.1:8430/",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				url, err := url.Parse(tc.targetURL)
				require.NoError(t, err)
				expectedHost := url.Host
				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, targetURL, nil)

				prepareRequest(req, url, headerName)

				require.NotNil(t, req.Header[headerName])
				require.Equal(t, expectedHost, req.Header[headerName][0])
			})
		}
	})

	t.Run("shall refuse paths outside the read-only allow-list", func(t *testing.T) {
		t.Parallel()

		// Grafana forwards arbitrary sub-paths to the data source, so the proxy is the
		// only thing standing between a dashboard user and the rest of VictoriaMetrics.
		allowed := []string{
			"/api/v1/query",
			"/api/v1/query_range",
			"/api/v1/query_exemplars",
			"/api/v1/series",
			"/api/v1/labels",
			"/api/v1/label/node_name/values",
			"/api/v1/metadata",
			"/api/v1/rules",
			"/api/v1/alerts",
			"/api/v1/status/buildinfo",
			"/api/v1/export",
			// Documented admin diagnostics.
			"/targets",
			"/api/v1/targets",
			"/api/v1/status/tsdb",
			// nginx passes the original URI for the /prometheus/api/v1 location.
			"/prometheus/api/v1/query",
			"/prometheus/api/v1/label/node_name/values",
			"/prometheus/api/v1/status/tsdb",
		}
		refused := []string{
			"/",
			"/snapshot/create",
			"/snapshot/list",
			"/prometheus/snapshot/create",
			"/api/v1/admin/tsdb/delete_series",
			"/api/v1/write",
			"/api/v1/import",
			// The whole scrape configuration, unlike the diagnostics allowed above.
			"/api/v1/status/config",
			"/metrics",
			"/flags",
			"/debug/pprof/heap",
			// Traversal must not walk out of an allowed prefix.
			"/api/v1/query/../../snapshot/create",
			"/api/v1/label/../../snapshot/create/values",
			// The label name is a single segment, so these are not label lookups.
			"/api/v1/label//values",
			"/api/v1/label/a/b/values",
		}

		for _, p := range allowed {
			assert.Truef(t, isPathAllowed(p, false), "expected %s to be allowed", p)
		}
		for _, p := range refused {
			assert.Falsef(t, isPathAllowed(p, false), "expected %s to be refused", p)
		}
	})

	t.Run("shall gate the admin-only diagnostics on the marker", func(t *testing.T) {
		t.Parallel()

		const adminPath = "/api/v1/status/config"

		assert.False(t, isPathAllowed(adminPath, false), "must be refused without the marker")
		assert.True(t, isPathAllowed(adminPath, true), "must be allowed with the marker")
		assert.True(t, isPathAllowed("/prometheus"+adminPath, true), "nginx passes the prefixed form")

		// The marker widens the allow-list, it does not disable it: an admin still cannot
		// reach an endpoint nobody listed.
		for _, p := range []string{"/snapshot/create", "/api/v1/admin/tsdb/delete_series", "/debug/pprof/heap"} {
			assert.Falsef(t, isPathAllowed(p, true), "expected %s to stay refused for an admin", p)
		}
	})

	t.Run("shall honour the marker only from the configured header", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, requestURL, nil)
		assert.False(t, isAdminRequest(req, "X-Proxy-Admin"), "absent header is not an admin")

		req.Header.Set("X-Proxy-Admin", "1")
		assert.True(t, isAdminRequest(req, "X-Proxy-Admin"))
		assert.False(t, isAdminRequest(req, ""), "an unconfigured header name never matches")

		req.Header.Set("X-Proxy-Admin", "true")
		assert.False(t, isAdminRequest(req, "X-Proxy-Admin"), "only the exact value counts")
	})

	t.Run("shall serve an admin-only path when the marker is present", func(t *testing.T) {
		t.Parallel()

		proxied := false
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			proxied = true
		}))
		t.Cleanup(server.Close)

		uri, err := url.Parse(server.URL)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/api/v1/status/config", nil)
		req.Header.Set("X-Proxy-Admin", "1")

		getHandler(Config{HeaderName: headerName, AdminHeaderName: "X-Proxy-Admin", TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, proxied, "an admin request must reach VictoriaMetrics")
	})

	t.Run("shall refuse an admin-only path without the marker", func(t *testing.T) {
		t.Parallel()

		proxied := false
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			proxied = true
		}))
		t.Cleanup(server.Close)

		uri, err := url.Parse(server.URL)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/api/v1/status/config", nil)

		getHandler(Config{HeaderName: headerName, AdminHeaderName: "X-Proxy-Admin", TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		require.False(t, proxied, "refused request must not reach VictoriaMetrics")
	})

	t.Run("shall answer a refused path with 403 without proxying", func(t *testing.T) {
		t.Parallel()

		proxied := false
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			proxied = true
		}))
		t.Cleanup(server.Close)

		uri, err := url.Parse(server.URL)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/snapshot/create", nil)

		getHandler(Config{HeaderName: headerName, TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		require.False(t, proxied, "refused request must not reach VictoriaMetrics")
	})

	t.Run("prepareRequest: add credentials to request", func(t *testing.T) {
		t.Parallel()

		uri, err := url.Parse(targetURL)
		require.NoError(t, err)

		username := "user"
		password := "password"
		uri.User = url.UserPassword(username, password)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, targetURL, nil)
		prepareRequest(req, uri, headerName)

		require.Equal(t, "Basic dXNlcjpwYXNzd29yZA==", req.Header.Get("Authorization"))
	})
}

// Failures used to be invisible: the invalid-header path logged nothing at all, and
// upstream errors went through httputil's default handler to the standard logger,
// arriving without a level. Both now log at warn (PR 5822 review).
//
// Deliberately not parallel: the hook attaches to the standard logger, which the
// parallel tests above also write to. Entries are matched by message so unrelated
// concurrent output cannot break the assertions.
func TestLogsFailuresAtWarn(t *testing.T) { //nolint:paralleltest
	hasWarn := func(t *testing.T, hook *test.Hook, msg string) {
		t.Helper()
		for _, e := range hook.AllEntries() {
			if e.Message == msg && e.Level == logrus.WarnLevel {
				assert.NotNil(t, e.Data[logrus.ErrorKey], "expected the cause to be attached")
				return
			}
		}
		t.Fatalf("no warn entry %s among %d entries", msg, len(hook.AllEntries()))
	}

	t.Run("unparsable filter header", func(t *testing.T) { //nolint:paralleltest
		hook := test.NewGlobal()
		uri, err := url.Parse(targetURL)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, requestURL, nil)
		req.Header.Set(headerName, "not-base64")

		getHandler(Config{HeaderName: headerName, TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })
		require.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)
		hasWarn(t, hook, "Rejecting request with unparsable filter header")
	})

	t.Run("upstream unreachable", func(t *testing.T) { //nolint:paralleltest
		// A closed server gives an address that is guaranteed to refuse connections.
		dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		deadURL := dead.URL
		dead.Close()

		hook := test.NewGlobal()
		uri, err := url.Parse(deadURL)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, deadURL+"/api/v1/query", nil)

		getHandler(Config{HeaderName: headerName, TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })
		require.Equal(t, http.StatusBadGateway, resp.StatusCode)
		hasWarn(t, hook, "Failed to proxy request")
	})

	t.Run("client cancellation is not a warning", func(t *testing.T) { //nolint:paralleltest
		// context.Canceled reaches ErrorHandler through the RoundTrip path. Nothing
		// failed upstream, so it must not warn -- Grafana cancelling superseded
		// queries would otherwise be a steady stream of them.
		dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		deadURL := dead.URL
		dead.Close()

		hook := test.NewGlobal()
		uri, err := url.Parse(deadURL)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, deadURL+"/api/v1/query", nil)
		getHandler(Config{HeaderName: headerName, TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })
		require.Equal(t, http.StatusBadGateway, resp.StatusCode)
		for _, e := range hook.AllEntries() {
			assert.NotEqual(t, logrus.WarnLevel, e.Level, "cancellation must not warn, got: %s", e.Message)
		}
	})

	t.Run("deadline exceeded is a gateway timeout", func(t *testing.T) { //nolint:paralleltest
		hook := test.NewGlobal()
		uri, err := url.Parse("http://192.0.2.1:9")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
		defer cancel()

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://192.0.2.1:9/api/v1/query", nil)
		getHandler(Config{HeaderName: headerName, TargetURL: uri}).ServeHTTP(rec, req)

		resp := rec.Result()
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })
		require.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
		hasWarn(t, hook, "Timed out proxying request")
	})
}
