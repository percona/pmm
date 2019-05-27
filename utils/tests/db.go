// pmm-agent
// Copyright (C) 2018 Percona LLC
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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

// OpenTestMySQL opens connection to MySQL test database.
func OpenTestMySQL(tb testing.TB) *sql.DB {
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

	// required for reform
	cfg.ClientFoundRows = true
	cfg.ParseTime = true

	dsn := cfg.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		db.SetMaxIdleConns(10)
		db.SetMaxOpenConns(10)
		db.SetConnMaxLifetime(0)

		// Wait until MySQL is running up to 30 seconds.
		for i := 0; i < 30; i++ {
			if err = db.Ping(); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
	}
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
	err := db.QueryRow(`SHOW GLOBAL VARIABLES WHERE Variable_name = 'version'`).Scan(&varName, &version)
	require.NoError(tb, err)
	mm := regexp.MustCompile(`^\d+\.\d+`).FindString(version)

	var comment string
	err = db.QueryRow(`SHOW GLOBAL VARIABLES WHERE Variable_name = 'version_comment'`).Scan(&varName, &comment)
	require.NoError(tb, err)
	var vendor MySQLVendor
	switch {
	case strings.Contains(comment, "Percona"):
		vendor = PerconaMySQL
	case strings.Contains(comment, "mariadb"):
		vendor = MariaDBMySQL
	default:
		vendor = OracleMySQL
	}

	tb.Logf("version = %q (mm = %q), version_comment = %q (vendor = %q)", version, mm, comment, vendor)
	return mm, vendor
}
