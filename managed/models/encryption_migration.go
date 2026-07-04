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

	"github.com/lib/pq"
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
	encryption.SetDefaultCipher(cipher)

	return nil
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
		var s struct {
			EncryptedItems []string `json:"encrypted_items"`
		}
		err = json.Unmarshal(settingsJSON, &s)
		if err != nil {
			return false, fmt.Errorf("failed to probe settings for encrypted data: %w", err)
		}
		if len(s.EncryptedItems) != 0 {
			return true, nil
		}
	}

	var exists bool
	err = db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM agents WHERE username LIKE '"+encryption.EnvelopePrefix+"%'"+
			" OR password LIKE '"+encryption.EnvelopePrefix+"%'"+
			" OR agent_password LIKE '"+encryption.EnvelopePrefix+"%')").Scan(&exists)
	if isUndefinedTable(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to probe agents for encrypted data: %w", err)
	}

	return exists, nil
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

	ids, err := AgentsNeedingReencryption(q, cipher)
	if err != nil {
		return err
	}

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
func AgentsNeedingReencryption(q *reform.Querier, cipher *encryption.Cipher) ([]string, error) {
	rows, err := q.Query(`
		SELECT agent_id, username, password, agent_password,
			aws_options, azure_options, mongo_options, mysql_options, postgresql_options, valkey_options
		FROM agents
		ORDER BY agent_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ids []string
	for rows.Next() {
		var id string
		var username, password, agentPassword sql.NullString
		var aws, azure, mongo, mysql, postgresql, valkey sql.NullString
		err = rows.Scan(&id, &username, &password, &agentPassword, &aws, &azure, &mongo, &mysql, &postgresql, &valkey)
		if err != nil {
			return nil, fmt.Errorf("failed to read agents: %w", err)
		}

		needs, err := agentNeedsReencryption(cipher, agentSecrets{
			strings: []sql.NullString{username, password, agentPassword},
			options: []optionsColumn{
				{aws, &AWSOptions{}},
				{azure, &AzureOptions{}},
				{mongo, &MongoDBOptions{}},
				{mysql, &MySQLOptions{}},
				{postgresql, &PostgreSQLOptions{}},
				{valkey, &ValkeyOptions{}},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("agent %s: %w", id, err)
		}
		if needs {
			ids = append(ids, id)
		}
	}

	return ids, rows.Err()
}

// LocationsNeedingReencryption returns IDs of backup locations whose stored
// S3 credentials are not encrypted with the primary key in the envelope format.
func LocationsNeedingReencryption(q *reform.Querier, cipher *encryption.Cipher) ([]string, error) {
	rows, err := q.Query(`SELECT id, s3_config FROM backup_locations ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup locations: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ids []string
	for rows.Next() {
		var id string
		var s3Config sql.NullString
		err = rows.Scan(&id, &s3Config)
		if err != nil {
			return nil, fmt.Errorf("failed to read backup locations: %w", err)
		}

		needs, err := optionsNeedReencryption(cipher, s3Config, &S3LocationConfig{})
		if err != nil {
			return nil, fmt.Errorf("backup location %s: %w", id, err)
		}
		if needs {
			ids = append(ids, id)
		}
	}

	return ids, rows.Err()
}

type optionsColumn struct {
	raw    sql.NullString
	target any
}

type agentSecrets struct {
	strings []sql.NullString
	options []optionsColumn
}

func agentNeedsReencryption(cipher *encryption.Cipher, secrets agentSecrets) (bool, error) {
	for _, v := range secrets.strings {
		if v.Valid && cipher.NeedsReencrypt(v.String) {
			return true, nil
		}
	}

	for _, o := range secrets.options {
		needs, err := optionsNeedReencryption(cipher, o.raw, o.target)
		if err != nil {
			return false, err
		}
		if needs {
			return true, nil
		}
	}

	return false, nil
}

// optionsNeedReencryption unmarshals a stored options blob without decrypting
// it and reports whether any of its tagged secret sub-fields needs a rewrite.
func optionsNeedReencryption(cipher *encryption.Cipher, raw sql.NullString, target any) (bool, error) {
	if !raw.Valid {
		return false, nil
	}

	err := json.Unmarshal([]byte(raw.String), target)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	needs := false
	err = applyToSecretFields(target, func(s string) (string, error) {
		if cipher.NeedsReencrypt(s) {
			needs = true
		}
		return s, nil
	})

	return needs, err
}
