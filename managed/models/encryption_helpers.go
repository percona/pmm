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
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// ErrEncryptionKeyMismatch is returned when this node's encryption key is not the key the data
// in the database was encrypted with.
var ErrEncryptionKeyMismatch = errors.New("encryption key does not match the database")

// encryptionKeyLockID is the PostgreSQL advisory lock that serializes the encryption key check
// between PMM Server nodes starting against the same database.
const encryptionKeyLockID = 14979

// undecryptableWarned holds the Agent fields already warned about, since Agents are read many
// times per second.
var undecryptableWarned sync.Map

// EncryptAgent encrypt agent.
// A field DecryptAgent could not decrypt is written back with its stored ciphertext, unless the
// caller has set a new value in the meantime.
func EncryptAgent(agent Agent) (Agent, error) {
	undecrypted := agent.undecrypted
	agent.undecrypted = nil

	for _, s := range agentSecrets(&agent) {
		stored, ok := undecrypted[s.name]
		if ok && *s.val == "" {
			*s.val = stored
			continue
		}

		res, err := encryption.Encrypt(*s.val)
		if err != nil {
			return agent, fmt.Errorf("agent %s: %s: %w", agent.AgentID, s.name, err)
		}
		*s.val = res
	}

	return agent, nil
}

// DecryptAgent decrypt agent.
// A field this node's key cannot decrypt is logged and left empty, so that one unreadable row
// does not fail every query over Agents.
func DecryptAgent(agent Agent) Agent {
	var undecrypted map[string]string
	for _, s := range agentSecrets(&agent) {
		res, err := encryption.Decrypt(*s.val)
		if err != nil {
			l := logrus.WithFields(logrus.Fields{"agent_id": agent.AgentID, "field": s.name})
			_, warned := undecryptableWarned.LoadOrStore(agent.AgentID+" "+s.name, struct{}{})
			if warned {
				l.Debugf("Cannot decrypt agent credentials: %s.", err)
			} else {
				l.Warnf("Cannot decrypt agent credentials: %s.", err)
			}

			if undecrypted == nil {
				undecrypted = make(map[string]string)
			}
			undecrypted[s.name] = *s.val
		}
		*s.val = res
	}
	agent.undecrypted = undecrypted

	return agent
}

type agentSecret struct {
	name string
	val  *string
}

// agentSecrets returns the encrypted fields of the Agent. Its *string fields are shared with the
// Agent it was copied from, so they are replaced with copies that can be written through.
func agentSecrets(agent *Agent) []agentSecret {
	secrets := []agentSecret{
		{"aws_options.access_key", &agent.AWSOptions.AWSAccessKey},
		{"aws_options.secret_key", &agent.AWSOptions.AWSSecretKey},
		{"azure_options.client_id", &agent.AzureOptions.ClientID},
		{"azure_options.client_secret", &agent.AzureOptions.ClientSecret},
		{"azure_options.subscription_id", &agent.AzureOptions.SubscriptionID},
		{"azure_options.tenant_id", &agent.AzureOptions.TenantID},
		{"mongo_options.tls_certificate_key", &agent.MongoDBOptions.TLSCertificateKey},
		{"mongo_options.tls_certificate_key_file_password", &agent.MongoDBOptions.TLSCertificateKeyFilePassword},
		{"mysql_options.tls_cert", &agent.MySQLOptions.TLSCert},
		{"mysql_options.tls_key", &agent.MySQLOptions.TLSKey},
		{"postgresql_options.ssl_cert", &agent.PostgreSQLOptions.SSLCert},
		{"postgresql_options.ssl_key", &agent.PostgreSQLOptions.SSLKey},
	}

	for _, f := range []struct {
		name string
		val  **string
	}{
		{"username", &agent.Username},
		{"password", &agent.Password},
		{"agent_password", &agent.AgentPassword},
	} {
		if *f.val == nil {
			continue
		}

		*f.val = new(**f.val)
		secrets = append(secrets, agentSecret{f.name, *f.val})
	}

	return secrets
}

// CheckEncryptionKey returns ErrEncryptionKeyMismatch if this node does not hold the encryption
// key the database was encrypted with.
//
// Every node of an HA cluster shares one database but keeps its own key file, so a node that
// generated its own key cannot decrypt the credentials written by the others.
func CheckEncryptionKey(q reform.DBTX) error {
	_, _, err := checkEncryptionKey(q)
	return err
}

// VerifyEncryptionKey is CheckEncryptionKey that also records this node's key fingerprint when
// none is stored yet. It holds a lock until tx ends, so that nodes starting at the same time
// cannot each record their own key.
func VerifyEncryptionKey(tx *reform.TX) error {
	_, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", encryptionKeyLockID)
	if err != nil {
		return fmt.Errorf("failed to lock the encryption key check: %w", err)
	}

	settings, fingerprint, err := checkEncryptionKey(tx)
	if err != nil {
		return err
	}
	if settings.EncryptionKeyFingerprint != "" {
		return nil
	}

	settings.EncryptionKeyFingerprint = fingerprint

	return SaveSettings(tx, settings)
}

// adoptEncryptionKey replaces a foreign fingerprint with this node's own once this node's key
// decrypts every stored agent username and password, so that a server whose key was lost recovers
// after its credentials have been re-entered. It returns whether the key was adopted.
// Never call it in HA: while no credentials are stored, a node with its own key would take the
// database over from the others.
func adoptEncryptionKey(tx *reform.TX) (bool, error) {
	settings, err := GetSettings(tx)
	if err != nil {
		return false, err
	}

	err = checkStoredSecretsReadable(tx, settings)
	if errors.Is(err, ErrEncryptionKeyMismatch) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	fingerprint, err := encryption.Fingerprint()
	if err != nil {
		return false, err
	}

	logrus.Warnf("Adopting encryption key fingerprint %s in place of %s: this server's key decrypts every stored agent credential.",
		fingerprint, settings.EncryptionKeyFingerprint)
	settings.EncryptionKeyFingerprint = fingerprint

	return true, SaveSettings(tx, settings)
}

// checkEncryptionKey returns the settings and this node's key fingerprint.
func checkEncryptionKey(q reform.DBTX) (*Settings, string, error) {
	fingerprint, err := encryption.Fingerprint()
	if err != nil {
		return nil, "", err
	}

	settings, err := GetSettings(q)
	if err != nil {
		return nil, "", err
	}

	if settings.EncryptionKeyFingerprint == "" {
		// Either a fresh install or an upgrade from a version that did not record the
		// fingerprint. This node's key is accepted only if it can read everything already stored.
		return settings, fingerprint, checkStoredSecretsReadable(q, settings)
	}

	if settings.EncryptionKeyFingerprint != fingerprint {
		return nil, "", keyMismatchError(fingerprint, settings.EncryptionKeyFingerprint)
	}

	return settings, fingerprint, nil
}

func keyMismatchError(local, stored string) error {
	return fmt.Errorf("%w: this node's key fingerprint is %s, the database was encrypted with %s",
		ErrEncryptionKeyMismatch, local, stored)
}

// checkStoredSecretsReadable decrypts every stored agent username and password to tell a
// matching key from a foreign one on databases that carry no fingerprint yet.
func checkStoredSecretsReadable(q reform.DBTX, settings *Settings) error {
	var total, unreadable int
	for _, column := range []string{"username", "password"} {
		encrypted := slices.ContainsFunc(settings.EncryptedItems, func(item string) bool {
			return strings.HasSuffix(item, ".agents."+column)
		})
		if !encrypted {
			// The column holds plaintext, so nothing in it can contradict this key.
			continue
		}

		t, u, err := countUnreadable(q, column)
		if err != nil {
			return err
		}
		total += t
		unreadable += u
	}

	if unreadable > 0 {
		return fmt.Errorf("%w: %d of %d stored agent credentials cannot be decrypted with this node's key",
			ErrEncryptionKeyMismatch, unreadable, total)
	}

	return nil
}

// countUnreadable returns how many non-empty values the column holds and how many of them this
// node's key cannot decrypt.
func countUnreadable(q reform.DBTX, column string) (int, int, error) {
	rows, err := q.Query(fmt.Sprintf("SELECT %[1]s FROM agents WHERE %[1]s IS NOT NULL AND %[1]s != ''", column))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read stored agent credentials: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var total, unreadable int
	for rows.Next() {
		var value string
		err = rows.Scan(&value)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to read stored agent credentials: %w", err)
		}

		total++
		_, err = encryption.Decrypt(value)
		if err != nil {
			unreadable++
		}
	}

	err = rows.Err()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read stored agent credentials: %w", err)
	}

	return total, unreadable, nil
}

// EncryptAWSOptionsHandler returns encrypted AWS Options.
func EncryptAWSOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return awsOptionsHandler(val, e.Encrypt)
}

// DecryptAWSOptionsHandler returns decrypted AWS Options.
func DecryptAWSOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return awsOptionsHandler(val, e.Decrypt)
}

func awsOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := AWSOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.AWSAccessKey, err = handler(o.AWSAccessKey)
	if err != nil {
		return nil, err
	}
	o.AWSSecretKey, err = handler(o.AWSSecretKey)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptAzureOptionsHandler returns encrypted Azure Options.
func EncryptAzureOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return azureOptionsHandler(val, e.Encrypt)
}

// DecryptAzureOptionsHandler returns decrypted Azure Options.
func DecryptAzureOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return azureOptionsHandler(val, e.Decrypt)
}

func azureOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := AzureOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.ClientID, err = handler(o.ClientID)
	if err != nil {
		return nil, err
	}
	o.ClientSecret, err = handler(o.ClientSecret)
	if err != nil {
		return nil, err
	}
	o.SubscriptionID, err = handler(o.SubscriptionID)
	if err != nil {
		return nil, err
	}
	o.TenantID, err = handler(o.TenantID)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o) //nolint:gosec
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptMongoDBOptionsHandler returns encrypted MongoDB Options.
func EncryptMongoDBOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mongoDBOptionsHandler(val, e.Encrypt)
}

// DecryptMongoDBOptionsHandler returns decrypted MongoDB Options.
func DecryptMongoDBOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mongoDBOptionsHandler(val, e.Decrypt)
}

func mongoDBOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := MongoDBOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.TLSCertificateKey, err = handler(o.TLSCertificateKey)
	if err != nil {
		return nil, err
	}
	o.TLSCertificateKeyFilePassword, err = handler(o.TLSCertificateKeyFilePassword)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptMySQLOptionsHandler returns encrypted MySQL Options.
func EncryptMySQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mySQLOptionsHandler(val, e.Encrypt)
}

// DecryptMySQLOptionsHandler returns decrypted MySQL Options.
func DecryptMySQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return mySQLOptionsHandler(val, e.Decrypt)
}

func mySQLOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := MySQLOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.TLSCert, err = handler(o.TLSCert)
	if err != nil {
		return nil, err
	}
	o.TLSKey, err = handler(o.TLSKey)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EncryptPostgreSQLOptionsHandler returns encrypted PostgreSQL Options.
func EncryptPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return postgreSQLOptionsHandler(val, e.Encrypt)
}

// DecryptPostgreSQLOptionsHandler returns decrypted PostgreSQL Options.
func DecryptPostgreSQLOptionsHandler(e *encryption.Encryption, val any) (any, error) {
	return postgreSQLOptionsHandler(val, e.Decrypt)
}

func postgreSQLOptionsHandler(val any, handler func(string) (string, error)) (any, error) {
	o := PostgreSQLOptions{}
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	err := json.Unmarshal([]byte(value.String), &o)
	if err != nil {
		return nil, err
	}

	o.SSLCert, err = handler(o.SSLCert)
	if err != nil {
		return nil, err
	}
	o.SSLKey, err = handler(o.SSLKey)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}
