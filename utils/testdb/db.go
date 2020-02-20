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

package testdb

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

// Open recreates testing PostgreSQL database and returns an open connection to it.
func Open(tb testing.TB, setupFixtures models.SetupFixturesMode) *sql.DB {
	tb.Helper()

	const username, password = "postgres", ""
	const testDatabase = "pmm-managed-dev"

	db, err := models.OpenDB("127.0.0.1:5432", "", username, password)
	require.NoError(tb, err)

	_, err = db.Exec(`DROP DATABASE IF EXISTS "` + testDatabase + `"`)
	require.NoError(tb, err)
	_, err = db.Exec(`CREATE DATABASE "` + testDatabase + `"`)
	require.NoError(tb, err)

	err = db.Close()
	require.NoError(tb, err)

	db, err = models.OpenDB("127.0.0.1:5432", testDatabase, username, password)
	require.NoError(tb, err)
	_, err = models.SetupDB(db, &models.SetupDBParams{
		// Uncomment to see all setup queries:
		// Logf: tb.Logf,

		Username:      username,
		Password:      password,
		SetupFixtures: setupFixtures,
	})
	require.NoError(tb, err)
	return db
}
