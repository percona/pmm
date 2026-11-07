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
	"slices"
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
		for _, want := range []catalogEntry{
			{Name: "num_queries", Aggregation: "sum"},
			{Name: "query_time", Aggregation: "avg"},
			{Name: "no_index_used", Engine: engineMySQL, Aggregation: "sum"},
			{Name: "docs_examined", Engine: engineMongoDB, Aggregation: "avg"},
		} {
			i := slices.IndexFunc(out.Entries, func(e catalogEntry) bool { return e.Name == want.Name })
			require.GreaterOrEqual(t, i, 0, want.Name)
			assert.Equal(t, want, out.Entries[i])
		}

		text := textOf(t, res)
		assert.Contains(t, text, "docs_examined")
		assert.Contains(t, text, "shared_blks_hit")
		for _, line := range []string{"- num_queries sum", "- query_time avg", "- no_index_used [mysql] sum", "- docs_examined [mongodb] avg"} {
			assert.Contains(t, text, "\n"+line+"\n")
		}
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

	t.Run("MetricsEngine", func(t *testing.T) {
		t.Parallel()

		res := callTool(t, session, "pmm_catalog", map[string]any{"kind": "metrics", "engine": "postgresql"})
		require.False(t, res.IsError)
		var out catalogResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &out))
		assert.Equal(t, "metrics", out.Kind)
		assert.Equal(t, "postgresql", out.Engine)
		var want []string
		for _, m := range metricsCatalog {
			if m.Engine == enginePostgreSQL {
				want = append(want, m.Name)
			}
		}
		got := make([]string, 0, len(out.Entries))
		for _, e := range out.Entries {
			assert.Equal(t, enginePostgreSQL, e.Engine, e.Name)
			got = append(got, e.Name)
		}
		assert.Equal(t, want, got)

		i := slices.IndexFunc(out.Entries, func(e catalogEntry) bool { return e.Name == "pg_stat_database_xact_commit" })
		require.GreaterOrEqual(t, i, 0)
		e := out.Entries[i]
		assert.Equal(t, "counter", e.Type)
		assert.Equal(t, "ops", e.Unit)
		assert.Contains(t, e.Labels, "datname")
		assert.NotEmpty(t, e.Meaning)
		assert.Equal(t, `sum by (datname) (rate(pg_stat_database_xact_commit{service_name="<service>"}[5m]))`, e.Query)

		text := textOf(t, res)
		assert.Contains(t, text, "pg_stat_database_xact_commit [postgresql] counter ops: ")
		assert.Contains(t, text, "; query: sum by (datname) (rate(pg_stat_database_xact_commit{")
		assert.NotContains(t, text, "mysql_global_status")
	})

	t.Run("MetricsAllEngines", func(t *testing.T) {
		t.Parallel()

		res := callTool(t, session, "pmm_catalog", map[string]any{"kind": "metrics"})
		require.False(t, res.IsError)
		var out catalogResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &out))
		assert.Empty(t, out.Engine)
		require.Len(t, out.Entries, len(metricsCatalog))
		engines := map[string]int{}
		for i, e := range out.Entries {
			assert.Equal(t, metricsCatalog[i].Name, e.Name)
			engines[e.Engine]++
		}
		assert.Len(t, engines, 4)
		for _, engine := range []string{"mysql", "postgresql", "mongodb", "node"} {
			assert.NotZero(t, engines[engine], engine)
		}
	})

	t.Run("MetricsNode", func(t *testing.T) {
		t.Parallel()

		res := callTool(t, session, "pmm_catalog", map[string]any{"kind": "metrics", "engine": "node"})
		require.False(t, res.IsError)
		var out catalogResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &out))
		require.NotEmpty(t, out.Entries)
		for _, e := range out.Entries {
			assert.Equal(t, engineNode, e.Engine, e.Name)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			args   map[string]any
			prefix string
		}{
			{map[string]any{"kind": "bogus"}, "error: invalid_input\nkind must be one of qan_columns, metrics; got 'bogus'"},
			{
				map[string]any{"kind": "qan_columns", "engine": "oracle"},
				"error: invalid_input\nengine must be one of mysql, postgresql, mongodb; got 'oracle'",
			},
			{
				map[string]any{"kind": "qan_columns", "engine": "node"},
				"error: invalid_input\nengine must be one of mysql, postgresql, mongodb; got 'node'",
			},
			{
				map[string]any{"kind": "metrics", "engine": "oracle"},
				"error: invalid_input\nengine must be one of mysql, postgresql, mongodb, node; got 'oracle'",
			},
		} {
			text, isError := callText(t, session, "pmm_catalog", tc.args)
			assert.True(t, isError, "%v", tc.args)
			assert.True(t, strings.HasPrefix(text, tc.prefix), "%v: %s", tc.args, text)
		}
	})
}
