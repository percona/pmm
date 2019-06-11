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

func TestShowTableStatus(t *testing.T) {
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	mySQLVersion, _ := tests.MySQLVersion(t, db)

	_, err := db.Exec("ANALYZE TABLE city")
	require.NoError(t, err)

	t.Run("Default", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   "root:root-password@tcp(127.0.0.1:3306)/world",
			Table: "city",
		}
		a := NewMySQLShowTableStatusAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var actual [][]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		const createTime = "2019-06-10 12:04:29"
		switch mySQLVersion {
		case "5.6":
			assert.Equal(t, []interface{}{
				"Name", "Engine", "Version", "Row_format", "Rows", "Avg_row_length", "Data_length", "Max_data_length",
				"Index_length", "Data_free", "Auto_increment", "Create_time", "Update_time", "Check_time", "Collation",
				"Checksum", "Create_options", "Comment",
			}, actual[0])
			actual[1][11] = createTime
			assert.Equal(t, []interface{}{
				"city", "InnoDB", 10.0, "Compact", 4188.0, 97.0, 409600.0, 0.0,
				131072.0, 0.0, 4080.0, "2019-06-10 12:04:29", nil, nil, "latin1_swedish_ci",
				nil, "", "",
			}, actual[1])

		case "5.7", "8.0", "10.2":
			assert.Equal(t, []interface{}{
				"Name", "Engine", "Version", "Row_format", "Rows", "Avg_row_length", "Data_length", "Max_data_length",
				"Index_length", "Data_free", "Auto_increment", "Create_time", "Update_time", "Check_time", "Collation",
				"Checksum", "Create_options", "Comment",
			}, actual[0])
			actual[1][11] = createTime
			assert.Equal(t, []interface{}{
				"city", "InnoDB", 10.0, "Dynamic", 4188.0, 97.0, 409600.0, 0.0,
				131072.0, 0.0, 4080.0, "2019-06-10 12:04:29", nil, nil, "latin1_swedish_ci",
				nil, "", "",
			}, actual[1])

		case "10.3", "10.4":
			assert.Equal(t, []interface{}{
				"Name", "Engine", "Version", "Row_format", "Rows", "Avg_row_length", "Data_length", "Max_data_length",
				"Index_length", "Data_free", "Auto_increment", "Create_time", "Update_time", "Check_time", "Collation",
				"Checksum", "Create_options", "Comment", "Max_index_length", "Temporary",
			}, actual[0])
			actual[1][11] = createTime
			assert.Equal(t, []interface{}{
				"city", "InnoDB", 10.0, "Dynamic", 4188.0, 97.0, 409600.0, 0.0,
				131072.0, 0.0, 4080.0, "2019-06-10 12:04:29", nil, nil, "latin1_swedish_ci",
				nil, "", "", 0.0, "N",
			}, actual[1])

		default:
			t.Fatal("Unhandled version.")
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   "root:root-password@tcp(127.0.0.1:3306)/world",
			Table: "no_such_table",
		}
		a := NewMySQLShowTableStatusAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `table "no_such_table" not found`)
	})
}
