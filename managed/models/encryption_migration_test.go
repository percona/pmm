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
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestMigrateEncryption(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)

	// nothing is encrypted yet
	hasEncrypted, err := models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.False(t, hasEncrypted)

	// legacy format is base64 of the Tink ciphertext without the envelope prefix
	toLegacy := func(plaintext string) string {
		stored, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		return strings.TrimPrefix(stored, encryption.EnvelopePrefix)
	}

	awsOptions, err := json.Marshal(map[string]any{
		"aws_access_key":                toLegacy("AKIAIOSFODNN7EXAMPLE"),
		"aws_secret_key":                toLegacy("aws-secret-key"),
		"rds_basic_metrics_disabled":    true,
		"rds_enhanced_metrics_disabled": false,
	})
	require.NoError(t, err)
	// pre-encryption blob: plaintext secrets inside JSON
	mysqlOptions, err := json.Marshal(map[string]any{
		"tls_cert": "-----BEGIN CERTIFICATE-----plain-----END CERTIFICATE-----",
		"tls_key":  "-----BEGIN PRIVATE KEY-----plain-----END PRIVATE KEY-----",
	})
	require.NoError(t, err)

	now := time.Now()
	_, err = sqlDB.ExecContext(
		t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)",
		now, now,
	)
	require.NoError(t, err)
	// username in legacy format, password in pre-encryption plaintext
	//nolint:dupword
	_, err = sqlDB.ExecContext(
		t.Context(),
		`INSERT INTO agents (agent_id, agent_type, username, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, aws_options, mysql_options) `+
			`VALUES ('A1', 'pmm-agent', $1, $2, 'N1', false, '', $3, $4, false, false, $5, $6)`,
		toLegacy("legacy-user"), "plain-password", now, now, string(awsOptions), string(mysqlOptions),
	)
	require.NoError(t, err)
	// backup location with pre-encryption plaintext S3 credentials
	_, err = sqlDB.ExecContext(
		t.Context(),
		`INSERT INTO backup_locations (id, name, description, type, s3_config, created_at, updated_at) `+
			`VALUES ('L1', 'loc', '', 's3', '{"endpoint": "https://s3.example.com", "access_key": "s3-access-key", "secret_key": "s3-secret-key", "bucket_name": "b", "bucket_region": "r"}', $1, $2)`,
		now, now,
	)
	require.NoError(t, err)

	ids, err := models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	require.Equal(t, []string{"A1"}, ids)
	locationIDs, err := models.LocationsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	require.Equal(t, []string{"L1"}, locationIDs)

	require.NoError(t, models.MigrateEncryption(q))

	// the lost-key startup guard now detects encrypted data
	hasEncrypted, err = models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.True(t, hasEncrypted)

	// all stored secrets are now envelopes carrying the primary key ID
	readRaw := func() (username, password, aws, mysql string) {
		t.Helper()
		var u, p, a, m sql.NullString
		err := sqlDB.QueryRowContext(t.Context(), `SELECT username, password, aws_options, mysql_options FROM agents WHERE agent_id = 'A1'`).Scan(&u, &p, &a, &m)
		require.NoError(t, err)
		return u.String, p.String, a.String, m.String
	}
	username, password, aws, mysql := readRaw()
	for _, stored := range []string{username, password} {
		keyID, ok := encryption.StoredKeyID(stored)
		require.True(t, ok, "value %q is not an envelope", stored)
		assert.Equal(t, cipher.PrimaryKeyID(), keyID)
	}
	var storedAWS models.AWSOptions
	require.NoError(t, json.Unmarshal([]byte(aws), &storedAWS))
	assert.True(t, encryption.IsEncrypted(storedAWS.AWSAccessKey))
	assert.True(t, encryption.IsEncrypted(storedAWS.AWSSecretKey))
	assert.True(t, storedAWS.RDSBasicMetricsDisabled)
	var storedMySQL models.MySQLOptions
	require.NoError(t, json.Unmarshal([]byte(mysql), &storedMySQL))
	assert.True(t, encryption.IsEncrypted(storedMySQL.TLSCert))
	assert.True(t, encryption.IsEncrypted(storedMySQL.TLSKey))

	// backup location S3 credentials are encrypted at rest and readable via the model
	var rawS3 string
	err = sqlDB.QueryRowContext(t.Context(), `SELECT s3_config FROM backup_locations WHERE id = 'L1'`).Scan(&rawS3)
	require.NoError(t, err)
	var storedS3 models.S3LocationConfig
	require.NoError(t, json.Unmarshal([]byte(rawS3), &storedS3))
	assert.True(t, encryption.IsEncrypted(storedS3.AccessKey))
	assert.True(t, encryption.IsEncrypted(storedS3.SecretKey))
	assert.Equal(t, "https://s3.example.com", storedS3.Endpoint)
	location, err := models.FindBackupLocationByID(q, "L1")
	require.NoError(t, err)
	assert.Equal(t, "s3-access-key", location.S3Config.AccessKey)
	assert.Equal(t, "s3-secret-key", location.S3Config.SecretKey)

	// the model view decrypts to the original secrets
	agent, err := models.FindAgentByID(q, "A1")
	require.NoError(t, err)
	assert.Equal(t, "legacy-user", agent.Username.Reveal())
	assert.Equal(t, "plain-password", agent.Password.Reveal())
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", agent.AWSOptions.AWSAccessKey)
	assert.Equal(t, "aws-secret-key", agent.AWSOptions.AWSSecretKey)
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----plain-----END PRIVATE KEY-----", agent.MySQLOptions.TLSKey)

	// idempotency: a second run finds nothing and rewrites nothing
	ids, err = models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	assert.Empty(t, ids)
	locationIDs, err = models.LocationsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	assert.Empty(t, locationIDs)
	require.NoError(t, models.MigrateEncryption(q))
	username2, password2, aws2, mysql2 := readRaw()
	assert.Equal(t, username, username2)
	assert.Equal(t, password, password2)
	assert.Equal(t, aws, aws2)
	assert.Equal(t, mysql, mysql2)
}

// TestMigrateEncryptionWrongKey covers a key file that does not match the
// data: PMM 3.x recorded the column as encrypted, the value has the shape of
// Tink ciphertext, but its key is not in the keyset. The migration must refuse
// instead of storing the ciphertext as if it were the password.
func TestMigrateEncryptionWrongKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)

	// legacy ciphertext produced by an unrelated keyset
	other, err := encryption.CreateCipher(encryption.NewFileKeyProvider(t.TempDir() + "/other.key"))
	require.NoError(t, err)
	stored, err := other.Encrypt("password-under-another-key")
	require.NoError(t, err)
	foreign := strings.TrimPrefix(stored, encryption.EnvelopePrefix)

	now := time.Now()
	_, err = sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
	require.NoError(t, err)
	// PMM 3.x wrote pmm-agent rows back decrypted, so only other agent types
	// are known to hold ciphertext in recorded columns
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
			`VALUES ('PA', 'pmm-agent', $1, 'N1', false, '', $2, $3, false, false)`,
		foreign, now, now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, password, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
			`VALUES ('A1', 'node_exporter', $1, 'PA', 'N1', false, '', $2, $3, false, false)`,
		foreign, now, now)
	require.NoError(t, err)

	t.Run("without PMM 3.x bookkeeping the value is plaintext", func(t *testing.T) {
		ids, err := models.AgentsNeedingReencryption(q, cipher)
		require.NoError(t, err)
		assert.Equal(t, []string{"A1", "PA"}, ids)
	})

	t.Run("recorded as encrypted: refuse and change nothing", func(t *testing.T) {
		res, err := sqlDB.ExecContext(t.Context(),
			`UPDATE settings SET settings = settings || '{"encrypted_items": ["pmm-managed.agents.password"]}'::jsonb`)
		require.NoError(t, err)
		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), n, "settings row is missing")

		_, err = models.AgentsNeedingReencryption(q, cipher)
		require.ErrorIs(t, err, encryption.ErrLegacyUnknownKey)
		assert.Contains(t, err.Error(), "agent A1 password")
		assert.NotContains(t, err.Error(), "agent PA")

		err = models.MigrateEncryption(q)
		require.ErrorIs(t, err, encryption.ErrLegacyUnknownKey)
		require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)

		for _, id := range []string{"A1", "PA"} {
			var password string
			require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT password FROM agents WHERE agent_id = $1`, id).Scan(&password))
			assert.Equal(t, foreign, password, id)
		}
	})
}

// TestDatabaseHasEncryptedDataInOptions covers installs whose only secrets
// live inside JSON option blobs, e.g. RDS or Azure agents without username.
func TestDatabaseHasEncryptedDataInOptions(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	require.NoError(t, q.Insert(&models.Node{NodeID: "N1", NodeType: models.GenericNodeType, NodeName: "name"}))
	require.NoError(t, q.Insert(&models.Agent{
		AgentID:      "A1",
		AgentType:    models.PMMAgentType,
		RunsOnNodeID: new("N1"),
		AWSOptions:   models.AWSOptions{AWSAccessKey: "access", AWSSecretKey: "secret"},
	}))

	hasEncrypted, err := models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.True(t, hasEncrypted)
}

// TestMigrateEncryptionStackedLayers covers option fields corrupted by key
// rotation before PMM 3.9.1 (PMM-15188): the value is new(old(secret)) and
// the previous key is the one PMM 3.x rotation left next to the key file.
func TestMigrateEncryptionStackedLayers(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	current, err := encryption.DefaultCipher()
	require.NoError(t, err)
	oldProvider := encryption.NewFileKeyProvider(t.TempDir() + "/pmm-encryption_old.key")
	oldKey, err := encryption.CreateCipher(oldProvider)
	require.NoError(t, err)
	lostKey, err := encryption.CreateCipher(encryption.NewFileKeyProvider(t.TempDir() + "/lost.key"))
	require.NoError(t, err)

	withOld, err := current.WithLegacyKeys(oldProvider)
	require.NoError(t, err)
	encryption.SetDefaultCipher(withOld)
	t.Cleanup(func() { encryption.SetDefaultCipher(current) })

	layer := func(c *encryption.Cipher, plaintext string) string {
		stored, err := c.Encrypt(plaintext)
		require.NoError(t, err)
		return strings.TrimPrefix(stored, encryption.EnvelopePrefix)
	}
	const tlsKey = "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----\n"
	options := func(stored string) string {
		b, err := json.Marshal(map[string]any{"tls_key": stored})
		require.NoError(t, err)
		return string(b)
	}

	now := time.Now()
	_, err = sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
	require.NoError(t, err)
	for id, stored := range map[string]string{
		"repairable": layer(current, layer(oldKey, tlsKey)),
		"lost":       layer(current, layer(lostKey, tlsKey)),
	} {
		//nolint:dupword
		_, err = sqlDB.ExecContext(t.Context(),
			`INSERT INTO agents (agent_id, agent_type, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, mysql_options) `+
				`VALUES ($1, 'pmm-agent', 'N1', false, '', $2, $3, false, false, $4)`,
			id, now, now, options(stored))
		require.NoError(t, err)
	}

	// the lost secret is reported, not treated as a key mismatch
	require.NoError(t, models.MigrateEncryption(q))

	agent, err := models.FindAgentByID(q, "repairable")
	require.NoError(t, err)
	assert.Equal(t, tlsKey, agent.MySQLOptions.TLSKey)

	// the repaired value is a single envelope readable without the previous key
	var raw string
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT mysql_options FROM agents WHERE agent_id = 'repairable'`).Scan(&raw))
	var stored models.MySQLOptions
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	decrypted, err := current.Decrypt(stored.TLSKey)
	require.NoError(t, err)
	assert.Equal(t, tlsKey, decrypted)

	// the other agent stays readable, and the sweep has converged
	_, err = models.FindAgentByID(q, "lost")
	require.NoError(t, err)
	ids, err := models.AgentsNeedingReencryption(q, withOld)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// TestMigrateEncryptionConcurrency covers HA nodes: the migration waits for
// another node's migration, and writes only secret columns so changes made
// by live nodes to other columns are kept.
func TestMigrateEncryptionConcurrency(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	updatedAt := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	_, err := sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", updatedAt, updatedAt)
	require.NoError(t, err)
	//nolint:dupword
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, listen_port) `+
			`VALUES ('A1', 'pmm-agent', 'plain-password', 'N1', false, 'RUNNING', $1, $2, false, false, 42000)`,
		updatedAt, updatedAt)
	require.NoError(t, err)

	// another node holds the migration lock
	other, err := sqlDB.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	_, err = other.ExecContext(t.Context(), "SELECT pg_advisory_xact_lock($1)", 0x504d4d45)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- db.InTransaction(func(tx *reform.TX) error {
			return models.MigrateEncryption(tx.Querier)
		})
	}()
	select {
	case err = <-done:
		t.Fatalf("migration did not wait for the lock: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	require.NoError(t, other.Rollback())
	require.NoError(t, <-done)

	var password, status string
	var listenPort int
	var storedUpdatedAt time.Time
	err = sqlDB.QueryRowContext(t.Context(), `SELECT password, status, listen_port, updated_at FROM agents WHERE agent_id = 'A1'`).
		Scan(&password, &status, &listenPort, &storedUpdatedAt)
	require.NoError(t, err)
	assert.True(t, encryption.IsEncrypted(password))
	assert.Equal(t, "RUNNING", status)
	assert.Equal(t, 42000, listenPort)
	assert.True(t, updatedAt.Equal(storedUpdatedAt), "updated_at was rewritten: %s", storedUpdatedAt)
}

// TestMigrateEncryptionBackup covers the one-time backup of values rewritten
// from the pre-envelope format: older PMM versions cannot read envelopes, so
// the backup is what makes the upgrade reversible.
func TestMigrateEncryptionBackup(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	// the backup is written next to the key file
	dir := t.TempDir()
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(dir, "encryption.key"))

	now := time.Now()
	_, err := sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
	require.NoError(t, err)
	//nolint:dupword
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, mysql_options) `+
			`VALUES ('A1', 'pmm-agent', 'plain-password', 'N1', false, '', $1, $2, false, false, '{"tls_key": "plain-key"}')`,
		now, now)
	require.NoError(t, err)

	require.NoError(t, models.MigrateEncryption(q))

	files, err := filepath.Glob(filepath.Join(dir, models.MigrationBackupPattern))
	require.NoError(t, err)
	require.Len(t, files, 1)
	// PMM Server's data directory is listed too and may hold backups of its own
	assert.Contains(t, models.MigrationBackupFiles(), files[0])
	info, err := os.Stat(files[0])
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	var backup struct {
		Tables struct {
			Agents []struct {
				AgentID      string          `json:"agent_id"`
				Password     string          `json:"password"`
				MySQLOptions json.RawMessage `json:"mysql_options"`
			} `json:"agents"`
		} `json:"tables"`
	}
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &backup))
	agents := backup.Tables.Agents
	require.Len(t, agents, 1)
	assert.Equal(t, "A1", agents[0].AgentID)
	assert.Equal(t, "plain-password", agents[0].Password)
	assert.JSONEq(t, `{"tls_key": "plain-key"}`, string(agents[0].MySQLOptions))

	// rows already in the envelope format are not backed up again
	require.NoError(t, models.MigrateEncryption(q))
	files, err = filepath.Glob(filepath.Join(dir, models.MigrationBackupPattern))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

// TestMigrateEncryptionAfterOlderVersion covers PMM before the envelope
// format started against an upgraded database: settings no longer list the
// encrypted columns, so its startup wraps every envelope in a legacy layer.
// The next migration must restore single envelopes of the original secrets.
func TestMigrateEncryptionAfterOlderVersion(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)
	olderVersionLayer := func(plaintext string) string {
		envelope, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		wrapped, err := cipher.Encrypt(envelope)
		require.NoError(t, err)
		return strings.TrimPrefix(wrapped, encryption.EnvelopePrefix)
	}

	const tlsKey = "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----\n"
	mysqlOptions, err := json.Marshal(map[string]any{"tls_key": olderVersionLayer(tlsKey)})
	require.NoError(t, err)

	now := time.Now()
	_, err = sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
	require.NoError(t, err)
	//nolint:dupword
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, mysql_options) `+
			`VALUES ('A1', 'pmm-agent', $1, 'N1', false, '', $2, $3, false, false, $4)`,
		olderVersionLayer("password"), now, now, string(mysqlOptions))
	require.NoError(t, err)

	require.NoError(t, models.MigrateEncryption(q))

	agent, err := models.FindAgentByID(q, "A1")
	require.NoError(t, err)
	assert.Equal(t, "password", agent.Password.Reveal())
	assert.Equal(t, tlsKey, agent.MySQLOptions.TLSKey)

	// stored values are single envelopes again: nothing left to rewrite
	var password, rawOptions string
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT password, mysql_options FROM agents WHERE agent_id = 'A1'`).Scan(&password, &rawOptions))
	var stored models.MySQLOptions
	require.NoError(t, json.Unmarshal([]byte(rawOptions), &stored))
	for _, v := range []string{password, stored.TLSKey} {
		insp, err := cipher.Inspect(v)
		require.NoError(t, err)
		assert.Zero(t, insp.ExtraLayers)
		assert.True(t, encryption.IsEncrypted(v))
	}
	ids, err := models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// TestMigrateEncryptionAfterKeyLoss covers a PMM 3.x install that lost its key
// file and silently generated a new one: rows written since use the current
// key, untouched rows still hold ciphertext of the lost key. No key file reads
// everything, so the migration must not refuse to start; it reports the lost
// values, keeps them in the backup, and they become readable again if the
// lost key is placed as the previous key.
func TestMigrateEncryptionAfterKeyLoss(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	dir := t.TempDir()
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(dir, "encryption.key"))

	current, err := encryption.DefaultCipher()
	require.NoError(t, err)
	lostProvider := encryption.NewFileKeyProvider(filepath.Join(t.TempDir(), "lost.key"))
	lostKey, err := encryption.CreateCipher(lostProvider)
	require.NoError(t, err)
	legacy := func(c *encryption.Cipher, plaintext string) string {
		stored, err := c.Encrypt(plaintext)
		require.NoError(t, err)
		return strings.TrimPrefix(stored, encryption.EnvelopePrefix)
	}
	stale := legacy(lostKey, "password-before-key-loss")

	_, err = sqlDB.ExecContext(t.Context(),
		`UPDATE settings SET settings = settings || '{"encrypted_items": ["pmm-managed.agents.password"]}'::jsonb`)
	require.NoError(t, err)
	now := time.Now()
	_, err = sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
			`VALUES ('PA', 'pmm-agent', 'N1', false, '', $1, $2, false, false)`, now, now)
	require.NoError(t, err)
	for id, password := range map[string]string{
		"fresh": legacy(current, "password-after-key-loss"),
		"stale": stale,
	} {
		_, err = sqlDB.ExecContext(t.Context(),
			`INSERT INTO agents (agent_id, agent_type, password, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
				`VALUES ($1, 'node_exporter', $2, 'PA', 'N1', false, '', $3, $4, false, false)`,
			id, password, now, now)
		require.NoError(t, err)
	}

	require.NoError(t, models.MigrateEncryption(q))

	agent, err := models.FindAgentByID(q, "fresh")
	require.NoError(t, err)
	assert.Equal(t, "password-after-key-loss", agent.Password.Reveal())

	// the lost value is rewritten, so the sweep converges, and kept in the backup
	var storedStale string
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT password FROM agents WHERE agent_id = 'stale'`).Scan(&storedStale))
	assert.True(t, encryption.IsEncrypted(storedStale))
	ids, err := models.AgentsNeedingReencryption(q, current)
	require.NoError(t, err)
	assert.Empty(t, ids)
	files, err := filepath.Glob(filepath.Join(dir, models.MigrationBackupPattern))
	require.NoError(t, err)
	require.Len(t, files, 1)
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), stale)

	// placing the lost key as the previous key makes the value readable again
	withLostKey, err := current.WithLegacyKeys(lostProvider)
	require.NoError(t, err)
	encryption.SetDefaultCipher(withLostKey)
	t.Cleanup(func() { encryption.SetDefaultCipher(current) })
	agent, err = models.FindAgentByID(q, "stale")
	require.NoError(t, err)
	assert.Equal(t, "password-before-key-loss", agent.Password.Reveal())
}

// TestMigrateEncryptionValkeyTLS covers PMM-15375: PMM 3.x stored the Valkey
// exporter's TLS certificate and key in plaintext. The migration encrypts them
// like the other TLS options; the CA certificate is public and stays readable.
func TestMigrateEncryptionValkeyTLS(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	const (
		ca   = "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----\n"
		cert = "-----BEGIN CERTIFICATE-----\nclient\n-----END CERTIFICATE-----\n"
		key  = "-----BEGIN PRIVATE KEY-----\nclient\n-----END PRIVATE KEY-----\n"
	)
	valkeyOptions, err := json.Marshal(map[string]any{"tls": true, "ssl_ca": ca, "ssl_cert": cert, "ssl_key": key})
	require.NoError(t, err)

	now := time.Now()
	_, err = sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
			`VALUES ('PA', 'pmm-agent', 'N1', false, '', $1, $2, false, false)`, now, now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, valkey_options) `+
			`VALUES ('V1', 'valkey_exporter', 'PA', 'N1', false, '', $1, $2, true, false, $3)`,
		now, now, string(valkeyOptions))
	require.NoError(t, err)

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)
	ids, err := models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	require.Equal(t, []string{"V1"}, ids)

	require.NoError(t, models.MigrateEncryption(q))

	var raw string
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT valkey_options FROM agents WHERE agent_id = 'V1'`).Scan(&raw))
	var stored models.ValkeyOptions
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	assert.True(t, encryption.IsEncrypted(stored.SSLCert))
	assert.True(t, encryption.IsEncrypted(stored.SSLKey))
	assert.Equal(t, ca, stored.SSLCa)
	assert.True(t, stored.TLS)

	agent, err := models.FindAgentByID(q, "V1")
	require.NoError(t, err)
	assert.Equal(t, models.ValkeyOptions{TLS: true, SSLCa: ca, SSLCert: cert, SSLKey: key}, agent.ValkeyOptions)

	ids, err = models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	assert.Empty(t, ids)
}
