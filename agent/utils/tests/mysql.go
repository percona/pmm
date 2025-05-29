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
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

const (
	maxIdleConns = 10
	maxOpenConns = 10
)

// GetTestMySQLDSN returns DNS for MySQL test database.
func GetTestMySQLDSN(tb testing.TB) string {
	tb.Helper()

	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}

	cfg := mysql.NewConfig()
	cfg.User = "root"
	cfg.Passwd = "root-password"
	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"
	cfg.DBName = "world"

	// MultiStatements must not be used as it enables SQL injections in Actions
	cfg.MultiStatements = false

	// required for reform
	cfg.ClientFoundRows = true
	cfg.ParseTime = true

	return cfg.FormatDSN()
}

// OpenTestMySQL opens connection to MySQL test database.
func OpenTestMySQL(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("mysql", GetTestMySQLDSN(tb))
	require.NoError(tb, err)

	db.SetMaxIdleConns(maxIdleConns)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetConnMaxLifetime(0)

	waitForTestDataLoad(tb, db)

	// to make Actions tests more stable
	_, err = db.Exec(`ANALYZE /* pmm-agent-tests:OpenTestMySQL */ TABLE city`)
	require.NoError(tb, err)

	return db
}
