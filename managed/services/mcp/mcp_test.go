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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connect starts an httptest server around the handler and opens an MCP client
// session against it.
func connect(t *testing.T, s *Service) *mcp.ClientSession {
	t.Helper()

	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp"}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func TestInitializeAndListTools(t *testing.T) {
	t.Parallel()

	session := connect(t, New(Params{}))

	init := session.InitializeResult()
	require.NotNil(t, init)
	assert.Equal(t, serverName, init.ServerInfo.Name)
	assert.Contains(t, init.Instructions, "pmm_inventory")

	res, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	for _, tool := range res.Tools {
		assert.True(t, strings.HasPrefix(tool.Name, "pmm_"), "tool %s", tool.Name)
		require.NotNil(t, tool.Annotations, "tool %s", tool.Name)
		assert.True(t, tool.Annotations.ReadOnlyHint, "tool %s", tool.Name)
		require.NotNil(t, tool.Annotations.DestructiveHint, "tool %s", tool.Name)
		assert.False(t, *tool.Annotations.DestructiveHint, "tool %s", tool.Name)
	}
}

func TestDisabled(t *testing.T) {
	t.Parallel()

	s := New(Params{Enabled: func() bool { return false }})
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
