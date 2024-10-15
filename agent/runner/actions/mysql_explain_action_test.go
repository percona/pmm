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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/version"
	"github.com/percona/pmm/api/agentpb"
)

func TestMySQLExplain(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	sqlDB := tests.OpenTestMySQL(t)
	t.Cleanup(func() { sqlDB.Close() }) //nolint:errcheck

	q := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf)).WithTag(queryTag)
	ctx := context.Background()
	mySQLVersion, mySQLVendor, _ := version.GetMySQLVersion(ctx, q)

	const query = "SELECT * FROM city ORDER BY Population"

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          dsn,
			Query:        query,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
		}
		a, err := NewMySQLExplainAction("", time.Second, params)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		var er explainResponse
		err = json.Unmarshal(b, &er)
		assert.NoError(t, err)

		actual := strings.TrimSpace(string(er.ExplainResult))
		// Check some columns names
		assert.Contains(t, actual, "id |select_type |table")
		assert.Contains(t, actual, "|type |possible_keys |key  |key_len |ref  |rows")

		// Checks some stable values
		assert.Contains(t, actual, "1  |SIMPLE      |city")
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          dsn,
			Query:        query,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON,
		}
		a, err := NewMySQLExplainAction("", time.Second, params)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var er explainResponse
		err = json.Unmarshal(b, &er)
		assert.NoError(t, err)

		m, err := objx.FromJSON(string(er.ExplainResult))
		require.NoError(t, err)

		assert.Equal(t, 1, m.Get("query_block.select_id").Int())

		var table map[string]interface{}
		if mySQLVendor == version.MariaDBVendor {
			table = m.Get("query_block.read_sorted_file.filesort.table").MSI()
		} else {
			table = m.Get("query_block.ordering_operation.table").MSI()
		}

		require.NotNil(t, table)
		assert.Equal(t, "city", table["table_name"])
		if mySQLVersion.String() != "5.6" && mySQLVendor != version.MariaDBVendor {
			assert.Equal(t, []interface{}{"ID", "Name", "CountryCode", "District", "Population"}, table["used_columns"])
		}

		if mySQLVendor != version.MariaDBVendor {
			require.Len(t, m.Get("warnings").InterSlice(), 1)
			assert.Equal(t, 1003, m.Get("warnings[0].Code").Int())
			assert.Equal(t, "Note", m.Get("warnings[0].Level").String())
			assert.Contains(t, m.Get("warnings[0].Message").String(), "/* select#1 */")
		}
	})

	t.Run("TraditionalJSON", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          dsn,
			Query:        query,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON,
		}
		a, err := NewMySQLExplainAction("", time.Second, params)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var er explainResponse
		err = json.Unmarshal(b, &er)
		assert.NoError(t, err)

		var actual [][]interface{}
		err = json.Unmarshal(er.ExplainResult, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		// Check some columns names
		assert.Contains(t, actual[0], "id")
		assert.Contains(t, actual[0], "select_type")
		assert.Contains(t, actual[0], "table")
		assert.Contains(t, actual[0], "type")
		assert.Contains(t, actual[0], "possible_keys")
		assert.Contains(t, actual[0], "key")
		assert.Contains(t, actual[0], "key_len")
		assert.Contains(t, actual[0], "ref")
		assert.Contains(t, actual[0], "rows")
		assert.Contains(t, actual[0], "Extra")

		// Checks some stable values
		assert.Equal(t, float64(1), actual[1][0]) // id
		assert.Equal(t, "SIMPLE", actual[1][1])   // select_type
		assert.Equal(t, "city", actual[1][2])     // table
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/world",
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
		}
		a, err := NewMySQLExplainAction("", time.Second, params)
		assert.ErrorContains(t, err, `Query to EXPLAIN is empty`)
		assert.Nil(t, a)
	})

	t.Run("DML Query Insert", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          dsn,
			Query:        `INSERT INTO city (Name) VALUES ('Rosario')`,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
		}
		a, err := NewMySQLExplainAction("", time.Second, params)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
		defer cancel()

		resp, err := a.Run(ctx)
		require.NoError(t, err)
		var er explainResponse
		err = json.Unmarshal(resp, &er)
		assert.NoError(t, err)
		assert.Equal(t, er.IsDMLQuery, true)
		assert.Equal(t, er.Query, `SELECT * FROM city  WHERE Name='Rosario'`)
	})

	t.Run("Query longer than max-query-length", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          dsn,
			Query:        `INSERT INTO city (Name)...`,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
		}
		a, err := NewMySQLExplainAction("", time.Second, params)
		assert.ErrorContains(t, err, "EXPLAIN failed because the query exceeded max length and got trimmed. Set max-query-length to a larger value.")
		assert.Nil(t, a)
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		t.Parallel()

		checkCity := func(t *testing.T) {
			t.Helper()

			var count int
			err := q.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 4079, count)
		}

		t.Run("Drop", func(t *testing.T) {
			t.Parallel()

			params := &agentpb.StartActionRequest_MySQLExplainParams{
				Dsn:          dsn,
				Query:        `SELECT 1; DROP TABLE city; --`,
				OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
			}
			a, err := NewMySQLExplainAction("", time.Second, params)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
			defer cancel()

			_, err = a.Run(ctx)
			expected := "Error 1064 \\(42000\\): You have an error in your SQL syntax; check the manual that corresponds " +
				"to your (MySQL|MariaDB) server version for the right syntax to use near 'DROP TABLE city; --' at line 1"
			require.Error(t, err)
			assert.Regexp(t, expected, err.Error())
			checkCity(t)
		})

		t.Run("Delete", func(t *testing.T) {
			t.Parallel()

			params := &agentpb.StartActionRequest_MySQLExplainParams{
				Dsn:          dsn,
				Query:        `DELETE FROM city`,
				OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
			}
			a, err := NewMySQLExplainAction("", time.Second, params)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
			defer cancel()

			_, err = a.Run(ctx)
			require.NoError(t, err)
			checkCity(t)
		})

		t.Run("Stored function", func(t *testing.T) {
			t.Parallel()

			check := func(t *testing.T) {
				t.Helper()
				var count int
				err := q.QueryRow("SELECT COUNT(*) FROM test_explain_table").Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			}

			// setup
			func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
				defer cancel()
				conn, err := sqlDB.Conn(ctx)
				require.NoError(t, err)
				defer conn.Close() //nolint:errcheck

				_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS test_explain_table")
				require.NoError(t, err)
				_, err = conn.ExecContext(ctx, "CREATE TABLE test_explain_table(i int)")
				require.NoError(t, err)
				_, err = conn.ExecContext(ctx, "INSERT INTO test_explain_table (i) VALUES (42)")
				require.NoError(t, err)
				_, err = conn.ExecContext(ctx, "DROP FUNCTION IF EXISTS cleanup")
				require.NoError(t, err)
				_, err = conn.ExecContext(ctx, `CREATE FUNCTION cleanup() RETURNS char(50) CHARSET latin1
				DETERMINISTIC
				BEGIN
				delete from world.test_explain_table;
				RETURN 'OK';
				END
				`)
				require.NoError(t, err)
			}(t)

			params := &agentpb.StartActionRequest_MySQLExplainParams{
				Dsn:          dsn,
				Query:        `select * from (select cleanup()) as testclean;`,
				OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
			}
			a, err := NewMySQLExplainAction("", time.Second, params)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
			defer cancel()

			_, err = a.Run(ctx)
			require.NoError(t, err)
			check(t)
		})
	})
}

func TestParseRealTableNameMySQL(t *testing.T) {
	type testCase struct {
		Query    string
		Expected string
	}

	tests := []testCase{
		{"SELECT;", ""},
		{"SELECT `district` FROM `people`;", "`people`"},
		{"SELECT `district` FROM `people`", "`people`"},
		{"SELECT `district` FROM people", "people"},
		{"SELECT name FROM people WHERE city = 'Paris'", "people"},
		{"SELECT name FROM world.people WHERE city = 'Paris'", "world.people"},
		{"SELECT name FROM `world`.`people` WHERE city = 'Paris'", "`world`.`people`"},
		{"SELECT name FROM `world` . `people` WHERE city = 'Paris'", "`world`.`people`"},
		{"SELECT name FROM \"world\".\"people\" WHERE city = 'Paris'", "\"world\".\"people\""},
		{"SELECT name FROM \"world\" . \"people\" WHERE city = 'Paris'", "\"world\".\"people\""},
		{"SELECT name FROM 'world'.'people' WHERE city = 'Paris'", "'world'.'people'"},
		{"SELECT name FROM 'world' . 'people' WHERE city = 'Paris'", "'world'.'people'"},
		{"SELECT name FROM 'world' . \"people\" WHERE city = `Paris`", "'world'.\"people\""},
		{"SELECT DATE(`date`) AS `date` FROM (SELECT MIN(`date`) AS `date`, `player_name` FROM `people` GROUP BY `player_name`) AS t GROUP BY DATE(`date`);", "`people`"},
	}

	for _, test := range tests {
		t.Run(test.Query, func(t *testing.T) {
			assert.Equal(t, test.Expected, parseRealTableName(test.Query))
		})
	}
}
