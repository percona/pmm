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
	"slices"
	"strings"
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

var (
	// ErrLegacyAuthFailed is returned for a legacy-format value whose key ID
	// is in the keyset but which fails authentication: the value is corrupted
	// or was encrypted with a different key that happens to share the ID.
	ErrLegacyAuthFailed = errors.New("legacy ciphertext failed authentication")
	// ErrLegacyUnknownKey is reported by InspectLegacy for a value shaped like
	// legacy ciphertext whose key ID is not in the keyset. It is ambiguous on
	// its own — plaintext can take that shape by chance — so Decrypt passes
	// such values through and callers that know a value was encrypted decide.
	ErrLegacyUnknownKey = errors.New("legacy ciphertext encrypted with a key that is not in the keyset")
	// ErrLegacyInnerKeyLost is reported by Inspect for a value that decrypts,
	// but whose plaintext is itself legacy ciphertext of a key that is not
	// available. Key rotation before PMM 3.9.1 stacked such layers on option
	// fields (PMM-15188); without the key of the inner layer the secret
	// cannot be recovered.
	ErrLegacyInnerKeyLost = errors.New("secret is wrapped in an extra encryption layer whose key is not available")
)

const (
	// Tink AES-GCM ciphertext: output prefix, 12-byte IV and 16-byte tag.
	minLegacyCiphertextLen = tinkPrefixLen + 12 + 16
	// Upper bound of stacked legacy layers removed from one value; each
	// buggy rotation added two.
	maxLegacyLayers = 16
)

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
	// legacyKeys decrypt legacy ciphertext: the keyset itself first, then
	// decrypt-only keys added with WithLegacyKeys
	legacyKeys []legacyKeyset
}

type legacyKeyset struct {
	keyIDs    map[uint32]bool
	primitive tink.AEAD
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
	keyIDs, err := handleKeyIDs(handle)
	if err != nil {
		return nil, err
	}

	return &Cipher{
		provider:   p,
		handle:     handle,
		primitive:  primitive,
		primaryID:  primary.KeyID(),
		legacyKeys: []legacyKeyset{{keyIDs: keyIDs, primitive: primitive}},
	}, nil
}

// WithLegacyKeys returns a copy of the cipher that can additionally decrypt
// legacy ciphertext with the keys of the given keyset. Those keys are never
// used for new values or for envelopes. PMM 3.x rotation kept the previous
// key next to the current one (see LegacyBackupKeyPath); inner layers stacked
// by buggy rotations are encrypted with it.
func (c *Cipher) WithLegacyKeys(p KeyProvider) (*Cipher, error) {
	handle, err := p.Load()
	if err != nil {
		return nil, err
	}
	primitive, err := aead.New(handle)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD primitive: %w", err)
	}
	keyIDs, err := handleKeyIDs(handle)
	if err != nil {
		return nil, err
	}

	cp := *c
	cp.legacyKeys = append(slices.Clone(c.legacyKeys), legacyKeyset{keyIDs: keyIDs, primitive: primitive})

	return &cp, nil
}

// LegacyBackupKeyPath returns where PMM 3.x rotation left the previous key
// for the key file at keyPath.
func LegacyBackupKeyPath(keyPath string) string {
	return strings.TrimSuffix(keyPath, ".key") + "_old.key"
}

func handleKeyIDs(handle *keyset.Handle) (map[uint32]bool, error) {
	ids := make(map[uint32]bool, handle.Len())
	for i := range handle.Len() {
		entry, err := handle.Entry(i)
		if err != nil {
			return nil, fmt.Errorf("failed to read encryption keyset entry: %w", err)
		}
		ids[entry.KeyID()] = true
	}

	return ids, nil
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
//   - legacy format (base64 Tink ciphertext): decrypted; a value carrying the
//     ID of a key in the keyset that fails authentication is an error
//     (ErrLegacyAuthFailed), a value carrying an unknown key ID is returned
//     unchanged (see ErrLegacyUnknownKey)
//   - anything else: returned unchanged (plaintext written before encryption)
//
// Legacy layers stacked under the outermost one (PMM-15188) are removed as
// long as their keys are available.
func (c *Cipher) Decrypt(stored string) (string, error) {
	plaintext, _, err := c.decrypt(stored)
	switch {
	case errors.Is(err, ErrLegacyUnknownKey), errors.Is(err, ErrLegacyInnerKeyLost):
		return plaintext, nil
	case err != nil:
		return "", err
	}

	return plaintext, nil
}

// Inspection describes how Decrypt reads a stored value.
type Inspection struct {
	// ExtraLayers is the number of stacked legacy layers removed under the
	// outermost encryption layer; a value with extra layers needs a rewrite.
	ExtraLayers int
}

// Inspect reports how Decrypt reads a stored value. The error is nil for
// empty, plaintext and decryptable values; otherwise it is the decryption
// error, ErrLegacyUnknownKey or ErrLegacyInnerKeyLost. The startup migration
// uses it to refuse rewriting ciphertext it cannot read as if it were
// plaintext, and to report secrets lost to stacked layers.
func (c *Cipher) Inspect(stored string) (Inspection, error) {
	_, layers, err := c.decrypt(stored)
	if layers > 0 {
		layers--
	}

	return Inspection{ExtraLayers: layers}, err
}

// decrypt returns the plaintext and the number of encryption layers removed.
// With ErrLegacyUnknownKey it returns the value unchanged, with
// ErrLegacyInnerKeyLost the innermost ciphertext it could reach.
func (c *Cipher) decrypt(stored string) (string, int, error) {
	if c == nil {
		return "", 0, ErrEncryptionNotInitialized
	}
	if stored == "" {
		return "", 0, nil
	}

	v := stored
	layers := 0
	if IsEncrypted(stored) {
		ciphertext, err := unwrapEnvelope(stored)
		if err != nil {
			return "", 0, err
		}
		plaintext, err := c.primitive.Decrypt(ciphertext, nil)
		if err != nil {
			return "", 0, fmt.Errorf("decryption failed: %w", err)
		}
		v = string(plaintext)
		layers = 1
	}

	for layers < maxLegacyLayers {
		ciphertext, keyID, ok := legacyCiphertext(v)
		if !ok {
			break
		}

		var keysets []tink.AEAD
		for _, k := range c.legacyKeys {
			if k.keyIDs[keyID] {
				keysets = append(keysets, k.primitive)
			}
		}
		if len(keysets) == 0 {
			if layers == 0 {
				return stored, 0, fmt.Errorf("%w (key ID %d)", ErrLegacyUnknownKey, keyID)
			}
			return v, layers, fmt.Errorf("%w (key ID %d)", ErrLegacyInnerKeyLost, keyID)
		}

		plaintext, ok := decryptWithAny(keysets, ciphertext)
		if !ok {
			if layers == 0 {
				return "", 0, fmt.Errorf("%w (key ID %d)", ErrLegacyAuthFailed, keyID)
			}
			// an authenticated plaintext that only looks like ciphertext
			break
		}
		v = plaintext
		layers++
	}

	return v, layers, nil
}

func decryptWithAny(keysets []tink.AEAD, ciphertext []byte) (string, bool) {
	for _, k := range keysets {
		plaintext, err := k.Decrypt(ciphertext, nil)
		if err == nil {
			return string(plaintext), true
		}
	}

	return "", false
}

// legacyCiphertext decodes a value that has the shape of legacy ciphertext:
// base64 of a Tink ciphertext long enough to hold an AES-GCM IV and tag.
func legacyCiphertext(stored string) ([]byte, uint32, bool) {
	ciphertext, err := base64.StdEncoding.DecodeString(stored)
	if err != nil || len(ciphertext) < minLegacyCiphertextLen {
		return nil, 0, false
	}
	keyID, ok := tinkKeyID(ciphertext)

	return ciphertext, keyID, ok
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
