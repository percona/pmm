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

package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/mongo_fix"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/version"
)

// GetTestMongoDBDSN returns DNS for MongoDB test database.
func GetTestMongoDBDSN(tb testing.TB) string {
	tb.Helper()
	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}
	return "mongodb://root:root-password@localhost:27017/admin"
}

// GetTestMongoDBReplicatedDSN returns DNS for replicated MongoDB test database.
func GetTestMongoDBReplicatedDSN(tb testing.TB) string {
	tb.Helper()
	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}
	return "mongodb://127.0.0.1:27020,127.0.0.1:27021/admin?replicaSet=rs0"
}

// GetTestMongoDBWithSSLDSN returns DNS template and files for MongoDB test database with ssl.
func GetTestMongoDBWithSSLDSN(tb testing.TB, pathToRoot string) (string, *agentv1.TextFiles) {
	tb.Helper()

	pathToRoot = filepath.Clean(pathToRoot)
	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}

	dsn := "mongodb://localhost:27018/admin/?tls=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}"

	caFile, err := os.ReadFile(filepath.Join(pathToRoot, "utils/tests/testdata/", "mongodb/", "ca.crt")) //nolint:gosec
	require.NoError(tb, err)

	certificateKey, err := os.ReadFile(filepath.Join(pathToRoot, "utils/tests/testdata/", "mongodb/", "client.pem")) //nolint:gosec
	require.NoError(tb, err)

	return dsn, &agentv1.TextFiles{
		Files: map[string]string{
			"caFilePlaceholder":             string(caFile),
			"certificateKeyFilePlaceholder": string(certificateKey),
		},
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
	}
}

// GetTestMongoDBReplicatedWithSSLDSN returns DNS template and files for replicated MongoDB test database with ssl.
func GetTestMongoDBReplicatedWithSSLDSN(tb testing.TB, pathToRoot string) (string, *agentv1.TextFiles) {
	tb.Helper()

	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}

	dsn := "mongodb://localhost:27022,localhost:27023/admin/?tls=true&tlsCaFile=" +
		"{{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}"

	caFile, err := os.ReadFile(filepath.Join(filepath.Clean(pathToRoot), "utils/tests/testdata/", "mongodb/", "ca.crt"))
	require.NoError(tb, err)

	certificateKey, err := os.ReadFile(filepath.Join(filepath.Clean(pathToRoot), "utils/tests/testdata/", "mongodb/", "client.pem"))
	require.NoError(tb, err)

	return dsn, &agentv1.TextFiles{
		Files: map[string]string{
			"caFilePlaceholder":             string(caFile),
			"certificateKeyFilePlaceholder": string(certificateKey),
		},
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
	}
}

// OpenTestMongoDB opens connection to MongoDB test database.
func OpenTestMongoDB(tb testing.TB, dsn string) *mongo.Client {
	tb.Helper()

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		require.NoError(tb, err)
	}

	client, err := mongo.Connect(context.Background(), opts)
	require.NoError(tb, err)

	require.NoError(tb, client.Ping(context.Background(), nil))

	return client
}

// MongoDBVersion returns Mongo DB version.
func MongoDBVersion(tb testing.TB, client *mongo.Client) (*version.Parsed, bool) {
	tb.Helper()

	res := client.Database("admin").RunCommand(context.Background(), primitive.M{"buildInfo": 1})
	if res.Err() != nil {
		tb.Fatalf("Cannot get buildInfo: %s", res.Err())
	}
	bi := struct {
		Version      string `bson:"version"`
		PSMDBVersion string `bson:"psmdbVersion"`
	}{}
	if err := res.Decode(&bi); err != nil {
		tb.Fatalf("Cannot decode buildInfo response: %s", err)
	}
	parsed, err := version.Parse(bi.Version)
	if err != nil {
		tb.Fatalf("Cannot parse version: %s", err)
	}

	var isPercona bool
	if bi.PSMDBVersion != "" {
		isPercona = true
	}

	return parsed, isPercona
}
