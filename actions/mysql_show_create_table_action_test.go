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

package actions

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestShowCreateTable(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	_, mySQLVendor := tests.MySQLVersion(t, db)

	t.Run("Default", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "city",
		}
		a := NewMySQLShowCreateTableAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		var expected string
		switch mySQLVendor {
		case tests.MariaDBMySQL:
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
		params := &agentpb.StartActionRequest_MySQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "no_such_table",
		}
		a := NewMySQLShowCreateTableAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, "Error 1146: Table 'world.no_such_table' doesn't exist")
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowCreateTableParams{
			Dsn:   dsn,
			Table: `city"; DROP TABLE city; --`,
		}
		a := NewMySQLShowCreateTableAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		expected := "Error 1146: Table 'world.city\"; DROP TABLE city; --' doesn't exist"
		assert.EqualError(t, err, expected)

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
