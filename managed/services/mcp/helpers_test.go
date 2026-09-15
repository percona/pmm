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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// testToken is the caller credential every fixture test forwards.
const testToken = "Bearer glsa_test_token"

// fixture is one canned PMM API response, keyed by "METHOD /path".
type fixture struct {
	file   string
	status int
}

// recordedRequest is what the fake PMM received for one call.
type recordedRequest struct {
	method string
	path   string
	query  string
	header http.Header
	body   []byte
}

// fakePMM replays testdata fixtures and records every request so tests can
// assert the forwarded credentials and request bodies.
type fakePMM struct {
	t      *testing.T
	server *httptest.Server

	mu       sync.Mutex
	routes   map[string]fixture
	requests []recordedRequest
}

func newFakePMM(t *testing.T, routes map[string]fixture) *fakePMM {
	t.Helper()

	f := &fakePMM{t: t, routes: routes}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakePMM) serve(rw http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)

	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{
		method: req.Method,
		path:   req.URL.Path,
		query:  req.URL.RawQuery,
		header: req.Header.Clone(),
		body:   body,
	})
	fx, ok := f.routes[req.Method+" "+req.URL.Path]
	f.mu.Unlock()

	if !ok {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(`{"code":5,"error":"no fixture for ` + req.Method + " " + req.URL.Path + `","message":"not found"}`))
		return
	}

	status := fx.status
	if status == 0 {
		status = http.StatusOK
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	_, _ = rw.Write(readFixture(f.t, fx.file))
}

// requestsTo returns the recorded requests for a path, in order.
func (f *fakePMM) requestsTo(path string) []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []recordedRequest
	for _, r := range f.requests {
		if r.path == path {
			out = append(out, r)
		}
	}
	return out
}

// assertForwardedAuth checks that every recorded request carried the caller's token.
func (f *fakePMM) assertForwardedAuth(t *testing.T) {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotEmpty(t, f.requests)
	for _, r := range f.requests {
		require.Equal(t, testToken, r.header.Get("Authorization"), "%s %s", r.method, r.path)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}

func unmarshalBody[T any](t *testing.T, r recordedRequest) T {
	t.Helper()

	var v T
	require.NoError(t, json.Unmarshal(r.body, &v), "body: %s", r.body)
	return v
}

// newTestService builds a Service whose API client points at the fake PMM.
func newTestService(t *testing.T, fake *fakePMM) *Service {
	t.Helper()

	c, err := newClient(fake.server.URL+"/", logrus.WithField("test", t.Name()))
	require.NoError(t, err)

	s, err := New(Params{API: c})
	require.NoError(t, err)
	return s
}

// connect starts an httptest server around the MCP handler and opens a client
// session that sends the test token with every request.
func connect(t *testing.T, s *Service) *mcp.ClientSession {
	t.Helper()

	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   srv.URL + "/mcp",
		HTTPClient: &http.Client{Transport: &headerTransport{next: http.DefaultTransport}},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

// headerTransport adds the test token to every MCP request.
type headerTransport struct {
	next http.RoundTripper
}

func (h *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", testToken)
	return h.next.RoundTrip(req)
}

// callText calls a tool through the session and returns its text and error flag.
func callText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "unexpected content %T", res.Content[0])
	return text.Text, res.IsError
}
