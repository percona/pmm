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

package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOptions(t *testing.T, env map[string]string) Options {
	t.Helper()
	return Options{
		Getenv: func(k string) string { return env[k] },
		RunDir: filepath.Join(t.TempDir(), "run", "pmm", "secrets"),
	}
}

func byName(t *testing.T, resolved []Resolved) map[Name]Resolved {
	t.Helper()
	m := make(map[Name]Resolved, len(resolved))
	for _, r := range resolved {
		m[r.Name] = r
	}
	return m
}

func linked(t *testing.T, opts Options, name Name) string {
	t.Helper()
	v, err := readFile(filepath.Join(opts.RunDir, string(name)))
	require.NoError(t, err)
	return v
}

func TestPrepare(t *testing.T) {
	t.Run("keeps the default values when nothing is configured", func(t *testing.T) {
		opts := testOptions(t, nil)

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		m := byName(t, resolved)
		assert.Len(t, m, len(catalog))
		for name, value := range map[Name]string{
			PostgresPassword:             "pmm-managed",
			GrafanaDBPassword:            "grafana",
			ClickHousePassword:           "clickhouse",
			ClickHouseDatasourcePassword: "grafana",
		} {
			assert.Equal(t, SourceDefault, m[name].Source, name)
			assert.Equal(t, value, linked(t, opts, name), name)
		}
		for _, name := range []Name{AdminPassword, GrafanaSecretKey} {
			assert.Equal(t, SourceUnset, m[name].Source, name)
			_, err = os.Lstat(filepath.Join(opts.RunDir, string(name)))
			require.ErrorIs(t, err, os.ErrNotExist, name)
		}

		fi, err := os.Stat(m[ClickHousePassword].Path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	})

	t.Run("moves legacy values into files", func(t *testing.T) {
		opts := testOptions(t, map[string]string{
			"PMM_CLICKHOUSE_PASSWORD":    "ch-value",
			"PMM_ADMIN_PASSWORD":         "admin-value",
			"GF_SECURITY_ADMIN_PASSWORD": "admin-value",
		})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		m := byName(t, resolved)
		assert.Equal(t, SourceEnv, m[ClickHousePassword].Source)
		assert.Equal(t, "ch-value", linked(t, opts, ClickHousePassword))
		assert.Equal(t, SourceEnv, m[AdminPassword].Source)
		assert.Equal(t, "admin-value", linked(t, opts, AdminPassword))
	})

	t.Run("uses a file and the secrets directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(GrafanaDBPassword)), []byte("from-dir\n"), 0o440))
		file := filepath.Join(dir, "custom")
		require.NoError(t, os.WriteFile(file, []byte("from-file"), 0o440))
		opts := testOptions(t, map[string]string{
			SecretsDirEnvVar:               dir,
			"PMM_CLICKHOUSE_PASSWORD_FILE": file,
		})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		m := byName(t, resolved)
		assert.Equal(t, SourceDir, m[GrafanaDBPassword].Source)
		assert.Equal(t, filepath.Join(dir, string(GrafanaDBPassword)), m[GrafanaDBPassword].Path)
		assert.Equal(t, "from-dir", linked(t, opts, GrafanaDBPassword))
		assert.Equal(t, SourceFile, m[ClickHousePassword].Source)
		assert.Equal(t, "from-file", linked(t, opts, ClickHousePassword))
	})

	t.Run("drops values from an earlier start", func(t *testing.T) {
		opts := testOptions(t, map[string]string{"GF_SECURITY_SECRET_KEY": "earlier"})
		_, err := Prepare(opts)
		require.NoError(t, err)
		require.Equal(t, "earlier", linked(t, opts, GrafanaSecretKey))

		opts.Getenv = func(string) string { return "" }
		resolved, err := Prepare(opts)
		require.NoError(t, err)
		assert.Equal(t, SourceUnset, byName(t, resolved)[GrafanaSecretKey].Source)
		assert.NoFileExists(t, filepath.Join(opts.RunDir, envSubdir, string(GrafanaSecretKey)))
		_, err = os.Lstat(filepath.Join(opts.RunDir, string(GrafanaSecretKey)))
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("reports every configuration error without the values", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "custom")
		require.NoError(t, os.WriteFile(file, []byte("x"), 0o440))
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(PostgresPassword)), []byte("q7v-dir"), 0o440))
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(ClickHouseDatasourcePassword)), []byte(" \n"), 0o440))
		opts := testOptions(t, map[string]string{
			SecretsDirEnvVar:               dir,
			"PMM_CLICKHOUSE_PASSWORD_FILE": file,
			"PMM_CLICKHOUSE_PASSWORD":      "q7v-value",
			"PMM_ADMIN_PASSWORD":           "q7v-first",
			"GF_SECURITY_ADMIN_PASSWORD":   "q7v-second",
			"PMM_POSTGRES_DBPASSWORD":      "q7v-env",
			"PMM_GRAFANA_SECRET_KEY_FILE":  filepath.Join(dir, "missing"),
		})

		_, err := Prepare(opts)
		require.Error(t, err)
		for _, s := range []string{
			"clickhouse-password: both a file",
			"admin-password: PMM_ADMIN_PASSWORD and GF_SECURITY_ADMIN_PASSWORD are set to different values",
			"postgres-password: both " + filepath.Join(dir, string(PostgresPassword)) + " and a value",
			"clickhouse-datasource-password: " + filepath.Join(dir, string(ClickHouseDatasourcePassword)) + " is empty",
			"grafana-secret-key:",
		} {
			assert.Contains(t, err.Error(), s)
		}
		assert.NotContains(t, err.Error(), "q7v")
	})
}

func TestValue(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, string(GrafanaDBPassword)), []byte("from-file\n"), 0o600))

	v, err := valueIn(dir, "explicit", GrafanaDBPassword, "default")
	require.NoError(t, err)
	assert.Equal(t, "explicit", v)

	v, err = valueIn(dir, "", GrafanaDBPassword, "default")
	require.NoError(t, err)
	assert.Equal(t, "from-file", v)

	v, err = valueIn(dir, "", ClickHousePassword, "default")
	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Equal(t, "default", v)
}

func TestEnvVarNames(t *testing.T) {
	names := EnvVarNames()
	assert.Contains(t, names, SecretsDirEnvVar)
	assert.Contains(t, names, "PMM_CLICKHOUSE_PASSWORD_FILE")
	assert.NotContains(t, names, "PMM_CLICKHOUSE_PASSWORD")
}
