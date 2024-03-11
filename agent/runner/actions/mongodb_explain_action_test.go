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
	"fmt"
	"os"
	"path/filepath"
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

		ex := NewMongoDBExplainAction(id, 0, params, os.TempDir())
		res, err := ex.Run(ctx)
		assert.Nil(t, err)

		want := map[string]interface{}{
			"indexFilterSet": false,
			"namespace":      "test.coll",
			"parsedQuery": map[string]interface{}{
				"k": map[string]interface{}{"$lte": map[string]interface{}{"$numberInt": "1"}},
			},
			"plannerVersion": map[string]interface{}{"$numberInt": "1"},
			"rejectedPlans":  []interface{}{},
			"winningPlan":    map[string]interface{}{"stage": "EOF"},
		}

		explainM := make(map[string]interface{})
		err = json.Unmarshal(res, &explainM)
		assert.Nil(t, err)
		queryPlanner, ok := explainM["queryPlanner"]
		assert.Equal(t, ok, true)
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
			in:         "aggregate.json",
			minVersion: "3.4.0",
		},
		{
			in: "count.json",
		},
		{
			in: "find_and_modify.json",
		},
	}
	mongoDBVersion := tests.MongoDBVersion(t, client)
	for _, tf := range testFiles {
		// Not all MongoDB versions allow explaining all commands
		if tf.minVersion != "" {
			c, err := lessThan(tf.minVersion, mongoDBVersion)
			require.NoError(t, err)
			if c {
				continue
			}
		}

		t.Run(tf.in, func(t *testing.T) {
			query, err := os.ReadFile(filepath.Join("testdata/", filepath.Clean(tf.in)))
			assert.NoError(t, err)
			params := &agentv1.StartActionRequest_MongoDBExplainParams{
				Dsn:   tests.GetTestMongoDBDSN(t),
				Query: string(query),
			}

			ex := NewMongoDBExplainAction(id, 0, params, os.TempDir())
			res, err := ex.Run(ctx)
			assert.NoError(t, err)

			explainM := make(map[string]interface{})
			err = json.Unmarshal(res, &explainM)
			assert.Nil(t, err)

			// Just test not empty because different versions and environments return different
			// explain results
			assert.NotEmpty(t, explainM)
		})
	}
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
