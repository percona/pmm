// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/youmark/pkcs8"
)

func generateRSAKey(t *testing.T) []byte {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return privateKeyPEM
}

func generateEncryptedRSAKey(t *testing.T, password string) []byte {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyDER, err := pkcs8.MarshalPrivateKey(privateKey, []byte(password), nil)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ENCRYPTED PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return privateKeyPEM
}

func writeKey(t *testing.T, keyname string, key []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), keyname)
	require.NoError(t, os.WriteFile(path, key, 0o600))
	return path
}

func TestEncryption(t *testing.T) {
	t.Run("Encrypted", func(t *testing.T) {
		keyPEM := generateRSAKey(t)
		key := writeKey(t, "key", keyPEM)
		enc := Encryption{
			KeyFile: key,
		}
		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: enc})
		cfg, err := loadFromFile(configfilef, &enc)
		require.NoError(t, err)
		assert.Equal(t, &Config{ID: "agent-id"}, cfg)
	})

	t.Run("EncryptedPassword", func(t *testing.T) {
		password := "abcdefgh"
		keyPEM := generateEncryptedRSAKey(t, password)
		key := writeKey(t, "key", keyPEM)
		enc := Encryption{
			KeyFile:         key,
			KeyFilePassword: password,
		}
		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: enc})
		cfg, err := loadFromFile(configfilef, &enc)
		require.NoError(t, err)
		assert.Equal(t, &Config{ID: "agent-id"}, cfg)
	})

	t.Run("EncryptedWrongPassword", func(t *testing.T) {
		password := "abcdefgh"
		keyPEM := generateEncryptedRSAKey(t, password)
		key := writeKey(t, "key", keyPEM)
		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: Encryption{
			KeyFile:         key,
			KeyFilePassword: password,
		}})

		cfg, err := loadFromFile(configfilef, &Encryption{
			KeyFile:         key,
			KeyFilePassword: "hgfedcba",
		})
		require.EqualError(t, err, "unable to get RSA key from KeyFile: unable to parse private key: pkcs8: incorrect password")
		assert.Nil(t, cfg)
	})

	t.Run("EncryptedWrongKey", func(t *testing.T) {
		password := "abcdefgh"
		key1PEM := generateEncryptedRSAKey(t, password)
		key2PEM := generateRSAKey(t)
		key1 := writeKey(t, "key1", key1PEM)
		key2 := writeKey(t, "key2", key2PEM)

		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: Encryption{
			KeyFile: key2,
		}})
		cfg, err := loadFromFile(configfilef, &Encryption{
			KeyFile:         key1,
			KeyFilePassword: password,
		})
		require.EqualError(t, err, "unable to RSA-unwrap AES key: crypto/rsa: decryption error")
		assert.Nil(t, cfg)
	})
}
