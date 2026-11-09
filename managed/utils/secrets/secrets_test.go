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
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/utils/secretfile"
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
	v, err := secretfile.Read(filepath.Join(opts.RunDir, string(name)))
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
		opts := testOptions(t, map[string]string{"GF_SECURITY_ADMIN_PASSWORD": "admin\n"})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		r := byName(t, resolved)[AdminPassword]
		assert.True(t, r.Padded)
		assert.False(t, r.Kept)
		assert.Equal(t, "GF_SECURITY_ADMIN_PASSWORD", r.Var)
		assert.Equal(t, "admin", linked(t, opts, AdminPassword))
	})

	t.Run("leaves a padded secret key in its variable", func(t *testing.T) {
		opts := testOptions(t, map[string]string{"GF_SECURITY_SECRET_KEY": "key"})
		_, err := Prepare(opts)
		require.NoError(t, err)
		require.Equal(t, "key", linked(t, opts, GrafanaSecretKey))

		opts.Getenv = func(k string) string { return map[string]string{"GF_SECURITY_SECRET_KEY": "key\n"}[k] }
		resolved, err := Prepare(opts)
		require.NoError(t, err)
		r := byName(t, resolved)[GrafanaSecretKey]
		assert.Equal(t, SourceEnv, r.Source)
		assert.True(t, r.Padded)
		assert.True(t, r.Kept)
		assert.Empty(t, r.Path)
		assert.Equal(t, "GF_SECURITY_SECRET_KEY", r.Var)
		// No link, so the entrypoint keeps the variable and supervisord gives Grafana no file.
		_, err = os.Lstat(filepath.Join(opts.RunDir, string(GrafanaSecretKey)))
		require.ErrorIs(t, err, os.ErrNotExist)
		assert.NoFileExists(t, filepath.Join(opts.RunDir, envSubdir, string(GrafanaSecretKey)))

		// Grafana used a key that is only whitespace as written too.
		opts.Getenv = func(k string) string { return map[string]string{"GF_SECURITY_SECRET_KEY": "\n"}[k] }
		resolved, err = Prepare(opts)
		require.NoError(t, err)
		assert.True(t, byName(t, resolved)[GrafanaSecretKey].Kept)
	})

	t.Run("refuses a padded database password", func(t *testing.T) {
		for k, name := range map[string]Name{
			"PMM_POSTGRES_DBPASSWORD":            PostgresPassword,
			"GF_DATABASE_PASSWORD":               GrafanaDBPassword,
			"PMM_CLICKHOUSE_PASSWORD":            ClickHousePassword,
			"PMM_CLICKHOUSE_DATASOURCE_PASSWORD": ClickHouseDatasourcePassword,
		} {
			_, err := Prepare(testOptions(t, map[string]string{k: " q7v-padded"}))
			require.ErrorContains(t, err, string(name)+": "+k+" has leading or trailing whitespace", k)
			assert.NotContains(t, err.Error(), "q7v", k)
		}
	})

	t.Run("prefers GF_SECURITY_ADMIN_PASSWORD over a different PMM_ADMIN_PASSWORD", func(t *testing.T) {
		opts := testOptions(t, map[string]string{
			"PMM_ADMIN_PASSWORD":         "from-pmm",
			"GF_SECURITY_ADMIN_PASSWORD": "from-grafana",
		})

		resolved, err := Prepare(opts)
		require.NoError(t, err)
		r := byName(t, resolved)[AdminPassword]
		assert.Equal(t, "GF_SECURITY_ADMIN_PASSWORD", r.Var)
		assert.Equal(t, "PMM_ADMIN_PASSWORD", r.Ignored)
		assert.Equal(t, "from-grafana", linked(t, opts, AdminPassword))

		opts.Getenv = func(k string) string { return map[string]string{"PMM_ADMIN_PASSWORD": "only"}[k] }
		resolved, err = Prepare(opts)
		require.NoError(t, err)
		r = byName(t, resolved)[AdminPassword]
		assert.Equal(t, "PMM_ADMIN_PASSWORD", r.Var)
		assert.Empty(t, r.Ignored)
		assert.Equal(t, "only", linked(t, opts, AdminPassword))
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

	t.Run("leaves credentials mounted at the run directory alone", func(t *testing.T) {
		opts := testOptions(t, nil)
		mounted := filepath.Join(opts.RunDir, string(AdminPassword))
		require.NoError(t, os.MkdirAll(opts.RunDir, 0o700))
		require.NoError(t, os.WriteFile(mounted, []byte("mounted"), 0o400))

		for dir, msg := range map[string]string{
			"":          opts.RunDir + " holds admin-password, which PMM Server didn't create",
			opts.RunDir: "PMM_SECRETS_DIR (" + opts.RunDir + ") is inside " + opts.RunDir,
		} {
			opts.Getenv = func(k string) string {
				if k == SecretsDirEnvVar {
					return dir
				}
				return ""
			}
			_, err := Prepare(opts)
			require.ErrorContains(t, err, msg)
			v, err := secretfile.Read(mounted)
			require.NoError(t, err)
			assert.Equal(t, "mounted", v)
		}
	})

	t.Run("makes an existing run directory private", func(t *testing.T) {
		opts := testOptions(t, nil)
		require.NoError(t, os.MkdirAll(opts.RunDir, 0o700))
		// as a tmpfs mounted without a mode
		require.NoError(t, os.Chmod(opts.RunDir, 0o1777))

		_, err := Prepare(opts)
		require.NoError(t, err)
		fi, err := os.Stat(opts.RunDir)
		require.NoError(t, err)
		assert.Equal(t, os.ModeDir|0o700, fi.Mode())
	})

	t.Run("refuses a Kubernetes Secret volume at the run directory", func(t *testing.T) {
		opts := testOptions(t, nil)
		data := filepath.Join(opts.RunDir, "..data")
		require.NoError(t, os.MkdirAll(data, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(data, string(AdminPassword)), []byte("mounted"), 0o400))
		require.NoError(t, os.Symlink(filepath.Join("..data", string(AdminPassword)), filepath.Join(opts.RunDir, string(AdminPassword))))

		_, err := Prepare(opts)
		require.ErrorContains(t, err, "holds ..data, admin-password, which PMM Server didn't create")
		assert.Equal(t, "mounted", linked(t, opts, AdminPassword))
	})

	t.Run("refuses a file inside the run directory", func(t *testing.T) {
		opts := testOptions(t, map[string]string{"GF_SECURITY_SECRET_KEY": "earlier", "GF_SECURITY_ADMIN_PASSWORD": "earlier"})
		_, err := Prepare(opts)
		require.NoError(t, err)

		key, admin := filepath.Join(opts.RunDir, string(GrafanaSecretKey)), filepath.Join(opts.RunDir, string(AdminPassword))
		env := map[string]string{"PMM_GRAFANA_SECRET_KEY_FILE": key, "GF_SECURITY_ADMIN_PASSWORD": "$__file{" + admin + "}"}
		opts.Getenv = func(k string) string { return env[k] }
		_, err = Prepare(opts)
		require.Error(t, err)
		for _, s := range []string{
			"grafana-secret-key: " + key + " is inside " + opts.RunDir,
			"admin-password: " + admin + " is inside " + opts.RunDir,
		} {
			assert.Contains(t, err.Error(), s)
		}
	})

	t.Run("refuses credential links it didn't create", func(t *testing.T) {
		opts := testOptions(t, nil)
		file := filepath.Join(t.TempDir(), "admin")
		require.NoError(t, os.WriteFile(file, []byte("mounted"), 0o400))
		require.NoError(t, os.MkdirAll(opts.RunDir, 0o700))
		require.NoError(t, os.Symlink(file, filepath.Join(opts.RunDir, string(AdminPassword))))

		_, err := Prepare(opts)
		require.ErrorContains(t, err, opts.RunDir+" holds admin-password, which PMM Server didn't create")
		assert.Equal(t, "mounted", linked(t, opts, AdminPassword))
	})

	t.Run("refuses anything else in the directory of its own values", func(t *testing.T) {
		opts := testOptions(t, nil)
		env := filepath.Join(opts.RunDir, envSubdir)
		require.NoError(t, os.MkdirAll(filepath.Join(env, "keep"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(env, "notes"), []byte("mounted"), 0o600))
		require.NoError(t, os.Symlink(filepath.Join(env, "notes"), filepath.Join(env, string(AdminPassword))))

		_, err := Prepare(opts)
		require.Error(t, err)
		for _, s := range []string{".env/admin-password", ".env/keep", ".env/notes", "which PMM Server didn't create"} {
			assert.Contains(t, err.Error(), s)
		}
		assert.DirExists(t, filepath.Join(env, "keep"))
		assert.FileExists(t, filepath.Join(env, "notes"))
	})

	t.Run("removes a temporary file that an interrupted start left", func(t *testing.T) {
		opts := testOptions(t, nil)
		tmp := filepath.Join(opts.RunDir, envSubdir, "."+string(PostgresPassword)+".tmp123")
		require.NoError(t, os.MkdirAll(filepath.Dir(tmp), 0o700))
		require.NoError(t, os.WriteFile(tmp, []byte("pmm-man"), 0o600))

		_, err := Prepare(opts)
		require.NoError(t, err)
		assert.NoFileExists(t, tmp)
		assert.Equal(t, "pmm-managed", linked(t, opts, PostgresPassword))
	})

	t.Run("starts again after a start killed while it dropped the earlier values", func(t *testing.T) {
		opts := testOptions(t, map[string]string{"GF_SECURITY_SECRET_KEY": "earlier"})
		_, err := Prepare(opts)
		require.NoError(t, err)
		env := filepath.Join(opts.RunDir, envSubdir)
		before, err := os.Stat(env)
		require.NoError(t, err)
		// What a start killed right after emptying .env leaves: the earlier links lead nowhere.
		entries, err := os.ReadDir(env)
		require.NoError(t, err)
		for _, e := range entries {
			require.NoError(t, os.Remove(filepath.Join(env, e.Name())))
		}

		_, err = Prepare(opts)
		require.NoError(t, err)
		assert.Equal(t, "earlier", linked(t, opts, GrafanaSecretKey))
		after, err := os.Stat(env)
		require.NoError(t, err)
		assert.True(t, os.SameFile(before, after), ".env must stay, or a start killed meanwhile leaves links without it")
	})

	t.Run("refuses a dangling link in the secrets directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, string(PostgresPassword))))

		_, err := Prepare(testOptions(t, map[string]string{SecretsDirEnvVar: dir}))
		require.ErrorIs(t, err, os.ErrNotExist)
		require.ErrorContains(t, err, "postgres-password: open "+filepath.Join(dir, string(PostgresPassword)))
	})

	t.Run("refuses a $__file{} value with a file in the secrets directory", func(t *testing.T) {
		dir, other := t.TempDir(), filepath.Join(t.TempDir(), "other")
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(AdminPassword)), []byte("from-dir"), 0o440))
		require.NoError(t, os.WriteFile(other, []byte("from-other"), 0o440))

		_, err := Prepare(testOptions(t, map[string]string{SecretsDirEnvVar: dir, "GF_SECURITY_ADMIN_PASSWORD": "$__file{" + other + "}"}))
		require.ErrorContains(t, err, "admin-password: both "+filepath.Join(dir, string(AdminPassword))+" and a value")
	})

	t.Run("refuses a secrets directory that is missing or not a directory", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, []byte("x"), 0o440))
		missing := filepath.Join(t.TempDir(), "missing")
		for dir, msg := range map[string]string{
			missing: "PMM_SECRETS_DIR: lstat " + missing + ": no such file or directory",
			file:    "PMM_SECRETS_DIR (" + file + ") is not a directory",
		} {
			_, err := Prepare(testOptions(t, map[string]string{SecretsDirEnvVar: dir}))
			require.EqualError(t, err, msg)
		}
	})

	t.Run("refuses a source that leads back through its link", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(GrafanaDBPassword)), []byte("from-dir"), 0o440))
		opts := testOptions(t, map[string]string{SecretsDirEnvVar: dir})
		_, err := Prepare(opts)
		require.NoError(t, err)

		alias := filepath.Join(t.TempDir(), "alias")
		require.NoError(t, os.Symlink(filepath.Join(opts.RunDir, string(GrafanaDBPassword)), alias))
		opts.Getenv = func(k string) string { return map[string]string{"PMM_GRAFANA_DB_PASSWORD_FILE": alias}[k] }
		_, err = Prepare(opts)
		require.ErrorIs(t, err, syscall.ELOOP)
		require.ErrorContains(t, err, "grafana-db-password: ")

		// a fixed configuration replaces the loop
		opts.Getenv = func(k string) string { return map[string]string{SecretsDirEnvVar: dir}[k] }
		_, err = Prepare(opts)
		require.NoError(t, err)
		assert.Equal(t, "from-dir", linked(t, opts, GrafanaDBPassword))
	})

	t.Run("refuses a secrets directory that leads into the run directory", func(t *testing.T) {
		env := map[string]string{"PMM_POSTGRES_DBPASSWORD": "from-env"}
		opts := testOptions(t, env)
		_, err := Prepare(opts)
		require.NoError(t, err)

		for _, target := range []string{opts.RunDir, filepath.Join(opts.RunDir, envSubdir)} {
			alias := filepath.Join(t.TempDir(), "secrets")
			require.NoError(t, os.Symlink(target, alias))
			env[SecretsDirEnvVar] = alias
			_, err = Prepare(opts)
			require.EqualError(t, err, "PMM_SECRETS_DIR ("+alias+") is inside "+opts.RunDir+", where PMM Server links the credentials")
			// refused before anything changed
			assert.Equal(t, "from-env", linked(t, opts, PostgresPassword))
		}
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
			"PMM_POSTGRES_DBPASSWORD":      "q7v-env",
			"PMM_GRAFANA_SECRET_KEY_FILE":  filepath.Join(dir, "missing"),
		})

		_, err := Prepare(opts)
		require.Error(t, err)
		for _, s := range []string{
			"clickhouse-password: both a file",
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

func TestInside(t *testing.T) {
	for path, want := range map[string]bool{
		"/run/pmm/secrets":                true,
		"/run/pmm/secrets/admin-password": true,
		"/run/pmm/other/../secrets/x":     true,
		"/run/pmm":                        false,
		"/run/pmm/secrets-old/x":          false,
		"/etc/pmm/secrets":                false,
	} {
		assert.Equal(t, want, inside("/run/pmm/secrets", path), path)
	}
}

func TestEnvVarNames(t *testing.T) {
	names := EnvVarNames()
	assert.Contains(t, names, SecretsDirEnvVar)
	assert.Contains(t, names, "PMM_CLICKHOUSE_PASSWORD_FILE")
	assert.NotContains(t, names, "PMM_CLICKHOUSE_PASSWORD")
}

func TestEntrypointUnsetsLegacyValues(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "build", "docker", "server", "entrypoint.sh"))
	require.NoError(t, err)
	unset := make(map[string]bool)
	for _, m := range regexp.MustCompile(`(?m)^\s*unset((?:[ \t]+[A-Z][A-Z0-9_]*|[ \t]*\\\n)+)`).FindAllStringSubmatch(string(b), -1) {
		for name := range strings.FieldsSeq(strings.ReplaceAll(m[1], "\\\n", " ")) {
			unset[name] = true
		}
	}
	for _, s := range catalog {
		for _, k := range s.valueEnvs {
			assert.True(t, unset[k], "build/docker/server/entrypoint.sh doesn't unset %s", k)
		}
	}
}
