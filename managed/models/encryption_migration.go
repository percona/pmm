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
	"os"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// initDefaultCipher loads the encryption keyset from the default path
// (PMM_ENCRYPTION_KEY_PATH or DefaultEncryptionKeyPath) and installs it as
// the process-wide cipher. It must run after the database connection is
// bootstrapped and before migrations. A missing keyset is generated only when
// the database holds no data encrypted with the missing key (see keyInUse);
// otherwise startup is refused to avoid making that data permanently
// unreadable. It returns whether a keyset was generated.
func initDefaultCipher(ctx context.Context, db *sql.DB, ha bool) (bool, error) {
	keyPath := encryption.DefaultKeyPath()
	provider := encryption.NewFileKeyProvider(keyPath)

	var created bool
	cipher, err := encryption.LoadCipher(provider)
	if errors.Is(err, encryption.ErrKeysetNotFound) {
		var inUse bool
		inUse, err = keyInUse(ctx, db, ha)
		if err != nil {
			return false, err
		}
		if inUse {
			return false, missingKeyError(keyPath, ha)
		}
		cipher, err = encryption.CreateCipher(provider)
		created = err == nil
	}
	if err != nil {
		return false, err
	}

	cipher = withLegacyBackupKey(cipher, keyPath)
	if acceptKeyLoss(cipher, keyPath, ha) {
		cipher = cipher.AcceptingKeyLoss()
	}
	encryption.SetDefaultCipher(cipher)

	return created, nil
}

// missingKeyError refuses to generate a key while the database holds data
// encrypted with the missing one. Accepting the loss takes a key to accept it
// for, so a standalone server is told to create one first.
func missingKeyError(keyPath string, ha bool) error {
	msg := fmt.Sprintf("encryption key not found at %s, but the database contains data encrypted with it; "+
		"restore the key file or point %s at it — generating a new key would make that data unreadable",
		keyPath, encryption.CustomEncryptionKeyPathEnvVar)
	if !ha {
		msg += fmt.Sprintf(". If the key is lost for good, create a new one with `pmm-encryption-rotation --generate-key > %s` "+
			"and restart: PMM Server then names the value of %s that accepts the loss", keyPath, encryption.AcceptKeyLossEnvVar)
	}

	return &keyError{msg}
}

// acceptKeyLoss reports whether the administrator has accepted that the key
// the stored data was encrypted with is lost, by setting
// encryption.AcceptKeyLossEnvVar to the ID of the current key. Never in HA:
// the other nodes still hold the key, and accepting its loss on one node would
// rewrite the data they read.
func acceptKeyLoss(cipher *encryption.Cipher, keyPath string, ha bool) bool {
	value := os.Getenv(encryption.AcceptKeyLossEnvVar)
	keyID := keyIDValue(cipher)
	switch {
	case value == "":
		return false
	case ha:
		logrus.Errorf("%s is ignored in HA mode: every PMM Server node must use the same encryption key, "+
			"so copy %s from a node that works instead.", encryption.AcceptKeyLossEnvVar, keyPath)
		return false
	case value != keyID:
		logrus.Warnf("%s=%s is ignored: it does not name the encryption key at %s, whose ID is %s.",
			encryption.AcceptKeyLossEnvVar, value, keyPath, keyID)
		return false
	default:
		logrus.Warnf("%s names the encryption key at %s: stored values it cannot decrypt are treated as lost. "+
			"Unset it once PMM Server has started.", encryption.AcceptKeyLossEnvVar, keyPath)
		return true
	}
}

// keyIDValue returns the value of encryption.AcceptKeyLossEnvVar that accepts
// the loss for the cipher's key.
func keyIDValue(cipher *encryption.Cipher) string {
	return strconv.FormatUint(uint64(cipher.PrimaryKeyID()), 10)
}

// keyInUse reports whether the database holds data encrypted with a missing
// key: encrypted values, or in HA the key check of the node that set the
// database up, whose key every node must share. A standalone server that
// holds no encrypted values loses nothing with a new key.
func keyInUse(ctx context.Context, db *sql.DB, ha bool) (bool, error) {
	hasEncryptedData, err := DatabaseHasEncryptedData(ctx, db)
	if err != nil || hasEncryptedData || !ha {
		return hasEncryptedData, err
	}

	return databaseHasKeyCheck(ctx, db)
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

// encryptedDataProbe finds any envelope value in the tables holding secrets.
// It runs before schema migrations, so it must not name columns that older
// schemas lack: every column of the row, including JSON blobs, is rendered as
// JSON, where an envelope value appears as a string starting with the prefix.
var encryptedDataProbe = "SELECT EXISTS (SELECT 1 FROM agents t WHERE to_jsonb(t)::text LIKE '%\"" + encryption.EnvelopePrefix + "%')" +
	" OR EXISTS (SELECT 1 FROM backup_locations t WHERE to_jsonb(t)::text LIKE '%\"" + encryption.EnvelopePrefix + "%')"

// legacyEncryptedColumns returns the agents columns that PMM 3.x recorded as
// encrypted in settings.encrypted_items (entries are "database.table.column").
// PMM 3.x encrypted values in those columns when writing agents, except that
// it wrote pmm-agent rows back decrypted on every pmm-agent connection; so
// values of other agent types there are ciphertext, see secretTable.
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
		return map[string]bool{}, nil
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

// MigrateEncryption rewrites rows whose stored secrets are not yet encrypted
// with the primary key in the envelope format. Reading through the model
// decrypts any supported format, so a plain reload-and-update converts
// plaintext (never encrypted), legacy ciphertext (pre-envelope) and stale-key
// envelopes alike. Rows already in the desired state are left untouched,
// which makes the migration idempotent and safe to run at every startup.
//
// The querier must belong to a transaction; outside one the locks below last
// for a single statement and protect nothing. An advisory lock serializes the
// migration across HA nodes, and each row is locked and only its secret
// columns are written, so concurrent changes to other columns by live nodes
// are kept.
//
// The key check in the settings (see keyCheckPlaintext) must decrypt with the
// key; it is written on the first start and follows key rotation.
func MigrateEncryption(q *reform.Querier) error {
	return migrateEncryption(q, migrationParams{})
}

// migrationParams describe the start migrateEncryption runs at.
type migrationParams struct {
	// keyCreated is set when the key was generated at this start because the
	// key file was missing and the database held no data encrypted with it
	keyCreated bool
	// ha is set on a node of an HA cluster
	ha bool
}

// acceptValue returns the value of encryption.AcceptKeyLossEnvVar a refusal
// to start offers: the ID of the key, unless its loss is accepted already or
// the node is in HA, see acceptKeyLoss.
func (p migrationParams) acceptValue(cipher *encryption.Cipher) string {
	if p.ha || cipher.AcceptsKeyLoss() {
		return ""
	}

	return keyIDValue(cipher)
}

// migrateEncryption is MigrateEncryption for the start described by params.
func migrateEncryption(q *reform.Querier, params migrationParams) error {
	cipher, err := encryption.DefaultCipher()
	if err != nil {
		return err
	}

	// re-entrant: migrateDB already holds it in the same transaction
	_, err = q.Exec("SELECT pg_advisory_xact_lock($1)", migrationLockID)
	if err != nil {
		return fmt.Errorf("failed to lock encryption migration: %w", err)
	}

	var undecryptable, lost, unknownKey []error
	writeCheck, keyMismatch, err := checkKey(q, cipher, params)
	if err != nil {
		return err
	}
	if keyMismatch {
		undecryptable = append(undecryptable, errors.New("settings encryption key check: the key this database was set up with is required"))
	}

	scans := make([]*secretsScan, len(secretTables))
	var keyReadsData bool
	backup := make(map[string][]map[string]any)
	for i, t := range secretTables {
		scans[i], err = scanSecrets(q, cipher, t)
		if err != nil {
			return err
		}
		undecryptable = append(undecryptable, scans[i].undecryptable...)
		lost = append(lost, scans[i].lost...)
		unknownKey = append(unknownKey, scans[i].unknownKey...)
		keyReadsData = keyReadsData || scans[i].keyReadsData
		if len(scans[i].backup) != 0 {
			backup[t.name] = scans[i].backup
		}
	}
	u, l := resolveUnknownKeys(unknownKey, keyReadsData || cipher.AcceptsKeyLoss())
	undecryptable = append(undecryptable, u...)
	lost = append(lost, l...)
	if len(undecryptable) != 0 {
		return errUndecryptable(undecryptable, params.acceptValue(cipher))
	}
	if len(lost) != 0 {
		logrus.Warnf("%d stored credential(s) cannot be recovered: they are encrypted with a key that is not available, "+
			"either a lost key file or the inner layer added by key rotation in PMM before 3.9.1 (%s). "+
			"Their stored values are kept in the migration backup. "+
			"Re-enter the credentials of the affected services; if you still have that key, place it at %s and restart PMM Server.",
			len(lost), errors.Join(lost...), encryption.LegacyBackupKeyPath(encryption.DefaultKeyPath()))
	}

	if len(backup) != 0 {
		path, err := writeMigrationBackup(backup)
		if err != nil {
			return fmt.Errorf("refusing to migrate encrypted data without a backup: %w", err)
		}
		logrus.Infof("Stored the pre-migration values of rewritten rows in %s", path)
	}

	for i, t := range secretTables {
		for _, id := range scans[i].needs {
			record := t.newRecord()
			err = q.SelectOneTo(record, "WHERE "+t.idColumn+" = $1 FOR UPDATE", id)
			if errors.Is(err, reform.ErrNoRows) {
				continue // removed in the meantime
			}
			if err == nil {
				err = q.UpdateColumns(record, t.columnNames()...)
			}
			if err != nil {
				return fmt.Errorf("failed to re-encrypt %s %s: %w", t.label, id, err)
			}
		}
	}

	if writeCheck {
		return writeKeyCheck(q, cipher)
	}

	return nil
}

// secretColumn is a column holding an encrypted secret: a scalar
// EncryptedString, or a JSON options blob with encrypt-tagged fields.
type secretColumn struct {
	name string
	// options returns a new options struct for the blob; nil for a scalar
	options func() any
}

// secretTable describes a table whose rows hold encrypted secrets.
type secretTable struct {
	name      string
	idColumn  string
	label     string
	newRecord func() reform.Record
	// mayHoldPlaintext is an SQL expression over the row: true when PMM 3.x
	// may have stored its secrets as plaintext even in columns it recorded as
	// encrypted (see legacyEncryptedColumns)
	mayHoldPlaintext string
	columns          []secretColumn
}

func (t secretTable) columnNames() []string {
	names := make([]string, len(t.columns))
	for i, c := range t.columns {
		names[i] = c.name
	}

	return names
}

func options[T any]() func() any { return func() any { return new(T) } }

var (
	agentsSecretTable = secretTable{
		name:             "agents",
		idColumn:         "agent_id",
		label:            "agent",
		newRecord:        func() reform.Record { return &Agent{} },
		mayHoldPlaintext: "agent_type = '" + string(PMMAgentType) + "'",
		columns: []secretColumn{
			{"username", nil},
			{"password", nil},
			{"agent_password", nil},
			{"aws_options", options[AWSOptions]()},
			{"azure_options", options[AzureOptions]()},
			{"mongo_options", options[MongoDBOptions]()},
			{"mysql_options", options[MySQLOptions]()},
			{"postgresql_options", options[PostgreSQLOptions]()},
			{"valkey_options", options[ValkeyOptions]()},
		},
	}
	locationsSecretTable = secretTable{
		name:      "backup_locations",
		idColumn:  "id",
		label:     "backup location",
		newRecord: func() reform.Record { return &BackupLocation{} },
		// PMM 3.x never encrypted S3 credentials
		mayHoldPlaintext: "TRUE",
		columns:          []secretColumn{{"s3_config", options[S3LocationConfig]()}},
	}
	secretTables = []secretTable{agentsSecretTable, locationsSecretTable}
)

// AgentsNeedingReencryption returns IDs of agents with at least one stored
// secret that is not encrypted with the primary key in the envelope format.
// It fails when a stored secret is ciphertext the cipher cannot decrypt (see
// errUndecryptable): rewriting such a row would store the ciphertext as if it
// were the secret.
func AgentsNeedingReencryption(q *reform.Querier, cipher *encryption.Cipher) ([]string, error) {
	return needingReencryption(q, cipher, agentsSecretTable)
}

// LocationsNeedingReencryption is AgentsNeedingReencryption for backup
// locations' S3 credentials.
func LocationsNeedingReencryption(q *reform.Querier, cipher *encryption.Cipher) ([]string, error) {
	return needingReencryption(q, cipher, locationsSecretTable)
}

func needingReencryption(q *reform.Querier, cipher *encryption.Cipher, t secretTable) ([]string, error) {
	scan, err := scanSecrets(q, cipher, t)
	if err != nil {
		return nil, err
	}
	undecryptable, _ := resolveUnknownKeys(scan.unknownKey, scan.keyReadsData)
	undecryptable = append(undecryptable, scan.undecryptable...)
	if len(undecryptable) != 0 {
		return nil, errUndecryptable(undecryptable, "")
	}

	return scan.needs, nil
}

// resolveUnknownKeys decides about ciphertext of keys the keyset does not hold,
// found in columns PMM 3.x recorded as encrypted. If the keyset reads none of
// the stored ciphertext, the key file does not match the database: restoring
// the right one fixes everything, so these values are undecryptable. If it
// reads some, it is the key PMM 3.x silently generated after the original was
// lost; no key file reads everything then, so refusing to start would only
// take PMM Server down, and these values are reported as lost instead.
func resolveUnknownKeys(unknownKey []error, keyReadsData bool) ([]error, []error) {
	if keyReadsData {
		return nil, unknownKey
	}

	return unknownKey, nil
}

// ErrEncryptionKeyMismatch is returned when the encryption key cannot decrypt
// the stored data, or is missing while the database holds data encrypted with
// it. Nothing is changed, and retrying does not help until the key the data
// was encrypted with is in place.
var ErrEncryptionKeyMismatch = errors.New("encryption key does not match the database")

// keyError is a refusal to start that only the right key file resolves.
type keyError struct {
	msg string
}

func (e *keyError) Error() string {
	return e.msg
}

// Is reports the error as ErrEncryptionKeyMismatch.
func (e *keyError) Is(target error) bool {
	return target == ErrEncryptionKeyMismatch
}

// errUndecryptable explains that the key does not match the stored data.
// Its causes can be matched with errors.Is, e.g. encryption.ErrLegacyUnknownKey.
// A non-empty acceptValue is the value of encryption.AcceptKeyLossEnvVar that
// accepts the loss of the key the data was encrypted with.
func errUndecryptable(problems []error, acceptValue string) error {
	return &undecryptableError{problems: problems, acceptValue: acceptValue}
}

type undecryptableError struct {
	problems    []error
	acceptValue string
}

func (e *undecryptableError) Error() string {
	const shown = 10
	list := make([]string, 0, shown)
	for _, p := range e.problems[:min(len(e.problems), shown)] {
		list = append(list, p.Error())
	}
	more := ""
	if len(e.problems) > shown {
		more = fmt.Sprintf(" and %d more", len(e.problems)-shown)
	}

	msg := fmt.Sprintf("the encryption key at %s cannot decrypt stored credentials (%s%s): "+
		"the key file does not match this database, or the values are corrupted; "+
		"if the key file was replaced, restore the original one (or point %s at it) and restart — no data was changed",
		encryption.DefaultKeyPath(), strings.Join(list, "; "), more, encryption.CustomEncryptionKeyPathEnvVar)
	if e.acceptValue != "" {
		msg += fmt.Sprintf(". If the key is lost for good, set %s=%s and restart: the values it cannot decrypt are kept "+
			"in a backup file and rewritten as lost; placing the lost key at %s later makes them readable again",
			encryption.AcceptKeyLossEnvVar, e.acceptValue, encryption.LegacyBackupKeyPath(encryption.DefaultKeyPath()))
	}

	return msg
}

func (e *undecryptableError) Unwrap() []error {
	return e.problems
}

// Is reports the error as ErrEncryptionKeyMismatch.
func (e *undecryptableError) Is(target error) bool {
	return target == ErrEncryptionKeyMismatch
}

type secretsScan struct {
	// needs lists the IDs of rows to rewrite
	needs         []string
	undecryptable []error
	// lost lists secrets whose innermost layer's key is gone; they cannot be
	// recovered by PMM and are reported, not treated as a key mismatch
	lost []error
	// unknownKey lists ciphertext of keys the keyset does not hold, in columns
	// PMM 3.x recorded as encrypted; see resolveUnknownKeys
	unknownKey []error
	// keyReadsData is set when the keyset decrypted at least one stored value
	keyReadsData bool
	// backup holds the stored columns of rows rewritten from the pre-envelope
	// format or holding lost secrets, see writeMigrationBackup
	backup []map[string]any
}

// scanSecrets classifies the stored secrets of every row of t without
// decrypting them into the model.
func scanSecrets(q *reform.Querier, cipher *encryption.Cipher, t secretTable) (*secretsScan, error) {
	strict, err := storedLegacyEncryptedColumns(q)
	if err != nil {
		return nil, err
	}

	rows, err := q.Query("SELECT " + t.idColumn + ", " + t.mayHoldPlaintext + ", " + strings.Join(t.columnNames(), ", ") +
		" FROM " + t.name + " ORDER BY " + t.idColumn)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", t.name, err)
	}
	defer rows.Close() //nolint:errcheck

	scan := &secretsScan{}
	for rows.Next() {
		var id string
		var mayHoldPlaintext bool
		values := make([]sql.NullString, len(t.columns))
		dest := []any{&id, &mayHoldPlaintext}
		for i := range values {
			dest = append(dest, &values[i])
		}
		err = rows.Scan(dest...)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", t.name, err)
		}

		insp := &inspector{cipher: cipher}
		if !mayHoldPlaintext {
			insp.strict = strict
		}
		stored := map[string]any{t.idColumn: id}
		for i, c := range t.columns {
			err = insp.column(c, values[i])
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", t.label, id, err)
			}
			stored[c.name] = storedValue(c, values[i])
		}

		scan.keyReadsData = scan.keyReadsData || insp.decrypted
		for _, p := range insp.undecryptable {
			scan.undecryptable = append(scan.undecryptable, fmt.Errorf("%s %s %s: %w", t.label, id, p.column, p.err))
		}
		for _, p := range insp.unknownKey {
			scan.unknownKey = append(scan.unknownKey, fmt.Errorf("%s %s %s: %w", t.label, id, p.column, p.err))
		}
		if !insp.needs || len(insp.undecryptable) != 0 {
			continue
		}
		// reported once, when the row is rewritten; later reads see an
		// envelope around the unrecoverable inner layer
		for _, p := range insp.lost {
			scan.lost = append(scan.lost, fmt.Errorf("%s %s %s", t.label, id, p.column))
		}
		scan.needs = append(scan.needs, id)
		if insp.preEnvelope || len(insp.lost) != 0 {
			scan.backup = append(scan.backup, stored)
		}
	}

	return scan, rows.Err()
}

// storedValue returns a column as stored, for the migration backup: JSON
// blobs stay JSON.
func storedValue(c secretColumn, v sql.NullString) any {
	switch {
	case !v.Valid:
		return nil
	case c.options != nil && json.Valid([]byte(v.String)):
		return json.RawMessage(v.String)
	default:
		return v.String
	}
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
	// unknownKey holds ciphertext of keys the keyset does not hold in strict
	// columns; whether that is a key mismatch is decided across all rows
	unknownKey []columnProblem
	// decrypted is set when the keyset decrypted one of the row's values
	decrypted bool
	// preEnvelope is set when a secret is not in the envelope format yet:
	// the row is written by PMM before this migration existed
	preEnvelope bool
}

func (i *inspector) value(column, stored string) {
	if stored == "" {
		return
	}
	if !encryption.IsEncrypted(stored) {
		i.preEnvelope = true
	}
	insp, err := i.cipher.Inspect(stored)
	if i.cipher.NeedsReencrypt(stored) || insp.ExtraLayers > 0 {
		i.needs = true
	}
	if insp.Decrypted {
		i.decrypted = true
	}

	switch {
	case err == nil:
	case errors.Is(err, encryption.ErrLegacyUnknownKey) && !i.strict[column]:
		// plaintext that happens to look like ciphertext
	case errors.Is(err, encryption.ErrLegacyUnknownKey):
		i.unknownKey = append(i.unknownKey, columnProblem{column, err})
	case errors.Is(err, encryption.ErrLegacyInnerKeyLost):
		// the key matches; the secret was lost to stacked layers before
		i.lost = append(i.lost, columnProblem{column, err})
	case errors.Is(err, encryption.ErrUnknownKey) && i.cipher.AcceptsKeyLoss():
		// the administrator has accepted that its key is gone
		i.lost = append(i.lost, columnProblem{column, err})
	default:
		i.undecryptable = append(i.undecryptable, columnProblem{column, err})
	}
}

// column inspects a scalar secret, or the tagged secret sub-fields of an
// options blob, unmarshaled without decrypting them.
func (i *inspector) column(c secretColumn, v sql.NullString) error {
	if !v.Valid {
		return nil
	}
	if c.options == nil {
		i.value(c.name, v.String)
		return nil
	}

	target := c.options()
	err := json.Unmarshal([]byte(v.String), target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", c.name, err)
	}

	return applyToSecretFields(target, func(s string) (string, error) {
		i.value(c.name, s)
		return s, nil
	})
}
