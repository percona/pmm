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

package clickhouse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfigFiles creates the given files in dir with placeholder content.
func writeConfigFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("<clickhouse/>"), 0o600))
	}
}

func TestGetClickHouseConfig(t *testing.T) {
	t.Parallel()

	// Empty input falls back to the default config.
	got, err := GetClickHouseConfig("")
	require.NoError(t, err)
	assert.Equal(t, defaultClickHouseConfig, got)
}

func TestValidateClickHouseConfigAt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// A config is valid as long as <name>-config.xml exists.
	writeConfigFiles(
		t, dir,
		"default-config.xml",
		"low-memory-config.xml",
	)

	tests := []struct {
		name        string
		config      string
		errContains []string
	}{
		{name: "default", config: "default"},
		{name: "low-memory", config: "low-memory"},
		{
			name:   "missing",
			config: "nonexistent",
			errContains: []string{
				`invalid PMM_CLICKHOUSE_CONFIG=nonexistent`,
				"available configs:",
				"default", "low-memory",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateClickHouseConfigAt(tt.config, dir)
			if tt.errContains == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, substr := range tt.errContains {
				assert.Contains(t, err.Error(), substr)
			}
		})
	}

	t.Run("invalid config dir", func(t *testing.T) {
		t.Parallel()

		base := t.TempDir()
		// "notdir" is a regular file; using it as the config dir makes os.Stat fail
		require.NoError(t, os.WriteFile(filepath.Join(base, "notdir"), nil, 0o600))

		err := validateClickHouseConfigAt("default", filepath.Join(base, "notdir"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot stat")
		assert.NotContains(t, err.Error(), "available configs:")
	})
}

func TestAvailableClickHouseConfigs(t *testing.T) {
	t.Parallel()

	t.Run("empty dir", func(t *testing.T) {
		t.Parallel()

		got, err := availableClickHouseConfigs(t.TempDir())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("lists config names sorted, ignoring non-config files", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeConfigFiles(
			t, dir,
			"low-memory-config.xml",
			"default-config.xml",
			"dhparam.pem", // not a *-config.xml, must be ignored
		)

		got, err := availableClickHouseConfigs(dir)
		require.NoError(t, err)
		assert.Equal(t, []string{"default", "low-memory"}, got)
	})
}
