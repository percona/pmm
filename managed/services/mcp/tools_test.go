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
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredAndErrorResults(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, map[string]fixture{"GET /v1/server/version": {file: "version.json"}})
		session := connect(t, newTestService(t, fake))
		res := callTool(t, session, "pmm_version", nil)
		require.False(t, res.IsError)
		var v versionResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &v))
		require.Equal(t, versionResult{Version: "3.8.1", ManagedVersion: "3.8.1-269-gfad5f58"}, v)
		require.Equal(t, "PMM Server 3.8.1 (pmm-managed 3.8.1-269-gfad5f58)", res.Content[0].(*mcp.TextContent).Text)
		require.NotNil(t, listTools(t, session)["pmm_version"].OutputSchema)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		// No route for /v1/server/version.
		fake := newFakePMM(t, nil)
		session := connect(t, newTestService(t, fake))
		res := callTool(t, session, "pmm_version", nil)
		require.True(t, res.IsError)
		require.Nil(t, res.StructuredContent)
		require.Equal(t, map[string]any{"code": "not_found", "retryable": false}, res.Meta["com.percona/error"])
		require.True(t, strings.HasPrefix(res.Content[0].(*mcp.TextContent).Text, "error: not_found\n"))
	})
}

func TestRetryable(t *testing.T) {
	t.Parallel()

	all := []errorCode{
		codeNotFound, codeInvalidInput, codeUnauthorized, codeTimeout, codeNoQuerySource,
		codeInsufficientPrivileges, codeAgentUnreachable, codePMMUnavailable, codeRateLimited,
	}
	want := map[errorCode]bool{codeTimeout: true, codeAgentUnreachable: true, codePMMUnavailable: true, codeRateLimited: true}
	for _, code := range all {
		assert.Equal(t, want[code], (&toolError{code: code}).retryable(), string(code))
	}
}
