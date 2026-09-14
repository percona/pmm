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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeAndListTools(t *testing.T) {
	t.Parallel()

	s, err := New(Params{})
	require.NoError(t, err)
	session := connect(t, s)

	init := session.InitializeResult()
	require.NotNil(t, init)
	assert.Equal(t, serverName, init.ServerInfo.Name)
	assert.Contains(t, init.Instructions, "pmm_inventory")

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
	assert.Equal(t, []string{"pmm_inventory", "pmm_version"}, names)
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
