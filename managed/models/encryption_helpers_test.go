// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package models_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

// agentSecrets holds every column of DefaultAgentEncryptionColumnsV3 for a single agent.
type agentSecrets struct {
	Username      sql.NullString
	Password      sql.NullString
	AgentPassword sql.NullString
	AWS           models.AWSOptions
	Azure         models.AzureOptions
	MongoDB       models.MongoDBOptions
	MySQL         models.MySQLOptions
	PostgreSQL    models.PostgreSQLOptions
}

// TestDefaultAgentEncryptionColumnsRoundTrip guards the invariant documented on
// encryption.Column: every column of DefaultAgentEncryptionColumnsV3 must decrypt back
// to its original value. A column wired with a single handler for both directions
// re-encrypts instead of decrypting and fails here, which is what broke encryption key
// rotation in https://perconadev.atlassian.net/browse/PMM-15188.
func TestDefaultAgentEncryptionColumnsRoundTrip(t *testing.T) {
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "encryption.key"))

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})

	ctx := t.Context()
	insertAgentWithSecrets(ctx, t, sqlDB)
	original := readAgentSecrets(ctx, t, sqlDB)

	e := encryption.New()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return e.EncryptItems(tx, models.DefaultAgentEncryptionColumnsV3)
	})
	require.NoError(t, err)

	// Every column of the set must have been processed, otherwise the round trip below
	// could pass just because both handlers did nothing.
	encrypted := readAgentSecrets(ctx, t, sqlDB)
	assert.NotEqual(t, original.Username, encrypted.Username)
	assert.NotEqual(t, original.Password, encrypted.Password)
	assert.NotEqual(t, original.AgentPassword, encrypted.AgentPassword)
	assert.NotEqual(t, original.AWS, encrypted.AWS)
	assert.NotEqual(t, original.Azure, encrypted.Azure)
	assert.NotEqual(t, original.MongoDB, encrypted.MongoDB)
	assert.NotEqual(t, original.MySQL, encrypted.MySQL)
	assert.NotEqual(t, original.PostgreSQL, encrypted.PostgreSQL)

	err = db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return e.DecryptItems(tx, models.DefaultAgentEncryptionColumnsV3)
	})
	require.NoError(t, err)

	assert.Equal(t, original, readAgentSecrets(ctx, t, sqlDB))
}

// TestEncryptDecryptAgentRoundTrip covers the happy path and the invariant that the caller's
// Agent is left untouched: the *string fields are shared with the caller, so the handlers must
// replace the pointers instead of writing through them.
func TestEncryptDecryptAgentRoundTrip(t *testing.T) {
	agent := models.Agent{
		AgentID:  "/agent_id/1",
		Username: new("username"),
		Password: new("password"),
		MySQLOptions: models.MySQLOptions{
			TLSCert: "mysql-tls-cert",
			TLSKey:  "mysql-tls-key",
		},
	}

	encrypted, err := models.EncryptAgent(agent)
	require.NoError(t, err)
	require.NotNil(t, encrypted.Username)
	assert.NotEqual(t, "username", *encrypted.Username)
	assert.NotEqual(t, "mysql-tls-cert", encrypted.MySQLOptions.TLSCert)

	require.NotNil(t, agent.Username)
	assert.Equal(t, "username", *agent.Username, "input agent must not be mutated")

	decrypted, err := models.DecryptAgent(encrypted)
	require.NoError(t, err)
	require.NotNil(t, decrypted.Username)
	require.NotNil(t, decrypted.Password)
	assert.Equal(t, "username", *decrypted.Username)
	assert.Equal(t, "password", *decrypted.Password)
	assert.Equal(t, "mysql-tls-cert", decrypted.MySQLOptions.TLSCert)
	assert.Equal(t, "mysql-tls-key", decrypted.MySQLOptions.TLSKey)
}

// TestDecryptAgentDoesNotReturnCiphertext guards the fix for
// https://perconadev.atlassian.net/browse/PMM-14979: a value that this node's key cannot
// decrypt must surface as an error, and the undecrypted value must not be handed back to the
// caller. It previously came back as the field value with only a warning logged, so the
// ciphertext reached pmm-agent as a username and produced
// `password authentication failed for user "AQ+rKT/..."`.
func TestDecryptAgentDoesNotReturnCiphertext(t *testing.T) {
	// Valid base64 but not ciphertext produced by this node's key, which is what an HA
	// follower reads when the row was encrypted with another node's key.
	foreignCiphertext := base64.StdEncoding.EncodeToString([]byte("encrypted-with-another-key"))

	agent := models.Agent{
		AgentID:  "/agent_id/1",
		Username: new(foreignCiphertext),
		Password: new(foreignCiphertext),
	}

	decrypted, err := models.DecryptAgent(agent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/agent_id/1", "error should identify the agent")
	assert.Contains(t, err.Error(), "username", "error should identify the field")
	assert.Nil(t, decrypted.Username, "ciphertext must not be returned as the decrypted value")
}

//nolint:dupword
func insertAgentWithSecrets(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	awsOptions, err := json.Marshal(models.AWSOptions{
		AWSAccessKey:            "aws-access-key",
		AWSSecretKey:            "aws-secret-key",
		RDSBasicMetricsDisabled: true,
	})
	require.NoError(t, err)

	azureOptions, err := json.Marshal(models.AzureOptions{
		SubscriptionID: "azure-subscription-id",
		ClientID:       "azure-client-id",
		ClientSecret:   "azure-client-secret",
		TenantID:       "azure-tenant-id",
		ResourceGroup:  "azure-resource-group",
	})
	require.NoError(t, err)

	mongoOptions, err := json.Marshal(models.MongoDBOptions{
		TLSCertificateKey:             "mongo-tls-certificate-key",
		TLSCertificateKeyFilePassword: "mongo-tls-certificate-key-file-password",
		TLSCa:                         "mongo-tls-ca",
	})
	require.NoError(t, err)

	mysqlOptions, err := json.Marshal(models.MySQLOptions{
		TLSCa:   "mysql-tls-ca",
		TLSCert: "mysql-tls-cert",
		TLSKey:  "mysql-tls-key",
	})
	require.NoError(t, err)

	postgresqlOptions, err := json.Marshal(models.PostgreSQLOptions{
		SSLCa:   "postgresql-ssl-ca",
		SSLCert: "postgresql-ssl-cert",
		SSLKey:  "postgresql-ssl-key",
	})
	require.NoError(t, err)

	now := time.Now()
	_, err = db.ExecContext(
		ctx,
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('1', 'generic', 'name', '', '', '', '', $1, $2)",
		now, now,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(
		ctx,
		"INSERT INTO agents (agent_id, agent_type, username, password, agent_password, runs_on_node_id, disabled, status, created_at, updated_at, "+
			"tls, tls_skip_verify, qan_options, exporter_options, aws_options, azure_options, mongo_options, mysql_options, postgresql_options) "+
			"VALUES ('1', 'pmm-agent', 'username', 'password', 'agent-password', '1', false, '', $1, $2, false, false, '{}', '{}', $3, $4, $5, $6, $7)",
		now, now, awsOptions, azureOptions, mongoOptions, mysqlOptions, postgresqlOptions,
	)
	require.NoError(t, err)
}

func readAgentSecrets(ctx context.Context, t *testing.T, db *sql.DB) agentSecrets {
	t.Helper()

	var s agentSecrets
	var aws, azure, mongo, mysql, postgresql []byte
	err := db.QueryRowContext(ctx,
		"SELECT username, password, agent_password, aws_options, azure_options, mongo_options, mysql_options, postgresql_options "+
			"FROM agents WHERE agent_id = $1", "1").
		Scan(&s.Username, &s.Password, &s.AgentPassword, &aws, &azure, &mongo, &mysql, &postgresql)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(aws, &s.AWS))
	require.NoError(t, json.Unmarshal(azure, &s.Azure))
	require.NoError(t, json.Unmarshal(mongo, &s.MongoDB))
	require.NoError(t, json.Unmarshal(mysql, &s.MySQL))
	require.NoError(t, json.Unmarshal(postgresql, &s.PostgreSQL))

	return s
}
