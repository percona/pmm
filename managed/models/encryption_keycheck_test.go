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
	"strconv"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

// newKeyFile creates a key file in a directory of its own.
func newKeyFile(t *testing.T) (string, *encryption.Cipher) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "pmm-encryption.key")
	c, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)

	return path, c
}

// acceptKeyLoss is the setting that accepts the loss of the key the data was
// encrypted with, for the key of the cipher.
func acceptKeyLoss(c *encryption.Cipher) string {
	return encryption.AcceptKeyLossEnvVar + "=" + strconv.FormatUint(uint64(c.PrimaryKeyID()), 10)
}

// setAcceptKeyLoss accepts the loss for the key of the cipher.
func setAcceptKeyLoss(t *testing.T, c *encryption.Cipher) {
	t.Helper()

	t.Setenv(encryption.AcceptKeyLossEnvVar, strconv.FormatUint(uint64(c.PrimaryKeyID()), 10))
}

// setupHADB runs pmm-managed's startup as a node of an HA cluster.
func setupHADB(t *testing.T, sqlDB *sql.DB) error {
	t.Helper()

	_, err := models.SetupDB(t.Context(), sqlDB, models.SetupDBParams{
		Logf:          t.Logf,
		Address:       models.DefaultPostgreSQLAddr,
		Username:      "postgres",
		HANodeID:      "pmm-server-2",
		SetupFixtures: models.SkipFixtures,
	})

	return err
}

// newPMMServerDB describes a database PMM Server connects to with TLS, whose
// files hold their names.
func newPMMServerDB(t *testing.T) models.SetupDBParams {
	t.Helper()

	params := models.SetupDBParams{
		Address:  models.DefaultPostgreSQLAddr,
		Username: "pmm-managed",
		Password: "pmm-managed-password",
		SSLMode:  models.RequireSSLMode,
	}
	dir := t.TempDir()
	for name, path := range map[string]*string{"ssl-ca": &params.SSLCAPath, "ssl-cert": &params.SSLCertPath, "ssl-key": &params.SSLKeyPath} {
		*path = filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(*path, []byte(name), 0o600))
	}

	return params
}

// setupDBWithFixtures runs pmm-managed's startup like setupDB, but with PMM
// Server's own Node, Service and Agents, connected to the given database.
func setupDBWithFixtures(t *testing.T, sqlDB *sql.DB, params models.SetupDBParams) error {
	t.Helper()

	params.Logf = t.Logf
	params.SetupFixtures = models.SetupFixtures
	_, err := models.SetupDB(t.Context(), sqlDB, params)

	return err
}

// pmmServerPostgreSQLAgents returns the Agents of PMM Server's own PostgreSQL
// Service.
func pmmServerPostgreSQLAgents(t *testing.T, q *reform.Querier) []*models.Agent {
	t.Helper()

	service, err := models.FindServiceByName(q, models.PMMServerPostgreSQLServiceName)
	require.NoError(t, err)
	agents, err := models.FindAgents(q, models.AgentFilters{ServiceID: service.ServiceID})
	require.NoError(t, err)

	return agents
}

func backupFiles(t *testing.T, dir string) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, models.MigrationBackupPattern))
	require.NoError(t, err)

	return files
}

// TestKeyCheckRefusesAnotherKey covers PMM-14979 on a fresh HA database: it
// holds no secrets yet, so only the key check tells a node that generated its
// own key from one that shares the key of the node that set the database up.
func TestKeyCheckRefusesAnotherKey(t *testing.T) {
	// set up by the first node, with the test key
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	firstNodeKey := os.Getenv(encryption.CustomEncryptionKeyPathEnvVar)
	require.NotEmpty(t, firstNodeKey)
	check := storedKeyCheck(t, sqlDB)
	require.NotEmpty(t, check, "stored at the first start")

	otherKey, other := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, otherKey)
	err := setupDB(t, sqlDB)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, "encryption key check")
	require.ErrorContains(t, err, acceptKeyLoss(other), "names the setting that accepts the loss")
	assert.Equal(t, check, storedKeyCheck(t, sqlDB), "nothing is changed")

	err = setupHADB(t, sqlDB)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	assert.NotContains(t, err.Error(), encryption.AcceptKeyLossEnvVar, "never offered in HA")

	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, firstNodeKey)
	require.NoError(t, setupDB(t, sqlDB), "a node with the shared key starts")
}

// TestKeyCheckHANodeWithoutKey covers an HA node without a key file joining a
// database another node set up: generating its own key would split the
// cluster, so it refuses. A standalone server generates one, see
// TestSetupDBCreatesMissingKey.
func TestKeyCheckHANodeWithoutKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	path := filepath.Join(t.TempDir(), "pmm-encryption.key")
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)

	err := setupHADB(t, sqlDB)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, "encryption key not found")
	assert.NotContains(t, err.Error(), encryption.AcceptKeyLossEnvVar, "never offered in HA")
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "no key must be generated")
}

// TestKeyCheckFollowsRotation covers the key check through key rotation: the
// startup migration re-encrypts it with the new primary key, so pruning the
// retired keys keeps the server starting, while a node that kept the key from
// before the rotation refuses.
func TestKeyCheckFollowsRotation(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	forgetKeyCheck(t, sqlDB)
	path, _ := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)

	require.NoError(t, setupDB(t, sqlDB))
	assert.NotEmpty(t, storedKeyCheck(t, sqlDB), "stored at the first start after the upgrade")
	beforeRotation, err := os.ReadFile(path)
	require.NoError(t, err)

	provider := encryption.NewFileKeyProvider(path)
	_, err = encryption.AddNewPrimaryKey(provider)
	require.NoError(t, err)
	rotated, err := encryption.LoadCipher(provider)
	require.NoError(t, err)
	stale, err := models.KeyCheckNeedsReencryption(q, rotated)
	require.NoError(t, err)
	assert.True(t, stale, "until the restart")

	require.NoError(t, setupDB(t, sqlDB))
	stale, err = models.KeyCheckNeedsReencryption(q, rotated)
	require.NoError(t, err)
	assert.False(t, stale)

	_, err = encryption.PruneRetiredKeys(provider)
	require.NoError(t, err)
	require.NoError(t, setupDB(t, sqlDB))

	oldKey := filepath.Join(t.TempDir(), "pmm-encryption.key")
	require.NoError(t, os.WriteFile(oldKey, beforeRotation, 0o600))
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, oldKey)
	require.ErrorIs(t, setupDB(t, sqlDB), models.ErrEncryptionKeyMismatch)
}

// TestAcceptKeyLossReplacesKeyCheck covers a server started with a key that
// is not the one its database was set up with, after the administrator has
// accepted that the original is lost.
func TestAcceptKeyLossReplacesKeyCheck(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	check := storedKeyCheck(t, sqlDB)
	path, newKey := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	setAcceptKeyLoss(t, newKey)

	require.NoError(t, setupDB(t, sqlDB))
	assert.NotEqual(t, check, storedKeyCheck(t, sqlDB))
	stale, err := models.KeyCheckNeedsReencryption(reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier, newKey)
	require.NoError(t, err)
	assert.False(t, stale)

	t.Setenv(encryption.AcceptKeyLossEnvVar, "")
	require.NoError(t, setupDB(t, sqlDB), "the new key is the database's key now")
}

// TestAcceptKeyLossLegacy covers a key lost under PMM 3.x with no service
// added since: the key PMM 3.x generated in its place reads none of the stored
// values, so the upgrade refuses until the administrator accepts the loss.
func TestAcceptKeyLossLegacy(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	forgetKeyCheck(t, sqlDB)
	dir := t.TempDir()
	path := filepath.Join(dir, "pmm-encryption.key")
	current, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	lostPath, lost := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)

	_, err = sqlDB.ExecContext(t.Context(),
		`UPDATE settings SET settings = settings || '{"encrypted_items": ["pmm-managed.agents.password"]}'::jsonb`)
	require.NoError(t, err)
	stale := legacyCiphertext(t, lost, "password-before-key-loss")
	insertExporter(t, sqlDB, "E1", stale, "")

	err = setupDB(t, sqlDB)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, acceptKeyLoss(current))
	assert.Equal(t, stale, storedPassword(t, sqlDB, "E1"), "nothing is changed")

	setAcceptKeyLoss(t, current)
	require.NoError(t, setupDB(t, sqlDB))
	keyID, ok := encryption.StoredKeyID(storedPassword(t, sqlDB, "E1"))
	require.True(t, ok)
	assert.Equal(t, current.PrimaryKeyID(), keyID)
	files := backupFiles(t, dir)
	require.Len(t, files, 1)
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), stale)

	// the lost key placed next to the key file makes the value readable again
	lostKey, err := os.ReadFile(lostPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(encryption.LegacyBackupKeyPath(path), lostKey, 0o600))
	t.Setenv(encryption.AcceptKeyLossEnvVar, "")
	require.NoError(t, setupDB(t, sqlDB))
	agent, err := models.FindAgentByID(reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier, "E1")
	require.NoError(t, err)
	assert.Equal(t, "password-before-key-loss", agent.Password.Reveal())
}

// TestAcceptKeyLossMissingKey covers a server whose key file is gone for good
// after the upgrade: its secrets are envelopes of the lost key. Accepting the
// loss takes a key to accept it for, so the administrator creates one first.
func TestAcceptKeyLossMissingKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	forgetKeyCheck(t, sqlDB)
	dir := t.TempDir()
	path := filepath.Join(dir, "pmm-encryption.key")
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)

	// set up with the key that gets lost
	lost, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	require.NoError(t, setupDB(t, sqlDB))
	stored, err := lost.Encrypt("password-before-key-loss", []byte("agents.password"))
	require.NoError(t, err)
	insertExporter(t, sqlDB, "E1", stored, "")
	lostKey, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))

	err = setupDB(t, sqlDB)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, "encryption key not found")
	require.ErrorContains(t, err, "--generate-key")
	require.ErrorContains(t, err, encryption.AcceptKeyLossEnvVar)
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "no key must be generated")

	created, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	err = setupDB(t, sqlDB)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, acceptKeyLoss(created))
	assert.Equal(t, stored, storedPassword(t, sqlDB, "E1"), "nothing is changed")

	setAcceptKeyLoss(t, created)
	require.NoError(t, setupDB(t, sqlDB))
	keyID, ok := encryption.StoredKeyID(storedPassword(t, sqlDB, "E1"))
	require.True(t, ok)
	assert.Equal(t, created.PrimaryKeyID(), keyID)
	ids, err := models.AgentsNeedingReencryption(q, created)
	require.NoError(t, err)
	assert.Empty(t, ids, "the sweep converges")
	files := backupFiles(t, dir)
	require.Len(t, files, 1)
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), stored)

	// the lost key placed next to the key file makes the value readable again
	require.NoError(t, os.WriteFile(encryption.LegacyBackupKeyPath(path), lostKey, 0o600))
	t.Setenv(encryption.AcceptKeyLossEnvVar, "")
	require.NoError(t, setupDB(t, sqlDB))
	agent, err := models.FindAgentByID(q, "E1")
	require.NoError(t, err)
	assert.Equal(t, "password-before-key-loss", agent.Password.Reveal())
	keyID, ok = encryption.StoredKeyID(storedPassword(t, sqlDB, "E1"))
	require.True(t, ok)
	assert.Equal(t, created.PrimaryKeyID(), keyID)
}

// TestAcceptKeyLossPMMServerAgents covers PMM Server's own PostgreSQL Service
// after its key is lost: PMM Server set up its Agents with the credentials of
// its own database, which no user knows, so they get those back. The Agent of
// a user's Service keeps its lost value until the user re-enters it.
func TestAcceptKeyLossPMMServerAgents(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	forgetKeyCheck(t, sqlDB)
	path := filepath.Join(t.TempDir(), "pmm-encryption.key")
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	db := newPMMServerDB(t)

	// set up with the key that gets lost
	lost, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
	agents := pmmServerPostgreSQLAgents(t, q)
	require.Len(t, agents, 2, "postgres_exporter and QAN")
	stored, err := lost.Encrypt("password-before-key-loss", []byte("agents.password"))
	require.NoError(t, err)
	insertExporter(t, sqlDB, "E1", stored, "")
	lookAlikes := pmmServerAgentLookAlikes(t, q, pointer.GetString(agents[0].ServiceID))
	require.NoError(t, os.Remove(path))

	created, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	setAcceptKeyLoss(t, created)
	hook := logtest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))

	warning := unrecoverableWarning(t, hook)
	assert.Contains(t, warning, "agent E1 password")
	for _, agent := range agents {
		assert.NotContains(t, warning, agent.AgentID, agent.AgentType)
		agent, err = models.FindAgentByID(q, agent.AgentID)
		require.NoError(t, err)
		assert.Equal(t, "pmm-managed", agent.Username.Reveal(), agent.AgentType)
		assert.Equal(t, "pmm-managed-password", agent.Password.Reveal(), agent.AgentType)
		assert.Equal(t, "ssl-ca", agent.PostgreSQLOptions.SSLCa, agent.AgentType)
		assert.Equal(t, "ssl-cert", agent.PostgreSQLOptions.SSLCert, agent.AgentType)
		assert.Equal(t, "ssl-key", agent.PostgreSQLOptions.SSLKey, agent.AgentType)
		keyID, ok := encryption.StoredKeyID(storedPassword(t, sqlDB, agent.AgentID))
		require.True(t, ok)
		assert.Equal(t, created.PrimaryKeyID(), keyID, agent.AgentType)
	}
	agent, err := models.FindAgentByID(q, "E1")
	require.NoError(t, err)
	assert.Equal(t, stored, agent.Password.Reveal(), "a user's Agent keeps the lost value")
	for name, id := range lookAlikes {
		agent, err = models.FindAgentByID(q, id)
		require.NoError(t, err)
		assert.NotEqual(t, "pmm-managed-password", agent.Password.Reveal(), name)
		assert.Contains(t, warning, "agent "+id+" password", name)
	}

	t.Setenv(encryption.AcceptKeyLossEnvVar, "")
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db), "the new key is the database's key now")
}

// pmmServerAgentLookAlikes adds Agents that share one trait of the Agents of PMM Server's own PostgreSQL
// Service, with credentials encrypted with the default cipher, and returns their IDs by what they share.
func pmmServerAgentLookAlikes(t *testing.T, q *reform.Querier, pmmServerServiceID string) map[string]string {
	t.Helper()

	// at the address of PMM Server's database, so that only the name tells them apart
	remote, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "remote-postgresql",
		NodeID:      models.PMMServerNodeID,
		Address:     pointer.ToString(models.LocalhostAddr),
		Port:        pointer.ToUint16(5432),
	})
	require.NoError(t, err)

	ids := make(map[string]string)
	for name, params := range map[string]*models.CreateAgentParams{
		"PMM Server's pmm-agent":          {PMMAgentID: models.PMMServerAgentID, ServiceID: remote.ServiceID},
		"PMM Server's PostgreSQL Service": {PMMAgentID: "PA", ServiceID: pmmServerServiceID},
	} {
		params.Username = "user"
		params.Password = "user-password"
		agent, err := models.CreateAgent(q, models.PostgresExporterType, params)
		require.NoError(t, err, name)
		ids[name] = agent.AgentID
	}

	return ids
}

// unrecoverableWarning returns the warning the startup migration logs about credentials it cannot recover.
func unrecoverableWarning(t *testing.T, hook *logtest.Hook) string {
	t.Helper()

	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.WarnLevel && strings.Contains(entry.Message, "cannot be recovered") {
			return entry.Message
		}
	}
	require.Fail(t, "no warning about credentials that cannot be recovered")

	return ""
}

// TestAcceptKeyLossServiceNameTaken covers a user's PostgreSQL Service under the name of PMM Server's own,
// e.g. after the built-in one was removed: its Agents must not get the credentials of PMM Server's database.
func TestAcceptKeyLossServiceNameTaken(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	forgetKeyCheck(t, sqlDB)
	path := filepath.Join(t.TempDir(), "pmm-encryption.key")
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	db := newPMMServerDB(t)

	_, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
	_, err = sqlDB.ExecContext(t.Context(), `UPDATE services SET address = '192.0.2.1' WHERE service_name = $1`,
		models.PMMServerPostgreSQLServiceName)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))

	created, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	setAcceptKeyLoss(t, created)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))

	for _, agent := range pmmServerPostgreSQLAgents(t, q) {
		assert.NotEqual(t, "pmm-managed-password", agent.Password.Reveal(), agent.AgentType)
	}
}

// TestRestorePMMServerAgentsLostColumnsOnly covers a secret of PMM Server's own PostgreSQL Agents lost to the
// stacked layers of PMM before 3.9.1 (PMM-15188): only that column is restored, not credentials still readable.
func TestRestorePMMServerAgentsLostColumnsOnly(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	cipher := keepDefaultCipher(t)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	db := newPMMServerDB(t)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
	exporter := pmmServerPostgreSQLAgents(t, q)[0]
	exporter.Username = models.EncryptedStringOrNil[models.AgentUsername]("custom-user")
	exporter.Password = models.EncryptedStringOrNil[models.AgentPassword]("custom-password")
	require.NoError(t, q.Update(exporter))

	// as PMM before 3.9.1 left it, before the upgrade
	forgetKeyCheck(t, sqlDB)
	_, gone := newKeyFile(t)
	options, err := json.Marshal(map[string]string{
		"ssl_cert": legacyCiphertext(t, cipher, legacyCiphertext(t, gone, "old-cert")),
		"ssl_key":  legacyCiphertext(t, cipher, legacyCiphertext(t, gone, "old-key")),
	})
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(), `UPDATE agents SET postgresql_options = $1 WHERE agent_id = $2`,
		string(options), exporter.AgentID)
	require.NoError(t, err)

	hook := logtest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))

	var restoreWarnings []string
	for _, entry := range hook.AllEntries() {
		if strings.HasPrefix(entry.Message, "Restored the lost") {
			restoreWarnings = append(restoreWarnings, entry.Message)
		}
	}
	require.Len(t, restoreWarnings, 1)
	assert.Contains(t, restoreWarnings[0], "Restored the lost postgresql_options of PMM Server's own PostgreSQL agent "+exporter.AgentID)
	exporter, err = models.FindAgentByID(q, exporter.AgentID)
	require.NoError(t, err)
	assert.Equal(t, "ssl-cert", exporter.PostgreSQLOptions.SSLCert)
	assert.Equal(t, "ssl-key", exporter.PostgreSQLOptions.SSLKey)
	assert.Equal(t, "custom-user", exporter.Username.Reveal())
	assert.Equal(t, "custom-password", exporter.Password.Reveal())
}

// TestAcceptKeyLossMissingTLSFiles covers TLS files of PMM Server's database that no longer exist, which its own
// connection tolerates: a lost credential of a user's Agent must not make startup read them.
func TestAcceptKeyLossMissingTLSFiles(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	db := newPMMServerDB(t)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
	_, gone := newKeyFile(t)
	stored, err := gone.Encrypt("password-before-key-loss", []byte("agents.password"))
	require.NoError(t, err)
	insertExporter(t, sqlDB, "E1", stored, "")
	require.NoError(t, os.Remove(db.SSLKeyPath))

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)
	setAcceptKeyLoss(t, cipher)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
}

// TestAcceptKeyLossLegacyPMMServerAgents is TestAcceptKeyLossPMMServerAgents
// for a key lost under PMM 3.x, see TestAcceptKeyLossLegacy.
func TestAcceptKeyLossLegacyPMMServerAgents(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier
	forgetKeyCheck(t, sqlDB)
	path, current := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	db := newPMMServerDB(t)
	db.SSLMode = models.DisableSSLMode
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
	forgetKeyCheck(t, sqlDB)

	// as PMM 3.x stored them, with the key that got lost
	_, lost := newKeyFile(t)
	_, err := sqlDB.ExecContext(t.Context(), `UPDATE settings SET settings = settings || `+
		`'{"encrypted_items": ["pmm-managed.agents.username", "pmm-managed.agents.password"]}'::jsonb`)
	require.NoError(t, err)
	for _, agent := range pmmServerPostgreSQLAgents(t, q) {
		_, err = sqlDB.ExecContext(t.Context(), `UPDATE agents SET username = $1, password = $2 WHERE agent_id = $3`,
			legacyCiphertext(t, lost, "pmm-managed"), legacyCiphertext(t, lost, "pmm-managed-password"), agent.AgentID)
		require.NoError(t, err)
	}

	err = setupDBWithFixtures(t, sqlDB, db)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, acceptKeyLoss(current))

	setAcceptKeyLoss(t, current)
	require.NoError(t, setupDBWithFixtures(t, sqlDB, db))
	agents := pmmServerPostgreSQLAgents(t, q)
	require.Len(t, agents, 2, "postgres_exporter and QAN")
	for _, agent := range agents {
		assert.Equal(t, "pmm-managed", agent.Username.Reveal(), agent.AgentType)
		assert.Equal(t, "pmm-managed-password", agent.Password.Reveal(), agent.AgentType)
	}
}

// TestAcceptKeyLossNamesTheKey covers the setting left behind after a
// recovery: it accepts the loss for the key it names only, so a key file
// replaced later is refused again.
func TestAcceptKeyLossNamesTheKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	check := storedKeyCheck(t, sqlDB)
	path, newKey := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	_, earlier := newKeyFile(t)

	for _, value := range []string{"1", "true", strconv.FormatUint(uint64(earlier.PrimaryKeyID()), 10)} {
		t.Setenv(encryption.AcceptKeyLossEnvVar, value)
		err := setupDB(t, sqlDB)
		require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch, value)
		require.ErrorContains(t, err, acceptKeyLoss(newKey), value)
		assert.Equal(t, check, storedKeyCheck(t, sqlDB), "nothing is changed with %s", value)
	}
}

// TestAcceptKeyLossIgnoredInHA covers the setting on an HA node with a key of
// its own: the other nodes still hold the database's key, so accepting its
// loss would rewrite the data they read. The node refuses like without it.
func TestAcceptKeyLossIgnoredInHA(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	check := storedKeyCheck(t, sqlDB)
	path, other := newKeyFile(t)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)
	setAcceptKeyLoss(t, other)

	require.ErrorIs(t, setupHADB(t, sqlDB), models.ErrEncryptionKeyMismatch)
	assert.Equal(t, check, storedKeyCheck(t, sqlDB), "nothing is changed")
}

// TestHALegacyValuesOfAnotherNodeKey covers an HA database holding legacy
// ciphertext of two keys, as HA nodes of older PMM versions could each use
// their own key. Unlike a standalone server after a key loss, the node must
// not rewrite the other node's values as lost, since that node still holds
// the key; placing it as the previous key makes them readable.
func TestHALegacyValuesOfAnotherNodeKey(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	keepDefaultCipher(t)
	current, err := encryption.DefaultCipher()
	require.NoError(t, err)

	// a key file of this test's own, so that the other key can be placed next to it
	data, err := os.ReadFile(os.Getenv(encryption.CustomEncryptionKeyPathEnvVar))
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "pmm-encryption.key")
	require.NoError(t, os.WriteFile(keyPath, data, 0o600))
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, keyPath)
	otherPath, other := newKeyFile(t)

	_, err = sqlDB.ExecContext(t.Context(),
		`UPDATE settings SET settings = settings || '{"encrypted_items": ["pmm-managed.agents.password"]}'::jsonb`)
	require.NoError(t, err)
	insertExporter(t, sqlDB, "own", legacyCiphertext(t, current, "own-password"), "")
	stale := legacyCiphertext(t, other, "other-node-password")
	insertExporter(t, sqlDB, "other", stale, "")

	err = setupHADB(t, sqlDB)
	require.ErrorIs(t, err, encryption.ErrLegacyUnknownKey)
	require.ErrorIs(t, err, models.ErrEncryptionKeyMismatch)
	require.ErrorContains(t, err, encryption.LegacyBackupKeyPath(keyPath))
	assert.Equal(t, stale, storedPassword(t, sqlDB, "other"), "nothing is rewritten")

	otherKey, err := os.ReadFile(otherPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(encryption.LegacyBackupKeyPath(keyPath), otherKey, 0o600))
	require.NoError(t, setupHADB(t, sqlDB))
	agent, err := models.FindAgentByID(reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier, "other")
	require.NoError(t, err)
	assert.Equal(t, "other-node-password", agent.Password.Reveal())
}
