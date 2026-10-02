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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	"github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func newTestServer(t *testing.T, failures int32, failStatus int) (*httptest.Server, *atomic.Int32, *[]string) {
	t.Helper()

	var calls atomic.Int32
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		if calls.Add(1) <= failures {
			rw.WriteHeader(failStatus)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls, &bodies
}

func post(t *testing.T, client *http.Client, url string, body io.Reader) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, body)
	require.NoError(t, err)
	return client.Do(req)
}

func TestRetryTransport(t *testing.T) {
	t.Parallel()

	rt := &retryTransport{
		next:       http.DefaultTransport,
		maxRetries: 3,
		baseDelay:  time.Millisecond,
		maxDelay:   time.Millisecond,
	}
	client := &http.Client{Transport: rt}

	t.Run("retries 503 and replays the body", func(t *testing.T) {
		t.Parallel()

		srv, calls, bodies := newTestServer(t, 2, http.StatusServiceUnavailable)
		resp, err := post(t, client, srv.URL, strings.NewReader(`{"service_name":"pg"}`))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int32(3), calls.Load())
		assert.Equal(t, []string{`{"service_name":"pg"}`, `{"service_name":"pg"}`, `{"service_name":"pg"}`}, *bodies)
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		t.Parallel()

		srv, calls, _ := newTestServer(t, 100, http.StatusServiceUnavailable)
		resp, err := post(t, client, srv.URL, strings.NewReader(`{}`))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, int32(4), calls.Load())
	})

	t.Run("does not retry other errors", func(t *testing.T) {
		t.Parallel()

		srv, calls, _ := newTestServer(t, 1, http.StatusInternalServerError)
		resp, err := post(t, client, srv.URL, strings.NewReader(`{}`))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("stops on context cancel", func(t *testing.T) {
		t.Parallel()

		slow := &retryTransport{next: http.DefaultTransport, maxRetries: 3, baseDelay: time.Hour, maxDelay: time.Hour}
		srv, calls, _ := newTestServer(t, 100, http.StatusServiceUnavailable)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, strings.NewReader(`{}`))
		require.NoError(t, err)
		_, err = (&http.Client{Transport: slow}).Do(req) //nolint:bodyclose
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, int32(1), calls.Load())
	})
}

func TestSetupClientsRetriesUnavailable(t *testing.T) {
	srv, calls, bodies := newTestServer(t, 1, http.StatusServiceUnavailable)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	SetupClients(&flags.GlobalFlags{ServerURL: u})
	_, err = managementClient.Default.ManagementService.AddService(&management_service.AddServiceParams{
		Body: management_service.AddServiceBody{
			External: &management_service.AddServiceParamsBodyExternal{ServiceName: "pg-patroni-external"},
		},
		Context: t.Context(),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	require.Len(t, *bodies, 2)
	assert.Contains(t, (*bodies)[0], `"service_name":"pg-patroni-external"`)
	assert.Equal(t, (*bodies)[0], (*bodies)[1])
}
