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

// Package testdb provides test DB utils.
package testdb

import (
	"context"
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

const (
	username, password = "postgres", ""
	testDatabase       = "pmm-managed-dev"
)

// Open recreates testing PostgreSQL database and returns an open connection to it.
func Open(tb testing.TB, setupFixtures models.SetupFixturesMode, migrationVersion *int) *sql.DB {
	tb.Helper()

	setupParams := models.SetupDBParams{
		Address:  "127.0.0.1:5432",
		Username: username,
		Password: password,
	}

	db, err := models.OpenDB(setupParams)
	require.NoError(tb, err)

	_, err = db.Exec(`DROP DATABASE IF EXISTS "` + testDatabase + `"`)
	require.NoError(tb, err)
	_, err = db.Exec(`CREATE DATABASE "` + testDatabase + `"`)
	require.NoError(tb, err)

	err = db.Close()
	require.NoError(tb, err)

	setupParams.Name = testDatabase
	db, err = models.OpenDB(setupParams)
	require.NoError(tb, err)
	SetupDB(tb, db, setupFixtures, migrationVersion)

	tb.Cleanup(func() {
		_ = db.Close() // let tests call db.Close() themselves if they want to
	})

	return db
}

// SetupDB runs PostgreSQL database migrations and optionally adds initial data for testing DB.
// Please use Open method to recreate DB for each test if you don't need to control migrations.
func SetupDB(tb testing.TB, db *sql.DB, setupFixtures models.SetupFixturesMode, migrationVersion *int) {
	tb.Helper()
	ctx := context.TODO()
	params := models.SetupDBParams{
		// Uncomment to see all setup queries:
		// Logf: tb.Logf,
		Address:          models.DefaultPostgreSQLAddr,
		Name:             newName(11),
		Username:         username,
		Password:         password,
		SetupFixtures:    setupFixtures,
		MigrationVersion: migrationVersion,
	}

	_, err := models.SetupDB(ctx, db, params)
	require.NoError(tb, err)
}

func newName(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	const alp = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = alp[r.Intn(len(alp))]
	}
	return string(b)
}
