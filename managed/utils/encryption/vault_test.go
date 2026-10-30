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
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

type keyStore map[string]string

func (s keyStore) Get(_ context.Context, path, key string) (string, error) {
	v, ok := s[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

func (s keyStore) WriteRef(_ context.Context, ref, value string) error {
	r, err := vault.ParseRef(ref)
	if err != nil {
		return err
	}
	s[r.Path+"#"+r.Key] = value
	return nil
}

// Not parallel: it sets the environment and the process-wide key and Vault client.
func TestKeyFromVault(t *testing.T) {
	const ref = "vault:secret/data/pmm/server#encryption_key"
	key, err := (&Encryption{}).GenerateKey()
	require.NoError(t, err)

	store := keyStore{"secret/data/pmm/server#encryption_key": key + "\n"}
	secrets.Configure(store, time.Minute)
	t.Setenv(KeyEnvVar, ref)
	t.Cleanup(func() {
		secrets.Configure(nil, 0)
		defaultEncryptionMtx.Lock()
		defaultEncryption = nil
		defaultEncryptionMtx.Unlock()
	})
	defaultEncryptionMtx.Lock()
	defaultEncryption = nil
	defaultEncryptionMtx.Unlock()

	e := New()
	assert.Equal(t, key, e.Key, "a trailing newline is not part of the key")
	assert.Empty(t, e.Path)
	_, err = os.Stat(KeyPath())
	require.ErrorIs(t, err, os.ErrNotExist, "no key file is written")

	encrypted, err := Encrypt("s3cr3t")
	require.NoError(t, err)

	// Rotation stores the new key in Vault, and a failed one puts the old key back there.
	require.NoError(t, RotateEncryptionKey())
	rotated := store["secret/data/pmm/server#encryption_key"]
	assert.NotEqual(t, key+"\n", rotated)
	assert.Equal(t, rotated, getDefaultEncryption().Key)

	_, err = Decrypt(encrypted)
	require.Error(t, err, "the new key does not decrypt what the old one encrypted")

	require.NoError(t, RestoreOldEncryptionKey())
	assert.Equal(t, key, store["secret/data/pmm/server#encryption_key"])

	t.Run("KeyNotInVault", func(t *testing.T) {
		t.Setenv(KeyEnvVar, key)
		assert.Equal(t, key, New().Key)
		require.ErrorContains(t, RotateEncryptionKey(), "store it in Vault to rotate it")
	})
}
