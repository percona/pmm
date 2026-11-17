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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/pi/starlark"
)

// runBuiltinCheck runs a bundled advisor from managed/data/checks the way pmm-managed-starlark does.
func runBuiltinCheck(t *testing.T, name string, queryResults ...[]map[string]any) []check.Result {
	t.Helper()

	f, err := os.Open(filepath.Join("..", "..", "data", "checks", name+".yml"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	checks, err := check.ParseChecks(f, &check.ParseParams{DisallowUnknownFields: true, DisallowInvalidChecks: true})
	require.NoError(t, err)
	require.Len(t, checks, 1)
	c := checks[0]

	funcs, err := GetFuncsForVersion(c.Version)
	require.NoError(t, err)
	env, err := starlark.NewEnv(c.Name, c.Script, funcs)
	require.NoError(t, err)

	input := make([]any, len(queryResults))
	for i, qr := range queryResults {
		input[i] = qr
	}
	results, err := env.Run(c.Name, input, GetAdditionalContext(), func(...any) {})
	require.NoError(t, err)

	return results
}

func TestMySQLReplicaSkippingErrorsCheck(t *testing.T) {
	t.Parallel()

	const name = "mysql_replica_running_skipping_errors_or_idempotent_mode"

	// The check's query returns the replica_* columns only from 8.0.26 on (gated by /*!80026 */).
	slaveOnly := func(version, execMode, skipErrors string) map[string]any {
		return map[string]any{
			"version": version, "service": "db1",
			"slave_exec_mode": execMode, "slave_skip_errors": skipErrors,
		}
	}
	both := func(version, execMode, skipErrors string) map[string]any {
		return map[string]any{
			"version": version, "service": "db1",
			"slave_exec_mode": execMode, "slave_skip_errors": skipErrors,
			"replica_exec_mode": execMode, "replica_skip_errors": skipErrors,
		}
	}

	type finding struct {
		summary  string
		severity common.Severity
	}

	for _, tc := range []struct {
		name string
		row  map[string]any
		want []finding
	}{
		{name: "5.7 healthy", row: slaveOnly("5.7.44-log", "STRICT", "OFF")},
		{name: "8.0.20 healthy", row: slaveOnly("8.0.20", "STRICT", "OFF")},
		{name: "8.0.26 healthy", row: both("8.0.26", "STRICT", "OFF")},
		{name: "8.4 healthy", row: both("8.4.13", "STRICT", "OFF")},
		{name: "9.7 healthy", row: both("9.7.4", "STRICT", "OFF")},
		{name: "26.7 healthy", row: both("26.7.0", "STRICT", "OFF")},
		{name: "26.10 healthy", row: both("26.10.0", "STRICT", "OFF")},
		{name: "Percona Server 26.7 healthy", row: both("26.7.0-1", "STRICT", "OFF")},
		{
			name: "8.0.20 skips errors",
			row:  slaveOnly("8.0.20", "STRICT", "1062"),
			want: []finding{{"slave_skip_error is configured as 1062.", common.Error}},
		},
		{
			name: "8.0.25 skips errors",
			row:  slaveOnly("8.0.25", "STRICT", "1062"),
			want: []finding{{"slave_skip_error is configured as 1062.", common.Error}},
		},
		{
			name: "8.0.26 skips errors",
			row:  both("8.0.26", "STRICT", "1062"),
			want: []finding{{"replica_skip_error is configured as 1062.", common.Error}},
		},
		{
			name: "8.4 idempotent",
			row:  both("8.4.13", "IDEMPOTENT", "OFF"),
			want: []finding{{"replica_exec_mode is IDEMPOTENT.", common.Error}},
		},
		{
			name: "9.7 skips errors",
			row:  both("9.7.4", "STRICT", "1062"),
			want: []finding{{"replica_skip_error is configured as 1062.", common.Error}},
		},
		{
			name: "26.10 idempotent and skips errors",
			row:  both("26.10.0", "IDEMPOTENT", "1062,1032"),
			want: []finding{
				{"replica_exec_mode is IDEMPOTENT.", common.Error},
				{"replica_skip_error is configured as 1062,1032.", common.Error},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			results := runBuiltinCheck(t, name, []map[string]any{tc.row})

			var got []finding
			for _, r := range results {
				got = append(got, finding{r.Summary, r.Severity})
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
