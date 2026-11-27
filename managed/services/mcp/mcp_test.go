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

package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeAndListTools(t *testing.T) {
	t.Parallel()

	s, err := New(Params{Enabled: func() bool { return true }})
	require.NoError(t, err)
	session := connect(t, s)

	init := session.InitializeResult()
	require.NotNil(t, init)
	assert.Equal(t, serverName, init.ServerInfo.Name)
	assert.Contains(t, init.Instructions, "pmm_inventory")
	assert.Contains(t, init.Instructions, "sent metrics in the last hour")
	assert.Contains(t, init.Instructions, "pmm_catalog")
	assert.Contains(t, init.Instructions, "offset")
	assert.Contains(t, init.Instructions, "Untrusted")
	assert.Contains(t, init.Instructions, "query_ref")
	assert.Contains(t, init.Instructions, "client host")
	assert.Contains(t, init.Instructions, "pmm_alerts")
	assert.Contains(t, init.Instructions, "alert names, labels, summaries and descriptions")
	assert.Contains(t, init.Instructions, "environment")
	assert.Contains(t, init.Instructions, "PMM's remediation in the message")
	assert.Contains(t, init.Instructions, "pmm_service_health")
	assert.Contains(t, init.Instructions, "MetricsQL")
	assert.Contains(t, init.Instructions, "pmm_catalog(kind: metrics, engine:", "one engine's catalogue, not all of it")
	assert.Less(t, strings.Count(init.Instructions, "\n"), 36)

	res, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		assert.True(t, strings.HasPrefix(tool.Name, "pmm_"), "tool %s", tool.Name)
		require.NotNil(t, tool.Annotations, "tool %s", tool.Name)
		assert.True(t, tool.Annotations.ReadOnlyHint, "tool %s", tool.Name)
		require.NotNil(t, tool.Annotations.DestructiveHint, "tool %s", tool.Name)
		assert.False(t, *tool.Annotations.DestructiveHint, "tool %s", tool.Name)
	}
	assert.Equal(t, []string{"pmm_alerts", "pmm_catalog", "pmm_get_config", "pmm_get_explain", "pmm_get_schema", "pmm_inventory", "pmm_list_metrics", "pmm_query_detail", "pmm_query_metrics", "pmm_service_health", "pmm_top_queries", "pmm_version"}, names)
}

func TestDisabled(t *testing.T) {
	t.Parallel()

	s, err := New(Params{Enabled: func() bool { return false }})
	require.NoError(t, err)
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/mcp", strings.NewReader("{}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// postMCP sends body to the MCP handler at srv as protocol version 2025-03-26, the last one with JSON-RPC batches,
// and returns the status and the body of the answer.
func postMCP(t *testing.T, srv *httptest.Server, body string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/mcp", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")
	req.Header.Set("Authorization", testToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(b)
}

func TestBatchRefused(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, map[string]fixture{"GET /v1/server/version": {file: "version.json"}})
	srv := httptest.NewServer(newTestService(t, fake).Handler())
	t.Cleanup(srv.Close)

	call := func(id int) string {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"pmm_version","arguments":{}}}`, id)
	}

	status, body := postMCP(t, srv, " \r\n\t["+call(1)+","+call(2)+"]")
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "JSON-RPC batches are not supported\n", body)
	assert.Zero(t, fake.requestCount(), "no tool of a batch may run")

	status, body = postMCP(t, srv, call(3))
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "PMM Server 3.8.1")
	assert.Len(t, fake.requestsTo("/v1/server/version"), 1)

	status, _ = postMCP(t, srv, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"pmm_version","arguments":{"pad":"`+
		strings.Repeat("x", maxRequestBodyBytes)+`"}}}`)
	assert.Equal(t, http.StatusRequestEntityTooLarge, status)
	assert.Len(t, fake.requestsTo("/v1/server/version"), 1)
}

func TestBatchOriginCheckedFirst(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, map[string]fixture{"GET /v1/server/version": {file: "version.json"}})
	s := newTestService(t, fake)
	call := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pmm_version","arguments":{}}}`

	for _, tc := range []struct {
		name     string
		headers  map[string]string
		want     int
		wantRead bool
	}{
		{"CrossSite", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}, http.StatusForbidden, false},
		{"SameOrigin", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusBadRequest, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var read atomic.Int64
			body := countingBody{r: strings.NewReader("[" + call + "]"), n: &read}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", body)
			req.RemoteAddr = "127.0.0.1:40000"
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("Mcp-Protocol-Version", "2025-03-26")
			req.Header.Set("Authorization", testToken)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()

			s.Handler().ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code)
			assert.Equal(t, tc.wantRead, read.Load() > 0, "body read")
			assert.Zero(t, fake.requestCount(), "no tool of a batch may run")
		})
	}
}

// droppableClient is an MCP client session whose test can close the session's connections, as a client that goes away does.
type droppableClient struct {
	session *mcp.ClientSession

	mu    sync.Mutex
	conns []net.Conn
	// callVersions are the Mcp-Protocol-Version headers of the tools/call requests the server received.
	callVersions []string
}

func (d *droppableClient) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	c, err := (&net.Dialer{}).DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	d.conns = append(d.conns, c)
	d.mu.Unlock()
	return c, nil
}

// drop closes every connection of the client.
func (d *droppableClient) drop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.conns {
		_ = c.Close()
	}
}

// connectDroppable is connect with a client that asks for protocol version and whose connections the test can drop.
func connectDroppable(t *testing.T, s *Service, version string) *droppableClient {
	t.Helper()

	d := &droppableClient{}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.Contains(string(body), `"method":"tools/call"`) {
			d.mu.Lock()
			d.callVersions = append(d.callVersions, req.Header.Get("Mcp-Protocol-Version"))
			d.mu.Unlock()
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		s.Handler().ServeHTTP(rw, req)
	}))
	t.Cleanup(srv.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   srv.URL + "/mcp",
		HTTPClient: &http.Client{Transport: &headerTransport{next: &http.Transport{DialContext: d.dial}}},
		MaxRetries: -1,
	}, &mcp.ClientSessionOptions{ProtocolVersion: version})
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	require.Equal(t, version, session.InitializeResult().ProtocolVersion, "the session negotiated another version")
	d.session = session
	return d
}

// TestDisconnectCancelsCall pins that a client that closes its connection cancels its call, and the call's unfinished
// actions, on every protocol version, so that the call's slots are free at once.
func TestDisconnectCancelsCall(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"2025-03-26", "2025-06-18", "2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			// call starts a tool call, waits until started reports it in flight, drops the client's connections and
			// returns once the call's client gave up.
			call := func(t *testing.T, s *Service, tool string, args map[string]any, started func() bool) {
				t.Helper()

				d := connectDroppable(t, s, version)
				ctx, cancel := context.WithCancel(t.Context())
				done := make(chan struct{})
				go func() {
					defer close(done)
					_, _ = d.session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
				}()
				t.Cleanup(func() {
					cancel()
					<-done
				})

				require.Eventually(t, started, 5*time.Second, 10*time.Millisecond, "the call did not start")
				d.mu.Lock()
				assert.Equal(t, []string{version}, d.callVersions)
				d.mu.Unlock()
				d.drop()
			}

			t.Run("BackendNeverAnswers", func(t *testing.T) {
				t.Parallel()

				fake := newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{"POST " + reportPath: {block: true}}))
				s := newQANService(t, fake, true)
				call(t, s, "pmm_top_queries", map[string]any{}, func() bool { return len(fake.requestsTo(reportPath)) > 0 })

				assert.Eventually(t, func() bool {
					s.metrics.mu.Lock()
					defer s.metrics.mu.Unlock()
					return s.metrics.total == 0
				}, time.Second, 10*time.Millisecond, "the call kept its metric slot")
			})

			t.Run("LiveAction", func(t *testing.T) {
				t.Parallel()

				fake, stub := newActionStub(t, 0, "ddl-customers")
				s := newQANService(t, fake, true)
				args := map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}}
				call(t, s, "pmm_get_schema", args, func() bool {
					stub.mu.Lock()
					defer stub.mu.Unlock()
					return stub.polls > 0
				})

				assert.Eventually(t, func() bool {
					s.live.mu.Lock()
					defer s.live.mu.Unlock()
					s.liveByService.mu.Lock()
					defer s.liveByService.mu.Unlock()
					return s.live.total == 0 && s.liveByService.total == 0
				}, time.Second, 10*time.Millisecond, "the call kept its live-inspection slots")
				assert.Equal(t, []string{"ddl-customers"}, stub.cancelled(), "the running action was not cancelled")
			})
		})
	}
}
