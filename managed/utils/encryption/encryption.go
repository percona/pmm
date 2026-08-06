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

// Package encryption contains functions to encrypt/decrypt items or DB.
package encryption

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/tink-crypto/tink-go/aead"
	"github.com/tink-crypto/tink-go/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/keyset"
	"github.com/tink-crypto/tink-go/tink"
	"gopkg.in/reform.v1"
)

var (
	// DefaultEncryptionKeyPath contains default PMM encryption key path.
	DefaultEncryptionKeyPath = "/srv/pmm-encryption.key"
	// ErrEncryptionNotInitialized is error in case of encryption is not initialized.
	ErrEncryptionNotInitialized = errors.New("encryption is not initialized")
	// DefaultEncryption is the default implementation of encryption, lazily initialized.
	defaultEncryption    *Encryption
	defaultEncryptionMtx sync.Mutex
)

// CustomEncryptionKeyPathEnvVar is an environment variable to set custom encryption key path.
const CustomEncryptionKeyPathEnvVar = "PMM_ENCRYPTION_KEY_PATH"

// getDefaultEncryption returns the default encryption instance, initializing it lazily if needed.
func getDefaultEncryption() *Encryption {
	defaultEncryptionMtx.Lock()
	defer defaultEncryptionMtx.Unlock()

	if defaultEncryption == nil {
		defaultEncryption = New()
	}
	return defaultEncryption
}

// Encryption contains fields required for encryption.
type Encryption struct {
	Path      string
	Key       string
	Primitive tink.AEAD
}

// Table represents table name, it's identifiers and columns to be encrypted/decrypted.
type Table struct {
	Name        string
	Identifiers []string
	Columns     []Column
}

// Column represents column name and column's custom handlers (if needed).
// CustomEncryptHandler and CustomDecryptHandler must be set as a matching pair:
// EncryptItems uses the encrypt handler, DecryptItems uses the decrypt one. Reusing
// a single handler for both directions adds another encryption layer during the
// decrypt phase instead of removing one.
type Column struct {
	Name                 string
	CustomEncryptHandler func(e *Encryption, val any) (any, error)
	CustomDecryptHandler func(e *Encryption, val any) (any, error)
}

// QueryValues represents query to update row after encrypt/decrypt.
type QueryValues struct {
	Query       string
	SetValues   [][]any
	WhereValues [][]any
}

func isTest() bool {
	// Check if tests are running by inspecting os.Args
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}

// New creates an encryption; if key on path doesn't exist, it will be generated.
func New() *Encryption {
	e := &Encryption{}
	customKeyPath := os.Getenv(CustomEncryptionKeyPathEnvVar)
	if customKeyPath != "" {
		e.Path = customKeyPath
	} else {
		if isTest() {
			e.Path = "./encryption.key"
		} else {
			e.Path = DefaultEncryptionKeyPath
		}
	}

	bytes, err := os.ReadFile(e.Path)
	switch {
	case os.IsNotExist(err):
		err = e.generateAndPersistKey()
		if err != nil {
			logrus.Panicf("Encryption: %v", err)
		}
	case err != nil:
		logrus.Panicf("Encryption: %v", err)
	default:
		e.Key = string(bytes)
	}

	primitive, err := e.getPrimitive()
	if err != nil {
		logrus.Panicf("Encryption: %v", err)
	}
	e.Primitive = primitive

	return e
}

// RotateEncryptionKey is a wrapper around DefaultEncryption.RotateEncryptionKey.
func RotateEncryptionKey() error {
	err := backupOldEncryptionKey()
	if err != nil {
		return err
	}

	defaultEncryptionMtx.Lock()
	defaultEncryption = New()
	defaultEncryptionMtx.Unlock()

	return nil
}

// RestoreOldEncryptionKey is a wrapper around DefaultEncryption.RestoreOldEncryptionKey.
func RestoreOldEncryptionKey() error {
	err := os.Rename(strings.TrimSuffix(KeyPath(), ".key")+"_old.key", KeyPath())
	if err != nil {
		return fmt.Errorf("could not restore old encryption key: %w", err)
	}

	return nil
}

func backupOldEncryptionKey() error {
	err := os.Rename(KeyPath(), strings.TrimSuffix(KeyPath(), ".key")+"_old.key")
	if err != nil {
		return fmt.Errorf("failed to backup old encryption key: %w", err)
	}

	return nil
}

// GenerateKey generates a new encryption key.
func (e *Encryption) GenerateKey() (string, error) {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return "", fmt.Errorf("failed to create keyset: %w", err)
	}

	buff := &bytes.Buffer{}
	err = insecurecleartextkeyset.Write(handle, keyset.NewBinaryWriter(buff))
	if err != nil {
		return "", fmt.Errorf("failed to write encryption key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buff.Bytes()), nil
}

// Fingerprint is a wrapper around DefaultEncryption.Fingerprint.
func Fingerprint() (string, error) {
	return getDefaultEncryption().Fingerprint()
}

// Fingerprint returns a stable identifier of the encryption key.
//
// It is recorded next to the encrypted data so a node can tell whether it holds the key the
// data was encrypted with without decrypting anything, which also works when there is no
// encrypted row to test against yet. It is a digest of the key, so it cannot be used to
// reconstruct it.
func (e *Encryption) Fingerprint() (string, error) {
	if e == nil || e.Key == "" {
		return "", ErrEncryptionNotInitialized
	}

	serializedKeyset, err := base64.StdEncoding.DecodeString(e.Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode keyset: %w", err)
	}

	sum := sha256.Sum256(serializedKeyset)

	return hex.EncodeToString(sum[:]), nil
}

func (e *Encryption) generateAndPersistKey() error {
	key, err := e.GenerateKey()
	if err != nil {
		return err
	}
	e.Key = key
	return e.saveKeyToFile()
}

func (e *Encryption) saveKeyToFile() error {
	return os.WriteFile(e.Path, []byte(e.Key), 0o644) //nolint:gosec,mnd
}

// Encrypt is a wrapper around DefaultEncryption.Encrypt.
func Encrypt(secret string) (string, error) {
	return getDefaultEncryption().Encrypt(secret)
}

// Encrypt returns input string encrypted.
// On failure it returns an empty string rather than the plaintext, so that a caller which
// ignores the error cannot persist an unencrypted secret.
func (e *Encryption) Encrypt(secret string) (string, error) {
	if e == nil || e.Primitive == nil {
		return "", ErrEncryptionNotInitialized
	}
	if secret == "" {
		return secret, nil
	}
	cipherText, err := e.Primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return "", fmt.Errorf("encryption: %w", err)
	}

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// EncryptItems is a wrapper around DefaultEncryption.EncryptItems.
func EncryptItems(tx *reform.TX, tables []Table) error {
	return getDefaultEncryption().EncryptItems(tx, tables)
}

// EncryptItems will encrypt all columns provided in DB connection.
func (e *Encryption) EncryptItems(tx *reform.TX, tables []Table) error {
	if len(tables) == 0 {
		return nil
	}

	for _, table := range tables {
		res, err := table.read(tx)
		if err != nil {
			return fmt.Errorf("failed to read table %s: %w", table.Name, err)
		}

		for k, v := range res.SetValues {
			for i, val := range v {
				var encrypted any
				var err error
				if handler := table.Columns[i].CustomEncryptHandler; handler != nil {
					encrypted, err = handler(e, val)
				} else {
					encrypted, err = encryptColumnStringHandler(e, val)
				}

				if err != nil {
					return fmt.Errorf("failed to encrypt table %s: %w", table.Name, err)
				}
				res.SetValues[k][i] = encrypted
			}
			data := slices.Concat([]any{}, v)
			data = slices.Concat(data, res.WhereValues[k])
			_, err := tx.Exec(res.Query, data...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Decrypt is wrapper around DefaultEncryption.Decrypt.
func Decrypt(cipherText string) (string, error) {
	return getDefaultEncryption().Decrypt(cipherText)
}

// Decrypt returns input string decrypted.
// On failure it returns an empty string rather than the input ciphertext. Returning the
// ciphertext let callers that only logged the error pass it on as if it were the decrypted
// value, which is how encrypted credentials reached pmm-agent as usernames and passwords
// when a node's encryption key did not match the data in a shared database.
func (e *Encryption) Decrypt(cipherText string) (string, error) {
	if e == nil || e.Primitive == nil {
		return "", ErrEncryptionNotInitialized
	}
	if cipherText == "" {
		return cipherText, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("decryption: %w", err)
	}
	secret, err := e.Primitive.Decrypt(decoded, []byte(""))
	if err != nil {
		return "", fmt.Errorf("decryption: %w", err)
	}

	return string(secret), nil
}

// DecryptItems is wrapper around DefaultEncryption.DecryptItems.
func DecryptItems(tx *reform.TX, tables []Table) error {
	return getDefaultEncryption().DecryptItems(tx, tables)
}

// DecryptItems will decrypt all columns provided in DB connection.
func (e *Encryption) DecryptItems(tx *reform.TX, tables []Table) error {
	if len(tables) == 0 {
		return nil
	}

	for _, table := range tables {
		res, err := table.read(tx)
		if err != nil {
			return fmt.Errorf("failed to read table %s: %w", table.Name, err)
		}

		for k, v := range res.SetValues {
			for i, val := range v {
				var decrypted any
				var err error
				if handler := table.Columns[i].CustomDecryptHandler; handler != nil {
					decrypted, err = handler(e, val)
				} else {
					decrypted, err = decryptColumnStringHandler(e, val)
				}

				if err != nil {
					return fmt.Errorf("failed to decrypt table %s: %w", table.Name, err)
				}
				res.SetValues[k][i] = decrypted
			}
			data := slices.Concat([]any{}, v)
			data = slices.Concat(data, res.WhereValues[k])
			_, err := tx.Exec(res.Query, data...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Encryption) getPrimitive() (tink.AEAD, error) { //nolint:ireturn
	serializedKeyset, err := base64.StdEncoding.DecodeString(e.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode keyset: %w", err)
	}

	binaryReader := keyset.NewBinaryReader(bytes.NewBuffer(serializedKeyset))
	parsedHandle, err := insecurecleartextkeyset.Read(binaryReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyset: %w", err)
	}

	t, err := aead.New(parsedHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD primitive: %w", err)
	}
	return t, err
}
