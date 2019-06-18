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
	"encoding/json"
	"testing"
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestShowIndex(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	mySQLVersion, mySQLVendor := tests.MySQLVersion(t, db)

	t.Run("Default", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowIndexParams{
			Dsn:   dsn,
			Table: "city",
		}
		a := NewMySQLShowIndexAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var actual [][]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 3)

		// cardinality sometimes changes between runs; fix it to the most common value for that version
		switch {
		case mySQLVersion == "5.6" || mySQLVendor == tests.MariaDBMySQL:
			actual[2][6] = "465"
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", "0", "PRIMARY", "1", "ID", "A", "4188", nil, nil, "", "BTREE", "", ""}, actual[1])
			assert.Equal(t, []interface{}{"city", "1", "CountryCode", "1", "CountryCode", "A", "465", nil, nil, "", "BTREE", "", ""}, actual[2])

		case mySQLVersion == "5.7":
			actual[2][6] = "232"
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", "0", "PRIMARY", "1", "ID", "A", "4188", nil, nil, "", "BTREE", "", ""}, actual[1])
			assert.Equal(t, []interface{}{"city", "1", "CountryCode", "1", "CountryCode", "A", "232", nil, nil, "", "BTREE", "", ""}, actual[2])

		case mySQLVersion == "8.0":
			actual[2][6] = "232"
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment", "Visible", "Expression",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", "0", "PRIMARY", "1", "ID", "A", "4188", nil, nil, "", "BTREE", "", "", "YES", nil}, actual[1])
			assert.Equal(t, []interface{}{"city", "1", "CountryCode", "1", "CountryCode", "A", "232", nil, nil, "", "BTREE", "", "", "YES", nil}, actual[2])

		default:
			t.Fatal("Unhandled version.")
		}
	})

	t.Run("Error", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowIndexParams{
			Dsn:   dsn,
			Table: "no_such_table",
		}
		a := NewMySQLShowIndexAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `Error 1146: Table 'world.no_such_table' doesn't exist`)
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowIndexParams{
			Dsn:   dsn,
			Table: `city"; DROP TABLE city; --`,
		}
		a := NewMySQLShowIndexAction("", params)
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
