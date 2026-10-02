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
	"encoding/base64"
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

// keepDefaultCipher restores the process-wide cipher after a test that lets
// SetupDB install another one.
func keepDefaultCipher(t *testing.T) *encryption.Cipher {
	t.Helper()

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)
	t.Cleanup(func() { encryption.SetDefaultCipher(cipher) })

	return cipher
}

// setupDB runs pmm-managed's startup on an already migrated test database.
func setupDB(t *testing.T, sqlDB *sql.DB) error {
	t.Helper()

	_, err := models.SetupDB(t.Context(), sqlDB, models.SetupDBParams{
		Logf:          t.Logf,
		Address:       models.DefaultPostgreSQLAddr,
		Username:      "postgres",
		SetupFixtures: models.SkipFixtures,
	})

	return err
}

// insertExporter stores an exporter row with the given stored values as they
// are, next to the pmm-agent and node it needs.
func insertExporter(t *testing.T, sqlDB *sql.DB, id, password, mysqlOptions string) {
	t.Helper()

	now := time.Now()
	_, err := sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2) ON CONFLICT DO NOTHING", now, now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
			`VALUES ('PA', 'pmm-agent', 'N1', false, '', $1, $2, false, false) ON CONFLICT DO NOTHING`, now, now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, password, mysql_options, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
			`VALUES ($1, 'mysqld_exporter', NULLIF($2, ''), NULLIF($3, '')::jsonb, 'PA', 'N1', false, '', $4, $5, false, false)`,
		id, password, mysqlOptions, now, now)
	require.NoError(t, err)
}

func legacyCiphertext(t *testing.T, c *encryption.Cipher, plaintext string) string {
	t.Helper()

	stored, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	return strings.TrimPrefix(stored, encryption.EnvelopePrefix)
}

// corruptEnvelope flips a byte of the authentication tag, keeping the envelope
// well-formed.
func corruptEnvelope(t *testing.T, stored string) string {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encryption.EnvelopePrefix))
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xff

	return encryption.EnvelopePrefix + base64.StdEncoding.EncodeToString(raw)
}

// forgetKeyCheck removes the encryption key check that testdb's setup stored
// with the test key, so that the database can be started with another key,
// like one set up before the key check existed.
func forgetKeyCheck(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	_, err := sqlDB.ExecContext(t.Context(), "UPDATE settings SET settings = settings - 'encryption_key_check'")
	require.NoError(t, err)
}

func storedKeyCheck(t *testing.T, sqlDB *sql.DB) string {
	t.Helper()

	var check sql.NullString
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), "SELECT settings->>'encryption_key_check' FROM settings").Scan(&check))

	return check.String
}

func storedPassword(t *testing.T, sqlDB *sql.DB, id string) string {
	t.Helper()

	var password string
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT password FROM agents WHERE agent_id = $1`, id).Scan(&password))

	return password
}

func TestSetupDBCreatesMissingKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	path := filepath.Join(t.TempDir(), "encryption.key")
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	previousCheck := storedKeyCheck(t, sqlDB)
	require.NotEmpty(t, previousCheck)

	// nothing is encrypted yet, so a new key is generated, and the key check
	// of the previous one is replaced
	require.NoError(t, setupDB(t, sqlDB))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	created, err := encryption.LoadCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	active, err := encryption.DefaultCipher()
	require.NoError(t, err)
	assert.Equal(t, created.PrimaryKeyID(), active.PrimaryKeyID())
	assert.NotEqual(t, previousCheck, storedKeyCheck(t, sqlDB))
	stale, err := models.KeyCheckNeedsReencryption(reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier, created)
	require.NoError(t, err)
	assert.False(t, stale)
}

func TestSetupDBLoadsExistingKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	path := filepath.Join(t.TempDir(), "encryption.key")
	existing, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	forgetKeyCheck(t, sqlDB)

	require.NoError(t, setupDB(t, sqlDB))

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the key file is used unchanged")
	active, err := encryption.DefaultCipher()
	require.NoError(t, err)
	assert.Equal(t, existing.PrimaryKeyID(), active.PrimaryKeyID())
}

func TestSetupDBRefusesMissingKeyWithEncryptedData(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	cipher := keepDefaultCipher(t)
	stored, err := cipher.Encrypt("password")
	require.NoError(t, err)
	insertExporter(t, sqlDB, "E1", stored, "")

	path := filepath.Join(t.TempDir(), "encryption.key")
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)

	err = setupDB(t, sqlDB)
	require.ErrorContains(t, err, "encryption key not found")
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "no new key must be generated")
	assert.Equal(t, stored, storedPassword(t, sqlDB, "E1"))
}

func TestSetupDBUsesPreviousKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "pmm-encryption.key")
	current, err := encryption.CreateCipher(encryption.NewFileKeyProvider(keyPath))
	require.NoError(t, err)
	previous, err := encryption.CreateCipher(encryption.NewFileKeyProvider(encryption.LegacyBackupKeyPath(keyPath)))
	require.NoError(t, err)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, keyPath)

	// written before the previous PMM 3.x rotation, with the key it kept as *_old.key
	forgetKeyCheck(t, sqlDB)
	insertExporter(t, sqlDB, "E1", legacyCiphertext(t, previous, "password-before-rotation"), "")

	require.NoError(t, setupDB(t, sqlDB))

	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	agent, err := models.FindAgentByID(q, "E1")
	require.NoError(t, err)
	assert.Equal(t, "password-before-rotation", agent.Password.Reveal())
	keyID, ok := encryption.StoredKeyID(storedPassword(t, sqlDB, "E1"))
	require.True(t, ok)
	assert.Equal(t, current.PrimaryKeyID(), keyID)
}

func TestSetupDBIgnoresBrokenPreviousKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "pmm-encryption.key")
	current, err := encryption.CreateCipher(encryption.NewFileKeyProvider(keyPath))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(encryption.LegacyBackupKeyPath(keyPath), []byte("not a keyset"), 0o600))
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, keyPath)

	// the previous key is optional: a broken one is logged and ignored
	forgetKeyCheck(t, sqlDB)
	require.NoError(t, setupDB(t, sqlDB))

	active, err := encryption.DefaultCipher()
	require.NoError(t, err)
	assert.Equal(t, current.PrimaryKeyID(), active.PrimaryKeyID())
}

func TestDatabaseHasEncryptedDataLegacyBookkeeping(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	hasEncrypted, err := models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	require.False(t, hasEncrypted)

	// PMM 3.x recorded the encrypted columns in settings
	_, err = sqlDB.ExecContext(t.Context(),
		`UPDATE settings SET settings = settings || '{"encrypted_items": ["pmm-managed.agents.password"]}'::jsonb`)
	require.NoError(t, err)
	hasEncrypted, err = models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.True(t, hasEncrypted)
}

func TestDatabaseHasEncryptedDataWithoutSchema(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, new(0))
	keepDefaultCipher(t)

	hasEncrypted, err := models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.False(t, hasEncrypted)

	// a first start creates the whole schema and runs the encryption migration
	require.NoError(t, setupDB(t, sqlDB))
	hasEncrypted, err = models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.False(t, hasEncrypted)
}

func TestMigrateEncryptionWithoutSettingsRow(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	insertExporter(t, sqlDB, "E1", "plain-password", "")
	_, err := sqlDB.ExecContext(t.Context(), `DELETE FROM settings`)
	require.NoError(t, err)

	require.NoError(t, models.MigrateEncryption(q))
	assert.True(t, encryption.IsEncrypted(storedPassword(t, sqlDB, "E1")))
}

// TestCorruptedCiphertextFailsReads covers "a decryption failure fails the
// operation": a corrupted envelope is never returned as data, and the startup
// migration refuses to rewrite it.
func TestCorruptedCiphertextFailsReads(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)

	password, err := cipher.Encrypt("password")
	require.NoError(t, err)
	corruptPassword := corruptEnvelope(t, password)
	insertExporter(t, sqlDB, "bad-password", corruptPassword, "")

	tlsKey, err := cipher.Encrypt("tls-key")
	require.NoError(t, err)
	options, err := json.Marshal(map[string]string{"tls_key": corruptEnvelope(t, tlsKey)})
	require.NoError(t, err)
	insertExporter(t, sqlDB, "bad-tls-key", "", string(options))

	for _, id := range []string{"bad-password", "bad-tls-key"} {
		_, err = models.FindAgentByID(q, id)
		require.Error(t, err, id)
	}

	err = models.MigrateEncryption(q)
	require.ErrorContains(t, err, "cannot decrypt stored credentials")
	assert.Equal(t, corruptPassword, storedPassword(t, sqlDB, "bad-password"), "nothing is rewritten")
}

func TestMigrateEncryptionBackupLocation(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	insertExporter(t, sqlDB, "E1", "plain-password", "")

	// the key file's directory cannot be written (e.g. a read-only mount)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "missing", "encryption.key"))
	savedDefault := encryption.DefaultEncryptionKeyPath
	t.Cleanup(func() { encryption.DefaultEncryptionKeyPath = savedDefault })

	t.Run("no writable directory: refuse", func(t *testing.T) {
		encryption.DefaultEncryptionKeyPath = filepath.Join(t.TempDir(), "missing", "pmm-encryption.key")

		err := models.MigrateEncryption(q)
		require.ErrorContains(t, err, "refusing to migrate encrypted data without a backup")
		assert.Equal(t, "plain-password", storedPassword(t, sqlDB, "E1"), "nothing is rewritten")
	})

	t.Run("falls back to the data directory", func(t *testing.T) {
		srv := t.TempDir()
		encryption.DefaultEncryptionKeyPath = filepath.Join(srv, "pmm-encryption.key")

		require.NoError(t, models.MigrateEncryption(q))
		files, err := filepath.Glob(filepath.Join(srv, models.MigrationBackupPattern))
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.True(t, encryption.IsEncrypted(storedPassword(t, sqlDB, "E1")))
	})
}

func TestMigrateEncryptionManyUndecryptable(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	other, err := encryption.CreateCipher(encryption.NewFileKeyProvider(filepath.Join(t.TempDir(), "other.key")))
	require.NoError(t, err)

	_, err = sqlDB.ExecContext(t.Context(),
		`UPDATE settings SET settings = settings || '{"encrypted_items": ["pmm-managed.agents.password"]}'::jsonb`)
	require.NoError(t, err)
	for _, id := range []string{"E01", "E02", "E03", "E04", "E05", "E06", "E07", "E08", "E09", "E10", "E11"} {
		insertExporter(t, sqlDB, id, legacyCiphertext(t, other, "password"), "")
	}

	err = models.MigrateEncryption(q)
	require.ErrorIs(t, err, encryption.ErrLegacyUnknownKey)
	assert.Contains(t, err.Error(), "agent E01 password")
	assert.Contains(t, err.Error(), "and 1 more")
}

// TestFindersReturnDecryptedSecrets covers the finders that load agents
// without q.Reload: they must return secrets decrypted like FindAgentByID.
func TestFindersReturnDecryptedSecrets(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	q := db.Querier

	require.NoError(t, q.Insert(&models.Node{NodeID: "N1", NodeType: models.GenericNodeType, NodeName: "node"}))
	require.NoError(t, q.Insert(&models.Agent{AgentID: "PA", AgentType: models.PMMAgentType, RunsOnNodeID: new("N1")}))
	require.NoError(t, q.Insert(&models.Service{
		ServiceID:   "S1",
		ServiceType: models.PostgreSQLServiceType,
		ServiceName: models.PMMServerPostgreSQLServiceName,
		NodeID:      "N1",
		Address:     new("127.0.0.1"),
		Port:        new(uint16(5432)),
	}))
	require.NoError(t, q.Insert(&models.Agent{
		AgentID:           "QAN1",
		AgentType:         models.QANPostgreSQLPgStatementsAgentType,
		PMMAgentID:        new("PA"),
		ServiceID:         new("S1"),
		Username:          models.EncryptedStringOrNil("pmm"),
		Password:          models.EncryptedStringOrNil("qan-password"),
		PostgreSQLOptions: models.PostgreSQLOptions{SSLKey: "ssl-key"},
	}))
	require.True(t, encryption.IsEncrypted(storedPassword(t, sqlDB, "QAN1")))

	agent, err := models.FindInternalPgQANAgent(q)
	require.NoError(t, err)
	assert.Equal(t, "qan-password", agent.Password.Reveal())
	assert.Equal(t, "ssl-key", agent.PostgreSQLOptions.SSLKey)

	agents, err := models.FindAgentsByIDs(q, []string{"QAN1"})
	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "qan-password", agents[0].Password.Reveal())

	err = db.InTransaction(func(tx *reform.TX) error {
		agent, err := models.FindAgentByIDForUpdate(tx.Querier, "QAN1")
		require.NoError(t, err)
		assert.Equal(t, "qan-password", agent.Password.Reveal())
		assert.Equal(t, "ssl-key", agent.PostgreSQLOptions.SSLKey)
		return nil
	})
	require.NoError(t, err)
}

func TestEncryptedStringScan(t *testing.T) {
	testdb.SetupEncryption(t)
	cipher := keepDefaultCipher(t)

	var s models.EncryptedString
	require.NoError(t, s.Scan(nil))
	assert.Empty(t, s.Reveal())

	stored, err := cipher.Encrypt("from-bytes")
	require.NoError(t, err)
	require.NoError(t, s.Scan([]byte(stored)))
	assert.Equal(t, "from-bytes", s.Reveal())

	require.ErrorContains(t, s.Scan(42), "expected string or []byte")
	require.Error(t, new(models.MySQLOptions).Scan([]byte(`{"tls_key": `)))

	// before pmm-managed installed the cipher nothing is encrypted or decrypted
	encryption.SetDefaultCipher(nil)
	_, err = models.EncryptedString("secret").Value()
	require.ErrorIs(t, err, encryption.ErrEncryptionNotInitialized)
	require.ErrorIs(t, s.Scan(stored), encryption.ErrEncryptionNotInitialized)
	_, err = models.MySQLOptions{TLSKey: "secret"}.Value()
	require.ErrorIs(t, err, encryption.ErrEncryptionNotInitialized)
	require.ErrorIs(t, new(models.MySQLOptions).Scan([]byte(`{"tls_key": "secret"}`)), encryption.ErrEncryptionNotInitialized)
}
