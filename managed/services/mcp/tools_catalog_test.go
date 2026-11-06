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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogQANColumns(t *testing.T) {
	t.Parallel()

	session := connect(t, newTestService(t, newFakePMM(t, nil)))

	t.Run("All", func(t *testing.T) {
		t.Parallel()

		res := callTool(t, session, "pmm_catalog", map[string]any{"kind": "qan_columns"})
		require.False(t, res.IsError)
		var out catalogResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &out))
		assert.Equal(t, "qan_columns", out.Kind)
		assert.Empty(t, out.Engine)
		require.Len(t, out.Entries, len(qanColumns))
		for i, c := range qanColumns {
			assert.Equal(t, c.name, out.Entries[i].Name)
			assert.Equal(t, c.engine, out.Entries[i].Engine)
			want := "sum"
			if c.avg {
				want = "avg"
			}
			assert.Equal(t, want, out.Entries[i].Aggregation, c.name)
		}

		text := textOf(t, res)
		assert.Contains(t, text, "docs_examined")
		assert.Contains(t, text, "shared_blks_hit")
	})

	t.Run("Engine", func(t *testing.T) {
		t.Parallel()

		res := callTool(t, session, "pmm_catalog", map[string]any{"kind": "qan_columns", "engine": "mongodb"})
		require.False(t, res.IsError)
		var out catalogResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &out))
		assert.Equal(t, "mongodb", out.Engine)
		var want []string
		for _, c := range qanColumns {
			if c.engine == "" || c.engine == engineMongoDB {
				want = append(want, c.name)
			}
		}
		got := make([]string, 0, len(out.Entries))
		for _, e := range out.Entries {
			assert.Contains(t, []string{"", engineMongoDB}, e.Engine)
			got = append(got, e.Name)
		}
		assert.Equal(t, want, got)

		text := textOf(t, res)
		assert.Contains(t, text, "docs_examined")
		assert.NotContains(t, text, "shared_blks_hit")
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			args   map[string]any
			prefix string
		}{
			{map[string]any{"kind": "bogus"}, "error: invalid_input\nkind must be one of qan_columns; got 'bogus'"},
			{
				map[string]any{"kind": "qan_columns", "engine": "oracle"},
				"error: invalid_input\nengine must be one of mysql, postgresql, mongodb; got 'oracle'",
			},
		} {
			text, isError := callText(t, session, "pmm_catalog", tc.args)
			assert.True(t, isError, "%v", tc.args)
			assert.True(t, strings.HasPrefix(text, tc.prefix), "%v: %s", tc.args, text)
		}
	})
}
