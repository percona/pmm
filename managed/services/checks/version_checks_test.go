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

package checks

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/starlark"
)

type finding struct {
	Severity string
	Summary  string
	Labels   map[string]string
}

func findings(results []check.Result) []finding {
	out := make([]finding, 0, len(results))
	for _, r := range results {
		f := finding{Severity: r.Severity.String(), Summary: r.Summary}
		if len(r.Labels) > 0 {
			f.Labels = r.Labels
		}

		out = append(out, f)
	}

	return out
}

// runBuiltinCheck runs a built-in check script the way pmm-managed-starlark does, with the clock fixed to noon UTC of today.
func runBuiltinCheck(t *testing.T, name, today string, docs ...[]map[string]any) []check.Result {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("..", "..", "data", "checks", name+".yml"))
	require.NoError(t, err)

	parsed, err := check.ParseChecks(bytes.NewReader(b), &check.ParseParams{
		DisallowUnknownFields: true,
		DisallowInvalidChecks: true,
	})
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	c := parsed[0]

	day, err := time.Parse("2006-01-02", today)
	require.NoError(t, err)
	fixed := day.Add(12 * time.Hour)

	orig := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = orig })

	funcs, err := GetFuncsForVersion(c.Version)
	require.NoError(t, err)

	env, err := starlark.NewEnv(name, c.Script, funcs)
	require.NoError(t, err)

	input := make([]any, len(docs))
	for i, d := range docs {
		input[i] = d
	}

	results, err := env.Run(name, input, GetAdditionalContext(), t.Log)
	require.NoError(t, err)

	return results
}

func mysqlVariables(version, comment, basedir, datadir string, extra ...string) []map[string]any {
	vars := [][2]string{
		{"version", version},
		{"version_comment", comment},
		{"basedir", basedir},
		{"datadir", datadir},
	}
	for i := 0; i+1 < len(extra); i += 2 {
		vars = append(vars, [2]string{extra[i], extra[i+1]})
	}

	rows := make([]map[string]any, 0, len(vars))
	for _, v := range vars {
		rows = append(rows, map[string]any{"Variable_name": v[0], "Value": v[1]})
	}

	return rows
}

func mongoBuildInfo(version string) []map[string]any {
	return []map[string]any{{"version": version}}
}

func mongoCmdLineOpts(parsed map[string]any) []map[string]any {
	return []map[string]any{{"parsed": parsed}}
}

func pgVersionRows(serverVersionNum int) []map[string]any {
	return []map[string]any{{
		"setting": strconv.Itoa(serverVersionNum),
		"major":   int64(serverVersionNum / 10000),
	}}
}

func promVector(labels map[string]string, value string) []map[string]any {
	return []map[string]any{{
		"metric": labels,
		"value":  []any{float64(1762948800), value},
	}}
}

func TestBuiltinVersionChecks(t *testing.T) {
	const today = "2026-11-12"

	for _, tc := range []struct {
		name  string
		check string
		today string
		docs  [][]map[string]any
		want  []finding
	}{
		{
			name:  "mysql_version Oracle 8.0.37",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("8.0.37", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want: []finding{{
				Severity: "warning",
				Summary:  "Newer version of MySQL is available",
				Labels:   map[string]string{"current": "8.0.37", "latest": "8.0.38"},
			}},
		},
		{
			name:  "mongodb_EOL 4.4.29",
			check: "mongodb_EOL",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("4.4.29")},
			want:  []finding{{Severity: "warning", Summary: "MongoDB version 4.4 is about to reach EOL"}},
		},
		{
			name:  "mongodb_version 6.0.19-16",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("6.0.19-16")},
			want:  []finding{{Severity: "warning", Summary: "Newer version of MongoDB is available."}},
		},
		{
			name:  "postgresql_eol_check 100023",
			check: "postgresql_eol_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(100023)},
			want:  []finding{{Severity: "warning", Summary: "Currently installed version is (10.3)"}},
		},
		{
			name:  "mysql_unsupported_version_check 5.6.51",
			check: "mysql_unsupported_version_check",
			today: today,
			docs: [][]map[string]any{{{
				"version": "5.6.51",
				"service": "db1",
				"comment": "MySQL Community Server (GPL)",
				"basedir": "/usr/",
				"datadir": "/var/lib/mysql/",
				"days":    "0",
			}}},
			want: []finding{{
				Severity: "warning",
				Summary:  "Unsupported Version Check",
				Labels:   map[string]string{"mysql_version": "5.6.51", "service": "db1"},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := findings(runBuiltinCheck(t, tc.check, tc.today, tc.docs...))
			assert.Equal(t, tc.want, got)
		})
	}
}
