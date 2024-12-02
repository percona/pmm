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
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	encryptionKeyTestPath = "/srv/pmm-encryption-rotation-test.key"
	originEncryptionKey   = `CMatkOIIEmQKWAowdHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuQWVzR2NtS2V5EiIaIKDxOKZxwiJl5Hj6oPZ/unTzmAvfwHWzZ1Wli0vac15YGAEQARjGrZDiCCAB`
	// pmm-managed-username encrypted with originEncryptionKey
	originUsernameHash = `AYxEFsZsg7lp9+eSy6+wPFHlaNNy0ZpTbYN0NuCLPnQOZUYf2S6H9B+XJdF4+DscxC/pJwI=`
	// pmm-managed-password encrypted with originEncryptionKey
	originPasswordHash = `AYxEFsZuL5xZb5IxGGh8NI6GrjDxCzFGxIcHe94UXcg+dnZphu7GQSgmZm633XvZ8CBU2wo=` //nolint:gosec
)

func TestEncryptionRotation(t *testing.T) {
	db := testdb.Open(t, models.SkipFixtures, nil)
	defer db.Close() //nolint:errcheck

	err := createOriginEncryptionKey()
	require.NoError(t, err)

	err = insertTestData(db)
	require.NoError(t, err)

	statusCode := RotateEncryptionKey(db, "pmm-managed-dev")
	require.Equal(t, 0, statusCode)

	newEncryptionKey, err := os.ReadFile(encryptionKeyTestPath)
	require.NoError(t, err)
	require.NotEqual(t, newEncryptionKey, []byte(originEncryptionKey))

	err = checkNewlyEncryptedData(db)
	require.NoError(t, err)

	err = os.Remove(encryptionKeyTestPath)
	require.NoError(t, err)
}

func createOriginEncryptionKey() error {
	encryption.DefaultEncryptionKeyPath = encryptionKeyTestPath
	err := os.WriteFile(encryptionKeyTestPath, []byte(originEncryptionKey), 0o600)
	if err != nil {
		return err
	}
	encryption.DefaultEncryption = encryption.New()
	return nil
}

//nolint:dupword
func insertTestData(db *sql.DB) error {
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
		EncryptedItems: []string{"pmm-managed-dev.agents.username", "pmm-managed-dev.agents.password", "pmm-managed-dev.agents.agent_password", "pmm-managed-dev.agents.aws_options", "pmm-managed-dev.agents.azure_options", "pmm-managed-dev.agents.mongo_options", "pmm-managed-dev.agents.mysql_options", "pmm-managed-dev.agents.postgresql_options"},
	})
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = db.Exec(
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ('1', 'generic', 'name', '', '', '', '', $1, $2)",
		now, now)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, socket, external_group, created_at, updated_at) "+
			"VALUES ('1', 'mysql', 'name', '1', '', '', '', '/var/run/mysqld/mysqld.sock', '', $1, $2)",
		now, now)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO agents (agent_id, agent_type, username, password, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, qan_options, mysql_options, aws_options, exporter_options) `+
			`VALUES ('1', 'pmm-agent', $1, $2, '1', NULL, false, '', $3, $4, false, false, '{"max_query_length": 0, "query_examples_disabled": false, "comments_parsing_disabled": true, "max_query_log_size": 0}', '{"table_count_tablestats_group_limit": 0}', '{"rds_basic_metrics_disabled": true, "rds_enhanced_metrics_disabled": true}', '{"push_metrics": false, "expose_exporter": false}')`,
		originUsernameHash, originPasswordHash, now, now)
	if err != nil {
		return err
	}

	return nil
}

func checkNewlyEncryptedData(db *sql.DB) error {
	var newlyEncryptedUsername string
	var newlyEncryptedPassword string
	err := db.QueryRow(`SELECT username, password FROM agents WHERE agent_id = $1`, "1").Scan(&newlyEncryptedUsername, &newlyEncryptedPassword)
	if err != nil {
		return err
	}
	if newlyEncryptedUsername == originUsernameHash {
		return errors.New("username hash not rotated properly")
	}
	if newlyEncryptedPassword == originPasswordHash {
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
