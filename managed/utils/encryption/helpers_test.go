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
	"database/sql"
	"path/filepath"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrepareRowPointersBoolIdentifier verifies that a BOOLEAN identifier column (e.g. the
// adre_provisioning singleton id) is scanned into a *sql.NullBool and that pointers are returned in
// declared SELECT order even when column types are mixed.
func TestPrepareRowPointersBoolIdentifier(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	cols := []*sqlmock.Column{
		sqlmock.NewColumn("id").OfType("BOOL", true),
		sqlmock.NewColumn("holmes_api_key").OfType("VARCHAR", ""),
		sqlmock.NewColumn("pmm_sa_token").OfType("VARCHAR", ""),
	}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRowsWithColumnDefinition(cols...).AddRow(true, "enc1", "enc2"),
	)

	rows, err := db.Query("SELECT id, holmes_api_key, pmm_sa_token FROM adre_provisioning")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	require.True(t, rows.Next())

	ptrs, err := prepareRowPointers(rows)
	require.NoError(t, err)
	require.Len(t, ptrs, 3)

	// Ordered: BOOL identifier first, then the two VARCHAR secret columns.
	_, isBool := ptrs[0].(*sql.NullBool)
	assert.True(t, isBool, "id should scan into *sql.NullBool")
	for i := 1; i < 3; i++ {
		_, isStr := ptrs[i].(*sql.NullString)
		assert.Truef(t, isStr, "column %d should scan into *sql.NullString", i)
	}

	// Scanning into the prepared pointers must succeed (mixed types, correct order).
	require.NoError(t, rows.Scan(ptrs...))
	assert.True(t, ptrs[0].(*sql.NullBool).Bool)
	assert.Equal(t, "enc1", ptrs[1].(*sql.NullString).String)
}

// TestPrepareRowPointersUnsupportedType ensures unsupported column types are still rejected.
func TestPrepareRowPointersUnsupportedType(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRowsWithColumnDefinition(
			sqlmock.NewColumn("n").OfType("INT8", int64(0)),
		).AddRow(int64(1)),
	)
	rows, err := db.Query("SELECT n FROM t")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	require.True(t, rows.Next())

	_, err = prepareRowPointers(rows)
	assert.Error(t, err)
}

// TestEncryptDecryptRoundTrip covers the app-layer secret round-trip and the graceful behaviour on
// non-ciphertext input that the ADRE helpers rely on.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	e := &Encryption{Path: filepath.Join(t.TempDir(), "test.key")}
	require.NoError(t, e.generateAndPersistKey())
	prim, err := e.getPrimitive()
	require.NoError(t, err)
	e.Primitive = prim

	for _, s := range []string{"", "xoxb-secret", "another-token"} {
		enc, err := e.Encrypt(s)
		require.NoError(t, err)
		if s != "" {
			assert.NotEqual(t, s, enc, "ciphertext must differ from plaintext")
		}
		dec, err := e.Decrypt(enc)
		require.NoError(t, err)
		assert.Equal(t, s, dec)
	}

	// Decrypting non-ciphertext returns the original value plus an error (graceful fallback used by
	// decryptField so legacy/garbled values degrade to themselves rather than crashing).
	out, err := e.Decrypt("not-ciphertext")
	assert.Error(t, err)
	assert.Equal(t, "not-ciphertext", out)
}
