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
	"path/filepath"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/utils/encryption"
)

// TestCheckKeyGeneratedKey covers a key generated at this start while the
// database holds the key check of another key. A standalone server replaces
// it. An HA node refuses: another node set the database up after this one
// found no key check, and this node must use that node's key.
func TestCheckKeyGeneratedKey(t *testing.T) {
	t.Parallel()

	newCipher := func(name string) *encryption.Cipher {
		c, err := encryption.CreateCipher(encryption.NewFileKeyProvider(filepath.Join(t.TempDir(), name)))
		require.NoError(t, err)

		return c
	}
	generated := newCipher("generated.key")
	check, err := newCipher("other.key").Encrypt(keyCheckPlaintext)
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		ha       bool
		write    bool
		mismatch bool
	}{
		{name: "standalone replaces the key check", write: true},
		{name: "HA refuses", ha: true, mismatch: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = mock.ExpectClose()
				assert.NoError(t, sqlDB.Close())
			})
			mock.ExpectQuery(regexp.QuoteMeta(selectKeyCheck)).
				WillReturnRows(sqlmock.NewRows([]string{"encryption_key_check"}).AddRow(check))
			q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

			write, mismatch, err := checkKey(q, generated, migrationParams{keyCreated: true, ha: tc.ha})
			require.NoError(t, err)
			assert.Equal(t, tc.write, write)
			assert.Equal(t, tc.mismatch, mismatch)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
