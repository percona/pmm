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

package main

import (
	"os"
	"path/filepath"
	"testing"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClickhouseDSN(t *testing.T) {
	t.Parallel()

	// Characters that would end the userinfo or the host if they were not escaped.
	const password = "p@ss:w/rd%#?"

	t.Run("escapes the password", func(t *testing.T) {
		t.Parallel()

		dsn, err := clickhouseDSN("default", password, "", "127.0.0.1:9000", "pmm")
		require.NoError(t, err)
		opts, err := clickhouse.ParseDSN(dsn)
		require.NoError(t, err)
		assert.Equal(t, "default", opts.Auth.Username)
		assert.Equal(t, password, opts.Auth.Password)
		assert.Equal(t, "pmm", opts.Auth.Database)
		assert.Equal(t, []string{"127.0.0.1:9000"}, opts.Addr)
	})

	t.Run("reads the password from the file in place of the value", func(t *testing.T) {
		t.Parallel()

		file := filepath.Join(t.TempDir(), "clickhouse-password")
		require.NoError(t, os.WriteFile(file, []byte(password+"\n"), 0o600))

		dsn, err := clickhouseDSN("default", "clickhouse", file, "127.0.0.1:9000", "pmm")
		require.NoError(t, err)
		opts, err := clickhouse.ParseDSN(dsn)
		require.NoError(t, err)
		assert.Equal(t, password, opts.Auth.Password)
	})

	t.Run("refuses a missing or empty file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		_, err := clickhouseDSN("default", "clickhouse", filepath.Join(dir, "missing"), "127.0.0.1:9000", "pmm")
		require.ErrorIs(t, err, os.ErrNotExist)

		empty := filepath.Join(dir, "empty")
		require.NoError(t, os.WriteFile(empty, []byte("\n"), 0o600))
		_, err = clickhouseDSN("default", "clickhouse", empty, "127.0.0.1:9000", "pmm")
		require.ErrorContains(t, err, "is empty")
	})
}
