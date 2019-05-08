// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/tests"
)

// see https://www.postgresql.org/docs/10/errcodes-appendix.html for error codes

func assertUniqueViolation(t *testing.T, err error, constraint string) {
	t.Helper()

	require.IsType(t, &pq.Error{}, err)
	pgErr := err.(*pq.Error)
	assert.EqualValues(t, pq.ErrorCode("23505"), pgErr.Code)
	assert.Equal(t, fmt.Sprintf(`duplicate key value violates unique constraint %q`, constraint), pgErr.Message)
}

func assertCheckViolation(t *testing.T, err error, table, constraint string) {
	t.Helper()

	require.IsType(t, &pq.Error{}, err)
	pgErr := err.(*pq.Error)
	assert.EqualValues(t, pq.ErrorCode("23514"), pgErr.Code)
	assert.Equal(t, fmt.Sprintf(`new row for relation %q violates check constraint %q`, table, constraint), pgErr.Message)
}

func TestDatabaseUniqueIndexes(t *testing.T) {
	t.Run("Nodes", func(t *testing.T) {
		db := tests.OpenTestDB(t)
		defer func() {
			require.NoError(t, db.Close())
		}()
		var err error
		now := models.Now()

		// node_id
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('1', 'generic', 'name', '', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('1', 'generic', 'other name', '', '', '', '', $1, $2)", now, now,
		)
		assertUniqueViolation(t, err, "nodes_pkey")

		// node_name
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('2', 'generic', 'name', '', '', '', '', $1, $2)", now, now,
		)
		assertUniqueViolation(t, err, "nodes_node_name_key")

		// machine_id
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('31', 'generic', 'name31', 'machine-id', '', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('32', 'generic', 'name32', 'machine-id', '', '', '', '', $1, $2)", now, now,
		)
		assertUniqueViolation(t, err, "nodes_machine_id_generic_key")

		// machine_id for container
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('31-container', 'container', 'name31-container', 'machine-id', '', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('32-container', 'container', 'name32-container', 'machine-id', '', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)

		// container_id
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, container_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('41', 'generic', 'name41', 'docker-container-id', '', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, container_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('42', 'generic', 'name42', 'docker-container-id', '', '', '', '', $1, $2)", now, now,
		)
		assertUniqueViolation(t, err, "nodes_container_id_key")

		// (address, region)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, region, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('51', 'generic', 'name51', 'instance1', 'region1', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, region, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('52', 'generic', 'name52', 'instance1', 'region1', '', '', '', $1, $2)", now, now,
		)
		assertUniqueViolation(t, err, "nodes_address_region_key")
		// same address, NULL region is fine
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('53', 'generic', 'name53', 'instance1', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('54', 'generic', 'name54', 'instance1', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
	})

	t.Run("Services", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("Agents", func(t *testing.T) {
		db := tests.OpenTestDB(t)
		defer func() {
			require.NoError(t, db.Close())
		}()
		var err error
		now := models.Now()

		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('/node_id/1', 'generic', 'name', '', '', '', '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/1', 'pmm-agent', '/node_id/1', false, '', $1, $2)", now, now,
		)
		require.NoError(t, err)

		// runs_on_node_id XOR pmm_agent_id
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/2', 'pmm-agent', '/node_id/1', NULL, false, '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/3', 'mysqld_exporter', NULL, '/agent_id/1', false, '', $1, $2)", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/4', 'mysqld_exporter', NULL, NULL, false, '', $1, $2)", now, now,
		)
		assertCheckViolation(t, err, "agents", "runs_on_node_id_xor_pmm_agent_id")
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/5', 'pmm-agent', '/node_id/1', '/agent_id/1', false, '', $1, $2)", now, now,
		)
		assertCheckViolation(t, err, "agents", "runs_on_node_id_xor_pmm_agent_id")

		// runs_on_node_id only for pmm-agent
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/6', 'mysqld_exporter', '/node_id/1', NULL, false, '', $1, $2)", now, now,
		)
		assertCheckViolation(t, err, "agents", "runs_on_node_id_only_for_pmm_agent")
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at) "+
				"VALUES ('/agent_id/7', 'pmm-agent', NULL, '/agent_id/1', false, '', $1, $2)", now, now,
		)
		assertCheckViolation(t, err, "agents", "runs_on_node_id_only_for_pmm_agent")
	})
}
