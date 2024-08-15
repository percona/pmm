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
	"crypto/rand"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestQueryExplain(t *testing.T) {
	database := "testdb"
	ctx := context.TODO()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	t.Cleanup(func() {
		defer client.Disconnect(ctx)              //nolint:errcheck
		defer client.Database(database).Drop(ctx) //nolint:errcheck
	})

	t.Run("Find", func(t *testing.T) {
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
			  "$db": "config"
			}
		  }`
		runExplain(ctx, t, prepareParams(t, query))
	})

	t.Run("Count with aggregate", func(t *testing.T) {
		query := `{
			"ns": "testdb.collection",
			"op": "command",
			"command": {
			  "aggregate": "collection",
			  "pipeline": [
				{
				  "$group": {
					"_id": null,
					"count": {
					  "$sum": {
						"$numberDouble": "1.0"
					  }
					}
				  }
				},
				{
				  "$project": {
					"_id": {
					  "$numberDouble": "0.0"
					}
				  }
				}
			  ],
			  "cursor": {},
			  "$db": "testdb"
			}
		  }`
		runExplain(ctx, t, prepareParams(t, query))
	})

	t.Run("Update", func(t *testing.T) {
		query := `{
			"ns": "testdb.inventory",
			"op": "update",
			"command": {
			  "q": {
				"item": "paper"
			  },
			  "u": {
				"$set": {
				  "size.uom": "cm",
				  "status": "P"
				},
				"$currentDate": {
				  "lastModified": true
				}
			  },
			  "multi": false,
			  "upsert": false
			}
		  }`
		runExplain(ctx, t, prepareParams(t, query))
	})

	t.Run("Remove", func(t *testing.T) {
		query := `{
			"ns": "testdb.inventory",
			"op": "remove",
			"command": {
			  "q": {
				"_id": {
				  "id": {
					"$binary": {
					  "base64": "vN9ImShsRBaCIFJ23YkysA==",
					  "subType": "04"
					}
				  },
				  "uid": {
					"$binary": {
					  "base64": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=",
					  "subType": "00"
					}
				  }
				}
			  },
			  "limit": {
				"$numberInt": "0"
			  }
			}
		  }`
		runExplain(ctx, t, prepareParams(t, query))
	})

	t.Run("Insert - no support", func(t *testing.T) {
		query := `{
			"ns": "testdb.inventory",
			"op": "command",
			"command": {
			  "distinct": "inventory",
			  "key": "dept",
			  "query": {},
			  "lsid": {
				"id": {
				  "$binary": {
					"base64": "54EXxw1pRPqx/+4fCiJLJw==",
					"subType": "04"
				  }
				}
			  },
			  "$db": "testdb"
			}
		  }`
		runExplainExpectError(ctx, t, prepareParams(t, query))
	})

	t.Run("Drop - no support", func(t *testing.T) {
		query := `{
			"ns": "testdb.listingsAndReviews",
			"op": "command",
			"command": {
			  "drop": "listingsAndReviews",
			  "lsid": {
				"id": {
				  "$binary": {
					"base64": "54DCxw1pRPqx/+4fCiJLJw==",
					"subType": "04"
				  }
				}
			  },
			  "$db": "testdb"
			}
		  }`
		runExplainExpectError(ctx, t, prepareParams(t, query))
	})

	t.Run("PMM-12451", func(t *testing.T) {
		// Mongo driver 1.6.0 is able to parse, 1.6.1 not.
		// Query from customer to prevent regression and wrong driver bump in future.
		query := `{"ns":"testdb.testDoc","op":"query","command":{"find":"testDoc","filter":{"$and":[{"c23":{"$ne":""}},{"c23":{"$ne":null},"delete":{"$ne":true}},{"$and":[{"c23":"985662747"},{"c15":{"$gte":{"$date":"2023-09-19T22:00:00.000Z"}}},{"c15":{"$lte":{"$date":"2023-10-20T21:59:59.000Z"}}},{"c8":{"$in":["X1118630710X","X1118630720X","X1118630730X","X1118630740X","X1118630750X","X1118630760X"]},"c22":{"$in":["X1118630710X","X1118630710XA","X1118630710XB","X1118630710XC","X1118630710XD","X1118630710XE"]},"c34":{"$in":["X1118630710X","X1118630710Y","X1118630710Z","X1118630710U","X1118630710V","X1118630710W"]}},{"c2":"xxxxxxx"},{"c29":{"$in":["X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X"]}}]}]},"lsid":{"id":{"$binary":{"base64":"n/f5RI2jTTCoyt0y+8D9Cw==","subType":"04"}}},"$db":"testdb"}}` //nolint:lll
		runExplain(ctx, t, prepareParams(t, query))
	})
}

func prepareParams(t *testing.T, query string) *agentpb.StartActionRequest_MongoDBExplainParams {
	t.Helper()

	return &agentpb.StartActionRequest_MongoDBExplainParams{
		Dsn:   tests.GetTestMongoDBDSN(t),
		Query: query,
	}
}

func runExplain(ctx context.Context, t *testing.T, params *agentpb.StartActionRequest_MongoDBExplainParams) {
	t.Helper()

	big, err := rand.Int(rand.Reader, big.NewInt(27))
	require.NoError(t, err)
	id := strconv.FormatUint(big.Uint64(), 10)
	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)
	res, err := ex.Run(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, string(res))
}

func runExplainExpectError(ctx context.Context, t *testing.T, params *agentpb.StartActionRequest_MongoDBExplainParams) {
	t.Helper()

	big, err := rand.Int(rand.Reader, big.NewInt(27))
	require.NoError(t, err)
	id := strconv.FormatUint(big.Uint64(), 10)
	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)
	_, err = ex.Run(ctx)
	assert.Error(t, err)
}
