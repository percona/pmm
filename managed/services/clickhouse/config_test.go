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
		{
			name:        "path traversal",
			config:      "../../tmp/evil",
			errContains: []string{"must be a name, not a path"},
		},
		{
			name:        "absolute path",
			config:      "/tmp/evil",
			errContains: []string{"must be a name, not a path"},
		},
		{
			name:        "parent dir",
			config:      "..",
			errContains: []string{"must be a name, not a path"},
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

func TestLinkClickHouseConfigAt(t *testing.T) {
	t.Parallel()

	// newConfigDir returns a dir holding the config files of both shipped configs.
	newConfigDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeConfigFiles(
			t, dir,
			"default-config.xml", "default-users.xml",
			"low-memory-config.xml", "low-memory-users.xml",
		)
		return dir
	}

	assertLinks := func(t *testing.T, dir, config string) {
		t.Helper()
		for _, l := range stableConfigLinks {
			target, err := os.Readlink(filepath.Join(dir, l.link))
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(dir, config+l.suffix), target)
		}
	}

	t.Run("creates links when absent", func(t *testing.T) {
		t.Parallel()

		dir := newConfigDir(t)
		require.NoError(t, linkClickHouseConfigAt("default", dir))
		assertLinks(t, dir, "default")
	})

	t.Run("repoints existing links when the config changes", func(t *testing.T) {
		t.Parallel()

		dir := newConfigDir(t)
		require.NoError(t, linkClickHouseConfigAt("default", dir))
		require.NoError(t, linkClickHouseConfigAt("low-memory", dir))
		assertLinks(t, dir, "low-memory")

		// Switching back must work too, and leave no temporary files behind.
		require.NoError(t, linkClickHouseConfigAt("default", dir))
		assertLinks(t, dir, "default")
		for _, l := range stableConfigLinks {
			_, err := os.Lstat(filepath.Join(dir, l.link+".tmp"))
			assert.ErrorIs(t, err, os.ErrNotExist)
		}
	})

	t.Run("leaves a regular file untouched", func(t *testing.T) {
		t.Parallel()

		// Anything that replaced the symlink with a regular file, such as an in-place `sed -i`,
		// is left alone rather than silently discarded.
		dir := newConfigDir(t)
		edited := filepath.Join(dir, "config.xml")
		require.NoError(t, os.WriteFile(edited, []byte("<clickhouse>edited</clickhouse>"), 0o600))

		require.NoError(t, linkClickHouseConfigAt("low-memory", dir))

		content, err := os.ReadFile(edited) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "<clickhouse>edited</clickhouse>", string(content))

		// The remaining link is still repointed.
		target, err := os.Readlink(filepath.Join(dir, "users.xml"))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "low-memory-users.xml"), target)
	})

	t.Run("fails when the target is missing", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeConfigFiles(t, dir, "default-config.xml") // default-users.xml deliberately absent

		err := linkClickHouseConfigAt("default", dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default-users.xml")
	})

	t.Run("leaves links untouched when a target is missing", func(t *testing.T) {
		t.Parallel()

		// config.xml must not be repointed when users.xml cannot be, otherwise the two links
		// straddle different configs.
		dir := newConfigDir(t)
		require.NoError(t, linkClickHouseConfigAt("default", dir))
		require.NoError(t, os.Remove(filepath.Join(dir, "low-memory-users.xml")))

		require.Error(t, linkClickHouseConfigAt("low-memory", dir))
		assertLinks(t, dir, "default")
	})

	t.Run("does not rewrite links that are already correct", func(t *testing.T) {
		t.Parallel()

		// Writing to /etc may be denied under an arbitrary UID, so an unchanged config must not
		// need any write at all.
		dir := newConfigDir(t)
		require.NoError(t, linkClickHouseConfigAt("default", dir))

		// A directory needs its execute bit, so these are wider than gosec's file limit.
		require.NoError(t, os.Chmod(dir, 0o500))       //nolint:gosec
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec

		require.NoError(t, linkClickHouseConfigAt("default", dir))
		assertLinks(t, dir, "default")
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
