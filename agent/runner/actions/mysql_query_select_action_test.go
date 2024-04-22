// Copyright (C) 2024 Percona LLC
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
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestMySQLQuerySelect(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLQuerySelectParams{
			Dsn:   dsn,
			Query: "COUNT(*) AS count FROM mysql.user WHERE plugin NOT IN ('caching_sha2_password')",
		}
		a := NewMySQLQuerySelectAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.Len(t, b, 13)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.InDelta(t, 1, len(data), 0)
		assert.Contains(t, data[0], "count")
	})

	t.Run("Binary", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLQuerySelectParams{
			Dsn:   dsn,
			Query: `x'0001feff' AS bytes`,
		}
		a := NewMySQLQuerySelectAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.Len(t, b, 17)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.InDelta(t, 1, len(data), 0)
		expected := map[string]interface{}{
			"bytes": "\x00\x01\xfe\xff",
		}
		assert.Equal(t, expected, data[0])
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_MySQLQuerySelectParams{
			Dsn:   dsn,
			Query: "* FROM city; DROP TABLE city; --",
		}
		a := NewMySQLQuerySelectAction("", 0, params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		expected := "Error 1064 \\(42000\\): You have an error in your SQL syntax; check the manual that corresponds " +
			"to your (MySQL|MariaDB) server version for the right syntax to use near 'DROP TABLE city; --' at line 1"
		require.Error(t, err)
		assert.Regexp(t, expected, err.Error())
		assert.Nil(t, b)

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
