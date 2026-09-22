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

package models

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/utils/encryption"
)

// TestVerifyEncryptionKey covers detection of the HA misconfiguration behind
// https://perconadev.atlassian.net/browse/PMM-14979, where each node generated its own
// encryption key while sharing one database.
func TestVerifyEncryptionKey(t *testing.T) {
	localFingerprint, err := encryption.Fingerprint()
	require.NoError(t, err)
	require.NotEmpty(t, localFingerprint)

	foreignCiphertext := base64.StdEncoding.EncodeToString([]byte("encrypted-with-another-key"))

	readableCiphertext, err := encryption.Encrypt("pmm-managed")
	require.NoError(t, err)

	settingsJSON := func(t *testing.T, s Settings) []byte {
		t.Helper()
		b, err := json.Marshal(s) //nolint:musttag
		require.NoError(t, err)

		return b
	}

	newMock := func(t *testing.T) (*reform.DB, sqlmock.Sqlmock) {
		t.Helper()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, mock.ExpectationsWereMet())
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		return reform.NewDB(sqlDB, postgresql.Dialect, nil), mock
	}

	expectSettings := func(t *testing.T, mock sqlmock.Sqlmock, s Settings) {
		t.Helper()
		mock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settingsJSON(t, s)))
	}

	verify := func(db *reform.DB) error {
		return db.InTransaction(VerifyEncryptionKey)
	}

	expectLock := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectExec("SELECT pg_advisory_xact_lock").
			WithArgs(encryptionKeyLockID).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	expectSave := func(mock sqlmock.Sqlmock) {
		mock.ExpectExec("UPDATE settings SET settings").
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	encryptedCredentials := []string{"pmm-managed.agents.username", "pmm-managed.agents.password"}

	t.Run("matching fingerprint is accepted", func(t *testing.T) {
		db, mock := newMock(t)
		expectSettings(t, mock, Settings{EncryptionKeyFingerprint: localFingerprint})

		assert.NoError(t, CheckEncryptionKey(db))
	})

	t.Run("matching fingerprint is not rewritten", func(t *testing.T) {
		db, mock := newMock(t)
		expectLock(mock)
		expectSettings(t, mock, Settings{EncryptionKeyFingerprint: localFingerprint})
		mock.ExpectCommit()

		assert.NoError(t, verify(db))
	})

	t.Run("foreign fingerprint is reported as a mismatch", func(t *testing.T) {
		db, mock := newMock(t)
		expectSettings(t, mock, Settings{EncryptionKeyFingerprint: "0123456789abcdef"})

		err := CheckEncryptionKey(db)
		require.ErrorIs(t, err, ErrEncryptionKeyMismatch)
		assert.Contains(t, err.Error(), localFingerprint)
		assert.Contains(t, err.Error(), "0123456789abcdef")
	})

	t.Run("fingerprint is recorded when nothing is encrypted yet", func(t *testing.T) {
		db, mock := newMock(t)
		expectLock(mock)
		expectSettings(t, mock, Settings{})
		expectSave(mock)
		mock.ExpectCommit()

		assert.NoError(t, verify(db))
	})

	t.Run("fingerprint is adopted when all stored credentials decrypt", func(t *testing.T) {
		db, mock := newMock(t)
		expectLock(mock)
		expectSettings(t, mock, Settings{EncryptedItems: encryptedCredentials})
		mock.ExpectQuery("SELECT username FROM agents").
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow(readableCiphertext).AddRow(readableCiphertext))
		mock.ExpectQuery("SELECT password FROM agents").
			WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(readableCiphertext))
		expectSave(mock)
		mock.ExpectCommit()

		assert.NoError(t, verify(db))
	})

	// The upgrade path for the reported cluster: a follower with its own key, against a database
	// whose rows were encrypted by another node and which carries no fingerprint yet.
	t.Run("fingerprint is not adopted when stored credentials cannot be decrypted", func(t *testing.T) {
		db, mock := newMock(t)
		expectLock(mock)
		expectSettings(t, mock, Settings{EncryptedItems: encryptedCredentials})
		mock.ExpectQuery("SELECT username FROM agents").
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow(foreignCiphertext))
		mock.ExpectQuery("SELECT password FROM agents").
			WillReturnRows(sqlmock.NewRows([]string{"password"}))
		mock.ExpectRollback()

		require.ErrorIs(t, verify(db), ErrEncryptionKeyMismatch)
	})

	// Rows written by several nodes with different keys: whichever row the database returns first
	// must not decide the outcome.
	t.Run("fingerprint is not adopted when any stored credential cannot be decrypted", func(t *testing.T) {
		db, mock := newMock(t)
		expectLock(mock)
		expectSettings(t, mock, Settings{EncryptedItems: encryptedCredentials})
		mock.ExpectQuery("SELECT username FROM agents").
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow(readableCiphertext).AddRow(foreignCiphertext))
		mock.ExpectQuery("SELECT password FROM agents").
			WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(readableCiphertext))
		mock.ExpectRollback()

		err := verify(db)
		require.ErrorIs(t, err, ErrEncryptionKeyMismatch)
		assert.Contains(t, err.Error(), "1 of 3")
	})

	t.Run("plaintext columns are not probed", func(t *testing.T) {
		db, mock := newMock(t)
		expectSettings(t, mock, Settings{EncryptedItems: []string{"pmm-managed.agents.aws_options"}})

		assert.NoError(t, CheckEncryptionKey(db))
	})
}
