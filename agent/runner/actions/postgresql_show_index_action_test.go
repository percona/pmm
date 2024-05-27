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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestPostgreSQLShowIndex(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestPostgreSQLDSN(t)
	db := tests.OpenTestPostgreSQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_PostgreSQLShowIndexParams{
			Dsn:   dsn,
			Table: "city",
		}
		a, err := NewPostgreSQLShowIndexAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var actual [][]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		assert.Equal(t, [][]interface{}{
			{"schemaname", "tablename", "indexname", "tablespace", "indexdef"},
			{"public", "city", "city_pkey", nil, "CREATE UNIQUE INDEX city_pkey ON public.city USING btree (id)"},
		}, actual)
	})

	t.Run("WithSchemaName", func(t *testing.T) {
		t.Parallel()

		params := &agentpb.StartActionRequest_PostgreSQLShowIndexParams{
			Dsn:   dsn,
			Table: "public.city",
		}
		a, err := NewPostgreSQLShowIndexAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		t.Logf("Full JSON:\n%s", b)

		var actual [][]interface{}
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		assert.Equal(t, [][]interface{}{
			{"schemaname", "tablename", "indexname", "tablespace", "indexdef"},
			{"public", "city", "city_pkey", nil, "CREATE UNIQUE INDEX city_pkey ON public.city USING btree (id)"},
		}, actual)
	})
}
