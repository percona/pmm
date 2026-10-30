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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

const (
	username     = "postgres"
	testDatabase = "pmm-managed-dev"

	// Variable postgresPasswordFile defines the file where PMM Server versions that gave the postgres
	// superuser a password stored it.
	postgresPasswordFile = "/srv/.postgres_password" //nolint:gosec
	// Variable postgresSocket defines the Unix socket of the embedded PostgreSQL of PMM Server.
	postgresSocket = "/run/postgresql/.s.PGSQL.5432"
)

// postgresPassword returns the PostgreSQL password, reading it from the password file if available.
func postgresPassword() string {
	data, err := os.ReadFile(postgresPasswordFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// postgresAddress returns the address to connect to PostgreSQL at: the Unix socket of the embedded
// PostgreSQL of a PMM Server dev container, whose postgres superuser has no password since it is
// trusted there, or the TCP address of the PostgreSQL that the tests are run with otherwise.
func postgresAddress() string {
	if postgresPassword() != "" {
		return "127.0.0.1:5432"
	}
	_, err := os.Stat(postgresSocket)
	if err != nil {
		return "127.0.0.1:5432"
	}
	return filepath.Dir(postgresSocket)
}

// Open recreates testing PostgreSQL database and returns an open connection to it.
func Open(tb testing.TB, setupFixtures models.SetupFixturesMode, migrationVersion *int) *sql.DB {
	tb.Helper()

	setupParams := models.SetupDBParams{
		Address:  postgresAddress(),
		Username: username,
		Password: postgresPassword(),
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
		Name:             newName(11), //nolint:mnd
		Username:         username,
		Password:         postgresPassword(),
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
