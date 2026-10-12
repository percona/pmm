// Copyright (C) 2023 Percona LLC
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

package base

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryTransport(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		unavailable   int32
		contentType   string
		status        int
		expectedCode  int
		expectedCalls int32
	}{
		"recovers": {
			unavailable:   1,
			contentType:   "application/json",
			status:        http.StatusOK,
			expectedCode:  http.StatusOK,
			expectedCalls: 2,
		},
		"gives up": {
			unavailable:   1000,
			contentType:   "application/json",
			status:        http.StatusOK,
			expectedCode:  http.StatusServiceUnavailable,
			expectedCalls: 2,
		},
		"maintenance page": {
			unavailable:   1000,
			contentType:   "text/html",
			status:        http.StatusOK,
			expectedCode:  http.StatusServiceUnavailable,
			expectedCalls: 1,
		},
		"other error": {
			status:        http.StatusInternalServerError,
			expectedCode:  http.StatusInternalServerError,
			expectedCalls: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Equal(t, "request body", string(body))

				if calls.Add(1) <= tc.unavailable {
					rw.Header().Set("Content-Type", tc.contentType)
					rw.WriteHeader(http.StatusServiceUnavailable)
					_, _ = rw.Write([]byte("unavailable"))
					return
				}
				rw.WriteHeader(tc.status)
			}))
			t.Cleanup(srv.Close)

			// The budget fits one delay but not two, so "gives up" makes exactly two calls.
			rt := &retryTransport{
				next:     http.DefaultTransport,
				delay:    50 * time.Millisecond,
				duration: 75 * time.Millisecond,
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, strings.NewReader("request body"))
			require.NoError(t, err)

			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			assert.Equal(t, tc.expectedCalls, calls.Load())

			if tc.expectedCode == http.StatusServiceUnavailable {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, "unavailable", string(body))
			}
		})
	}
}
