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
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// keyCheckPlaintext is a known value stored encrypted in the settings
// (Settings.EncryptionKeyCheck), so that a node can tell whether its key is
// the one the database was set up with before any secret is stored: a fresh
// HA database holds none, and two nodes with different keys would otherwise
// both start and write rows the other cannot read. The startup migration
// re-encrypts it with the primary key like every other value, so it follows
// key rotation.
const keyCheckPlaintext = "pmm-encryption-key-check"

const selectKeyCheck = "SELECT settings->>'encryption_key_check' FROM settings"

// readKeyCheck returns the stored key check; ok is false when there is no
// settings row.
func readKeyCheck(q reform.DBTX) (string, bool, error) {
	var check sql.NullString
	err := q.QueryRow(selectKeyCheck).Scan(&check)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("failed to read the encryption key check: %w", err)
	}

	return check.String, true, nil
}

// writeKeyCheck stores the key check encrypted with the primary key, leaving
// the other settings as they are.
func writeKeyCheck(q reform.DBTX, cipher *encryption.Cipher) error {
	check, err := cipher.Encrypt(keyCheckPlaintext)
	if err != nil {
		return err
	}

	_, err = q.Exec("UPDATE settings SET settings = jsonb_set(COALESCE(settings, '{}'), '{encryption_key_check}', to_jsonb($1::text))", check)
	if err != nil {
		return fmt.Errorf("failed to store the encryption key check: %w", err)
	}

	return nil
}

// checkKey compares the stored key check with the cipher. It returns whether
// to store a new key check, and whether the key is not the one the database
// was set up with. On a standalone server, a key generated at this start
// replaces the key check of the missing one, as the database held no data
// encrypted with it; so does a key the administrator accepted after losing
// the original one. In HA a generated key never does: another node set the
// database up after this one found no key check (keyInUse runs before the
// migration lock), and this node must use that node's key.
func checkKey(q *reform.Querier, cipher *encryption.Cipher, params migrationParams) (bool, bool, error) {
	check, hasSettings, err := readKeyCheck(q)
	switch {
	case err != nil:
		return false, false, err
	case !hasSettings:
		return false, false, nil
	case check == "": // the first start, or the upgrade
		return true, false, nil
	case keyCheckMatches(cipher, check):
		return cipher.NeedsReencrypt(check), false, nil
	case params.keyCreated && !params.ha:
		logrus.Infof("Replacing the encryption key check of the missing encryption key.")
		return true, false, nil
	case cipher.AcceptsKeyLoss():
		logrus.Warnf("The encryption key is not the one this database was set up with; replacing the key check, as %s is set.",
			encryption.AcceptKeyLossEnvVar)
		return true, false, nil
	default:
		return false, true, nil
	}
}

// keyCheckMatches reports whether the stored key check decrypts with the cipher.
func keyCheckMatches(cipher *encryption.Cipher, check string) bool {
	plaintext, err := cipher.Decrypt(check)

	return err == nil && plaintext == keyCheckPlaintext
}

// KeyCheckNeedsReencryption reports whether the stored key check is not
// encrypted with the primary key yet; key rotation waits for it like for the
// rows, so that pruning never removes the key it is encrypted with.
func KeyCheckNeedsReencryption(q *reform.Querier, cipher *encryption.Cipher) (bool, error) {
	check, _, err := readKeyCheck(q)
	if err != nil {
		return false, err
	}

	return cipher.NeedsReencrypt(check), nil
}

// databaseHasKeyCheck reports whether another node has already set the
// database up with its key. It runs before schema migrations: a database with
// no schema yet has no key check.
func databaseHasKeyCheck(ctx context.Context, db *sql.DB) (bool, error) {
	var check sql.NullString
	err := db.QueryRowContext(ctx, selectKeyCheck).Scan(&check)
	switch {
	case isUndefinedTable(err) || errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to probe settings for the encryption key check: %w", err)
	}

	return check.String != "", nil
}
