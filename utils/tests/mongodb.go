// pmm-agent
// Copyright 2019 Percona LLC
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

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetTestMongoDBDSN returns DNS for MongoDB test database.
func GetTestMongoDBDSN(tb testing.TB) string {
	tb.Helper()

	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}

	return "mongodb://root:root-password@127.0.0.1:27017/admin"
}

// OpenTestMongoDB opens connection to MongoDB test database.
func OpenTestMongoDB(tb testing.TB) *mongo.Client {
	tb.Helper()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(GetTestMongoDBDSN(tb)))
	require.NoError(tb, err)

	require.NoError(tb, client.Ping(context.Background(), nil))

	return client
}
