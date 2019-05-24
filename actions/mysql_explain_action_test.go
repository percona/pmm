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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestMySQLExplain(t *testing.T) {
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	mySQLVersion := tests.MySQLVersion(t, db)

	const query = "SELECT * FROM `city`"

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
		switch mySQLVersion {
		case "5.6":
			expected = strings.TrimSpace(`
id |select_type |table |type |possible_keys |key  |key_len |ref  |rows |Extra
1  |SIMPLE      |city  |ALL  |NULL          |NULL |NULL    |NULL |\d+  |NULL
			`)
		case "5.7", "8.0":
			expected = strings.TrimSpace(`
id |select_type |table |partitions |type |possible_keys |key  |key_len |ref  |rows |filtered |Extra
1  |SIMPLE      |city  |NULL       |ALL  |NULL          |NULL |NULL    |NULL |\d+  |100.00   |NULL
			`)
		default:
			t.Log("Unhandled version, assuming dummy results.")
			expected = "TODO"
		}

		actual := strings.TrimSpace(string(b))
		assert.Regexp(t, expected, actual)
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

		var actual map[string]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		actual = actual["query_block"].(map[string]interface{})
		actualTable := actual["table"].(map[string]interface{})

		switch mySQLVersion {
		case "5.6":
			assert.Equal(t, 1.0, actual["select_id"])
			assert.Equal(t, "city", actualTable["table_name"])
		case "5.7", "8.0":
			assert.Equal(t, 1.0, actual["select_id"])
			assert.Equal(t, "city", actualTable["table_name"])
			assert.Equal(t, []interface{}{"ID", "Name", "CountryCode", "District", "Population"}, actualTable["used_columns"])
		default:
			assert.Failf(t, "Unhandled version %q.", mySQLVersion)
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
