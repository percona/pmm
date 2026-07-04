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
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestMigrateEncryption(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	cipher, err := encryption.DefaultCipher()
	require.NoError(t, err)

	// nothing is encrypted yet
	hasEncrypted, err := models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.False(t, hasEncrypted)

	// legacy format is base64 of the Tink ciphertext without the envelope prefix
	toLegacy := func(plaintext string) string {
		stored, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		return strings.TrimPrefix(stored, encryption.EnvelopePrefix)
	}

	awsOptions, err := json.Marshal(map[string]any{
		"aws_access_key":                toLegacy("AKIAIOSFODNN7EXAMPLE"),
		"aws_secret_key":                toLegacy("aws-secret-key"),
		"rds_basic_metrics_disabled":    true,
		"rds_enhanced_metrics_disabled": false,
	})
	require.NoError(t, err)
	// pre-encryption blob: plaintext secrets inside JSON
	mysqlOptions, err := json.Marshal(map[string]any{
		"tls_cert": "-----BEGIN CERTIFICATE-----plain-----END CERTIFICATE-----",
		"tls_key":  "-----BEGIN PRIVATE KEY-----plain-----END PRIVATE KEY-----",
	})
	require.NoError(t, err)

	now := time.Now()
	_, err = sqlDB.ExecContext(t.Context(),
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('N1', 'generic', 'name', '', '', '', '', $1, $2)",
		now, now,
	)
	require.NoError(t, err)
	// username in legacy format, password in pre-encryption plaintext
	//nolint:dupword
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO agents (agent_id, agent_type, username, password, runs_on_node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, aws_options, mysql_options) `+
			`VALUES ('A1', 'pmm-agent', $1, $2, 'N1', false, '', $3, $4, false, false, $5, $6)`,
		toLegacy("legacy-user"), "plain-password", now, now, string(awsOptions), string(mysqlOptions),
	)
	require.NoError(t, err)
	// backup location with pre-encryption plaintext S3 credentials
	_, err = sqlDB.ExecContext(t.Context(),
		`INSERT INTO backup_locations (id, name, description, type, s3_config, created_at, updated_at) `+
			`VALUES ('L1', 'loc', '', 's3', '{"endpoint": "https://s3.example.com", "access_key": "s3-access-key", "secret_key": "s3-secret-key", "bucket_name": "b", "bucket_region": "r"}', $1, $2)`,
		now, now,
	)
	require.NoError(t, err)

	ids, err := models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	require.Equal(t, []string{"A1"}, ids)
	locationIDs, err := models.LocationsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	require.Equal(t, []string{"L1"}, locationIDs)

	require.NoError(t, models.MigrateEncryption(q))

	// the lost-key startup guard now detects encrypted data
	hasEncrypted, err = models.DatabaseHasEncryptedData(t.Context(), sqlDB)
	require.NoError(t, err)
	assert.True(t, hasEncrypted)

	// all stored secrets are now envelopes carrying the primary key ID
	readRaw := func() (username, password, aws, mysql string) {
		t.Helper()
		var u, p, a, m sql.NullString
		err := sqlDB.QueryRowContext(t.Context(), `SELECT username, password, aws_options, mysql_options FROM agents WHERE agent_id = 'A1'`).Scan(&u, &p, &a, &m)
		require.NoError(t, err)
		return u.String, p.String, a.String, m.String
	}
	username, password, aws, mysql := readRaw()
	for _, stored := range []string{username, password} {
		keyID, ok := encryption.StoredKeyID(stored)
		require.True(t, ok, "value %q is not an envelope", stored)
		assert.Equal(t, cipher.PrimaryKeyID(), keyID)
	}
	var storedAWS models.AWSOptions
	require.NoError(t, json.Unmarshal([]byte(aws), &storedAWS))
	assert.True(t, encryption.IsEncrypted(storedAWS.AWSAccessKey))
	assert.True(t, encryption.IsEncrypted(storedAWS.AWSSecretKey))
	assert.True(t, storedAWS.RDSBasicMetricsDisabled)
	var storedMySQL models.MySQLOptions
	require.NoError(t, json.Unmarshal([]byte(mysql), &storedMySQL))
	assert.True(t, encryption.IsEncrypted(storedMySQL.TLSCert))
	assert.True(t, encryption.IsEncrypted(storedMySQL.TLSKey))

	// backup location S3 credentials are encrypted at rest and readable via the model
	var rawS3 string
	err = sqlDB.QueryRowContext(t.Context(), `SELECT s3_config FROM backup_locations WHERE id = 'L1'`).Scan(&rawS3)
	require.NoError(t, err)
	var storedS3 models.S3LocationConfig
	require.NoError(t, json.Unmarshal([]byte(rawS3), &storedS3))
	assert.True(t, encryption.IsEncrypted(storedS3.AccessKey))
	assert.True(t, encryption.IsEncrypted(storedS3.SecretKey))
	assert.Equal(t, "https://s3.example.com", storedS3.Endpoint)
	location, err := models.FindBackupLocationByID(q, "L1")
	require.NoError(t, err)
	assert.Equal(t, "s3-access-key", location.S3Config.AccessKey)
	assert.Equal(t, "s3-secret-key", location.S3Config.SecretKey)

	// the model view decrypts to the original secrets
	agent, err := models.FindAgentByID(q, "A1")
	require.NoError(t, err)
	assert.Equal(t, "legacy-user", agent.Username.Reveal())
	assert.Equal(t, "plain-password", agent.Password.Reveal())
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", agent.AWSOptions.AWSAccessKey)
	assert.Equal(t, "aws-secret-key", agent.AWSOptions.AWSSecretKey)
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----plain-----END PRIVATE KEY-----", agent.MySQLOptions.TLSKey)

	// idempotency: a second run finds nothing and rewrites nothing
	ids, err = models.AgentsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	assert.Empty(t, ids)
	locationIDs, err = models.LocationsNeedingReencryption(q, cipher)
	require.NoError(t, err)
	assert.Empty(t, locationIDs)
	require.NoError(t, models.MigrateEncryption(q))
	username2, password2, aws2, mysql2 := readRaw()
	assert.Equal(t, username, username2)
	assert.Equal(t, password, password2)
	assert.Equal(t, aws, aws2)
	assert.Equal(t, mysql, mysql2)
}
