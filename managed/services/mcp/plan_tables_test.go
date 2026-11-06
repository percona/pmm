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
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noteExpect is what resolving one Task 1 fixture must give.
type noteExpect struct {
	resolved   []string
	unresolved []string
}

// fixtureExpectations are written from Task 1's recorded findings, per case.
// Format 2 differs from format 1 only in the name of the union.
var fixtureExpectations = map[string]noteExpect{
	"single":              {resolved: []string{"shop.orders"}},
	"join-no-aliases":     {resolved: []string{"shop.orders", "shop.customers"}},
	"join-aliases":        {resolved: []string{"shop.orders", "shop.customers"}},
	"schema-qualified":    {resolved: []string{"shop.orders"}},
	"subquery-where":      {resolved: []string{"shop.orders", "shop.customers"}},
	"alias-shadows-table": {resolved: []string{"shop.a", "shop.b"}},
	"derived-table":       {resolved: []string{"shop.orders", "shop.customers"}, unresolved: []string{"d"}},
	"cte":                 {resolved: []string{"shop.orders", "shop.customers"}, unresolved: []string{"t"}},
	"cte-shadows-table":   {resolved: []string{"shop.orders", "shop.customers"}, unresolved: []string{"orders"}},
	"union":               {resolved: []string{"shop.orders", "shop.customers"}, unresolved: []string{"<union1,2>"}},
}

var fixtureName = regexp.MustCompile(`^(?:mysql|ps)-(5\.7|8\.0|8\.4-v2|8\.4)-(.+)\.json$`)

func refStrings(refs []tableRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.database + "." + r.name
	}
	return out
}

func TestPlanTablesFixtures(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/explain/*.json")
	require.NoError(t, err)
	require.Len(t, files, 76)

	for _, file := range files {
		m := fixtureName.FindStringSubmatch(filepath.Base(file))
		require.NotNil(t, m, file)
		version, name := m[1], m[2]
		want, ok := fixtureExpectations[name]
		require.True(t, ok, "no expectation for case %s", name)
		if version == "8.4-v2" && name == "union" {
			want.unresolved = []string{"<union temporary>"}
		}

		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()

			plan, err := os.ReadFile(file)
			require.NoError(t, err)
			tables, err := planTables(plan)
			require.NoError(t, err)
			resolved, unresolved := resolveTables(tables, aliasMap(plan), "shop")
			assert.ElementsMatch(t, want.resolved, refStrings(resolved))
			assert.ElementsMatch(t, want.unresolved, unresolved)
		})
	}
}

func TestPlanTablesCTEShadowsTableKeepsBothEntries(t *testing.T) {
	t.Parallel()

	plan, err := os.ReadFile("testdata/explain/mysql-8.0-cte-shadows-table.json")
	require.NoError(t, err)
	tables, err := planTables(plan)
	require.NoError(t, err)

	var orders []planTable
	for _, tb := range tables {
		if tb.Name == "orders" {
			orders = append(orders, tb)
		}
	}
	require.Len(t, orders, 2)
	assert.True(t, orders[0].Derived != orders[1].Derived)
}

func TestPlanTablesFormat2(t *testing.T) {
	t.Parallel()

	plan, err := os.ReadFile("testdata/explain/mysql-8.4-v2-join-aliases.json")
	require.NoError(t, err)
	tables, err := planTables(plan)
	require.NoError(t, err)
	assert.Equal(t, []planTable{
		{Schema: "shop", Name: "customers", Alias: "c"},
		{Schema: "shop", Name: "orders", Alias: "o"},
	}, tables)
}

func TestPlanTablesRejectsNonJSON(t *testing.T) {
	t.Parallel()

	_, err := planTables([]byte("x"))
	require.Error(t, err)
}

func TestPlanTablesSkipsAgentKeys(t *testing.T) {
	t.Parallel()

	bare := `{"query_block":{"select_id":1,"table":{"table_name":"o","access_type":"ALL"}}}`
	withAgent := `{"query_block":{"select_id":1,"table":{"table_name":"o","real_table_name":"fake","access_type":"ALL"}},` +
		`"real_table_name":{"table_name":"bogus"},` +
		`"warnings":[{"Level":"Note","Code":1003,"Message":"x","table":{"table_name":"bogus2"}}]}`

	want, err := planTables([]byte(bare))
	require.NoError(t, err)
	got, err := planTables([]byte(withAgent))
	require.NoError(t, err)
	assert.Equal(t, []planTable{{Name: "o"}}, want)
	assert.Equal(t, want, got)
}

func notePlan(t *testing.T, message string) []byte {
	t.Helper()

	plan, err := json.Marshal(map[string]any{
		"query_block": map[string]any{"select_id": 1},
		"warnings":    []map[string]any{{"Level": "Note", "Code": 1003, "Message": message}},
	})
	require.NoError(t, err)
	return plan
}

func TestAliasMap(t *testing.T) {
	t.Parallel()

	o := tableRef{database: "shop", name: "orders"}
	c := tableRef{database: "shop", name: "customers"}

	tests := []struct {
		name string
		plan []byte
		want map[string]tableRef
	}{
		{
			name: "backquoted with alias",
			plan: notePlan(t, "/* select#1 */ select `shop`.`o`.`id` AS `id` from `shop`.`orders` `o` join `shop`.`customers` `c` where (`shop`.`o`.`cid` = `shop`.`c`.`id`)"),
			want: map[string]tableRef{"o": o, "c": c},
		},
		{
			name: "backquoted without alias",
			plan: notePlan(t, "/* select#1 */ select `shop`.`orders`.`id` AS `id` from `shop`.`orders` join `shop`.`customers` where 1"),
			want: map[string]tableRef{"orders": o, "customers": c},
		},
		{
			name: "alias shadows another table name",
			plan: notePlan(t, "select 1 from `shop`.`a` `b` join `shop`.`b` `a`"),
			want: map[string]tableRef{
				"b": {database: "shop", name: "a"},
				"a": {database: "shop", name: "b"},
			},
		},
		{
			name: "semi join",
			plan: notePlan(t, "select 1 from `shop`.`customers` semi join (`shop`.`orders`) where 1"),
			want: map[string]tableRef{"customers": c, "orders": o},
		},
		{
			name: "derived table select",
			plan: notePlan(t, "select `d`.`x` from (/* select#2 */ select 1 from `shop`.`orders` group by 1) `d` join `shop`.`customers` `c`"),
			want: map[string]tableRef{"orders": o, "c": c},
		},
		{
			name: "cte name is never an entry",
			plan: notePlan(t, "with `t` as (/* select#2 */ select 1 from `shop`.`orders`) /* select#1 */ select 1 from `t` join `shop`.`customers` `c`"),
			want: map[string]tableRef{"orders": o, "c": c},
		},
		{
			name: "no note",
			plan: []byte(`{"query_block":{"select_id":1},"warnings":[]}`),
			want: map[string]tableRef{},
		},
		{
			name: "no warnings key",
			plan: []byte(`{"query_block":{"select_id":1}}`),
			want: map[string]tableRef{},
		},
		{
			name: "other warning code",
			plan: []byte(`{"warnings":[{"Level":"Warning","Code":1235,"Message":"from ` + "`shop`.`orders`" + `"}]}`),
			want: map[string]tableRef{},
		},
		{
			name: "string literal",
			plan: notePlan(t, "select 1 from `shop`.`orders` where `note` = 'from secret_t s join other_t'"),
			want: map[string]tableRef{"orders": o},
		},
		{
			name: "double dash comment",
			plan: notePlan(t, "select 1 from `shop`.`orders` -- from `shop`.`secret` `s`\n where 1"),
			want: map[string]tableRef{"orders": o},
		},
		{
			name: "hash comment",
			plan: notePlan(t, "select 1 from `shop`.`orders` # join `shop`.`secret` `s`\n where 1"),
			want: map[string]tableRef{"orders": o},
		},
		{
			name: "block comment",
			plan: notePlan(t, "select 1 from `shop`.`orders` /* join `shop`.`secret` `s` */ where 1"),
			want: map[string]tableRef{"orders": o},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, aliasMap(tc.plan))
		})
	}
}

func TestAliasMapNoNoteLeavesFormat1Unresolved(t *testing.T) {
	t.Parallel()

	plan := []byte(`{"query_block":{"select_id":1,"table":{"table_name":"o"}}}`)
	tables, err := planTables(plan)
	require.NoError(t, err)
	resolved, unresolved := resolveTables(tables, aliasMap(plan), "shop")
	assert.Empty(t, resolved)
	assert.Equal(t, []string{"o"}, unresolved)
}

func TestResolveTablesOnlyPlanNames(t *testing.T) {
	t.Parallel()

	aliases := map[string]tableRef{
		"o":      {database: "shop", name: "orders"},
		"unused": {database: "shop", name: "secret"},
	}
	resolved, unresolved := resolveTables([]planTable{{Name: "o"}, {Name: "gone"}}, aliases, "db")
	assert.Equal(t, []tableRef{{database: "shop", name: "orders"}}, resolved)
	assert.Equal(t, []string{"gone"}, unresolved)
}

func TestResolveTablesDatabaseAndDerived(t *testing.T) {
	t.Parallel()

	tables := []planTable{
		{Schema: "s2", Name: "x", Alias: "y"},
		{Name: "t"},
		{Name: "d", Derived: true},
		{Name: "<subquery2>"},
		{Schema: "s2", Name: "x"},
	}
	aliases := map[string]tableRef{"t": {name: "t"}}
	resolved, unresolved := resolveTables(tables, aliases, "db")
	assert.Equal(t, []tableRef{{database: "s2", name: "x"}, {database: "db", name: "t"}}, resolved)
	assert.Equal(t, []string{"d", "<subquery2>"}, unresolved)
}

func TestDecodeExplainEnvelope(t *testing.T) {
	plan := `{"query_block":{"select_id":1}}`
	envelope := `{"explain_result":"` + base64.StdEncoding.EncodeToString([]byte(plan)) +
		`","explained_query":"SELECT secret FROM t","is_dml":false}`

	for _, raw := range []string{"true", "false"} {
		t.Setenv("PMM_MCP_RAW_SQL", raw)
		got, err := decodeExplainEnvelope(envelope)
		require.NoError(t, err)
		assert.Equal(t, plan, string(got))
		assert.NotContains(t, string(got), "secret")
	}

	_, err := decodeExplainEnvelope("not json")
	require.Error(t, err)
	_, err = decodeExplainEnvelope(`{"explain_result":"","explained_query":"x"}`)
	require.Error(t, err)
	_, err = decodeExplainEnvelope(`{"explain_result":"!!!"}`)
	require.Error(t, err)
}
