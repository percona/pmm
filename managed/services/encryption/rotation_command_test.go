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

package encryption

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

// fakeSupervisorctl puts a supervisorctl first on PATH that reports
// pmm-managed as RUNNING and exits with restartExit on restart.
func fakeSupervisorctl(t *testing.T, restartExit int) {
	t.Helper()

	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = restart ]; then exit %d; fi\necho 'pmm-managed RUNNING pid 1, uptime 0:00:01'\n", restartExit)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "supervisorctl"), []byte(script), 0o700))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// keyFile creates a key file at PMM_ENCRYPTION_KEY_PATH and returns its path
// and primary key ID.
func keyFile(t *testing.T) (string, uint32) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "encryption.key")
	c, err := encryption.CreateCipher(encryption.NewFileKeyProvider(path))
	require.NoError(t, err)
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, path)

	return path, c.PrimaryKeyID()
}

func keysetIDs(t *testing.T, path string) []uint32 {
	t.Helper()

	handle, err := encryption.NewFileKeyProvider(path).Load()
	require.NoError(t, err)
	ids := make([]uint32, 0, handle.Len())
	for i := range handle.Len() {
		entry, err := handle.Entry(i)
		require.NoError(t, err)
		ids = append(ids, entry.KeyID())
	}

	return ids
}

// TestRotateEncryptionKeyCommand runs the pmm-encryption-rotation flow with a
// fake supervisorctl; the restarted pmm-managed has nothing to re-encrypt.
func TestRotateEncryptionKeyCommand(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})
	// still live when a subtest's cleanup runs, unlike the subtest's own
	ctx := t.Context()
	// set up with the test key, unlike the subtests' key files; the restart is
	// fake, so no startup migration re-encrypts the key check
	_, err := sqlDB.ExecContext(ctx, "UPDATE settings SET settings = settings - 'encryption_key_check'")
	require.NoError(t, err)

	t.Run("adds a primary key and keeps the old one", func(t *testing.T) {
		fakeSupervisorctl(t, 0)
		path, oldKeyID := keyFile(t)

		code, err := RotateEncryptionKey(sqlDB, RotationParams{})
		require.NoError(t, err)
		assert.Equal(t, codeOK, code)

		ids := keysetIDs(t, path)
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, oldKeyID)
		rotated, err := encryption.LoadCipher(encryption.NewFileKeyProvider(path))
		require.NoError(t, err)
		assert.NotEqual(t, oldKeyID, rotated.PrimaryKeyID())
	})

	t.Run("prunes retired keys", func(t *testing.T) {
		fakeSupervisorctl(t, 0)
		path, oldKeyID := keyFile(t)

		code, err := RotateEncryptionKey(sqlDB, RotationParams{Prune: true})
		require.NoError(t, err)
		assert.Equal(t, codeOK, code)

		ids := keysetIDs(t, path)
		require.Len(t, ids, 1)
		assert.NotEqual(t, oldKeyID, ids[0])
	})

	t.Run("restart fails", func(t *testing.T) {
		fakeSupervisorctl(t, 1)
		keyFile(t)

		code, err := RotateEncryptionKey(sqlDB, RotationParams{})
		require.Error(t, err)
		assert.Equal(t, codeRestartFailed, code)
	})

	t.Run("sweep fails on data the key cannot decrypt", func(t *testing.T) {
		fakeSupervisorctl(t, 0)
		keyFile(t)
		other, err := encryption.CreateCipher(encryption.NewFileKeyProvider(filepath.Join(t.TempDir(), "other.key")))
		require.NoError(t, err)
		stored, err := other.Encrypt("password")
		require.NoError(t, err)
		now := time.Now()
		_, err = sqlDB.ExecContext(t.Context(),
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = sqlDB.ExecContext(t.Context(),
			`INSERT INTO agents (agent_id, agent_type, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify) `+
				`VALUES ('PA', 'pmm-agent', $1, 'N1', false, '', $2, $3, false, false)`, stored, now, now)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := sqlDB.ExecContext(ctx, `DELETE FROM agents; DELETE FROM nodes`)
			assert.NoError(t, err)
		})

		code, err := RotateEncryptionKey(sqlDB, RotationParams{Prune: true})
		require.ErrorContains(t, err, "cannot decrypt stored credentials")
		assert.Equal(t, codeSweepFailed, code)
	})

	t.Run("refused in HA while other nodes may run", func(t *testing.T) {
		fakeSupervisorctl(t, 0)
		path, oldKeyID := keyFile(t)
		t.Setenv("PMM_HA_ENABLE", "1")

		code, err := RotateEncryptionKey(sqlDB, RotationParams{Prune: true})
		require.ErrorContains(t, err, "stop PMM Server on every other node first")
		require.ErrorContains(t, err, "--ha-other-nodes-stopped")
		assert.Equal(t, codeRotationFailed, code)
		assert.Equal(t, []uint32{oldKeyID}, keysetIDs(t, path), "the keyset is unchanged")
	})

	t.Run("HA with the other nodes stopped", func(t *testing.T) {
		fakeSupervisorctl(t, 0)
		path, oldKeyID := keyFile(t)
		t.Setenv("PMM_HA_ENABLE", "1")

		code, err := RotateEncryptionKey(sqlDB, RotationParams{OtherHANodesStopped: true})
		require.NoError(t, err)
		assert.Equal(t, codeOK, code)
		ids := keysetIDs(t, path)
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, oldKeyID)
	})

	t.Run("no key file", func(t *testing.T) {
		fakeSupervisorctl(t, 0)
		t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "missing.key"))

		code, err := RotateEncryptionKey(sqlDB, RotationParams{})
		require.ErrorIs(t, err, encryption.ErrKeysetNotFound)
		assert.Equal(t, codeRotationFailed, code)
	})
}
