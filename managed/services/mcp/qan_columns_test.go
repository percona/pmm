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
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQANColumnsMatchQANAPI2 pins the catalogue to qan-api2's column maps: a name
// they don't know is dropped from a report without an error.
func TestQANColumnsMatchQANAPI2(t *testing.T) {
	t.Parallel()

	maps := qanAPI2Maps(t)
	// name -> reported as an average
	want := make(map[string]bool)
	for _, m := range []string{"sumColumnNames", "specialColumnNames"} {
		for name := range maps[m] {
			want[name] = false
		}
	}
	for name := range maps["commonColumnNames"] {
		want[name] = true
	}

	got := make(map[string]bool, len(qanColumns))
	for _, c := range qanColumns {
		got[c.name] = c.avg
		assert.Contains(t, []string{"", engineMySQL, enginePostgreSQL, engineMongoDB}, c.engine, c.name)
	}
	assert.Len(t, qanColumns, len(got), "a name is listed twice")
	assert.Equal(t, want, got)

	// qan-api2 rejects any other group_by with a bare HTTP 500.
	for _, d := range groupByDimensions {
		assert.Contains(t, maps["standartDimensions"], d)
	}
}

// qanAPI2Maps reads the map literals declared in qan-api2's analytics/base.go.
func qanAPI2Maps(t *testing.T) map[string]map[string]struct{} {
	t.Helper()

	path := filepath.Join("..", "..", "..", "qan-api2", "services", "analytics", "base.go")
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	require.NoError(t, err)

	maps := make(map[string]map[string]struct{})
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
			return true
		}
		lit, ok := vs.Values[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		keys := make(map[string]struct{})
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.BasicLit)
			if !ok || key.Kind != token.STRING {
				continue
			}
			name, err := strconv.Unquote(key.Value)
			require.NoError(t, err)
			keys[name] = struct{}{}
		}
		maps[vs.Names[0].Name] = keys
		return true
	})
	for _, m := range []string{"standartDimensions", "sumColumnNames", "specialColumnNames", "commonColumnNames"} {
		require.NotEmpty(t, maps[m], "%s not found in %s", m, path)
	}
	return maps
}
