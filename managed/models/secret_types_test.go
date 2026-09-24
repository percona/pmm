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

package models_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

// TestAgentSecretsRoundTrip guards the codec invariant that replaced the
// per-column encrypt/decrypt handlers: every secret an agent carries is stored
// as a single-layer envelope and scans back to its original value, while the
// untagged option fields stay plaintext. A double-encrypted option field is
// what broke encryption key rotation in
// https://perconadev.atlassian.net/browse/PMM-15188.
func TestAgentSecretsRoundTrip(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})
	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	require.NoError(t, q.Insert(&models.Node{NodeID: "N1", NodeType: models.GenericNodeType, NodeName: "name"}))

	original := &models.Agent{
		AgentID:       "A1",
		AgentType:     models.PMMAgentType,
		RunsOnNodeID:  new("N1"),
		Username:      models.EncryptedStringOrNil("username"),
		Password:      models.EncryptedStringOrNil("password"),
		AgentPassword: models.EncryptedStringOrNil("agent-password"),
		AWSOptions: models.AWSOptions{
			AWSAccessKey:            "aws-access-key",
			AWSSecretKey:            "aws-secret-key",
			RDSBasicMetricsDisabled: true,
		},
		AzureOptions: models.AzureOptions{
			SubscriptionID: "azure-subscription-id",
			ClientID:       "azure-client-id",
			ClientSecret:   "azure-client-secret",
			TenantID:       "azure-tenant-id",
			ResourceGroup:  "azure-resource-group",
		},
		MongoDBOptions: models.MongoDBOptions{
			TLSCertificateKey:             "mongo-tls-certificate-key",
			TLSCertificateKeyFilePassword: "mongo-tls-certificate-key-file-password",
			TLSCa:                         "mongo-tls-ca",
		},
		MySQLOptions: models.MySQLOptions{
			TLSCa:   "mysql-tls-ca",
			TLSCert: "mysql-tls-cert",
			TLSKey:  "mysql-tls-key",
		},
		PostgreSQLOptions: models.PostgreSQLOptions{
			SSLCa:   "postgresql-ssl-ca",
			SSLCert: "postgresql-ssl-cert",
			SSLKey:  "postgresql-ssl-key",
		},
		ValkeyOptions: models.ValkeyOptions{
			SSLCa:   "valkey-ssl-ca",
			SSLCert: "valkey-ssl-cert",
			SSLKey:  "valkey-ssl-key",
		},
	}
	require.NoError(t, q.Insert(original))
	// Value must not mutate the caller's struct.
	assert.Equal(t, "mysql-tls-cert", original.MySQLOptions.TLSCert)

	var username, password, agentPassword string
	var aws, azure, mongo, mysql, pg, valkey []byte
	err = sqlDB.QueryRowContext(t.Context(),
		"SELECT username, password, agent_password, aws_options, azure_options, mongo_options, mysql_options, postgresql_options, valkey_options "+
			"FROM agents WHERE agent_id = $1", "A1").
		Scan(&username, &password, &agentPassword, &aws, &azure, &mongo, &mysql, &pg, &valkey)
	require.NoError(t, err)

	for _, stored := range []string{username, password, agentPassword} {
		assertSingleEnvelope(t, cipher, stored)
	}
	assertStoredOptions(t, cipher, aws, original.AWSOptions)
	assertStoredOptions(t, cipher, azure, original.AzureOptions)
	assertStoredOptions(t, cipher, mongo, original.MongoDBOptions)
	assertStoredOptions(t, cipher, mysql, original.MySQLOptions)
	assertStoredOptions(t, cipher, pg, original.PostgreSQLOptions)
	assertStoredOptions(t, cipher, valkey, original.ValkeyOptions)

	agent, err := models.FindAgentByID(q, "A1")
	require.NoError(t, err)
	assert.Equal(t, "username", agent.Username.Reveal())
	assert.Equal(t, "password", agent.Password.Reveal())
	assert.Equal(t, "agent-password", agent.AgentPassword.Reveal())
	assert.Equal(t, original.AWSOptions, agent.AWSOptions)
	assert.Equal(t, original.AzureOptions, agent.AzureOptions)
	assert.Equal(t, original.MongoDBOptions, agent.MongoDBOptions)
	assert.Equal(t, original.MySQLOptions, agent.MySQLOptions)
	assert.Equal(t, original.PostgreSQLOptions, agent.PostgreSQLOptions)
	assert.Equal(t, original.ValkeyOptions, agent.ValkeyOptions)
}

// assertSingleEnvelope checks that stored is an envelope of the primary key
// whose plaintext is not itself ciphertext.
func assertSingleEnvelope(t *testing.T, cipher *encryption.Cipher, stored string) {
	t.Helper()

	keyID, ok := encryption.StoredKeyID(stored)
	require.True(t, ok, "not an envelope: %q", stored)
	assert.Equal(t, cipher.PrimaryKeyID(), keyID)

	plaintext, err := cipher.Decrypt(stored)
	require.NoError(t, err)
	assert.False(t, encryption.IsEncrypted(plaintext), "encrypted more than once")
}

// assertStoredOptions checks a raw JSON options column against the struct it
// was written from: fields tagged encrypt:"true" hold envelopes, all other
// string fields are stored as is.
func assertStoredOptions[T any](t *testing.T, cipher *encryption.Cipher, raw []byte, original T) {
	t.Helper()

	var stored T
	require.NoError(t, json.Unmarshal(raw, &stored))

	storedVal := reflect.ValueOf(stored)
	originalVal := reflect.ValueOf(original)
	typ := storedVal.Type()
	var secrets int
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type.Kind() != reflect.String {
			continue
		}
		if field.Tag.Get("encrypt") != "true" {
			assert.Equal(t, originalVal.Field(i).String(), storedVal.Field(i).String(), "%s.%s", typ.Name(), field.Name)
			continue
		}
		secrets++
		assertSingleEnvelope(t, cipher, storedVal.Field(i).String())
	}
	assert.Positive(t, secrets, "%s has no encrypted fields", typ.Name())
}
