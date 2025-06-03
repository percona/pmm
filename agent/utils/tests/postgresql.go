// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"database/sql"
	"net"
	"net/url"
	"strconv"
	"testing"

	_ "github.com/lib/pq" // register SQL driver
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/version"
)

const (
	defaultPostgresPort = 5432
	pgMaxIdleConns      = 10
)

// GetTestPostgreSQLDSN returns DNS for PostgreSQL test database.
func GetTestPostgreSQLDSN(tb testing.TB) string {
	tb.Helper()

	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}
	q := make(url.Values)
	q.Set("sslmode", "disable") // TODO: make it configurable

	u := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort("localhost", strconv.Itoa(defaultPostgresPort)),
		Path:     "pmm-agent",
		User:     url.UserPassword("pmm-agent", "pmm-agent-password"),
		RawQuery: q.Encode(),
	}

	return u.String()
}

// OpenTestPostgreSQL opens connection to PostgreSQL test database.
func OpenTestPostgreSQL(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("postgres", GetTestPostgreSQLDSN(tb))
	require.NoError(tb, err)

	db.SetMaxIdleConns(pgMaxIdleConns)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(0)

	waitForTestDataLoad(tb, db)

	return db
}

// PostgreSQLVersion returns major PostgreSQL version (e.g. "9.6", "10", etc.).
func PostgreSQLVersion(tb testing.TB, db *sql.DB) string {
	tb.Helper()

	var v string
	err := db.QueryRow("SELECT /* pmm-agent-tests:PostgreSQLVersion */ version()").Scan(&v)
	require.NoError(tb, err)

	m := version.ParsePostgreSQLVersion(v)
	require.NotEmpty(tb, m, "Failed to parse PostgreSQL version from %q.", v)
	tb.Logf("version = %q (m = %q)", v, m)
	return m
}
