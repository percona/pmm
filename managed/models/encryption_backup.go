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
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/percona/pmm/managed/utils/encryption"
)

// MigrationBackupPattern matches the files written by writeMigrationBackup.
const MigrationBackupPattern = "pmm-encryption-migration-backup-*.json"

// writeMigrationBackup stores the columns of rows the migration rewrites from
// the pre-envelope format, exactly as they are stored, next to the key file.
// The rewrite cannot be undone otherwise: older PMM versions cannot read the
// envelope format. Values are as sensitive as the database itself (legacy
// ciphertext, or secrets PMM 3.x stored as plaintext), so the file is
// readable by the owner only.
func writeMigrationBackup(agents, locations []map[string]any) (string, error) {
	data, err := json.MarshalIndent(map[string]any{
		"created_at":       time.Now().UTC().Format(time.RFC3339),
		"key_path":         encryption.DefaultKeyPath(),
		"agents":           agents,
		"backup_locations": locations,
	}, "", "  ")
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(encryption.DefaultKeyPath())
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
		return "", fmt.Errorf("failed to write migration backup: %w", err)
	}

	name := fmt.Sprintf("pmm-encryption-migration-backup-%s.json", time.Now().UTC().Format("20060102T150405.000000000Z"))
	path := filepath.Join(dir, name)
	err = os.Rename(tmp.Name(), path)
	if err != nil {
		return "", fmt.Errorf("failed to write migration backup: %w", err)
	}

	return path, nil
}

func nullString(v sql.NullString) any {
	if !v.Valid {
		return nil
	}

	return v.String
}

// rawJSON keeps a JSON column as JSON in the backup, or as a string if it is
// not valid JSON.
func rawJSON(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	if json.Valid([]byte(v.String)) {
		return json.RawMessage(v.String)
	}

	return v.String
}
