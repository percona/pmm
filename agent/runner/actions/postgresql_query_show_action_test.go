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
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestPostgreSQLQueryShow(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestPostgreSQLDSN(t)
	db := tests.OpenTestPostgreSQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLQueryShowParams{
			Dsn: dsn,
		}
		a, err := NewPostgreSQLQueryShowAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 22000, len(b))
		assert.LessOrEqual(t, len(b), 37989)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.LessOrEqual(t, 200, len(data))
		assert.LessOrEqual(t, len(data), 399)

		var found int
		for _, m := range data {
			setting := m["setting"]
			description := m["description"]
			switch m["name"].(string) {
			case "allow_system_table_mods":
				assert.Equal(t, "off", setting)
				assert.Equal(t, "Allows modifications of the structure of system tables.", description)
				found++
			case "autovacuum_freeze_max_age":
				assert.Equal(t, "200000000", setting)
				assert.Equal(t, "Age at which to autovacuum a table to prevent transaction ID wraparound.", description)
				found++
			}
		}
		assert.Equal(t, 2, found)
	})
}
