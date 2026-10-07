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
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"
)

// ErrKeysetNotFound is returned by KeyProvider.Load when no keyset exists yet.
var ErrKeysetNotFound = errors.New("encryption keyset not found")

// KeyProvider loads and stores the Tink keyset used by Cipher.
type KeyProvider interface {
	// Load returns the stored keyset, or an error wrapping ErrKeysetNotFound
	// when none exists.
	Load() (*keyset.Handle, error)
	// Store persists the keyset, replacing any previous one.
	Store(handle *keyset.Handle) error
}

// FileKeyProvider stores the keyset as a base64-encoded serialized Tink
// keyset in a single file — the same format PMM has used since 3.0, so
// existing key files load unchanged.
type FileKeyProvider struct {
	path string
}

// NewFileKeyProvider creates a FileKeyProvider for the given path.
func NewFileKeyProvider(path string) *FileKeyProvider {
	return &FileKeyProvider{path: path}
}

// DefaultKeyPath returns the encryption key path: PMM_ENCRYPTION_KEY_PATH if
// set, DefaultEncryptionKeyPath otherwise.
func DefaultKeyPath() string {
	if custom := os.Getenv(CustomEncryptionKeyPathEnvVar); custom != "" {
		return custom
	}

	return DefaultEncryptionKeyPath
}

// AcceptKeyLossEnvVar is an environment variable that lets a standalone
// pmm-managed start when the key the stored data was encrypted with is lost
// for good. Its value must be the ID of the current encryption key, which the
// refusal to start names, so that a setting left behind never accepts another
// key later. Values the key cannot decrypt are kept in the migration backup and
// rewritten as unrecoverable instead of refusing to start; placing the lost key
// next to the key file later makes them readable again (see
// LegacyBackupKeyPath).
const AcceptKeyLossEnvVar = "PMM_ENCRYPTION_ACCEPT_KEY_LOSS"

// Load reads and parses the keyset file.
func (p *FileKeyProvider) Load() (*keyset.Handle, error) {
	data, err := os.ReadFile(p.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w at %s", ErrKeysetNotFound, p.path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read encryption keyset: %w", err)
	}

	serialized, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption keyset %s: %w", p.path, err)
	}

	handle, err := insecurecleartextkeyset.Read(keyset.NewBinaryReader(bytes.NewReader(serialized)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse encryption keyset %s: %w", p.path, err)
	}

	return handle, nil
}

// keysetFileMode keeps the keyset readable by the owner only.
const keysetFileMode = os.FileMode(0o600)

// GenerateKeyset returns a new serialized keyset in the key-file format
// without persisting it anywhere.
func GenerateKeyset() (string, error) {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return "", fmt.Errorf("failed to generate encryption keyset: %w", err)
	}

	return serializeHandle(handle)
}

func serializeHandle(handle *keyset.Handle) (string, error) {
	buff := &bytes.Buffer{}
	err := insecurecleartextkeyset.Write(handle, keyset.NewBinaryWriter(buff))
	if err != nil {
		return "", fmt.Errorf("failed to serialize encryption keyset: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buff.Bytes()), nil
}

// Store writes the keyset atomically with owner-only permissions.
func (p *FileKeyProvider) Store(handle *keyset.Handle) error {
	encoded, err := serializeHandle(handle)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(p.path), ".pmm-encryption-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to store encryption keyset: %w", err)
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	err = tmp.Chmod(keysetFileMode)
	if err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return fmt.Errorf("failed to store encryption keyset: %w", err)
	}
	_, err = tmp.WriteString(encoded)
	if err == nil {
		// the key file is the only copy of the key: it must be on disk before it replaces the old one
		err = tmp.Sync()
	}
	if err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return fmt.Errorf("failed to store encryption keyset: %w", err)
	}
	err = tmp.Close()
	if err != nil {
		return fmt.Errorf("failed to store encryption keyset: %w", err)
	}

	err = os.Rename(tmp.Name(), p.path)
	if err != nil {
		return fmt.Errorf("failed to store encryption keyset: %w", err)
	}
	err = syncDir(filepath.Dir(p.path))
	if err != nil {
		return fmt.Errorf("failed to store encryption keyset: %w", err)
	}

	return nil
}

// syncDir makes a rename in the directory durable: after a crash, the data
// encrypted with a new key must not find the old key file, so a failure
// stops the caller before anything is encrypted with that key. A file system
// that cannot sync a directory at all says so (EINVAL, ENOTSUP); there the
// rename is as durable as it gets.
func syncDir(path string) error {
	dir, err := os.Open(path) //nolint:gosec
	if err != nil {
		return err
	}
	defer dir.Close() //nolint:errcheck

	err = dir.Sync()
	if errors.Is(err, errors.ErrUnsupported) || errors.Is(err, syscall.EINVAL) {
		return nil
	}

	return err
}
