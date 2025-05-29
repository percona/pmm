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
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	headerName = "X-Test-Header"
	targetURL  = "http://127.0.0.1"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, filters []string) http.HandlerFunc {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if filters != nil {
				assert.Equal(t, url.Values{"extra_filters[]": filters}.Encode(), r.URL.RawQuery)
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

	handler := setup(t, nil)

	t.Run("shall proxy request", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, targetURL, nil)
		url, err := url.Parse(targetURL)
		require.NoError(t, err)

		prepareRequest(req, url, headerName)

		handler.ServeHTTP(rec, req)
		resp := rec.Result()
		defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("shall properly handle filters", func(t *testing.T) {
		t.Parallel()

		type testParams struct {
			expectedFilters []string
			expectedStatus  int
			headerContent   string
			name            string
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
				targetURL:       "http://127.0.0.1/a?extra_filters[]=a&extra_filters[]=b",
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
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				testTargetURL := targetURL
				if tc.targetURL != "" {
					testTargetURL = tc.targetURL
				}

				handler := setup(t, tc.expectedFilters)

				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, testTargetURL, nil)

				url, err := url.Parse(testTargetURL)
				require.NoError(t, err)
				prepareRequest(req, url, headerName)
				req.Header.Set(headerName, tc.headerContent)

				handler.ServeHTTP(rec, req)
				resp := rec.Result()
				defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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
				req := httptest.NewRequest(http.MethodGet, targetURL, nil)

				prepareRequest(req, url, headerName)

				require.NotNil(t, req.Header[headerName])
				require.Equal(t, expectedHost, req.Header[headerName][0])
			})
		}
	})
}
