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

			srv, calls := newRetryTestServer(t, tc.unavailable, tc.contentType, tc.status)
			resp := roundTripWithRetry(t, srv, time.Minute)
			t.Cleanup(func() { _ = resp.Body.Close() })

			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			assert.Equal(t, tc.expectedCalls, calls.Load())
		})
	}
}

func TestRetryTransportGivesUp(t *testing.T) {
	t.Parallel()

	srv, calls := newRetryTestServer(t, 1000, "application/json", http.StatusOK)
	resp := roundTripWithRetry(t, srv, 100*time.Millisecond)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Greater(t, calls.Load(), int32(1))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "unavailable", string(body))
}

// newRetryTestServer returns a server answering its first unavailable requests with a 503 of
// contentType, and the rest with status, and the number of requests it received.
func newRetryTestServer(t *testing.T, unavailable int32, contentType string, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, "request body", string(body))

		if calls.Add(1) <= unavailable {
			rw.Header().Set("Content-Type", contentType)
			rw.WriteHeader(http.StatusServiceUnavailable)
			_, _ = rw.Write([]byte("unavailable"))
			return
		}
		rw.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	return srv, &calls
}

// roundTripWithRetry sends a POST to srv through a retryTransport retrying for duration.
func roundTripWithRetry(t *testing.T, srv *httptest.Server, duration time.Duration) *http.Response {
	t.Helper()

	rt := &retryTransport{
		// Not http.DefaultTransport: a parallel test's srv.Close drops its idle connections.
		next:     srv.Client().Transport,
		delay:    10 * time.Millisecond,
		duration: duration,
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, strings.NewReader("request body"))
	require.NoError(t, err)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)

	return resp
}
