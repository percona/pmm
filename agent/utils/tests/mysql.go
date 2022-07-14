// Copyright 2019 Percona LLC
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
	"regexp"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
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

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(0)

	waitForFixtures(tb, db)

	// to make Actions tests more stable
	_, err = db.Exec(`ANALYZE /* pmm-agent-tests:OpenTestMySQL */ TABLE city`)
	require.NoError(tb, err)

	return db
}

// MySQLVendor represents MySQL vendor (Oracle, Percona).
type MySQLVendor string

// MySQL vendors.
const (
	OracleMySQL  MySQLVendor = "oracle"
	PerconaMySQL MySQLVendor = "percona"
	MariaDBMySQL MySQLVendor = "mariadb"
)

// MySQLVersion returns MAJOR.MINOR MySQL version (e.g. "5.6", "8.0", etc.) and vendor.
func MySQLVersion(tb testing.TB, db *sql.DB) (string, MySQLVendor) {
	tb.Helper()

	var varName, version string
	err := db.QueryRow(`SHOW /* pmm-agent-tests:MySQLVersion */ GLOBAL VARIABLES WHERE Variable_name = 'version'`).Scan(&varName, &version)
	require.NoError(tb, err)
	mm := regexp.MustCompile(`^\d+\.\d+`).FindString(version)

	var comment string
	err = db.QueryRow(`SHOW /* pmm-agent-tests:MySQLVersion */ GLOBAL VARIABLES WHERE Variable_name = 'version_comment'`).Scan(&varName, &comment)
	require.NoError(tb, err)

	// SHOW /* pmm-agent-tests:MySQLVersion */ GLOBAL VARIABLES WHERE Variable_name = 'version_comment';
	// +-----------------+----------------+
	// | Variable_name   | Value          |
	// +-----------------+----------------+
	// | version_comment | MariaDB Server |
	// +-----------------+----------------+
	// convert comment to lowercase because not all MySQL flavors & versions return the same capitalization
	// but make it only in the switch-case to preserve the original value for debugging
	var vendor MySQLVendor
	switch {
	case strings.Contains(strings.ToLower(comment), "percona"):
		vendor = PerconaMySQL
	case strings.Contains(strings.ToLower(comment), "mariadb"):
		vendor = MariaDBMySQL
	default:
		vendor = OracleMySQL
	}

	tb.Logf("version = %q (mm = %q), version_comment = %q (vendor = %q)", version, mm, comment, vendor)
	return mm, vendor
}
