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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// initDefaultCipher loads the encryption keyset from the default path
// (PMM_ENCRYPTION_KEY_PATH or DefaultEncryptionKeyPath) and installs it as
// the process-wide cipher. It must run after the database connection is
// bootstrapped and before migrations: a missing keyset is generated only when
// the database contains no encrypted data; otherwise startup is refused to
// avoid making that data permanently unreadable.
func initDefaultCipher(ctx context.Context, db *sql.DB) error {
	keyPath := encryption.DefaultKeyPath()
	provider := encryption.NewFileKeyProvider(keyPath)

	cipher, err := encryption.LoadCipher(provider)
	if errors.Is(err, encryption.ErrKeysetNotFound) {
		var hasEncryptedData bool
		hasEncryptedData, err = DatabaseHasEncryptedData(ctx, db)
		if err != nil {
			return err
		}
		if hasEncryptedData {
			return fmt.Errorf("encryption key not found at %s, but the database contains encrypted data; "+
				"restore the key file or point %s at it — generating a new key would make that data unreadable",
				keyPath, encryption.CustomEncryptionKeyPathEnvVar)
		}
		cipher, err = encryption.CreateCipher(provider)
	}
	if err != nil {
		return err
	}

	cipher = withLegacyBackupKey(cipher, keyPath)
	encryption.SetDefaultCipher(cipher)

	return nil
}

// withLegacyBackupKey adds the key PMM 3.x rotation left next to the key file
// as decrypt-only, so the migration can remove the layers it stacked on
// option fields (PMM-15188). The file is optional and never modified.
func withLegacyBackupKey(cipher *encryption.Cipher, keyPath string) *encryption.Cipher {
	backupPath := encryption.LegacyBackupKeyPath(keyPath)
	withBackup, err := cipher.WithLegacyKeys(encryption.NewFileKeyProvider(backupPath))
	switch {
	case errors.Is(err, encryption.ErrKeysetNotFound):
		return cipher
	case err != nil:
		logrus.Warnf("Ignoring the previous encryption key at %s: %s", backupPath, err)
		return cipher
	}
	logrus.Infof("Using the previous encryption key at %s to decrypt data encrypted before key rotation", backupPath)

	return withBackup
}

// DatabaseHasEncryptedData reports whether the database contains values that
// can only be read with the existing encryption key: envelope-format secrets,
// or legacy ciphertext recorded by the pre-envelope PMM 3.x bookkeeping. It
// is used to refuse startup when the key file is lost — generating a fresh
// key would make that data permanently unreadable. A database with no schema
// yet is reported as having no encrypted data.
func DatabaseHasEncryptedData(ctx context.Context, db *sql.DB) (bool, error) {
	// legacy bookkeeping from PMM 3.x before the envelope format
	var settingsJSON []byte
	err := db.QueryRowContext(ctx, "SELECT settings FROM settings").Scan(&settingsJSON)
	switch {
	case isUndefinedTable(err) || errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return false, fmt.Errorf("failed to probe settings for encrypted data: %w", err)
	default:
		columns, err := legacyEncryptedColumns(settingsJSON)
		if err != nil {
			return false, fmt.Errorf("failed to probe settings for encrypted data: %w", err)
		}
		if len(columns) != 0 {
			return true, nil
		}
	}

	var exists bool
	err = db.QueryRowContext(ctx, encryptedDataProbe).Scan(&exists)
	if isUndefinedTable(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to probe for encrypted data: %w", err)
	}

	return exists, nil
}

// encryptedDataProbe finds any envelope value: scalar agent secrets start with
// the prefix, secrets inside JSON blobs appear as a string starting with it.
var encryptedDataProbe = func() string {
	scalar := "'" + encryption.EnvelopePrefix + "%'"
	inJSON := "'%\"" + encryption.EnvelopePrefix + "%'"

	return "SELECT EXISTS (SELECT 1 FROM agents WHERE" +
		" username LIKE " + scalar + " OR password LIKE " + scalar + " OR agent_password LIKE " + scalar +
		" OR aws_options::text LIKE " + inJSON + " OR azure_options::text LIKE " + inJSON +
		" OR mongo_options::text LIKE " + inJSON + " OR mysql_options::text LIKE " + inJSON +
		" OR postgresql_options::text LIKE " + inJSON + " OR valkey_options::text LIKE " + inJSON +
		") OR EXISTS (SELECT 1 FROM backup_locations WHERE s3_config::text LIKE " + inJSON + ")"
}()

// legacyEncryptedColumns returns the agents columns that PMM 3.x recorded as
// encrypted in settings.encrypted_items (entries are "database.table.column").
// Every non-empty value in those columns was written as ciphertext.
func legacyEncryptedColumns(settingsJSON []byte) (map[string]bool, error) {
	var s struct {
		EncryptedItems []string `json:"encrypted_items"`
	}
	err := json.Unmarshal(settingsJSON, &s)
	if err != nil {
		return nil, err
	}

	columns := make(map[string]bool, len(s.EncryptedItems))
	for _, item := range s.EncryptedItems {
		parts := strings.Split(item, ".")
		if len(parts) >= 2 && parts[len(parts)-2] == "agents" {
			columns[parts[len(parts)-1]] = true
		}
	}

	return columns, nil
}

// storedLegacyEncryptedColumns reads legacyEncryptedColumns from the settings
// row. The bookkeeping disappears from the row the first time settings are
// saved after the upgrade; by then the migration has already run.
func storedLegacyEncryptedColumns(q *reform.Querier) (map[string]bool, error) {
	var settingsJSON []byte
	err := q.QueryRow("SELECT settings FROM settings").Scan(&settingsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	return legacyEncryptedColumns(settingsJSON)
}

func isUndefinedTable(err error) bool {
	// undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
	var pErr *pq.Error
	return errors.As(err, &pErr) && pErr.Code == "42P01"
}

// MigrateEncryption rewrites agent rows whose stored secrets are not yet
// encrypted with the primary key in the envelope format. Reading through the
// model decrypts any supported format, so a plain reload-and-update converts
// plaintext (never encrypted), legacy ciphertext (pre-envelope) and stale-key
// envelopes alike. Rows already in the desired state are left untouched,
// which makes the migration idempotent and safe to run at every startup,
// including concurrently from several HA nodes.
func MigrateEncryption(q *reform.Querier) error {
	cipher, err := encryption.DefaultCipher()
	if err != nil {
		return err
	}

	scan, err := scanAgents(q, cipher)
	if err != nil {
		return err
	}
	if len(scan.undecryptable) != 0 {
		return errUndecryptable(scan.undecryptable)
	}
	if len(scan.lost) != 0 {
		logrus.Warnf("Credentials of %d agent field(s) cannot be recovered: they were encrypted more than once "+
			"by key rotation in PMM before 3.9.1 and the key of the inner layer is not available (%s). "+
			"Place that key at %s and restart PMM Server, or re-enter the credentials of the affected services.",
			len(scan.lost), strings.Join(scan.lost, ", "), encryption.LegacyBackupKeyPath(encryption.DefaultKeyPath()))
	}
	ids := scan.needs

	for _, id := range ids {
		agent := &Agent{AgentID: id}
		err = q.Reload(agent)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt agent %s: %w", id, err)
		}
		err = q.Update(agent)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt agent %s: %w", id, err)
		}
	}

	locationIDs, err := LocationsNeedingReencryption(q, cipher)
	if err != nil {
		return err
	}

	for _, id := range locationIDs {
		location := &BackupLocation{ID: id}
		err = q.Reload(location)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt backup location %s: %w", id, err)
		}
		err = q.Update(location)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt backup location %s: %w", id, err)
		}
	}

	return nil
}

// AgentsNeedingReencryption returns IDs of agents with at least one stored
// secret that is not encrypted with the primary key in the envelope format.
// It fails when a stored secret is ciphertext the cipher cannot decrypt (see
// errUndecryptable): rewriting such a row would store the ciphertext as if it
// were the secret.
func AgentsNeedingReencryption(q *reform.Querier, cipher *encryption.Cipher) ([]string, error) {
	scan, err := scanAgents(q, cipher)
	if err != nil {
		return nil, err
	}
	if len(scan.undecryptable) != 0 {
		return nil, errUndecryptable(scan.undecryptable)
	}

	return scan.needs, nil
}

// errUndecryptable explains that the key does not match the stored data.
func errUndecryptable(problems []string) error {
	const shown = 10
	list := problems
	more := ""
	if len(list) > shown {
		more = fmt.Sprintf(" and %d more", len(list)-shown)
		list = list[:shown]
	}

	return fmt.Errorf("the encryption key at %s cannot decrypt stored credentials (%s%s): "+
		"the key file does not match this database; restore the original key file "+
		"(or point %s at it) and restart — no data was changed",
		encryption.DefaultKeyPath(), strings.Join(list, "; "), more, encryption.CustomEncryptionKeyPathEnvVar)
}

type agentsScan struct {
	needs         []string
	undecryptable []string
	// lost lists secrets whose innermost layer's key is gone; they cannot be
	// recovered by PMM and are reported, not treated as a key mismatch
	lost []string
}

func scanAgents(q *reform.Querier, cipher *encryption.Cipher) (*agentsScan, error) {
	strict, err := storedLegacyEncryptedColumns(q)
	if err != nil {
		return nil, err
	}

	rows, err := q.Query(`
		SELECT agent_id, username, password, agent_password,
			aws_options, azure_options, mongo_options, mysql_options, postgresql_options, valkey_options
		FROM agents
		ORDER BY agent_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	scan := &agentsScan{}
	for rows.Next() {
		var id string
		var username, password, agentPassword sql.NullString
		var aws, azure, mongo, mysql, postgresql, valkey sql.NullString
		err = rows.Scan(&id, &username, &password, &agentPassword, &aws, &azure, &mongo, &mysql, &postgresql, &valkey)
		if err != nil {
			return nil, fmt.Errorf("failed to read agents: %w", err)
		}

		insp := &inspector{cipher: cipher, strict: strict}
		insp.scalar("username", username)
		insp.scalar("password", password)
		insp.scalar("agent_password", agentPassword)
		for _, o := range []optionsColumn{
			{"aws_options", aws, &AWSOptions{}},
			{"azure_options", azure, &AzureOptions{}},
			{"mongo_options", mongo, &MongoDBOptions{}},
			{"mysql_options", mysql, &MySQLOptions{}},
			{"postgresql_options", postgresql, &PostgreSQLOptions{}},
			{"valkey_options", valkey, &ValkeyOptions{}},
		} {
			err = insp.options(o)
			if err != nil {
				return nil, fmt.Errorf("agent %s: %w", id, err)
			}
		}

		for _, p := range insp.undecryptable {
			scan.undecryptable = append(scan.undecryptable, fmt.Sprintf("agent %s %s: %s", id, p.column, p.err))
		}
		for _, p := range insp.lost {
			scan.lost = append(scan.lost, fmt.Sprintf("agent %s %s", id, p.column))
		}
		if insp.needs && len(insp.undecryptable) == 0 {
			scan.needs = append(scan.needs, id)
		}
	}

	return scan, rows.Err()
}

// LocationsNeedingReencryption returns IDs of backup locations whose stored
// S3 credentials are not encrypted with the primary key in the envelope format.
// PMM 3.x stored them as plaintext, so only values that fail authentication
// under a known key are treated as undecryptable.
func LocationsNeedingReencryption(q *reform.Querier, cipher *encryption.Cipher) ([]string, error) {
	rows, err := q.Query(`SELECT id, s3_config FROM backup_locations ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup locations: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ids []string
	var undecryptable []string
	for rows.Next() {
		var id string
		var s3Config sql.NullString
		err = rows.Scan(&id, &s3Config)
		if err != nil {
			return nil, fmt.Errorf("failed to read backup locations: %w", err)
		}

		insp := &inspector{cipher: cipher}
		err = insp.options(optionsColumn{"s3_config", s3Config, &S3LocationConfig{}})
		if err != nil {
			return nil, fmt.Errorf("backup location %s: %w", id, err)
		}
		for _, p := range insp.undecryptable {
			undecryptable = append(undecryptable, fmt.Sprintf("backup location %s %s: %s", id, p.column, p.err))
		}
		if insp.needs && len(insp.undecryptable) == 0 {
			ids = append(ids, id)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(undecryptable) != 0 {
		return nil, errUndecryptable(undecryptable)
	}

	return ids, nil
}

type optionsColumn struct {
	name   string
	raw    sql.NullString
	target any
}

type columnProblem struct {
	column string
	err    error
}

// inspector classifies the stored secrets of one row without decrypting
// them into the model.
type inspector struct {
	cipher *encryption.Cipher
	// strict holds the columns PMM 3.x recorded as encrypted: a value there
	// that looks like ciphertext of an unknown key is ciphertext, not plaintext
	strict        map[string]bool
	needs         bool
	undecryptable []columnProblem
	lost          []columnProblem
}

func (i *inspector) value(column, stored string) {
	if stored == "" {
		return
	}
	insp, err := i.cipher.Inspect(stored)
	if i.cipher.NeedsReencrypt(stored) || insp.ExtraLayers > 0 {
		i.needs = true
	}

	switch {
	case err == nil:
	case errors.Is(err, encryption.ErrLegacyUnknownKey) && !i.strict[column]:
		// plaintext that happens to look like ciphertext
	case errors.Is(err, encryption.ErrLegacyInnerKeyLost):
		// the key matches; the secret was lost to stacked layers before
		i.lost = append(i.lost, columnProblem{column, err})
	default:
		i.undecryptable = append(i.undecryptable, columnProblem{column, err})
	}
}

func (i *inspector) scalar(column string, v sql.NullString) {
	if v.Valid {
		i.value(column, v.String)
	}
}

// options unmarshals a stored options blob without decrypting it and
// inspects its tagged secret sub-fields.
func (i *inspector) options(o optionsColumn) error {
	if !o.raw.Valid {
		return nil
	}

	err := json.Unmarshal([]byte(o.raw.String), o.target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", o.name, err)
	}

	return applyToSecretFields(o.target, func(s string) (string, error) {
		i.value(o.name, s)
		return s, nil
	})
}
