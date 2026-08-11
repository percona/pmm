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

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// EncryptAgent encrypt agent.
func EncryptAgent(agent Agent) (Agent, error) {
	return agentEncryption(agent, encryption.Encrypt)
}

// DecryptAgent decrypt agent.
// On error the returned Agent is only partially decrypted and must not be used: its remaining
// fields still hold ciphertext.
func DecryptAgent(agent Agent) (Agent, error) {
	return agentEncryption(agent, encryption.Decrypt)
}

// ErrEncryptionKeyMismatch is returned when this node's encryption key is not the key the data
// in the database was encrypted with.
var ErrEncryptionKeyMismatch = errors.New("encryption key does not match the database")

// VerifyEncryptionKey reports whether this node holds the encryption key the database was
// encrypted with, and records the key's fingerprint when none is stored yet.
//
// Every node of an HA cluster shares one database but keeps its own key file, so a node that
// generated its own key cannot decrypt the credentials written by the others.
func VerifyEncryptionKey(q reform.DBTX) error {
	fingerprint, err := encryption.Fingerprint()
	if err != nil {
		return err
	}

	settings, err := GetSettings(q)
	if err != nil {
		return err
	}

	if settings.EncryptionKeyFingerprint == "" {
		// Either a fresh install or an upgrade from a version that did not record the
		// fingerprint. This node's key is adopted only if it can read what is already stored.
		err = checkStoredSecretsReadable(q, settings)
		if err != nil {
			return err
		}

		settings.EncryptionKeyFingerprint = fingerprint

		return SaveSettings(q, settings)
	}

	if settings.EncryptionKeyFingerprint != fingerprint {
		return fmt.Errorf("%w: this node's key fingerprint is %s, the database was encrypted with %s",
			ErrEncryptionKeyMismatch, fingerprint, settings.EncryptionKeyFingerprint)
	}

	return nil
}

// checkStoredSecretsReadable decrypts one stored agent username to tell a matching key from a
// foreign one on databases that carry no fingerprint yet.
func checkStoredSecretsReadable(q reform.DBTX, settings *Settings) error {
	encrypted := slices.ContainsFunc(settings.EncryptedItems, func(item string) bool {
		return strings.HasSuffix(item, ".agents.username")
	})
	if !encrypted {
		// The probed column holds plaintext, so nothing there can contradict this key.
		return nil
	}

	var username string
	err := q.QueryRow("SELECT username FROM agents WHERE username IS NOT NULL AND username != '' LIMIT 1").Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read stored agent credentials: %w", err)
	}

	_, err = encryption.Decrypt(username)
	if err != nil {
		return fmt.Errorf("%w: stored agent credentials cannot be decrypted with this node's key: %w",
			ErrEncryptionKeyMismatch, err)
	}

	return nil
}

func agentEncryption(agent Agent, handler func(string) (string, error)) (Agent, error) {
	apply := func(name string, val *string) error {
		res, err := handler(*val)
		if err != nil {
			return fmt.Errorf("agent %s: %s: %w", agent.AgentID, name, err)
		}
		*val = res

		return nil
	}

	// The *string fields are shared with the caller's Agent, so a copy is assigned instead of
	// writing through the existing pointer.
	ptr := func(name string, val *string) (*string, error) {
		if val == nil {
			return nil, nil //nolint:nilnil
		}

		res := *val
		err := apply(name, &res)
		if err != nil {
			return nil, err
		}

		return &res, nil
	}

	var err error

	agent.Username, err = ptr("username", agent.Username)
	if err != nil {
		return agent, err
	}

	agent.Password, err = ptr("password", agent.Password)
	if err != nil {
		return agent, err
	}

	agent.AgentPassword, err = ptr("agent_password", agent.AgentPassword)
	if err != nil {
		return agent, err
	}

	for _, f := range []struct {
		nonEmpty bool
		name     string
		val      *string
	}{
		{!agent.AWSOptions.IsEmpty(), "aws_options.access_key", &agent.AWSOptions.AWSAccessKey},
		{!agent.AWSOptions.IsEmpty(), "aws_options.secret_key", &agent.AWSOptions.AWSSecretKey},
		{!agent.AzureOptions.IsEmpty(), "azure_options.client_id", &agent.AzureOptions.ClientID},
		{!agent.AzureOptions.IsEmpty(), "azure_options.client_secret", &agent.AzureOptions.ClientSecret},
		{!agent.AzureOptions.IsEmpty(), "azure_options.subscription_id", &agent.AzureOptions.SubscriptionID},
		{!agent.AzureOptions.IsEmpty(), "azure_options.tenant_id", &agent.AzureOptions.TenantID},
		{!agent.MongoDBOptions.IsEmpty(), "mongo_options.tls_certificate_key", &agent.MongoDBOptions.TLSCertificateKey},
		{!agent.MongoDBOptions.IsEmpty(), "mongo_options.tls_certificate_key_file_password", &agent.MongoDBOptions.TLSCertificateKeyFilePassword},
		{!agent.MySQLOptions.IsEmpty(), "mysql_options.tls_cert", &agent.MySQLOptions.TLSCert},
		{!agent.MySQLOptions.IsEmpty(), "mysql_options.tls_key", &agent.MySQLOptions.TLSKey},
		{!agent.PostgreSQLOptions.IsEmpty(), "postgresql_options.ssl_cert", &agent.PostgreSQLOptions.SSLCert},
		{!agent.PostgreSQLOptions.IsEmpty(), "postgresql_options.ssl_key", &agent.PostgreSQLOptions.SSLKey},
	} {
		if !f.nonEmpty {
			continue
		}

		err = apply(f.name, f.val)
		if err != nil {
			return agent, err
		}
	}

	return agent, nil
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

	res, err := json.Marshal(o)
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
