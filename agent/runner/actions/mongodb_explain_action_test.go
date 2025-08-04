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
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/tests"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/version"
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

	t.Run("Count", func(t *testing.T) {
		query := `{
			"ns": "testdb.collection",
			"op": "command",
			"command": {
			  "count": "collection",
			  "query": {
				"a": {
				  "$numberDouble": "5.0"
				},
				"b": {
				  "$numberDouble": "5.0"
				}
			  },
			  "$db": "testdb"
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

	t.Run("Distinct", func(t *testing.T) {
		query := `{
			"ns": "testdb.inventory",
			"op": "command",
			"command": {
			  "distinct": "inventory",
			  "key": "dept",
			  "query": {}
			  },
			  "$db": "testdb"
			}
		  }`
		runExplain(ctx, t, prepareParams(t, query))
	})

	t.Run("Insert - not supported", func(t *testing.T) {
		query := `{
			"ns": "testdb.listingsAndReviews",
			"op": "insert",
			"command": {
			  "insert": "listingsAndReviews",
			  "ordered": true,
			  "$db": "testdb"
			}
		  }`
		runExplainExpectError(ctx, t, prepareParams(t, query))
	})

	t.Run("Drop - not supported", func(t *testing.T) {
		query := `{
			"ns": "testdb.listingsAndReviews",
			"op": "command",
			"command": {
			  "drop": "listingsAndReviews",
			  "$db": "testdb"
			}
		  }`
		runExplainExpectError(ctx, t, prepareParams(t, query))
	})

	t.Run("PMM-12451", func(t *testing.T) {
		// Query from customer to prevent wrong date/time, timestamp parsing in future and prevent regression.
		query := `{"ns":"testdb.testDoc","op":"query","command":{"find":"testDoc","filter":{"$and":[{"c23":{"$ne":""}},{"c23":{"$ne":null},"delete":{"$ne":true}},{"$and":[{"c23":"985662747"},{"c15":{"$gte":{"$date":"2023-09-19T22:00:00.000Z"}}},{"c15":{"$lte":{"$date":"2023-10-20T21:59:59.000Z"}}},{"c8":{"$in":["X1118630710X","X1118630720X","X1118630730X","X1118630740X","X1118630750X","X1118630760X"]},"c22":{"$in":["X1118630710X","X1118630710XA","X1118630710XB","X1118630710XC","X1118630710XD","X1118630710XE"]},"c34":{"$in":["X1118630710X","X1118630710Y","X1118630710Z","X1118630710U","X1118630710V","X1118630710W"]}},{"c2":"xxxxxxx"},{"c29":{"$in":["X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X","X1118630710X"]}}]}]},"lsid":{"id":{"$binary":{"base64":"n/f5RI2jTTCoyt0y+8D9Cw==","subType":"04"}}},"$db":"testdb"}}`
		runExplain(ctx, t, prepareParams(t, query))
	})
}

func prepareParams(t *testing.T, query string) *agentv1.StartActionRequest_MongoDBExplainParams {
	t.Helper()

	return &agentv1.StartActionRequest_MongoDBExplainParams{
		Dsn:   tests.GetTestMongoDBDSN(t),
		Query: query,
	}
}

func runExplain(ctx context.Context, t *testing.T, params *agentv1.StartActionRequest_MongoDBExplainParams) {
	t.Helper()

	big, err := rand.Int(rand.Reader, big.NewInt(27))
	require.NoError(t, err)
	id := strconv.FormatUint(big.Uint64(), 10)
	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)
	res, err := ex.Run(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, string(res))
}

func runExplainExpectError(ctx context.Context, t *testing.T, params *agentv1.StartActionRequest_MongoDBExplainParams) {
	t.Helper()

	big, err := rand.Int(rand.Reader, big.NewInt(27))
	require.NoError(t, err)
	id := strconv.FormatUint(big.Uint64(), 10)
	ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
	require.NoError(t, err)
	_, err = ex.Run(ctx)
	require.Error(t, err)
}

func TestMongoDBExplain(t *testing.T) {
	database := "test"
	collection := "test_col"
	id := "abcd1234"
	ctx := context.TODO()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	defer client.Database(database).Drop(ctx) //nolint:errcheck

	err := prepareData(ctx, client, database, collection)
	require.NoError(t, err)

	t.Run("Valid MongoDB query", func(t *testing.T) {
		params := &agentv1.StartActionRequest_MongoDBExplainParams{
			Dsn:   tests.GetTestMongoDBDSN(t),
			Query: `{"ns":"test.coll","op":"query","query":{"k":{"$lte":{"$numberInt":"1"}}}}`,
		}

		ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
		require.NoError(t, err)

		res, err := ex.Run(ctx)
		assert.NoError(t, err)

		want := map[string]interface{}{
			"indexFilterSet": false,
			"namespace":      "test.coll",
			"parsedQuery": map[string]interface{}{
				"k": map[string]interface{}{"$lte": map[string]interface{}{"$numberInt": "1"}},
			},
			"rejectedPlans": []interface{}{},
			"winningPlan":   map[string]interface{}{"stage": "EOF"},
		}
		mongoDBVersion, _ := tests.MongoDBVersion(t, client)

		switch {
		case mongoDBVersion.Major < 5:
			want["plannerVersion"] = map[string]interface{}{"$numberInt": "1"}
		case mongoDBVersion.Major < 8:
			want["maxIndexedAndSolutionsReached"] = false
			want["maxIndexedOrSolutionsReached"] = false
			want["maxScansToExplodeReached"] = false
			if mongoDBVersion.Major == 7 {
				want["optimizationTimeMillis"] = map[string]interface{}{"$numberInt": "0"}
			}
		case mongoDBVersion.Major == 8:
			want["maxIndexedAndSolutionsReached"] = false
			want["maxIndexedOrSolutionsReached"] = false
			want["maxScansToExplodeReached"] = false
			want["optimizationTimeMillis"] = map[string]interface{}{"$numberInt": "0"}
			want["winningPlan"] = map[string]interface{}{"stage": "EOF", "isCached": false}
			want["prunedSimilarIndexes"] = false
		}

		explainM := make(map[string]interface{})
		err = json.Unmarshal(res, &explainM)
		assert.NoError(t, err)
		queryPlanner, ok := explainM["queryPlanner"]
		assert.True(t, ok)
		assert.NotEmpty(t, queryPlanner)
		assert.Equal(t, want, queryPlanner)
	})
}

// These tests are based on v3 tests. The previous ones are inherited from PMM 1/Toolkit.
func TestNewMongoDBExplain(t *testing.T) {
	database := "sbtest"
	id := "abcd1234"
	ctx := context.TODO()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	defer client.Database(database).Drop(ctx) //nolint:errcheck

	_, err := client.Database(database).Collection("people").InsertOne(ctx, bson.M{"last_name": "Brannigan", "first_name": "Zapp"})
	require.NoError(t, err)

	_, err = client.Database(database).Collection("orders").InsertOne(ctx, bson.M{"status": "A", "amount": 123.45})
	require.NoError(t, err)

	testFiles := []struct {
		in         string
		minVersion string
	}{
		{
			in: "distinct.json",
		},
		{
			in: "aggregate.json",
		},
		{
			in: "count.json",
		},
		{
			in: "find_and_modify.json",
		},
	}
	for _, tf := range testFiles {
		t.Run(tf.in, func(t *testing.T) {
			query, err := os.ReadFile(filepath.Join("testdata/", filepath.Clean(tf.in)))
			assert.NoError(t, err)
			params := &agentv1.StartActionRequest_MongoDBExplainParams{
				Dsn:   tests.GetTestMongoDBDSN(t),
				Query: string(query),
			}

			ex, err := NewMongoDBExplainAction(id, 0, params, os.TempDir())
			require.NoError(t, err)

			res, err := ex.Run(ctx)
			assert.NoError(t, err)

			explainM := make(map[string]interface{})
			err = json.Unmarshal(res, &explainM)
			assert.NoError(t, err)

			// Just test not empty because different versions and environments return different
			// explain results
			assert.NotEmpty(t, explainM)
		})
	}
}

func prepareData(ctx context.Context, client *mongo.Client, database, collection string) error {
	limit := int64(100)
	count, _ := client.Database(database).Collection(collection).CountDocuments(ctx, nil)

	if count < limit {
		for i := int64(0); i < limit; i++ {
			doc := primitive.M{"f1": i, "f2": fmt.Sprintf("text_%5d", limit-i)}
			if _, err := client.Database(database).Collection(collection).InsertOne(ctx, doc); err != nil {
				return err
			}
		}
	}

	return nil
}

func lessThan(minVersionStr, currentVersion string) (bool, error) {
	v, err := version.Parse(currentVersion)
	if err != nil {
		return false, err
	}
	v.Rest = ""

	// Check if version meets the conditions
	minVersion, err := version.Parse(minVersionStr)
	if err != nil {
		return false, err
	}
	return minVersion.Less(v), nil
}
