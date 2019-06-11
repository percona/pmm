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
	"strings"
	"testing"
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestMySQLExplain(t *testing.T) {
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	mySQLVersion, mySQLVendor := tests.MySQLVersion(t, db)

	_, err := db.Exec("ANALYZE TABLE city")
	require.NoError(t, err)

	const query = "SELECT * FROM city ORDER BY Population"

	t.Run("Default", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          "root:root-password@tcp(127.0.0.1:3306)/world",
			Query:        query,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
		}
		a := NewMySQLExplainAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		var expected string
		switch {
		case mySQLVersion == "5.6" || mySQLVendor == tests.MariaDBMySQL:
			expected = strings.TrimSpace(`
id |select_type |table |type |possible_keys |key  |key_len |ref  |rows |Extra
1  |SIMPLE      |city  |ALL  |NULL          |NULL |NULL    |NULL |4188 |Using filesort
			`)
		default:
			expected = strings.TrimSpace(`
id |select_type |table |partitions |type |possible_keys |key  |key_len |ref  |rows |filtered |Extra
1  |SIMPLE      |city  |NULL       |ALL  |NULL          |NULL |NULL    |NULL |4188 |100.00   |Using filesort
			`)
		}
		actual := strings.TrimSpace(string(b))
		assert.Equal(t, expected, actual)
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          "root:root-password@tcp(127.0.0.1:3306)/world",
			Query:        query,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON,
		}
		a := NewMySQLExplainAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)
		m, err := objx.FromJSON(string(b))
		require.NoError(t, err)

		assert.Equal(t, 1.0, m.Get("query_block.select_id").Float64())

		var table map[string]interface{}
		switch mySQLVendor {
		case tests.MariaDBMySQL:
			table = m.Get("query_block.read_sorted_file.filesort.table").MSI()
		default:
			table = m.Get("query_block.ordering_operation.table").MSI()
		}

		require.NotNil(t, table)
		assert.Equal(t, "city", table["table_name"])
		if mySQLVersion != "5.6" && mySQLVendor != tests.MariaDBMySQL {
			assert.Equal(t, []interface{}{"ID", "Name", "CountryCode", "District", "Population"}, table["used_columns"])
		}

		if mySQLVendor != tests.MariaDBMySQL {
			require.Len(t, m.Get("warnings").InterSlice(), 1)
			assert.Equal(t, 1003.0, m.Get("warnings[0].Code").Float64())
			assert.Equal(t, "Note", m.Get("warnings[0].Level").String())
			assert.Contains(t, m.Get("warnings[0].Message").String(), "/* select#1 */")
		}
	})

	t.Run("TraditionalJSON", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          "root:root-password@tcp(127.0.0.1:3306)/world",
			Query:        query,
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON,
		}
		a := NewMySQLExplainAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var actual [][]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		switch {
		case mySQLVersion == "5.6" || mySQLVendor == tests.MariaDBMySQL:
			assert.Equal(t, []interface{}{
				"id", "select_type", "table",
				"type", "possible_keys", "key", "key_len", "ref", "rows", "Extra",
			}, actual[0])
			assert.Equal(t, []interface{}{"1", "SIMPLE", "city", "ALL", nil, nil, nil, nil, "4188", "Using filesort"}, actual[1])
		default:
			assert.Equal(t, []interface{}{
				"id", "select_type", "table", "partitions",
				"type", "possible_keys", "key", "key_len", "ref", "rows", "filtered", "Extra",
			}, actual[0])
			assert.Equal(t, []interface{}{"1", "SIMPLE", "city", nil, "ALL", nil, nil, nil, nil, "4188", "100.00", "Using filesort"}, actual[1])
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLExplainParams{
			Dsn:          "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/world",
			OutputFormat: agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT,
		}
		a := NewMySQLExplainAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		require.Error(t, err)
		assert.Regexp(t, `Error 1045: Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`, err.Error())
	})
}
