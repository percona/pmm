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
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
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

func getTX(t *testing.T, db *sql.DB) (*sql.Tx, func()) {
	t.Helper()

	tx, err := db.Begin()
	require.NoError(t, err)
	rollback := func() {
		require.NoError(t, tx.Rollback())
	}
	return tx, rollback
}

//nolint:lll
func TestDatabaseChecks(t *testing.T) {
	t.Run("Nodes", func(t *testing.T) {
		db := testdb.Open(t, models.SkipFixtures, nil)
		defer func() {
			require.NoError(t, db.Close())
		}()
		var err error
		now := models.Now()

		// node_id
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('1', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('1', 'generic', 'other name', '', '', '', '', $1, $2)", now, now)
		assertUniqueViolation(t, err, "nodes_pkey")

		// node_name
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('2', 'generic', 'name', '', '', '', '', $1, $2)", now, now)
		assertUniqueViolation(t, err, "nodes_node_name_key")

		// machine_id for generic Node: https://jira.percona.com/browse/PMM-4196
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('31', 'generic', 'name31', 'machine-id', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('32', 'generic', 'name32', 'machine-id', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)

		// machine_id for container Node
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('31-container', 'container', 'name31-container', 'machine-id', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, machine_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('32-container', 'container', 'name32-container', 'machine-id', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)

		// container_id
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, container_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('41', 'generic', 'name41', 'docker-container-id', '', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, container_id, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('42', 'generic', 'name42', 'docker-container-id', '', '', '', '', $1, $2)", now, now)
		assertUniqueViolation(t, err, "nodes_container_id_key")

		// (address, region)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, region, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('51', 'generic', 'name51', 'instance1', 'region1', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, region, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('52', 'generic', 'name52', 'instance1', 'region1', '', '', '', $1, $2)", now, now)
		assertUniqueViolation(t, err, "nodes_address_region_key")
		// same address, NULL region is fine
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('53', 'generic', 'name53', 'instance1', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, address, distro, node_model, az, created_at, updated_at) "+
				"VALUES ('54', 'generic', 'name54', 'instance1', '', '', '', $1, $2)", now, now)
		require.NoError(t, err)
	})

	t.Run("Services", func(t *testing.T) {
		db := testdb.Open(t, models.SkipFixtures, nil)
		defer func() {
			require.NoError(t, db.Close())
		}()
		var err error
		now := models.Now()
		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('/node_id/1', 'generic', 'name', '', '', '', '', $1, $2)",
			now, now)
		require.NoError(t, err)

		// Try to insert both address and socket
		_, err = db.Exec(
			"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, address, port, socket, external_group, created_at, updated_at) "+
				"VALUES ('/service_id/1', 'mysql', 'name', '/node_id/1', '', '', '', '10.10.10.10', 3306, '/var/run/mysqld/mysqld.sock', '', $1, $2)",
			now, now)
		require.Error(t, err, `pq: new row for relation "services" violates check constraint "address_socket_check"`)

		// Try to insert both address and socket empty
		_, err = db.Exec(
			"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, address, port, socket, external_group, created_at, updated_at) "+
				"VALUES ('/service_id/1', 'mysql', 'name', '/node_id/1', '', '', '', NULL, NULL, NULL, '', $1, $2)",
			now, now)
		require.NoError(t, err)

		// Try to insert invalid port
		_, err = db.Exec(
			"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, address, port, socket, external_group, created_at, updated_at) "+
				"VALUES ('/service_id/1', 'mysql', 'name', '/node_id/1', '', '', '', '10.10.10.10', 999999, NULL, '', $1, $2)",
			now, now)
		require.Error(t, err, `pq: new row for relation "services" violates check constraint "port_check"`)

		// Try to insert empty group for external exporter
		_, err = db.Exec(
			"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, address, port, socket, external_group, created_at, updated_at) "+
				"VALUES ('/service_id/1', 'external', 'name', '/node_id/1', '', '', '', '10.10.10.10', 3333, NULL, '', $1, $2)",
			now, now)
		require.Error(t, err, `pq: new row for relation "services" violates check constraint "services_external_group_check"`)

		// Try to insert non empty group for mysql exporter
		_, err = db.Exec(
			"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, address, port, socket, external_group, created_at, updated_at) "+
				"VALUES ('/service_id/1', 'mysql', 'name', '/node_id/1', '', '', '', '10.10.10.10', 3306, NULL, 'non empty group', $1, $2)",
			now, now)
		require.Error(t, err, `pq: new row for relation "services" violates check constraint "services_external_group_check"`)
	})

	t.Run("Agents", func(t *testing.T) {
		db := testdb.Open(t, models.SkipFixtures, nil)
		defer func() {
			require.NoError(t, db.Close())
		}()
		var err error
		now := models.Now()

		_, err = db.Exec(
			"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
				"VALUES ('/node_id/1', 'generic', 'name', '', '', '', '', $1, $2)",
			now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO services (service_id, service_type, service_name, node_id, environment, cluster, replication_set, socket, external_group, created_at, updated_at) "+
				"VALUES ('/service_id/1', 'mysql', 'name', '/node_id/1', '', '', '', '/var/run/mysqld/mysqld.sock', '', $1, $2)",
			now, now)
		require.NoError(t, err)
		_, err = db.Exec(
			"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
				"VALUES ('/agent_id/1', 'pmm-agent', '/node_id/1', NULL, false, '', $1, $2, false, false, false, 0, 0, true, true, false)",
			now, now)
		require.NoError(t, err)

		t.Run("runs_on_node_id_xor_pmm_agent_id", func(t *testing.T) {
			t.Run("Normal", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/2', 'pmm-agent', '/node_id/1', NULL, false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				require.NoError(t, err)
				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/3', 'mysqld_exporter', NULL, '/agent_id/1', '/node_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				require.NoError(t, err)
			})

			t.Run("BothNULL", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/4', 'mysqld_exporter', NULL, NULL, '/node_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				assertCheckViolation(t, err, "agents", "runs_on_node_id_xor_pmm_agent_id")
			})

			t.Run("BothSet", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/5', 'pmm-agent', '/node_id/1', '/agent_id/1', '/node_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				assertCheckViolation(t, err, "agents", "runs_on_node_id_xor_pmm_agent_id")
			})
		})
		t.Run("runs_on_node_id_only_for_pmm_agent", func(t *testing.T) {
			t.Run("NotPMMAgent", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/6', 'mysqld_exporter', '/node_id/1', NULL, '/node_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				assertCheckViolation(t, err, "agents", "runs_on_node_id_only_for_pmm_agent")
			})

			t.Run("PMMAgent", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/7', 'pmm-agent', NULL, '/agent_id/1', '/node_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				assertCheckViolation(t, err, "agents", "runs_on_node_id_only_for_pmm_agent")
			})
		})
		t.Run("node_id_or_service_id_or_pmm_agent_id", func(t *testing.T) {
			// pmm_agent_id is always set in that test - NULL is tested above

			t.Run("node_id set", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, service_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/8', 'node_exporter', NULL, '/agent_id/1', '/node_id/1', NULL, false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)

				assert.NoError(t, err)
			})

			t.Run("service_id set", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, service_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/8', 'mysqld_exporter', NULL, '/agent_id/1', NULL, '/service_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)

				assert.NoError(t, err)
			})

			t.Run("Both NULL", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, service_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/8', 'mysqld_exporter', NULL, '/agent_id/1', NULL, NULL, false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)

				assertCheckViolation(t, err, "agents", "node_id_or_service_id_for_non_pmm_agent")
			})

			t.Run("Both set", func(t *testing.T) {
				tx, rollback := getTX(t, db)
				defer rollback()

				_, err = tx.Exec(
					"INSERT INTO agents (agent_id, agent_type, runs_on_node_id, pmm_agent_id, node_id, service_id, disabled, status, created_at, updated_at, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics) "+
						"VALUES ('/agent_id/8', 'mysqld_exporter', NULL, '/agent_id/1', '/node_id/1', '/service_id/1', false, '', $1, $2, false, false, false, 0, 0, false, false, false)",
					now, now)
				assertCheckViolation(t, err, "agents", "node_id_or_service_id_for_non_pmm_agent")
			})
		})
	})
}

func TestDatabaseMigrations(t *testing.T) {
	t.Run("Update metrics resolutions", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, pointer.ToInt(9))
		defer sqlDB.Close() //nolint:errcheck
		settings, err := models.GetSettings(sqlDB)
		require.NoError(t, err)
		metricsResolutions := models.MetricsResolutions{
			HR: 5 * time.Second,
			MR: 5 * time.Second,
			LR: 60 * time.Second,
		}
		settings.MetricsResolutions = metricsResolutions
		err = models.SaveSettings(sqlDB, settings)
		require.NoError(t, err)

		settings, err = models.GetSettings(sqlDB)
		require.NoError(t, err)
		require.Equal(t, metricsResolutions, settings.MetricsResolutions)

		testdb.SetupDB(t, sqlDB, models.SkipFixtures, pointer.ToInt(10))
		settings, err = models.GetSettings(sqlDB)
		require.NoError(t, err)
		require.Equal(t, models.MetricsResolutions{
			HR: 5 * time.Second,
			MR: 10 * time.Second,
			LR: 60 * time.Second,
		}, settings.MetricsResolutions)
	})
	t.Run("Shouldn' update metrics resolutions if it's already changed", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, pointer.ToInt(9))
		defer sqlDB.Close() //nolint:errcheck
		settings, err := models.GetSettings(sqlDB)
		require.NoError(t, err)
		metricsResolutions := models.MetricsResolutions{
			HR: 1 * time.Second,
			MR: 5 * time.Second,
			LR: 60 * time.Second,
		}
		settings.MetricsResolutions = metricsResolutions
		err = models.SaveSettings(sqlDB, settings)
		require.NoError(t, err)

		settings, err = models.GetSettings(sqlDB)
		require.NoError(t, err)
		require.Equal(t, metricsResolutions, settings.MetricsResolutions)

		testdb.SetupDB(t, sqlDB, models.SkipFixtures, pointer.ToInt(10))
		settings, err = models.GetSettings(sqlDB)
		require.NoError(t, err)
		require.Equal(t, models.MetricsResolutions{
			HR: 1 * time.Second,
			MR: 5 * time.Second,
			LR: 60 * time.Second,
		}, settings.MetricsResolutions)
	})
	t.Run("stats_collections field migration: string to string array", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, pointer.ToInt(57))
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		defer sqlDB.Close() //nolint:errcheck

		// Insert dummy node in DB
		_, err := sqlDB.ExecContext(context.Background(),
			`INSERT INTO
			nodes(node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at)
			VALUES
			('node_id', 'node_type', 'node_name', 'distro', 'node_model', 'az', 'address', '03/03/2014 02:03:04', '03/03/2014 02:03:04')`,
		)
		require.NoError(t, err)

		// Insert dummy agent in DB
		_, err = sqlDB.ExecContext(context.Background(),
			`INSERT INTO
			agents(mongo_db_tls_options, agent_id, agent_type, runs_on_node_id, created_at, updated_at, disabled, status, tls, tls_skip_verify, query_examples_disabled, max_query_log_size, table_count_tablestats_group_limit, rds_basic_metrics_disabled, rds_enhanced_metrics_disabled, push_metrics)
			VALUES
			('{"stats_collections": "db.col1,db.col2,db.col3"}', 'id', 'pmm-agent', 'node_id' , '03/03/2014 02:03:04', '03/03/2014 02:03:04', false, 'alive', false, false, false, 0, 0, false, false, false)`,
		)
		require.NoError(t, err)

		// Apply migration
		testdb.SetupDB(t, sqlDB, models.SkipFixtures, pointer.ToInt(63))

		agent, err := models.FindAgentByID(db.Querier, "id")
		require.NoError(t, err)
		require.Equal(t, "id", agent.AgentID)
		require.Equal(t, []string{"db.col1", "db.col2", "db.col3"}, agent.MongoDBOptions.StatsCollections)
	})
	t.Run("stats_collections field migration: string array to string array", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, pointer.ToInt(63))
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		defer sqlDB.Close() //nolint:errcheck

		// Insert dummy node in DB
		_, err := sqlDB.ExecContext(context.Background(),
			`INSERT INTO
			nodes(node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at)
			VALUES
			('node_id', 'generic', 'node_name', 'distro', 'node_model', 'az', 'address', '03/03/2014 02:03:04', '03/03/2014 02:03:04')`,
		)
		require.NoError(t, err)

		// Insert dummy agent in DB
		pmmAgent, err := models.CreatePMMAgent(db.Querier, "node_id", make(map[string]string))
		require.NoError(t, err)

		createdAgent, err := models.CreateAgent(db.Querier, models.NodeExporterType,
			&models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				NodeID:     "node_id",
				MongoDBOptions: &models.MongoDBOptions{StatsCollections: []string{
					"db.col1", "db.col2", "db.col3",
				}},
			})
		require.NoError(t, err)

		// Apply migration
		testdb.SetupDB(t, sqlDB, models.SkipFixtures, pointer.ToInt(58))

		agent, err := models.FindAgentByID(db.Querier, createdAgent.AgentID)
		require.NoError(t, err)
		require.Equal(t, []string{"db.col1", "db.col2", "db.col3"}, agent.MongoDBOptions.StatsCollections)
	})
}
