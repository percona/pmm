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

	"github.com/percona/pmm/agent/utils/tests"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func TestShowTableStatus(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentv1.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   dsn,
			Table: "city",
		}
		a := NewMySQLShowTableStatusAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var actual [][]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		// Check some columns names
		assert.Contains(t, actual[0], "Name")
		assert.Contains(t, actual[0], "Engine")
		assert.Contains(t, actual[0], "Version")
		assert.Contains(t, actual[0], "Row_format")
		assert.Contains(t, actual[0], "Rows")
		assert.Contains(t, actual[0], "Avg_row_length")
		assert.Contains(t, actual[0], "Data_length")
		assert.Contains(t, actual[0], "Max_data_length")
		assert.Contains(t, actual[0], "Index_length")
		assert.Contains(t, actual[0], "Data_free")
		assert.Contains(t, actual[0], "Auto_increment")
		assert.Contains(t, actual[0], "Create_time")
		assert.Contains(t, actual[0], "Update_time")
		assert.Contains(t, actual[0], "Check_time")
		assert.Contains(t, actual[0], "Collation")
		assert.Contains(t, actual[0], "Checksum")
		assert.Contains(t, actual[0], "Create_options")
		assert.Contains(t, actual[0], "Comment")

		// Checks some stable values
		assert.Equal(t, "city", actual[1][0])           // Name
		assert.Equal(t, "InnoDB", actual[1][1])         // Engine
		assert.InEpsilon(t, 10.0, actual[1][2], 0.0001) // Version
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		params := &agentv1.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   dsn,
			Table: "no_such_table",
		}
		a := NewMySQLShowTableStatusAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `table "no_such_table" not found`)
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		t.Parallel()
		params := &agentv1.StartActionRequest_MySQLShowTableStatusParams{
			Dsn:   dsn,
			Table: `city"; DROP TABLE city; --`,
		}
		a := NewMySQLShowTableStatusAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		assert.EqualError(t, err, `table "city; DROP TABLE city; --" not found`)

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
