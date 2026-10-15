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

func TestApplyPasswords(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "users.d"), 0o755))
	path := filepath.Join(dir, passwordsDropIn)
	env := map[string]string{"PMM_CLICKHOUSE_PASSWORD": "clickhouse", "PMM_CLICKHOUSE_DATASOURCE_PASSWORD": "grafana"}
	getenv := func(name string) string { return env[name] }

	require.NoError(t, applyPasswordsAt(dir, getenv))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	// The hashes of users.xml for the default passwords.
	assert.Contains(t, string(b), "<default>\n            <password_sha256_hex replace=\"replace\">"+
		"7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3</password_sha256_hex>")
	assert.Contains(t, string(b), "<grafana>\n            <password_sha256_hex replace=\"replace\">"+
		"cace491b69555e8d0f77747d47ae54e31ce4cc322fe51a7bdcf64402f3676ebf</password_sha256_hex>")
	assert.NotContains(t, string(b), ">clickhouse<", "only hashes are written")

	// A user that users.xml does not have is left alone.
	env = map[string]string{"PMM_CLICKHOUSE_USER": "qan", "PMM_CLICKHOUSE_PASSWORD": "x", "PMM_CLICKHOUSE_DATASOURCE_PASSWORD": "y"}
	require.NoError(t, applyPasswordsAt(dir, getenv))
	b, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "<default>")
	assert.Contains(t, string(b), "<grafana>")

	// Without the variables, users.xml applies again.
	env = nil
	require.NoError(t, applyPasswordsAt(dir, getenv))
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, applyPasswordsAt(dir, getenv))
}
