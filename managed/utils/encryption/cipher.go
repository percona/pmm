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

// Package encryption provides field-level encryption for PMM Server data.
package encryption

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

// DefaultEncryptionKeyPath contains default PMM encryption key path.
var DefaultEncryptionKeyPath = "/srv/pmm-encryption.key"

// ErrEncryptionNotInitialized is returned when the default cipher is used
// before SetDefaultCipher was called.
var ErrEncryptionNotInitialized = errors.New("encryption is not initialized")

// CustomEncryptionKeyPathEnvVar is an environment variable to set custom encryption key path.
const CustomEncryptionKeyPathEnvVar = "PMM_ENCRYPTION_KEY_PATH"

var (
	defaultCipher   *Cipher
	defaultCipherMu sync.RWMutex
)

// SetDefaultCipher installs the process-wide cipher used by the models field
// codec. It must be called once during startup, before any database access.
func SetDefaultCipher(c *Cipher) {
	defaultCipherMu.Lock()
	defer defaultCipherMu.Unlock()
	defaultCipher = c
}

// DefaultCipher returns the process-wide cipher installed by
// SetDefaultCipher, or ErrEncryptionNotInitialized if none is set.
func DefaultCipher() (*Cipher, error) {
	defaultCipherMu.RLock()
	defer defaultCipherMu.RUnlock()
	if defaultCipher == nil {
		return nil, ErrEncryptionNotInitialized
	}

	return defaultCipher, nil
}

// Cipher encrypts and decrypts field values using a Tink keyset.
//
// Values are stored in the self-describing envelope format (see
// EnvelopePrefix). Decrypt transparently handles the legacy format (base64
// Tink ciphertext without a prefix, written by PMM before the envelope was
// introduced) and passes through plaintext written before encryption was
// enabled. A Cipher is immutable and safe for concurrent use.
type Cipher struct {
	provider  KeyProvider
	handle    *keyset.Handle
	primitive tink.AEAD
	primaryID uint32
}

// LoadCipher loads an existing keyset from the provider.
func LoadCipher(p KeyProvider) (*Cipher, error) {
	handle, err := p.Load()
	if err != nil {
		return nil, err
	}

	return newCipher(p, handle)
}

// CreateCipher generates a new keyset and persists it via the provider. It
// refuses to replace an existing keyset — doing so would make previously
// encrypted values permanently unreadable.
func CreateCipher(p KeyProvider) (*Cipher, error) {
	_, err := p.Load()
	switch {
	case err == nil:
		return nil, errors.New("refusing to overwrite an existing encryption keyset")
	case !errors.Is(err, ErrKeysetNotFound):
		return nil, err
	}

	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption keyset: %w", err)
	}
	err = p.Store(handle)
	if err != nil {
		return nil, err
	}

	return newCipher(p, handle)
}

// LoadOrCreateCipher loads the keyset, generating one if none exists yet.
func LoadOrCreateCipher(p KeyProvider) (*Cipher, error) {
	c, err := LoadCipher(p)
	if errors.Is(err, ErrKeysetNotFound) {
		return CreateCipher(p)
	}

	return c, err
}

func newCipher(p KeyProvider, handle *keyset.Handle) (*Cipher, error) {
	primitive, err := aead.New(handle)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD primitive: %w", err)
	}
	primary, err := handle.Primary()
	if err != nil {
		return nil, fmt.Errorf("failed to determine primary encryption key: %w", err)
	}

	return &Cipher{
		provider:  p,
		handle:    handle,
		primitive: primitive,
		primaryID: primary.KeyID(),
	}, nil
}

// Encrypt returns the plaintext encrypted with the primary key in the
// envelope format. Empty input is passed through unchanged.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if c == nil {
		return "", ErrEncryptionNotInitialized
	}
	if plaintext == "" {
		return "", nil
	}

	ciphertext, err := c.primitive.Encrypt([]byte(plaintext), nil)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	return wrapEnvelope(ciphertext), nil
}

// Decrypt returns the plaintext for a stored value:
//   - envelope format: decrypted with the keyset; any failure is an error
//   - legacy format (base64 Tink ciphertext): decrypted; values that only
//     look like ciphertext fail authentication and fall through
//   - anything else: returned unchanged (plaintext written before encryption)
func (c *Cipher) Decrypt(stored string) (string, error) {
	if c == nil {
		return "", ErrEncryptionNotInitialized
	}
	if stored == "" {
		return "", nil
	}

	if IsEncrypted(stored) {
		ciphertext, err := unwrapEnvelope(stored)
		if err != nil {
			return "", err
		}
		plaintext, err := c.primitive.Decrypt(ciphertext, nil)
		if err != nil {
			return "", fmt.Errorf("decryption failed: %w", err)
		}

		return string(plaintext), nil
	}

	if plaintext, ok := c.tryLegacyDecrypt(stored); ok {
		return plaintext, nil
	}

	return stored, nil
}

func (c *Cipher) tryLegacyDecrypt(stored string) (string, bool) {
	ciphertext, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", false
	}
	if _, ok := tinkKeyID(ciphertext); !ok {
		return "", false
	}

	plaintext, err := c.primitive.Decrypt(ciphertext, nil)
	if err != nil {
		return "", false
	}

	return string(plaintext), true
}

// PrimaryKeyID returns the ID of the key new values are encrypted with.
func (c *Cipher) PrimaryKeyID() uint32 {
	return c.primaryID
}

// NeedsReencrypt reports whether a stored value should be rewritten to be
// encrypted with the primary key in the envelope format: non-empty plaintext,
// legacy values and envelopes carrying a stale key ID all qualify.
func (c *Cipher) NeedsReencrypt(stored string) bool {
	if stored == "" {
		return false
	}
	if id, ok := StoredKeyID(stored); ok {
		return id != c.primaryID
	}

	return true
}
