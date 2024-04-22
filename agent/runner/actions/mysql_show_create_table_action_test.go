// Copyright (C) 2024 Percona LLC
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

package actions

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/version"
	"github.com/percona/pmm/api/agentpb"
)

func TestMySQLShowCreateTable(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	sqlDB := tests.OpenTestMySQL(t)
	t.Cleanup(func() { sqlDB.Close() }) //nolint:errcheck

	q := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf)).WithTag(queryTag)
	ctx := context.Background()
	mySQLVersion, mySQLVendor, _ := version.GetMySQLVersion(ctx, q)
	t.Logf("version = %q, vendor = %q", mySQLVersion, mySQLVendor)

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "city",
		}
		a := NewMySQLShowCreateTableAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		var expected string
		switch {
		case mySQLVersion.String() == "8.0":
			// https://dev.mysql.com/doc/relnotes/mysql/8.0/en/news-8-0-19.html
			// Display width specification for integer data types was deprecated in MySQL 8.0.17,
			// and now statements that include data type definitions in their output no longer
			// show the display width for integer types [...]
			expected = strings.TrimSpace(`
CREATE TABLE "city" (
  "ID" int NOT NULL AUTO_INCREMENT,
  "Name" char(35) NOT NULL DEFAULT '',
  "CountryCode" char(3) NOT NULL DEFAULT '',
  "District" char(20) NOT NULL DEFAULT '',
  "Population" int NOT NULL DEFAULT '0',
  PRIMARY KEY ("ID"),
  KEY "CountryCode" ("CountryCode"),
  CONSTRAINT "city_ibfk_1" FOREIGN KEY ("CountryCode") REFERENCES "country" ("Code")
) ENGINE=InnoDB AUTO_INCREMENT=4080 DEFAULT CHARSET=latin1
			`)
		case mySQLVendor == version.MariaDBVendor && mySQLVersion.Float() > 10.2:
			// `DEFAULT 0` for Population
			expected = strings.TrimSpace(`
CREATE TABLE "city" (
  "ID" int(11) NOT NULL AUTO_INCREMENT,
  "Name" char(35) NOT NULL DEFAULT '',
  "CountryCode" char(3) NOT NULL DEFAULT '',
  "District" char(20) NOT NULL DEFAULT '',
  "Population" int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY ("ID"),
  KEY "CountryCode" ("CountryCode"),
  CONSTRAINT "city_ibfk_1" FOREIGN KEY ("CountryCode") REFERENCES "country" ("Code")
) ENGINE=InnoDB AUTO_INCREMENT=4080 DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci
			`)
		case mySQLVendor == version.MariaDBVendor && mySQLVersion.Float() < 10.3:
			// `DEFAULT 0` for Population
			expected = strings.TrimSpace(`
CREATE TABLE "city" (
  "ID" int(11) NOT NULL AUTO_INCREMENT,
  "Name" char(35) NOT NULL DEFAULT '',
  "CountryCode" char(3) NOT NULL DEFAULT '',
  "District" char(20) NOT NULL DEFAULT '',
  "Population" int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY ("ID"),
  KEY "CountryCode" ("CountryCode"),
  CONSTRAINT "city_ibfk_1" FOREIGN KEY ("CountryCode") REFERENCES "country" ("Code")
) ENGINE=InnoDB AUTO_INCREMENT=4080 DEFAULT CHARSET=latin1
			`)
		default:
			// `DEFAULT '0'` for Population
			expected = strings.TrimSpace(`
CREATE TABLE "city" (
  "ID" int(11) NOT NULL AUTO_INCREMENT,
  "Name" char(35) NOT NULL DEFAULT '',
  "CountryCode" char(3) NOT NULL DEFAULT '',
  "District" char(20) NOT NULL DEFAULT '',
  "Population" int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY ("ID"),
  KEY "CountryCode" ("CountryCode"),
  CONSTRAINT "city_ibfk_1" FOREIGN KEY ("CountryCode") REFERENCES "country" ("Code")
) ENGINE=InnoDB AUTO_INCREMENT=4080 DEFAULT CHARSET=latin1
			`)
		}

		assert.Equal(t, expected, string(b))
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "no_such_table",
		}
		a := NewMySQLShowCreateTableAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, "Error 1146 (42S02): Table 'world.no_such_table' doesn't exist")
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLShowCreateTableParams{
			Dsn:   dsn,
			Table: `city"; DROP TABLE city; --`,
		}
		a := NewMySQLShowCreateTableAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		expected := "Error 1146 (42S02): Table 'world.city; DROP TABLE city; --' doesn't exist"
		assert.EqualError(t, err, expected)

		var count int
		err = q.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
