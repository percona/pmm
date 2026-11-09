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

package encryption

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
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

const (
	encryptionKeyTestPath = "/srv/pmm-encryption-rotation-test.key"
	// Rotation backs the previous key up next to the current one.
	oldEncryptionKeyTestPath = "/srv/pmm-encryption-rotation-test_old.key"
	originalEncryptionKey    = `CMatkOIIEmQKWAowdHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuQWVzR2NtS2V5EiIaIKDxOKZxwiJl5Hj6oPZ/unTzmAvfwHWzZ1Wli0vac15YGAEQARjGrZDiCCAB`
	// pmm-managed-username encrypted with originalEncryptionKey
	originalUsernameHash = `AYxEFsZsg7lp9+eSy6+wPFHlaNNy0ZpTbYN0NuCLPnQOZUYf2S6H9B+XJdF4+DscxC/pJwI=`
	// pmm-managed-password encrypted with originalEncryptionKey
	originalPasswordHash = `AYxEFsZuL5xZb5IxGGh8NI6GrjDxCzFGxIcHe94UXcg+dnZphu7GQSgmZm633XvZ8CBU2wo=`
)

func TestEncryptionRotation(t *testing.T) {
	db := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	ctx := t.Context()
	createOriginalEncryptionKey(t)

	tlsCert, tlsKey := generateTLSKeyPair(t)
	tlsCa, _ := generateTLSKeyPair(t)

	err := insertTestData(ctx, db, tlsCa, tlsCert, tlsKey)
	require.NoError(t, err)

	statusCode, err := RotateEncryptionKey(db, "pmm-managed-dev")
	require.NoError(t, err)
	require.Equal(t, 0, statusCode)

	newEncryptionKey, err := os.ReadFile(encryptionKeyTestPath)
	require.NoError(t, err)
	require.NotEqual(t, newEncryptionKey, []byte(originalEncryptionKey))

	err = checkNewlyEncryptedData(ctx, db)
	require.NoError(t, err)

	err = checkMySQLOptions(ctx, db, tlsCa, tlsCert, tlsKey)
	require.NoError(t, err)

	// Repeated rotations must not accumulate encryption layers. Only the DB part is
	// exercised here - the exported wrapper additionally stops and starts pmm-managed
	// via supervisorctl, which the first rotation already covers.
	err = rotateEncryptionKey(reform.NewDB(db, postgresql.Dialect, nil), "pmm-managed-dev")
	require.NoError(t, err)

	err = checkMySQLOptions(ctx, db, tlsCa, tlsCert, tlsKey)
	require.NoError(t, err)
}

// generateTLSKeyPair returns a self-signed certificate and its private key, both PEM encoded.
func generateTLSKeyPair(t *testing.T) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "pmm-encryption-rotation-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return string(cert), string(privateKey)
}

// checkMySQLOptions verifies that agents.mysql_options survives key rotation: tls_ca
// stays plaintext PEM while tls_cert and tls_key decrypt back to the original material
// with a single Decrypt call. See https://perconadev.atlassian.net/browse/PMM-15188:
// re-encrypting these columns during the decrypt phase left pmm-agent unable to parse
// them as PEM.
func checkMySQLOptions(ctx context.Context, db *sql.DB, tlsCa, tlsCert, tlsKey string) error {
	var raw string
	err := db.QueryRowContext(ctx, `SELECT mysql_options FROM agents WHERE agent_id = $1`, "1").Scan(&raw)
	if err != nil {
		return err
	}

	o := models.MySQLOptions{}
	err = json.Unmarshal([]byte(raw), &o)
	if err != nil {
		return err
	}

	if o.TLSCa != tlsCa {
		return errors.New("mysql_options tls_ca changed during rotation")
	}

	cert, err := encryption.Decrypt(o.TLSCert)
	if err != nil {
		return err
	}
	if cert != tlsCert {
		return errors.New("mysql_options tls_cert not properly decrypted after rotation")
	}

	key, err := encryption.Decrypt(o.TLSKey)
	if err != nil {
		return err
	}
	if key != tlsKey {
		return errors.New("mysql_options tls_key not properly decrypted after rotation")
	}

	_, err = tls.X509KeyPair([]byte(cert), []byte(key))

	return err
}

func createOriginalEncryptionKey(t *testing.T) {
	t.Helper()

	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, encryptionKeyTestPath)
	require.NoError(t, os.WriteFile(encryptionKeyTestPath, []byte(originalEncryptionKey), 0o600))

	t.Cleanup(func() {
		for _, path := range []string{encryptionKeyTestPath, oldEncryptionKeyTestPath} {
			err := os.Remove(path)
			if err != nil && !os.IsNotExist(err) {
				t.Errorf("failed to remove %s: %v", path, err)
			}
		}
	})
}

//nolint:dupword
func insertTestData(ctx context.Context, db *sql.DB, tlsCa, tlsCert, tlsKey string) error {
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
		EncryptedItems: []string{"pmm-managed-dev.agents.username", "pmm-managed-dev.agents.password", "pmm-managed-dev.agents.agent_password", "pmm-managed-dev.agents.aws_options", "pmm-managed-dev.agents.azure_options", "pmm-managed-dev.agents.mongo_options", "pmm-managed-dev.agents.mysql_options", "pmm-managed-dev.agents.postgresql_options"},
	})
	if err != nil {
		return err
	}

	// Store tls_cert/tls_key encrypted with the original key (as pmm-managed does at
	// rest), while tls_ca stays plaintext PEM. A dedicated instance is used instead of
	// the package level helpers so that the test does not depend on when the lazily
	// initialized default encryption is created.
	e := encryption.New()
	encryptedCert, err := e.Encrypt(tlsCert)
	if err != nil {
		return err
	}
	encryptedKey, err := e.Encrypt(tlsKey)
	if err != nil {
		return err
	}
	mysqlOptions, err := json.Marshal(models.MySQLOptions{
		TLSCa:   tlsCa,
		TLSCert: encryptedCert,
		TLSKey:  encryptedKey,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = db.ExecContext(
		ctx,
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('1', 'generic', 'name', '', '', '', '', $1, $2)",
		now, now,
	)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(
		ctx,
		"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, socket, external_group, created_at, updated_at) "+
			"VALUES ('1', 'mysql', 'name', '1', '', '', '', '/var/run/mysqld/mysqld.sock', '', $1, $2)",
		now, now,
	)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO agents (agent_id, agent_type, username, password, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, qan_options, mysql_options, aws_options, exporter_options) `+
			`VALUES ('1', 'pmm-agent', $1, $2, '1', NULL, false, '', $3, $4, false, false, '{"max_query_length": 0, "query_examples_disabled": false, "comments_parsing_disabled": true, "max_query_log_size": 0}', $5, '{"rds_basic_metrics_disabled": true, "rds_enhanced_metrics_disabled": true}', '{"push_metrics": false, "expose_exporter": false}')`,
		originalUsernameHash, originalPasswordHash, now, now, string(mysqlOptions),
	)
	if err != nil {
		return err
	}

	return nil
}

func checkNewlyEncryptedData(ctx context.Context, db *sql.DB) error {
	var newlyEncryptedUsername string
	var newlyEncryptedPassword string
	err := db.QueryRowContext(ctx, `SELECT username, password FROM agents WHERE agent_id = $1`, "1").
		Scan(&newlyEncryptedUsername, &newlyEncryptedPassword)
	if err != nil {
		return err
	}
	if newlyEncryptedUsername == originalUsernameHash {
		return errors.New("username hash not rotated properly")
	}
	if newlyEncryptedPassword == originalPasswordHash {
		return errors.New("password hash not rotated properly")
	}

	username, err := encryption.Decrypt(newlyEncryptedUsername)
	if err != nil {
		return err
	}
	if username != "pmm-managed-username" {
		return errors.New("username not properly decrypted")
	}

	password, err := encryption.Decrypt(newlyEncryptedPassword)
	if err != nil {
		return err
	}
	if password != "pmm-managed-password" {
		return errors.New("password not properly decrypted")
	}

	return nil
}
