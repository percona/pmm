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

func TestPostgreSQLQuerySelect(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestPostgreSQLDSN(t)
	db := tests.OpenTestPostgreSQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLQuerySelectParams{
			Dsn:   dsn,
			Query: "* FROM pg_extension",
		}
		a, err := NewPostgreSQLQuerySelectAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 130, len(b))
		assert.LessOrEqual(t, len(b), 300)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.LessOrEqual(t, 1, len(data))
		assert.LessOrEqual(t, len(data), 3)
		delete(data[0], "oid")
		expected := map[string]interface{}{
			"extname":        "plpgsql",
			"extowner":       "10",
			"extnamespace":   "11",
			"extrelocatable": false,
			"extversion":     "1.0",
			"extconfig":      nil,
			"extcondition":   nil,
		}
		assert.Equal(t, expected, data[0])
	})

	t.Run("Binary", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLQuerySelectParams{
			Dsn:   dsn,
			Query: `'\x0001feff'::bytea AS bytes`,
		}
		a, err := NewPostgreSQLQuerySelectAction("", 0, params, os.TempDir())
		require.NoError(t, err)

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
		params := &agentpb.StartActionRequest_PostgreSQLQuerySelectParams{
			Dsn:   dsn,
			Query: "* FROM city; DROP TABLE city CASCADE; --",
		}
		a, err := NewPostgreSQLQuerySelectAction("", 0, params, os.TempDir())
		assert.EqualError(t, err, "query contains ';'")
		assert.Nil(t, a)
	})
}
