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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/percona/pmm/managed/utils/encryption"
)

// MigrationBackupPattern matches the files written by writeMigrationBackup.
const MigrationBackupPattern = "pmm-encryption-migration-backup-*.json"

// writeMigrationBackup stores the secret columns of the rows, by table, that
// the migration rewrites from the pre-envelope format, exactly as they are
// stored, next to the key file. If that directory is not writable, the backup
// goes to PMM Server's data directory. The rewrite cannot be undone otherwise:
// older PMM versions cannot read the envelope format. Values are as sensitive
// as the database itself (legacy ciphertext, or secrets PMM 3.x stored as
// plaintext), so the file is readable by the owner only.
func writeMigrationBackup(rows map[string][]map[string]any) (string, error) {
	data, err := json.MarshalIndent(map[string]any{
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"key_path":   encryption.DefaultKeyPath(),
		"tables":     rows,
	}, "", "  ")
	if err != nil {
		return "", err
	}

	// the key's directory may be read-only (e.g. a mounted secret); /srv is
	// PMM Server's data directory and always writable
	dirs := []string{filepath.Dir(encryption.DefaultKeyPath())}
	if srv := filepath.Dir(encryption.DefaultEncryptionKeyPath); srv != dirs[0] {
		dirs = append(dirs, srv)
	}

	var errs []error
	for _, dir := range dirs {
		path, err := writeFileAtomically(dir, data)
		if err == nil {
			return path, nil
		}
		errs = append(errs, err)
	}

	return "", errors.Join(errs...)
}

func writeFileAtomically(dir string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, ".pmm-encryption-migration-backup-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create migration backup in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	// CreateTemp already creates the file with owner-only permissions
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("failed to write migration backup in %s: %w", dir, err)
	}

	name := fmt.Sprintf("pmm-encryption-migration-backup-%s.json", time.Now().UTC().Format("20060102T150405.000000000Z"))
	path := filepath.Join(dir, name)
	err = os.Rename(tmp.Name(), path)
	if err != nil {
		return "", fmt.Errorf("failed to write migration backup in %s: %w", dir, err)
	}

	return path, nil
}
