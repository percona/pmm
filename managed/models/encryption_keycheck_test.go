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
	"os"
	"path/filepath"
	"strconv"
	"testing"

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
	stored, err := lost.Encrypt("password-before-key-loss")
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
