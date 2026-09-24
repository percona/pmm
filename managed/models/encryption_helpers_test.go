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

	decrypted := models.DecryptAgent(encrypted)
	require.NotNil(t, decrypted.Username)
	require.NotNil(t, decrypted.Password)
	assert.Equal(t, "username", *decrypted.Username)
	assert.Equal(t, "password", *decrypted.Password)
	assert.Equal(t, "mysql-tls-cert", decrypted.MySQLOptions.TLSCert)
	assert.Equal(t, "mysql-tls-key", decrypted.MySQLOptions.TLSKey)
}

// TestDecryptAgentUnreadableFields guards the handling of values this node's key cannot decrypt,
// which is what an HA node reads when a row was encrypted with another node's key
// (https://perconadev.atlassian.net/browse/PMM-14979). The ciphertext must not be handed back as
// if it were the decrypted value, and writing the Agent back must neither lose it nor encrypt it
// a second time.
func TestDecryptAgentUnreadableFields(t *testing.T) {
	foreignCiphertext := base64.StdEncoding.EncodeToString([]byte("encrypted-with-another-key"))

	agent := models.Agent{
		AgentID:  "/agent_id/1",
		Username: new(foreignCiphertext),
		Password: new(foreignCiphertext),
		MySQLOptions: models.MySQLOptions{
			TLSKey: foreignCiphertext,
		},
	}

	decrypted := models.DecryptAgent(agent)
	require.NotNil(t, decrypted.Username)
	require.NotNil(t, decrypted.Password)
	assert.Empty(t, *decrypted.Username, "ciphertext must not be returned as the decrypted value")
	assert.Empty(t, *decrypted.Password)
	assert.Empty(t, decrypted.MySQLOptions.TLSKey)
	assert.Equal(t, foreignCiphertext, *agent.Username, "input agent must not be mutated")

	t.Run("unchanged fields are written back as stored", func(t *testing.T) {
		encrypted, err := models.EncryptAgent(decrypted)
		require.NoError(t, err)
		assert.Equal(t, foreignCiphertext, *encrypted.Username)
		assert.Equal(t, foreignCiphertext, *encrypted.Password)
		assert.Equal(t, foreignCiphertext, encrypted.MySQLOptions.TLSKey)
	})

	t.Run("a new value replaces the stored one", func(t *testing.T) {
		changed := decrypted
		changed.Password = new("new-password")

		encrypted, err := models.EncryptAgent(changed)
		require.NoError(t, err)
		assert.Equal(t, foreignCiphertext, *encrypted.Username)

		password, err := encryption.Decrypt(*encrypted.Password)
		require.NoError(t, err)
		assert.Equal(t, "new-password", password)
	})
}

// TestEncryptionKeyInDatabase covers the parts of https://perconadev.atlassian.net/browse/PMM-14979
// that depend on PostgreSQL: the checks run while migrating, the lock between nodes, and how
// encrypted columns are recorded.
func TestEncryptionKeyInDatabase(t *testing.T) {
	// Initialize this node's key before the test switches the key path, so that it stays the
	// default key.
	localFingerprint, err := encryption.Fingerprint()
	require.NoError(t, err)

	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "foreign.key"))
	foreign := encryption.New()
	foreignFingerprint, err := foreign.Fingerprint()
	require.NoError(t, err)

	const dbName = "pmm-managed-dev"

	setup := func(t *testing.T, sqlDB *sql.DB, haNodeID string) error {
		t.Helper()
		_, err := models.SetupDB(t.Context(), sqlDB, models.SetupDBParams{
			Address:       models.DefaultPostgreSQLAddr,
			Name:          dbName,
			Username:      "postgres",
			SetupFixtures: models.SetupFixtures,
			HANodeID:      haNodeID,
		})

		return err
	}

	setFingerprint := func(t *testing.T, db *reform.DB, fingerprint string) {
		t.Helper()
		_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EncryptionKeyFingerprint: &fingerprint})
		require.NoError(t, err)
	}

	countAgents := func(t *testing.T, db *reform.DB) int {
		t.Helper()
		var n int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&n))

		return n
	}

	t.Run("HA node with a foreign key writes nothing", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		setFingerprint(t, db, foreignFingerprint)

		err := setup(t, sqlDB, "pmm-test-0")
		require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)

		settings, err := models.GetSettings(db)
		require.NoError(t, err)
		assert.Empty(t, settings.EncryptedItems)
		assert.Equal(t, foreignFingerprint, settings.EncryptionKeyFingerprint)
		assert.Zero(t, countAgents(t, db))
	})

	t.Run("standalone server with a foreign key boots and keeps the fingerprint", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		foreignCiphertext, err := foreign.Encrypt("postgres")
		require.NoError(t, err)
		_, err = db.Exec("UPDATE agents SET username = $1 WHERE agent_type = $2", foreignCiphertext, models.PostgresExporterType)
		require.NoError(t, err)
		setFingerprint(t, db, foreignFingerprint)

		before, err := models.GetSettings(db)
		require.NoError(t, err)

		require.NoError(t, setup(t, sqlDB, ""))

		settings, err := models.GetSettings(db)
		require.NoError(t, err)
		assert.Equal(t, before.EncryptedItems, settings.EncryptedItems)
		assert.Equal(t, foreignFingerprint, settings.EncryptionKeyFingerprint)
		require.ErrorIs(t, models.CheckEncryptionKey(db), models.ErrEncryptionKeyMismatch)
	})

	// A standalone server whose key was lost, after every credential was re-entered with its new key.
	t.Run("standalone server adopts its key once every credential decrypts", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		setFingerprint(t, db, foreignFingerprint)

		require.NoError(t, setup(t, sqlDB, ""))

		settings, err := models.GetSettings(db)
		require.NoError(t, err)
		assert.Equal(t, localFingerprint, settings.EncryptionKeyFingerprint)
		require.NoError(t, models.CheckEncryptionKey(db))
	})

	t.Run("columns are not encrypted with a foreign key", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		setFingerprint(t, db, foreignFingerprint)

		err := db.InTransaction(func(tx *reform.TX) error {
			return models.EncryptDB(tx, dbName, models.DefaultAgentEncryptionColumnsV3)
		})
		require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	})

	t.Run("a node checking the key waits for the node that holds the lock", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, models.VerifyEncryptionKey(tx))
		// Stands in for the first node having recorded a different key.
		_, err = models.UpdateSettings(tx, &models.ChangeSettingsParams{EncryptionKeyFingerprint: &foreignFingerprint})
		require.NoError(t, err)

		res := make(chan error, 1)
		go func() {
			res <- db.InTransaction(models.VerifyEncryptionKey)
		}()

		select {
		case err := <-res:
			require.FailNow(t, "the key was checked while another node held the lock", "error: %v", err)
		case <-time.After(500 * time.Millisecond):
		}

		require.NoError(t, tx.Commit())
		require.ErrorIs(t, <-res, models.ErrEncryptionKeyMismatch)
	})

	t.Run("unreadable credentials do not fail agent queries", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		agents, err := models.FindAgents(db.Querier, models.AgentFilters{AgentType: new(models.PostgresExporterType)})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		agentID := agents[0].AgentID

		foreignCiphertext, err := foreign.Encrypt("postgres")
		require.NoError(t, err)
		_, err = db.Exec("UPDATE agents SET username = $1 WHERE agent_id = $2", foreignCiphertext, agentID)
		require.NoError(t, err)

		all, err := models.FindAgents(db.Querier, models.AgentFilters{})
		require.NoError(t, err)
		assert.Greater(t, len(all), 1)

		agent, err := models.FindAgentByID(db.Querier, agentID)
		require.NoError(t, err)
		require.NotNil(t, agent.Username)
		assert.Empty(t, *agent.Username)

		agent.Disabled = true
		require.NoError(t, models.UpdateAgent(db.Querier, agent))

		var stored string
		require.NoError(t, db.QueryRow("SELECT username FROM agents WHERE agent_id = $1", agentID).Scan(&stored))
		assert.Equal(t, foreignCiphertext, stored, "writing the agent back must keep the stored credentials")
	})

	// Before the fix, the list was replaced by the columns of the last call only, so the next
	// start encrypted the rest a second time.
	t.Run("encrypted columns are added to the recorded ones", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		require.NoError(t, setup(t, sqlDB, ""))
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		settings, err := models.GetSettings(db)
		require.NoError(t, err)
		all := settings.EncryptedItems
		require.Contains(t, all, dbName+".agents.username")
		require.Contains(t, all, dbName+".agents.password")

		usernameOnly := []encryption.Table{{
			Name:        "agents",
			Identifiers: []string{"agent_id"},
			Columns:     []encryption.Column{{Name: "username"}},
		}}
		require.NoError(t, db.InTransaction(func(tx *reform.TX) error {
			return models.DecryptDB(tx, dbName, usernameOnly)
		}))

		settings, err = models.GetSettings(db)
		require.NoError(t, err)
		assert.NotContains(t, settings.EncryptedItems, dbName+".agents.username")
		assert.Contains(t, settings.EncryptedItems, dbName+".agents.password")
		assert.NotEmpty(t, settings.EncryptionKeyFingerprint, "other columns are still encrypted")

		require.NoError(t, setup(t, sqlDB, ""))

		settings, err = models.GetSettings(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, all, settings.EncryptedItems)

		agents, err := models.FindAgents(db.Querier, models.AgentFilters{AgentType: new(models.PostgresExporterType)})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		require.NotNil(t, agents[0].Username)
		assert.Equal(t, "postgres", *agents[0].Username, "username must be encrypted exactly once")

		require.NoError(t, db.InTransaction(func(tx *reform.TX) error {
			return models.DecryptDB(tx, dbName, models.DefaultAgentEncryptionColumnsV3)
		}))

		settings, err = models.GetSettings(db)
		require.NoError(t, err)
		assert.Empty(t, settings.EncryptedItems)
		assert.Empty(t, settings.EncryptionKeyFingerprint)
	})
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
