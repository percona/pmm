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
	"os"
	"testing"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestNewMongoExplain(t *testing.T) {
	database := "test"
	collection := "test_col"
	id := "abcd1234"
	ctx := context.TODO()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	t.Cleanup(func() { defer client.Disconnect(ctx) }) //nolint:errcheck
	defer client.Database(database).Drop(ctx)          //nolint:errcheck

	err := prepareData(ctx, client, database, collection)
	require.NoError(t, err)

	params := &agentpb.StartActionRequest_MongoDBExplainParams{
		Dsn: tests.GetTestMongoDBDSN(t),
		Query: `
		{
			"ns": "config.version",
			"op": "query",
			"command": {
			  "find": "version",
			  "filter": {},
			  "limit": {
				"$numberLong": "1"
			  },
			  "singleBatch": true,
			  "lsid": {
				"id": {
				  "$binary": {
					"base64": "YfWLS+S/RsWGvmk9Y5kfFg==",
					"subType": "04"
				  }
				}
			  },
			  "$db": "config"
			}
		  }`,
	}

	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)

	res, err := ex.Run(ctx)
	assert.NoError(t, err)
	assert.Empty(t, string(res))
}

func prepareData(ctx context.Context, client *mongo.Client, database, collection string) error {
	max := int64(100)
	count, _ := client.Database(database).Collection(collection).CountDocuments(ctx, nil)

	if count < max {
		for i := int64(0); i < max; i++ {
			doc := primitive.M{"f1": i, "f2": fmt.Sprintf("text_%5d", max-i)}
			if _, err := client.Database(database).Collection(collection).InsertOne(ctx, doc); err != nil {
				return err
			}
		}
	}

	return nil
}
