// pmm-agent
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

func TestMySQLShowIndex(t *testing.T) {
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

		// cardinality changes between runs
		actual[1][6] = "CARDINALITY"
		actual[2][6] = "CARDINALITY"

		switch {
		case mySQLVersion == "5.6" || mySQLVendor == tests.MariaDBMySQL:
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", "0", "PRIMARY", "1", "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[1])
			assert.Equal(t, []interface{}{"city", "1", "CountryCode", "1", "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[2])

		case mySQLVersion == "5.7":
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", "0", "PRIMARY", "1", "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[1])
			assert.Equal(t, []interface{}{"city", "1", "CountryCode", "1", "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[2])

		case mySQLVersion == "8.0":
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment", "Visible", "Expression",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", "0", "PRIMARY", "1", "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", "", "YES", nil}, actual[1])
			assert.Equal(t, []interface{}{"city", "1", "CountryCode", "1", "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", "", "YES", nil}, actual[2])

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
