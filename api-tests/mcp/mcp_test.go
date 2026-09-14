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

// Package mcp contains integration tests for PMM's MCP endpoint (/mcp).
package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/utils/tlsconfig"
)

// basicAuthTransport adds the credentials from the test server URL to every request.
type basicAuthTransport struct {
	next     http.RoundTripper
	user     string
	password string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.SetBasicAuth(t.user, t.password)
	return t.next.RoundTrip(req)
}

// httpClient returns an HTTP client for the PMM Server under test; withAuth
// adds the admin credentials from the server URL.
func httpClient(withAuth bool) *http.Client {
	transport := &http.Transport{}
	if pmmapitests.BaseURL.Scheme == "https" {
		transport.TLSClientConfig = tlsconfig.Get()
		transport.TLSClientConfig.ServerName = pmmapitests.BaseURL.Hostname()
		transport.TLSClientConfig.InsecureSkipVerify = pmmapitests.ServerInsecureTLS
	}
	var rt http.RoundTripper = transport
	if withAuth && pmmapitests.BaseURL.User != nil {
		password, _ := pmmapitests.BaseURL.User.Password()
		rt = &basicAuthTransport{next: transport, user: pmmapitests.BaseURL.User.Username(), password: password}
	}
	return &http.Client{Transport: rt}
}

func endpoint() string {
	return pmmapitests.BaseURL.ResolveReference(&url.URL{Path: "mcp"}).String()
}

// connect opens an authenticated MCP session against the server under test.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "pmm-api-tests", Version: "1"}, nil)
	session, err := client.Connect(pmmapitests.Context, &mcp.StreamableClientTransport{
		Endpoint:   endpoint(),
		HTTPClient: httpClient(true),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// callText calls a tool and returns its text content and error flag.
func callText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()

	res, err := session.CallTool(pmmapitests.Context, &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "unexpected content %T", res.Content[0])
	t.Logf("%s -> isError=%v\n%s", name, res.IsError, text.Text)
	return text.Text, res.IsError
}

func TestUnauthenticated(t *testing.T) {
	t.Parallel()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"pmm-api-tests","version":"1"}}}`
	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, endpoint(), strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := httpClient(false).Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Logf("Response: %s", b)

	// nginx auth_request answers 401 with pmm-managed's JSON body; the MCP
	// handler itself never emits 401.
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.EqualValues(t, 16, m["code"])
	assert.Equal(t, "Unauthorized", m["message"])
}

func TestListTools(t *testing.T) {
	t.Parallel()

	session := connect(t)
	assert.Equal(t, "pmm-mcp", session.InitializeResult().ServerInfo.Name)

	res, err := session.ListTools(pmmapitests.Context, nil)
	require.NoError(t, err)

	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		require.NotNil(t, tool.Annotations, tool.Name)
		assert.True(t, tool.Annotations.ReadOnlyHint, tool.Name)
	}
	for _, want := range []string{"pmm_version", "pmm_inventory"} {
		assert.Contains(t, names, want)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	text, isError := callText(t, connect(t), "pmm_version", nil)
	assert.False(t, isError)
	assert.True(t, strings.HasPrefix(text, "PMM Server 3."), text)
}

func TestInventory(t *testing.T) {
	t.Parallel()

	res, err := inventoryClient.Default.ServicesService.ListServices(&services_service.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	text, isError := callText(t, connect(t), "pmm_inventory", map[string]any{})
	assert.False(t, isError)

	for _, svc := range res.Payload.Mysql {
		assert.Contains(t, text, "- "+svc.ServiceName+" | mysql ")
		assert.Contains(t, text, "id="+svc.ServiceID)
	}
	for _, svc := range res.Payload.Postgresql {
		assert.Contains(t, text, "- "+svc.ServiceName+" | postgresql ")
		assert.Contains(t, text, "id="+svc.ServiceID)
	}
	for _, svc := range res.Payload.Mongodb {
		assert.Contains(t, text, "- "+svc.ServiceName+" | mongodb ")
	}

	t.Run("EngineFilter", func(t *testing.T) {
		t.Parallel()

		text, isError := callText(t, connect(t), "pmm_inventory", map[string]any{"engine": "mysql"})
		assert.False(t, isError)
		assert.NotContains(t, text, "| postgresql ")
	})

	t.Run("InvalidEngine", func(t *testing.T) {
		t.Parallel()

		text, isError := callText(t, connect(t), "pmm_inventory", map[string]any{"engine": "oracle"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input"), text)
	})
}
