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

package agents

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
)

func TestCheckPortChanged(t *testing.T) {
	t.Parallel()

	// Define the columns that reform expects when querying the agents table
	agentColumns := []string{
		"agent_id", "agent_type", "runs_on_node_id", "service_id", "node_id",
		"pmm_agent_id", "custom_labels", "environment_variables", "created_at", "updated_at",
		"disabled", "status", "listen_port", "version", "process_exec_path", "is_connected",
		"username", "password", "agent_password", "tls", "tls_skip_verify",
		"log_level", "exporter_options", "qan_options", "rta_options",
		"aws_options", "azure_options", "mongo_options", "mysql_options", "postgresql_options", "valkey_options",
	}

	t.Run("returns false when agent not found", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Mock the SELECT query that will be executed by FindAgentByID
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("non-existent-agent-id").
			WillReturnError(reform.ErrNoRows)

		changed := checkPortChanged(db.Querier, "non-existent-agent-id", 8080)
		assert.False(t, changed, "should return false when agent is not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when old port is zero", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Mock the SELECT query returning an agent with nil ListenPort
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-1").
			WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(
				"test-agent-1",              // agent_id
				string(models.PMMAgentType), // agent_type
				"test-node-1",               // runs_on_node_id
				nil,                         // service_id
				nil,                         // node_id
				nil,                         // pmm_agent_id
				nil,                         // custom_labels
				nil,                         // environment_variables
				time.Now(),                  // created_at
				time.Now(),                  // updated_at
				false,                       // disabled
				"",                          // status
				nil,                         // listen_port (NULL)
				nil,                         // version
				nil,                         // process_exec_path
				false,                       // is_connected
				nil,                         // username
				nil,                         // password
				nil,                         // agent_password
				false,                       // tls
				false,                       // tls_skip_verify
				nil,                         // log_level
				`{}`,                        // exporter_options
				`{}`,                        // qan_options
				`{}`,                        // rta_options
				`{}`,                        // aws_options
				`{}`,                        // azure_options
				`{}`,                        // mongo_options
				`{}`,                        // mysql_options
				`{}`,                        // postgresql_options
				`{}`,                        // valkey_options
			))

		changed := checkPortChanged(db.Querier, "test-agent-1", 8080)
		assert.False(t, changed, "should return false when old port is zero")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when old port equals new port", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Mock the SELECT query returning an agent with ListenPort = 8080
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-2").
			WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(
				"test-agent-2",              // agent_id
				string(models.PMMAgentType), // agent_type
				"test-node-2",               // runs_on_node_id
				nil,                         // service_id
				nil,                         // node_id
				nil,                         // pmm_agent_id
				nil,                         // custom_labels
				nil,                         // environment_variables
				time.Now(),                  // created_at
				time.Now(),                  // updated_at
				false,                       // disabled
				"",                          // status
				8080,                        // listen_port
				nil,                         // version
				nil,                         // process_exec_path
				false,                       // is_connected
				nil,                         // username
				nil,                         // password
				nil,                         // agent_password
				false,                       // tls
				false,                       // tls_skip_verify
				nil,                         // log_level
				`{}`,                        // exporter_options
				`{}`,                        // qan_options
				`{}`,                        // rta_options
				`{}`,                        // aws_options
				`{}`,                        // azure_options
				`{}`,                        // mongo_options
				`{}`,                        // mysql_options
				`{}`,                        // postgresql_options
				`{}`,                        // valkey_options
			))

		changed := checkPortChanged(db.Querier, "test-agent-2", 8080)
		assert.False(t, changed, "should return false when old port equals new port")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns true when old port differs from new port", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Mock the SELECT query returning an agent with ListenPort = 8080
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-3").
			WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(
				"test-agent-3",              // agent_id
				string(models.PMMAgentType), // agent_type
				"test-node-3",               // runs_on_node_id
				nil,                         // service_id
				nil,                         // node_id
				nil,                         // pmm_agent_id
				nil,                         // custom_labels
				nil,                         // environment_variables
				time.Now(),                  // created_at
				time.Now(),                  // updated_at
				false,                       // disabled
				"",                          // status
				8080,                        // listen_port
				nil,                         // version
				nil,                         // process_exec_path
				false,                       // is_connected
				nil,                         // username
				nil,                         // password
				nil,                         // agent_password
				false,                       // tls
				false,                       // tls_skip_verify
				nil,                         // log_level
				`{}`,                        // exporter_options
				`{}`,                        // qan_options
				`{}`,                        // rta_options
				`{}`,                        // aws_options
				`{}`,                        // azure_options
				`{}`,                        // mongo_options
				`{}`,                        // mysql_options
				`{}`,                        // postgresql_options
				`{}`,                        // valkey_options
			))

		changed := checkPortChanged(db.Querier, "test-agent-3", 9090)
		assert.True(t, changed, "should return true when old port differs from new port")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles port conversion from uint32 to uint16", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Mock the SELECT query returning an agent with ListenPort = 42000
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-4").
			WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(
				"test-agent-4",              // agent_id
				string(models.PMMAgentType), // agent_type
				"test-node-4",               // runs_on_node_id
				nil,                         // service_id
				nil,                         // node_id
				nil,                         // pmm_agent_id
				nil,                         // custom_labels
				nil,                         // environment_variables
				time.Now(),                  // created_at
				time.Now(),                  // updated_at
				false,                       // disabled
				"",                          // status
				42000,                       // listen_port
				nil,                         // version
				nil,                         // process_exec_path
				false,                       // is_connected
				nil,                         // username
				nil,                         // password
				nil,                         // agent_password
				false,                       // tls
				false,                       // tls_skip_verify
				nil,                         // log_level
				`{}`,                        // exporter_options
				`{}`,                        // qan_options
				`{}`,                        // rta_options
				`{}`,                        // aws_options
				`{}`,                        // azure_options
				`{}`,                        // mongo_options
				`{}`,                        // mysql_options
				`{}`,                        // postgresql_options
				`{}`,                        // valkey_options
			))

		// Test with the same port passed as uint32
		changed := checkPortChanged(db.Querier, "test-agent-4", uint32(42000))
		assert.False(t, changed, "should correctly compare ports after uint32 to uint16 conversion")

		// Expect the query again for the second test
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-4").
			WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(
				"test-agent-4",
				string(models.PMMAgentType),
				"test-node-4",
				nil, nil, nil, nil, nil,
				time.Now(), time.Now(),
				false, "", 42000, nil, nil, false,
				nil, nil, nil, false, false, nil,
				`{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`,
			))

		// Test with different port
		changed = checkPortChanged(db.Querier, "test-agent-4", uint32(42001))
		assert.True(t, changed, "should detect port change after uint32 to uint16 conversion")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
