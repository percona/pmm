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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestNewMongoExplain(t *testing.T) {
	database := "testdb"
	ctx := context.TODO()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	t.Cleanup(func() {
		defer client.Disconnect(ctx)              //nolint:errcheck
		defer client.Database(database).Drop(ctx) //nolint:errcheck
	})

	t.Run("Find collections query", func(t *testing.T) {
		query := `{
			"ns": "config.collections",
			"op": "query",
			"command": {
			  "find": "collections",
			  "filter": {
				"_id": {
				  "$regex": "^admin.",
				  "$options": "i"
				}
			  },
			  "lsid": {
				"id": {
				  "$binary": {
					"base64": "DSrmgdR2Sme3QAC5+9pTNA==",
					"subType": "04"
				  }
				}
			  },
			  "$db": "config"
			}
		  }`
		runExplain(t, ctx, prepareParams(t, query))
	})

	// TODO: More queries/commands
}

func prepareParams(t *testing.T, query string) *agentpb.StartActionRequest_MongoDBExplainParams {
	t.Helper()

	return &agentpb.StartActionRequest_MongoDBExplainParams{
		Dsn:   tests.GetTestMongoDBDSN(t),
		Query: query,
	}
}

func runExplain(t *testing.T, ctx context.Context, params *agentpb.StartActionRequest_MongoDBExplainParams) {
	id := fmt.Sprintf("%d", rand.Uint64())
	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)

	res, err := ex.Run(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, string(res))
}
