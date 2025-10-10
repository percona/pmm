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

package actions

import (
	"context"
	"encoding/json"
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

func TestMySQLShowIndex(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	sqlDB := tests.OpenTestMySQL(t)
	t.Cleanup(func() { sqlDB.Close() }) //nolint:errcheck

	q := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf)).WithTag(queryTag)
	ctx := context.Background()
	mySQLVersion, mySQLVendor, _ := version.GetMySQLVersion(ctx, q)

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLShowIndexParams{
			Dsn:   dsn,
			Table: "city",
		}
		a := NewMySQLShowIndexAction("", 0, params)
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
		case mySQLVendor == version.MariaDBVendor && mySQLVersion.Float() >= 10.5:
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment", "Ignored",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", float64(0), "PRIMARY", float64(1), "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", "", "NO"}, actual[1])
			assert.Equal(t, []interface{}{"city", float64(1), "CountryCode", float64(1), "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", "", "NO"}, actual[2])

		case mySQLVersion.String() == "5.6" || mySQLVendor == version.MariaDBVendor:
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", float64(0), "PRIMARY", float64(1), "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[1])
			assert.Equal(t, []interface{}{"city", float64(1), "CountryCode", float64(1), "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[2])

		case mySQLVersion.String() == "5.7":
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", float64(0), "PRIMARY", float64(1), "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[1])
			assert.Equal(t, []interface{}{"city", float64(1), "CountryCode", float64(1), "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", ""}, actual[2])

		case mySQLVersion.String() == "8.0":
			assert.Equal(t, []interface{}{
				"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name", "Collation", "Cardinality",
				"Sub_part", "Packed", "Null", "Index_type", "Comment", "Index_comment", "Visible", "Expression",
			}, actual[0])
			assert.Equal(t, []interface{}{"city", float64(0), "PRIMARY", float64(1), "ID", "A", "CARDINALITY", nil, nil, "", "BTREE", "", "", "YES", nil}, actual[1])
			assert.Equal(t, []interface{}{"city", float64(1), "CountryCode", float64(1), "CountryCode", "A", "CARDINALITY", nil, nil, "", "BTREE", "", "", "YES", nil}, actual[2])

		default:
			t.Fatal("Unhandled version.")
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLShowIndexParams{
			Dsn:   dsn,
			Table: "no_such_table",
		}
		a := NewMySQLShowIndexAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `Error 1146 (42S02): Table 'world.no_such_table' doesn't exist`)
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLShowIndexParams{
			Dsn:   dsn,
			Table: `city"; DROP TABLE city; --`,
		}
		a := NewMySQLShowIndexAction("", 0, params)
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
