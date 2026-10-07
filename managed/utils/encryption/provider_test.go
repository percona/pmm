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
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/mac"
)

// TestGenerateKeyset pins that `pmm-encryption-rotation --generate-key`
// output is a key file PMM can load and use.
func TestGenerateKeyset(t *testing.T) {
	encoded, err := GenerateKeyset()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "encryption.key")
	require.NoError(t, os.WriteFile(path, []byte(encoded+"\n"), 0o600))

	c, err := LoadCipher(NewFileKeyProvider(path))
	require.NoError(t, err)
	stored, err := c.Encrypt("secret")
	require.NoError(t, err)
	decrypted, err := c.Decrypt(stored)
	require.NoError(t, err)
	assert.Equal(t, "secret", decrypted)
}

func TestFileKeyProviderLoadErrors(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}

	for name, path := range map[string]string{
		"directory":      dir,
		"not base64":     write("not-base64.key", "%%% not base64 %%%"),
		"not a keyset":   write("not-keyset.key", base64.StdEncoding.EncodeToString([]byte("not a keyset"))),
		"empty key file": write("empty.key", ""),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadCipher(NewFileKeyProvider(path))
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrKeysetNotFound, "a broken key file must not look like a missing one")

			// a broken key file is never replaced by a new key
			_, err = CreateCipher(NewFileKeyProvider(path))
			require.Error(t, err)
		})
	}
}

func TestDefaultKeyPath(t *testing.T) {
	t.Setenv(CustomEncryptionKeyPathEnvVar, "")
	assert.Equal(t, DefaultEncryptionKeyPath, DefaultKeyPath())

	t.Setenv(CustomEncryptionKeyPathEnvVar, "/custom/encryption.key")
	assert.Equal(t, "/custom/encryption.key", DefaultKeyPath())
}

func TestStoreInMissingDirectory(t *testing.T) {
	provider := NewFileKeyProvider(filepath.Join(t.TempDir(), "missing", "encryption.key"))

	_, err := CreateCipher(provider)
	require.Error(t, err)

	handle, err := keyset.NewHandle(mac.HMACSHA256Tag256KeyTemplate())
	require.NoError(t, err)
	require.Error(t, provider.Store(handle))
}

func TestRotationWithoutKeyset(t *testing.T) {
	provider := NewFileKeyProvider(filepath.Join(t.TempDir(), "encryption.key"))

	_, err := AddNewPrimaryKey(provider)
	require.ErrorIs(t, err, ErrKeysetNotFound)
	_, err = PruneRetiredKeys(provider)
	require.ErrorIs(t, err, ErrKeysetNotFound)
}

// handleProvider serves a fixed keyset.
type handleProvider struct {
	handle *keyset.Handle
}

func (p handleProvider) Load() (*keyset.Handle, error) { return p.handle, nil }

func (p handleProvider) Store(*keyset.Handle) error { return nil }

// TestNonAEADKeyset covers a key file holding a valid Tink keyset of the wrong
// primitive: it must be rejected, not used.
func TestNonAEADKeyset(t *testing.T) {
	handle, err := keyset.NewHandle(mac.HMACSHA256Tag256KeyTemplate())
	require.NoError(t, err)

	_, err = LoadCipher(handleProvider{handle})
	require.Error(t, err)

	c := newTestCipher(t)
	_, err = c.WithLegacyKeys(handleProvider{handle})
	require.Error(t, err)
	_, err = c.WithLegacyKeys(NewFileKeyProvider(filepath.Join(t.TempDir(), "missing.key")))
	require.ErrorIs(t, err, ErrKeysetNotFound)
}

func TestSyncDir(t *testing.T) {
	require.NoError(t, syncDir(t.TempDir()))
	require.ErrorIs(t, syncDir(filepath.Join(t.TempDir(), "missing")), os.ErrNotExist, "a failure is reported")
}
