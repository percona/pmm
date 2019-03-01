// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package tests

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

// OpenTestDB recreates testing database and returns an open connection to it.
func OpenTestDB(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := models.OpenDB("", "pmm-managed", "pmm-managed", tb.Logf)
	require.NoError(tb, err)

	const testDatabase = "pmm-managed-dev"
	_, err = db.Exec("DROP DATABASE `" + testDatabase + "`")
	require.NoError(tb, err)
	_, err = db.Exec("CREATE DATABASE `" + testDatabase + "`")
	require.NoError(tb, err)

	err = db.Close()
	require.NoError(tb, err)

	db, err = models.OpenDB(testDatabase, "pmm-managed", "pmm-managed", tb.Logf)
	require.NoError(tb, err)
	return db
}

// OpenTestPostgresDB recreates testing postgres database and returns an open connection to it.
func OpenTestPostgresDB(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := models.OpenPostgresDB("", "pmm-managed", "pmm-managed", tb.Logf)
	require.NoError(tb, err)

	const testDatabase = "pmm-managed-dev"
	_, err = db.Exec(`DROP DATABASE IF EXISTS "` + testDatabase + `"`)
	require.NoError(tb, err)
	_, err = db.Exec(`CREATE DATABASE "` + testDatabase + `"`)
	require.NoError(tb, err)

	err = db.Close()
	require.NoError(tb, err)

	db, err = models.OpenPostgresDB(testDatabase, "pmm-managed", "pmm-managed", tb.Logf)
	require.NoError(tb, err)
	return db
}
