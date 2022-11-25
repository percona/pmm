// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	headerName = "x-test-header"
	targetURL  = "http://127.0.0.1"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, filters []string) http.HandlerFunc {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if filters != nil {
				require.Equal(t, r.URL.RawQuery, url.Values{"extra_filters": filters}.Encode())
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

		handler.ServeHTTP(rec, req)
		require.Equal(t, rec.Result().StatusCode, http.StatusOK)
	})

	t.Run("shall properly handle filters", func(t *testing.T) {
		t.Parallel()

		type testParams struct {
			expectedFilters []string
			expectedStatus  int
			headerContent   string
			name            string
		}

		testCases := []testParams{
			{
				name:            "shall process filters properly",
				expectedFilters: []string{"abc", "def"},
				expectedStatus:  http.StatusOK,
				headerContent:   base64.StdEncoding.EncodeToString([]byte(`["abc", "def"]`)),
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
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				handler := setup(t, tc.expectedFilters)

				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, targetURL, nil)
				req.Header.Set(headerName, tc.headerContent)

				handler.ServeHTTP(rec, req)
				require.Equal(t, rec.Result().StatusCode, tc.expectedStatus)
			})
		}
	})
}
