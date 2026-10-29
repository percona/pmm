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
		assert.False(t, m[AdminPassword].Padded)
	})

	t.Run("expands a value the way Grafana did", func(t *testing.T) {
		pw := filepath.Join(t.TempDir(), "pw")
		require.NoError(t, os.WriteFile(pw, []byte("from-file\n"), 0o440))
		opts := testOptions(t, map[string]string{
			"GF_SECURITY_ADMIN_PASSWORD":         "$__file{" + pw + "}",
			"GF_DATABASE_PASSWORD":               "${REF}-$__env{KEY}-$__vault{key}",
			"REF":                                "$__file{" + pw + "}",
			"KEY":                                "from-env",
			"PMM_CLICKHOUSE_DATASOURCE_PASSWORD": "${KEY}-$__file{" + pw + "}",
			"GF_SECURITY_SECRET_KEY":             "$__file{" + pw + "}",
			"PMM_POSTGRES_DBPASSWORD":            "$__file{" + pw + "}",
		})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		m := byName(t, resolved)
		// A value that is only $__file{} links the file, as a file variable does.
		assert.Equal(t, SourceFile, m[AdminPassword].Source)
		assert.Equal(t, pw, m[AdminPassword].Path)
		assert.Equal(t, "from-file", linked(t, opts, AdminPassword))
		// Variables first, then files; Grafana has no vault provider.
		assert.Equal(t, "from-file-from-env-$__vault{key}", linked(t, opts, GrafanaDBPassword))
		// Grafana's provisioning applied only the files pass to this one.
		assert.Equal(t, "${KEY}-from-file", linked(t, opts, ClickHouseDatasourcePassword))
		// Grafana encrypted with the secret key as written, and nothing expanded pmm-managed's own password.
		assert.Equal(t, SourceEnv, m[GrafanaSecretKey].Source)
		assert.Equal(t, "$__file{"+pw+"}", linked(t, opts, GrafanaSecretKey))
		assert.Equal(t, "$__file{"+pw+"}", linked(t, opts, PostgresPassword))
	})

	t.Run("ignores Grafana's __FILE variables, as earlier versions did", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "pw")
		require.NoError(t, os.WriteFile(file, []byte("from-file"), 0o440))
		opts := testOptions(t, map[string]string{
			"GF_SECURITY_ADMIN_PASSWORD__FILE": file,
			"GF_SECURITY_SECRET_KEY__FILE":     file,
			"GF_DATABASE_PASSWORD__FILE":       file,
		})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		m := byName(t, resolved)
		assert.Equal(t, SourceUnset, m[AdminPassword].Source)
		assert.Equal(t, SourceUnset, m[GrafanaSecretKey].Source)
		assert.Equal(t, SourceDefault, m[GrafanaDBPassword].Source)
	})

	t.Run("flags a value with surrounding whitespace", func(t *testing.T) {
		opts := testOptions(t, map[string]string{"GF_SECURITY_SECRET_KEY": "key\n"})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		r := byName(t, resolved)[GrafanaSecretKey]
		assert.True(t, r.Padded)
		assert.Equal(t, "GF_SECURITY_SECRET_KEY", r.Var)
		assert.Equal(t, "key", linked(t, opts, GrafanaSecretKey))
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

	t.Run("links a relative path by its absolute path", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ch"), []byte("from-relative"), 0o440))
		t.Chdir(dir)
		opts := testOptions(t, map[string]string{"PMM_CLICKHOUSE_PASSWORD_FILE": "ch"})

		_, err := Prepare(opts)
		require.NoError(t, err)
		target, err := os.Readlink(filepath.Join(opts.RunDir, string(ClickHousePassword)))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "ch"), target)
		assert.Equal(t, "from-relative", linked(t, opts, ClickHousePassword))
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

	t.Run("reports a value that does not expand", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		opts := testOptions(t, map[string]string{
			"GF_SECURITY_ADMIN_PASSWORD":         "$__file{" + missing + "}",
			"PMM_CLICKHOUSE_DATASOURCE_PASSWORD": "q7v-$__file{" + missing + "}",
			"GF_DATABASE_PASSWORD":               "${UNSET}",
		})

		_, err := Prepare(opts)
		require.Error(t, err)
		for _, s := range []string{
			"admin-password: open " + missing,
			"clickhouse-datasource-password: PMM_CLICKHOUSE_DATASOURCE_PASSWORD: open " + missing,
			"grafana-db-password: GF_DATABASE_PASSWORD resolves to an empty value",
		} {
			assert.Contains(t, err.Error(), s)
		}
		assert.NotContains(t, err.Error(), "q7v")
	})
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "credential")
	assert.False(t, exists(file))

	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))
	assert.True(t, exists(file))

	// An error other than absence, here "not a directory", leaves the credential to its reader rather than to a default.
	assert.True(t, exists(filepath.Join(file, "child")))

	// So does a link whose file went away.
	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(filepath.Join(dir, "gone"), link))
	assert.True(t, exists(link))
}

func TestValue(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, string(GrafanaDBPassword)), []byte("from-file\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, string(AdminPassword)), nil, 0o600))
	require.NoError(t, os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, string(PostgresPassword))))

	v, err := valueIn(dir, "explicit", GrafanaDBPassword, "default")
	require.NoError(t, err)
	assert.Equal(t, "explicit", v)

	v, err = valueIn(dir, "", GrafanaDBPassword, "default")
	require.NoError(t, err)
	assert.Equal(t, "from-file", v)

	v, err = valueIn(dir, "", ClickHousePassword, "default")
	require.ErrorIs(t, err, ErrNotResolved)
	assert.Equal(t, "default", v)

	// A resolved credential that can't be read is an error, never the default.
	for _, name := range []Name{AdminPassword, PostgresPassword} {
		v, err = valueIn(dir, "", name, "default")
		require.Error(t, err, name)
		require.NotErrorIs(t, err, ErrNotResolved, name)
		assert.Empty(t, v, name)
	}
}

func TestEnvVarNames(t *testing.T) {
	names := EnvVarNames()
	assert.Contains(t, names, SecretsDirEnvVar)
	assert.Contains(t, names, "PMM_CLICKHOUSE_PASSWORD_FILE")
	assert.NotContains(t, names, "PMM_CLICKHOUSE_PASSWORD")
}
