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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type legacyFixtures struct {
	Strings []struct {
		Plaintext  string `json:"plaintext"`
		Ciphertext string `json:"ciphertext"`
	} `json:"strings"`
	Options map[string]struct {
		Plaintext string `json:"plaintext"`
		Encrypted string `json:"encrypted"`
	} `json:"options"`
}

func loadLegacyFixtures(t *testing.T) *legacyFixtures {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "legacy-ciphertexts.json"))
	require.NoError(t, err)
	f := &legacyFixtures{}
	require.NoError(t, json.Unmarshal(data, f))
	require.NotEmpty(t, f.Strings)
	require.NotEmpty(t, f.Options)

	return f
}

func legacyCipher(t *testing.T) *Cipher {
	t.Helper()

	c, err := LoadCipher(NewFileKeyProvider(filepath.Join("testdata", "legacy-keyset.key")))
	require.NoError(t, err)

	return c
}

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()

	c, err := CreateCipher(NewFileKeyProvider(filepath.Join(t.TempDir(), "encryption.key")))
	require.NoError(t, err)

	return c
}

// The committed fixtures were generated with the pre-rewrite implementation
// (tink-go v1, base64 ciphertext without an envelope). They pin the promise
// that a PMM 3.x upgrade keeps all existing data readable.
func TestLegacyStringFixtures(t *testing.T) {
	c := legacyCipher(t)

	for _, pair := range loadLegacyFixtures(t).Strings {
		decrypted, err := c.Decrypt(pair.Ciphertext)
		require.NoError(t, err)
		assert.Equal(t, pair.Plaintext, decrypted)

		// legacy values must be picked up by migration/rotation sweeps
		assert.True(t, c.NeedsReencrypt(pair.Ciphertext))
		_, ok := StoredKeyID(pair.Ciphertext)
		assert.False(t, ok)

		// re-encrypting yields an envelope value that needs no further rewrite
		reencrypted, err := c.Encrypt(pair.Plaintext)
		require.NoError(t, err)
		assert.True(t, IsEncrypted(reencrypted))
		assert.False(t, c.NeedsReencrypt(reencrypted))
		roundTrip, err := c.Decrypt(reencrypted)
		require.NoError(t, err)
		assert.Equal(t, pair.Plaintext, roundTrip)
	}
}

func TestLegacyOptionsFixtures(t *testing.T) {
	c := legacyCipher(t)

	for column, pair := range loadLegacyFixtures(t).Options {
		var plain, encrypted map[string]any
		require.NoError(t, json.Unmarshal([]byte(pair.Plaintext), &plain), column)
		require.NoError(t, json.Unmarshal([]byte(pair.Encrypted), &encrypted), column)
		require.Len(t, encrypted, len(plain), column)

		secretFields := 0
		for key, plainValue := range plain {
			encryptedValue, ok := encrypted[key]
			require.True(t, ok, "%s: %s missing in encrypted blob", column, key)
			if assert.ObjectsAreEqual(plainValue, encryptedValue) {
				continue
			}

			// differing values are the encrypted sub-fields
			secretFields++
			ciphertext, ok := encryptedValue.(string)
			require.True(t, ok, "%s: %s is not a string", column, key)
			decrypted, err := c.Decrypt(ciphertext)
			require.NoError(t, err, "%s: %s", column, key)
			assert.Equal(t, plainValue, decrypted, "%s: %s", column, key)
		}
		assert.NotZero(t, secretFields, "%s: fixture has no encrypted sub-fields", column)
	}
}

func TestCipherRoundTrip(t *testing.T) {
	c := newTestCipher(t)

	for _, plaintext := range []string{
		"secret",
		"møøse-пароль-🔐",
		strings.Repeat("k", 10000),
		"-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
	} {
		stored, err := c.Encrypt(plaintext)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(stored, EnvelopePrefix))

		keyID, ok := StoredKeyID(stored)
		require.True(t, ok)
		assert.Equal(t, c.PrimaryKeyID(), keyID)

		decrypted, err := c.Decrypt(stored)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	}
}

func TestCipherEmptyString(t *testing.T) {
	c := newTestCipher(t)

	stored, err := c.Encrypt("")
	require.NoError(t, err)
	assert.Empty(t, stored)

	decrypted, err := c.Decrypt("")
	require.NoError(t, err)
	assert.Empty(t, decrypted)

	assert.False(t, c.NeedsReencrypt(""))
}

func TestCipherPlaintextPassthrough(t *testing.T) {
	c := newTestCipher(t)

	for _, plaintext := range []string{
		"just-a-password",
		"with spaces and $ymbols",
		// valid base64 decoding to bytes starting with the Tink prefix byte:
		// authentication fails, so it is still treated as plaintext
		"AeEBAg==",
	} {
		decrypted, err := c.Decrypt(plaintext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)

		// plaintext must be picked up by the migration sweep
		assert.True(t, c.NeedsReencrypt(plaintext))
	}
}

func TestCipherEnvelopeHardFailures(t *testing.T) {
	c := newTestCipher(t)

	t.Run("malformed base64", func(t *testing.T) {
		_, err := c.Decrypt(EnvelopePrefix + "%%%not-base64%%%")
		assert.Error(t, err)
	})

	t.Run("corrupted ciphertext", func(t *testing.T) {
		stored, err := c.Encrypt("secret")
		require.NoError(t, err)
		corrupted := stored[:len(stored)-4] + "AAAA"
		_, err = c.Decrypt(corrupted)
		assert.Error(t, err)
	})

	t.Run("wrong key", func(t *testing.T) {
		stored, err := c.Encrypt("secret")
		require.NoError(t, err)

		other := newTestCipher(t)
		_, err = other.Decrypt(stored)
		assert.Error(t, err)
	})
}

func TestCreateCipherRefusesOverwrite(t *testing.T) {
	provider := NewFileKeyProvider(filepath.Join(t.TempDir(), "encryption.key"))

	_, err := CreateCipher(provider)
	require.NoError(t, err)

	_, err = CreateCipher(provider)
	assert.ErrorContains(t, err, "refusing to overwrite")
}

func TestLoadCipherMissingKeyset(t *testing.T) {
	_, err := LoadCipher(NewFileKeyProvider(filepath.Join(t.TempDir(), "missing.key")))
	assert.ErrorIs(t, err, ErrKeysetNotFound)
}

func TestLoadOrCreateCipher(t *testing.T) {
	provider := NewFileKeyProvider(filepath.Join(t.TempDir(), "encryption.key"))

	created, err := LoadOrCreateCipher(provider)
	require.NoError(t, err)

	loaded, err := LoadOrCreateCipher(provider)
	require.NoError(t, err)
	assert.Equal(t, created.PrimaryKeyID(), loaded.PrimaryKeyID())
}

func TestFileKeyProviderPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "encryption.key")
	_, err := CreateCipher(NewFileKeyProvider(path))
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "encryption.key")
	provider := NewFileKeyProvider(path)

	oldCipher, err := CreateCipher(provider)
	require.NoError(t, err)
	oldValue, err := oldCipher.Encrypt("secret-under-key1")
	require.NoError(t, err)

	newKeyID, err := AddNewPrimaryKey(provider)
	require.NoError(t, err)
	assert.NotEqual(t, oldCipher.PrimaryKeyID(), newKeyID)

	newCipher, err := LoadCipher(provider)
	require.NoError(t, err)
	assert.Equal(t, newKeyID, newCipher.PrimaryKeyID())

	// values encrypted under the old key stay readable and are flagged stale
	decrypted, err := newCipher.Decrypt(oldValue)
	require.NoError(t, err)
	assert.Equal(t, "secret-under-key1", decrypted)
	assert.True(t, newCipher.NeedsReencrypt(oldValue))

	// new writes carry the new key ID
	newValue, err := newCipher.Encrypt(decrypted)
	require.NoError(t, err)
	keyID, ok := StoredKeyID(newValue)
	require.True(t, ok)
	assert.Equal(t, newKeyID, keyID)
	assert.False(t, newCipher.NeedsReencrypt(newValue))

	retired, err := PruneRetiredKeys(provider)
	require.NoError(t, err)
	assert.Equal(t, []uint32{oldCipher.PrimaryKeyID()}, retired)

	prunedCipher, err := LoadCipher(provider)
	require.NoError(t, err)
	decrypted, err = prunedCipher.Decrypt(newValue)
	require.NoError(t, err)
	assert.Equal(t, "secret-under-key1", decrypted)

	// old-key envelopes are unreadable after pruning — that's the point
	_, err = prunedCipher.Decrypt(oldValue)
	require.Error(t, err)

	// pruning again is a no-op
	retired, err = PruneRetiredKeys(provider)
	require.NoError(t, err)
	assert.Empty(t, retired)
}

func TestPruneRequiresNoStaleValues(t *testing.T) {
	// documents the contract: PruneRetiredKeys itself does not verify DB
	// state; callers must sweep first (NeedsReencrypt reports stale values)
	provider := NewFileKeyProvider(filepath.Join(t.TempDir(), "encryption.key"))
	c, err := CreateCipher(provider)
	require.NoError(t, err)
	stored, err := c.Encrypt("secret")
	require.NoError(t, err)

	_, err = AddNewPrimaryKey(provider)
	require.NoError(t, err)
	swept, err := LoadCipher(provider)
	require.NoError(t, err)
	require.True(t, swept.NeedsReencrypt(stored))
}

func TestLegacyWrongKey(t *testing.T) {
	c := newTestCipher(t)
	fixtures := loadLegacyFixtures(t)
	legacy := fixtures.Strings[0].Ciphertext

	// the key file does not match the data: the value is ambiguous on its own,
	// so Decrypt passes it through and InspectLegacy flags it
	decrypted, err := c.Decrypt(legacy)
	require.NoError(t, err)
	assert.Equal(t, legacy, decrypted)
	_, err = c.Inspect(legacy)
	assert.ErrorIs(t, err, ErrLegacyUnknownKey)
}

func TestLegacyAuthFailure(t *testing.T) {
	c := legacyCipher(t)
	legacy := loadLegacyFixtures(t).Strings[1].Ciphertext

	// flip a byte of the tag, keeping the Tink prefix and key ID intact
	raw, err := base64.StdEncoding.DecodeString(legacy)
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xff
	corrupted := base64.StdEncoding.EncodeToString(raw)

	_, err = c.Decrypt(corrupted)
	require.ErrorIs(t, err, ErrLegacyAuthFailed)
	_, err = c.Inspect(corrupted)
	assert.ErrorIs(t, err, ErrLegacyAuthFailed)
}

func TestInspectLegacyReadableValues(t *testing.T) {
	c := legacyCipher(t)
	envelope, err := c.Encrypt("secret")
	require.NoError(t, err)

	for _, stored := range []string{
		"",
		"plain-password",
		"AeEBAg==", // Tink prefix byte, but too short to be AES-GCM ciphertext
		envelope,
		loadLegacyFixtures(t).Strings[0].Ciphertext,
	} {
		insp, err := c.Inspect(stored)
		require.NoError(t, err, stored)
		assert.Zero(t, insp.ExtraLayers, stored)
	}
}

// legacyLayer encrypts plaintext in the legacy format (no envelope prefix).
func legacyLayer(t *testing.T, c *Cipher, plaintext string) string {
	t.Helper()

	stored, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	return strings.TrimPrefix(stored, EnvelopePrefix)
}

// TestStackedLegacyLayers covers values corrupted by key rotation before
// PMM 3.9.1 (PMM-15188): its decrypt phase encrypted option fields with the
// old key instead of decrypting them, then the encrypt phase added a layer
// with the new key. After one rotation a value is new(old(secret)), after two
// new2(new1(new1(old(secret)))).
func TestStackedLegacyLayers(t *testing.T) {
	dir := t.TempDir()
	oldProvider := NewFileKeyProvider(filepath.Join(dir, "encryption_old.key"))
	oldKey, err := CreateCipher(oldProvider)
	require.NoError(t, err)
	current := newTestCipher(t)

	const secret = "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----\n"
	oneRotation := legacyLayer(t, current, legacyLayer(t, oldKey, secret))

	t.Run("previous key available", func(t *testing.T) {
		c, err := current.WithLegacyKeys(oldProvider)
		require.NoError(t, err)

		decrypted, err := c.Decrypt(oneRotation)
		require.NoError(t, err)
		assert.Equal(t, secret, decrypted)
		insp, err := c.Inspect(oneRotation)
		require.NoError(t, err)
		assert.Equal(t, 1, insp.ExtraLayers)

		// a value migrated before the key was found still heals: the layer
		// under the envelope is removed on read
		migrated, err := current.Encrypt(legacyLayer(t, oldKey, secret))
		require.NoError(t, err)
		decrypted, err = c.Decrypt(migrated)
		require.NoError(t, err)
		assert.Equal(t, secret, decrypted)
		insp, err = c.Inspect(migrated)
		require.NoError(t, err)
		assert.Equal(t, 1, insp.ExtraLayers)

		// the decrypt-only key never encrypts
		reencrypted, err := c.Encrypt(secret)
		require.NoError(t, err)
		keyID, ok := StoredKeyID(reencrypted)
		require.True(t, ok)
		assert.Equal(t, current.PrimaryKeyID(), keyID)
	})

	t.Run("repeated layers of the same key", func(t *testing.T) {
		c, err := current.WithLegacyKeys(oldProvider)
		require.NoError(t, err)
		stacked := legacyLayer(t, current, legacyLayer(t, current, legacyLayer(t, oldKey, secret)))

		decrypted, err := c.Decrypt(stacked)
		require.NoError(t, err)
		assert.Equal(t, secret, decrypted)
	})

	t.Run("inner key lost", func(t *testing.T) {
		// without the previous key the secret cannot be recovered; reads do
		// not fail (other rows stay usable) but Inspect reports it
		_, err := current.Decrypt(oneRotation)
		require.NoError(t, err)
		_, err = current.Inspect(oneRotation)
		assert.ErrorIs(t, err, ErrLegacyInnerKeyLost)
	})
}

func TestLegacyBackupKeyPath(t *testing.T) {
	assert.Equal(t, "/srv/pmm-encryption_old.key", LegacyBackupKeyPath("/srv/pmm-encryption.key"))
	assert.Equal(t, "/etc/custom-key_old.key", LegacyBackupKeyPath("/etc/custom-key"))
}

// TestLegacyLayerOverEnvelope covers PMM before the envelope format started
// after an upgrade: at startup it re-encrypts every column in the legacy
// format, wrapping envelopes. Reads must remove that layer and report it, so
// the next migration rewrites the value, including values a migration without
// this handling already wrapped in a second envelope.
func TestLegacyLayerOverEnvelope(t *testing.T) {
	c := newTestCipher(t)
	envelope, err := c.Encrypt("secret")
	require.NoError(t, err)
	doubleEnvelope, err := c.Encrypt(envelope)
	require.NoError(t, err)

	for name, stored := range map[string]string{
		"legacy layer added by an older version":     legacyLayer(t, c, envelope),
		"envelope stored by a migration without fix": doubleEnvelope,
	} {
		t.Run(name, func(t *testing.T) {
			decrypted, err := c.Decrypt(stored)
			require.NoError(t, err)
			assert.Equal(t, "secret", decrypted)

			insp, err := c.Inspect(stored)
			require.NoError(t, err)
			assert.Equal(t, 1, insp.ExtraLayers)
		})
	}
}

// TestInspectDecrypted pins the evidence the startup migration uses to tell a
// key file that does not match the database from one that reads part of it.
func TestInspectDecrypted(t *testing.T) {
	c := newTestCipher(t)
	other := newTestCipher(t)
	envelope, err := c.Encrypt("secret")
	require.NoError(t, err)

	for stored, want := range map[string]bool{
		"":                            false,
		"plain-password":              false,
		envelope:                      true,
		legacyLayer(t, c, "secret"):   true,
		legacyLayer(t, other, "gone"): false,
	} {
		insp, _ := c.Inspect(stored)
		assert.Equal(t, want, insp.Decrypted, stored)
	}
}

func TestCipherNotInitialized(t *testing.T) {
	var c *Cipher
	_, err := c.Encrypt("secret")
	require.ErrorIs(t, err, ErrEncryptionNotInitialized)
	_, err = c.Decrypt("secret")
	require.ErrorIs(t, err, ErrEncryptionNotInitialized)
	_, err = c.Inspect("secret")
	require.ErrorIs(t, err, ErrEncryptionNotInitialized)

	defaultCipherMu.RLock()
	saved := defaultCipher
	defaultCipherMu.RUnlock()
	t.Cleanup(func() { SetDefaultCipher(saved) })
	SetDefaultCipher(nil)
	_, err = DefaultCipher()
	require.ErrorIs(t, err, ErrEncryptionNotInitialized)
}

// TestLookAlikeUnderLayer covers a decrypted value that has the shape of legacy
// ciphertext of a key in the keyset but does not authenticate: it is the
// secret itself, not another layer, and is returned unchanged.
func TestLookAlikeUnderLayer(t *testing.T) {
	c := newTestCipher(t)

	raw := make([]byte, minLegacyCiphertextLen+8)
	raw[0] = tinkPrefixByte
	raw[1], raw[2], raw[3], raw[4] = byte(c.primaryID>>24), byte(c.primaryID>>16), byte(c.primaryID>>8), byte(c.primaryID)
	lookAlike := base64.StdEncoding.EncodeToString(raw)

	// on its own it is a hard error: a known key ID that fails authentication
	_, err := c.Decrypt(lookAlike)
	require.ErrorIs(t, err, ErrLegacyAuthFailed)

	// under an envelope it is the secret
	stored, err := c.Encrypt(lookAlike)
	require.NoError(t, err)
	decrypted, err := c.Decrypt(stored)
	require.NoError(t, err)
	assert.Equal(t, lookAlike, decrypted)
	insp, err := c.Inspect(stored)
	require.NoError(t, err)
	assert.Zero(t, insp.ExtraLayers)
	assert.True(t, insp.Decrypted)
}

// TestUnknownKey pins how an envelope of a key the keyset does not hold is
// read: an error, unless the cipher accepts key loss.
func TestUnknownKey(t *testing.T) {
	lost := newTestCipher(t)
	stored, err := lost.Encrypt("secret")
	require.NoError(t, err)
	c := newTestCipher(t)

	_, err = c.Decrypt(stored)
	require.ErrorIs(t, err, ErrUnknownKey)
	_, err = c.Inspect(stored)
	require.ErrorIs(t, err, ErrUnknownKey)

	accepting := c.AcceptingKeyLoss()
	assert.True(t, accepting.AcceptsKeyLoss())
	assert.False(t, c.AcceptsKeyLoss(), "the original cipher is unchanged")
	decrypted, err := accepting.Decrypt(stored)
	require.NoError(t, err)
	assert.Equal(t, stored, decrypted)
	_, err = accepting.Inspect(stored)
	require.ErrorIs(t, err, ErrUnknownKey, "the migration still sees the loss")
}

// TestLostEnvelopeRecovered covers an envelope of a lost key that the startup
// migration wrapped in one of the primary key: placing the lost key next to
// the key file makes the secret readable again.
func TestLostEnvelopeRecovered(t *testing.T) {
	lostPath := filepath.Join(t.TempDir(), "lost.key")
	lost, err := CreateCipher(NewFileKeyProvider(lostPath))
	require.NoError(t, err)
	inner, err := lost.Encrypt("secret")
	require.NoError(t, err)
	c := newTestCipher(t)
	wrapped, err := c.Encrypt(inner)
	require.NoError(t, err)

	decrypted, err := c.Decrypt(wrapped)
	require.NoError(t, err)
	assert.Equal(t, inner, decrypted, "without the lost key")

	withLost, err := c.WithLegacyKeys(NewFileKeyProvider(lostPath))
	require.NoError(t, err)
	decrypted, err = withLost.Decrypt(wrapped)
	require.NoError(t, err)
	assert.Equal(t, "secret", decrypted)
	insp, err := withLost.Inspect(wrapped)
	require.NoError(t, err)
	assert.Equal(t, 1, insp.ExtraLayers, "the next migration rewrites it as a single envelope")
}
