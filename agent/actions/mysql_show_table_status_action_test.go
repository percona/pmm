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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestShowTableStatus(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck

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

		var actual map[string]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		// Check some columns names
		assert.Contains(t, actual, "Name")
		assert.Contains(t, actual, "Engine")
		assert.Contains(t, actual, "Version")
		assert.Contains(t, actual, "Row_format")
		assert.Contains(t, actual, "Rows")
		assert.Contains(t, actual, "Avg_row_length")
		assert.Contains(t, actual, "Data_length")
		assert.Contains(t, actual, "Max_data_length")
		assert.Contains(t, actual, "Index_length")
		assert.Contains(t, actual, "Data_free")
		assert.Contains(t, actual, "Auto_increment")
		assert.Contains(t, actual, "Create_time")
		assert.Contains(t, actual, "Update_time")
		assert.Contains(t, actual, "Check_time")
		assert.Contains(t, actual, "Collation")
		assert.Contains(t, actual, "Checksum")
		assert.Contains(t, actual, "Create_options")
		assert.Contains(t, actual, "Comment")

		// Checks some stable values
		assert.Equal(t, "city", actual["Name"])
		assert.Equal(t, "InnoDB", actual["Engine"])
		assert.Equal(t, 10.0, actual["Version"])
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
