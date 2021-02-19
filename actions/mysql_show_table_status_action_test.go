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

func TestShowTableStatus(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	mySQLVersion, _ := tests.MySQLVersion(t, db)

	t.Run("Default", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   dsn,
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

		case "5.7", "8.0":
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

		case "10.2":
			assert.Equal(t, []interface{}{
				"Name", "Engine", "Version", "Row_format", "Rows", "Avg_row_length", "Data_length", "Max_data_length",
				"Index_length", "Data_free", "Auto_increment", "Create_time", "Update_time", "Check_time", "Collation",
				"Checksum", "Create_options", "Comment",
			}, actual[0])
			actual[1][11] = createTime
			assert.Equal(t, []interface{}{
				"city", "InnoDB", 10.0, "Dynamic", 4079.0, 100.0, 409600.0, 0.0,
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
				"city", "InnoDB", 10.0, "Dynamic", 4079.0, 100.0, 409600.0, 0.0,
				131072.0, 0.0, 4080.0, "2019-06-10 12:04:29", nil, nil, "latin1_swedish_ci",
				nil, "", "", 0.0, "N",
			}, actual[1])

		default:
			t.Fatal("Unhandled version.")
		}
	})

	t.Run("Error", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   dsn,
			Table: "no_such_table",
		}
		a := NewMySQLShowTableStatusAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `table "no_such_table" not found`)
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		params := &agentpb.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   dsn,
			Table: `city"; DROP TABLE city; --`,
		}
		a := NewMySQLShowTableStatusAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `table "city\"; DROP TABLE city; --" not found`)

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
