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

func TestMySQLQueryShow(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMySQLDSN(t)
	db := tests.OpenTestMySQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_MySQLQueryShowParams{
			Dsn:   dsn,
			Query: "VARIABLES",
		}
		a := NewMySQLQueryShowAction("", time.Second, params)
		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout())
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 15000, len(b))
		assert.LessOrEqual(t, len(b), 28000)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.LessOrEqual(t, 400, len(data))
		assert.LessOrEqual(t, len(data), 800)

		var found int
		for _, m := range data {
			value := m["Value"]
			switch m["Variable_name"].(string) {
			case "auto_generate_certs":
				assert.Equal(t, "ON", value)
				found++
			case "auto_increment_increment":
				assert.Equal(t, "1", value)
				found++
			}
		}
		assert.GreaterOrEqual(t, found, 1)
	})
}
