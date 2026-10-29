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
	"context"
	"database/sql/driver"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

func TestShouldForceConfigurationUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		portsChanged   bool
		enteredRunning bool
		status         inventoryv1.AgentStatus
		expected       bool
	}{
		// A normal restart: the new port is already stored by the time RUNNING arrives, so
		// this is the only thing that distinguishes it. Missing it leaves VictoriaMetrics
		// scraping the old address until the batched update, which is what PMM-14267 fixed.
		{"Agent entering RUNNING", false, true, inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, true},

		// A reconnect: already running, port moved underneath it.
		{"running Agent whose port changed", true, false, inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, true},

		{"running Agent that changed nothing", false, false, inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, false},

		// An Agent that cannot start reports a new port on every attempt and never reaches
		// RUNNING, which is what made rebuilding here unbounded.
		{"Agent that is still starting", true, false, inventoryv1.AgentStatus_AGENT_STATUS_STARTING, false},
		{"Agent that failed to start", true, false, inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, false},
		{"Agent waiting to be restarted", true, false, inventoryv1.AgentStatus_AGENT_STATUS_WAITING, false},
		{"Agent that is stopping", true, false, inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, false},
		{"Agent that has stopped", true, false, inventoryv1.AgentStatus_AGENT_STATUS_DONE, false},
		{"Agent of unknown status", true, true, inventoryv1.AgentStatus_AGENT_STATUS_UNSPECIFIED, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, shouldForceConfigurationUpdate(tc.portsChanged, tc.enteredRunning, tc.status))
		})
	}
}

func TestCheckEnteredRunning(t *testing.T) {
	t.Parallel()

	agentColumns := []string{
		"agent_id", "agent_type", "runs_on_node_id", "service_id", "node_id",
		"pmm_agent_id", "custom_labels", "environment_variables", "created_at", "updated_at",
		"disabled", "status", "listen_port", "version", "process_exec_path", "is_connected",
		"username", "password", "agent_password", "tls", "tls_skip_verify",
		"log_level", "exporter_options", "qan_options", "rta_options",
		"aws_options", "azure_options", "mongo_options", "mysql_options", "postgresql_options", "valkey_options",
	}

	agentRowWith := func(agentID, status string, disabled bool, listenPort int, exporterOptions string) *sqlmock.Rows {
		return sqlmock.NewRows(agentColumns).AddRow(
			agentID, string(models.PMMAgentType), "test-node", nil, nil, nil, nil, nil,
			time.Now(), time.Now(), disabled, status, listenPort, nil, nil, false,
			nil, nil, nil, false, false, nil,
			exporterOptions, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`,
		)
	}
	agentRow := func(agentID, status string) *sqlmock.Rows {
		return agentRowWith(agentID, status, false, 8080, `{}`)
	}

	newDB := func(t *testing.T) (*reform.DB, sqlmock.Sqlmock) {
		t.Helper()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), mock
	}

	t.Run("true when a stored STARTING becomes RUNNING", func(t *testing.T) {
		t.Parallel()

		db, mock := newDB(t)
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-1").
			WillReturnRows(agentRow("test-agent-1", inventoryv1.AgentStatus_AGENT_STATUS_STARTING.String()))

		assert.True(t, checkEnteredRunning(db.Querier, "test-agent-1", inventoryv1.AgentStatus_AGENT_STATUS_RUNNING))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("false when it was already RUNNING", func(t *testing.T) {
		t.Parallel()

		db, mock := newDB(t)
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-2").
			WillReturnRows(agentRow("test-agent-2", inventoryv1.AgentStatus_AGENT_STATUS_RUNNING.String()))

		assert.False(t, checkEnteredRunning(db.Querier, "test-agent-2", inventoryv1.AgentStatus_AGENT_STATUS_RUNNING))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("false for a report that is not RUNNING, without querying", func(t *testing.T) {
		t.Parallel()

		db, mock := newDB(t)

		assert.False(t, checkEnteredRunning(db.Querier, "test-agent-3", inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("false when the agent is not found", func(t *testing.T) {
		t.Parallel()

		db, mock := newDB(t)
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("missing-agent").
			WillReturnError(reform.ErrNoRows)

		assert.False(t, checkEnteredRunning(db.Querier, "missing-agent", inventoryv1.AgentStatus_AGENT_STATUS_RUNNING))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// PMM Server never scrapes these, so a synchronous rebuild for them is pure cost.
	for _, tc := range []struct {
		name            string
		disabled        bool
		listenPort      int
		exporterOptions string
	}{
		{"false for a push-mode Agent, scraped by its own vmagent", false, 8080, `{"push_metrics": true}`},
		{"false for a built-in Agent, which has no port", false, 0, `{}`},
		{"false for a disabled Agent", true, 8080, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newDB(t)
			mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
				WithArgs("test-agent-4").
				WillReturnRows(agentRowWith("test-agent-4", inventoryv1.AgentStatus_AGENT_STATUS_STARTING.String(),
					tc.disabled, tc.listenPort, tc.exporterOptions))

			assert.False(t, checkEnteredRunning(db.Querier, "test-agent-4", inventoryv1.AgentStatus_AGENT_STATUS_RUNNING))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

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

	t.Run("returns false for a push-mode agent, which PMM Server does not scrape", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
			WithArgs("test-agent-5").
			WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(
				"test-agent-5",
				string(models.NodeExporterType),
				nil,
				nil, nil, nil, nil, nil,
				time.Now(), time.Now(),
				false, "", 42000, nil, nil, false,
				nil, nil, nil, false, false, nil,
				`{"push_metrics": true}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`,
			))

		changed := checkPortChanged(db.Querier, "test-agent-5", uint32(42001))
		assert.False(t, changed, "should ignore a port PMM Server's configuration does not contain")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// fakeVMDB counts how stateChanged asks for a VictoriaMetrics configuration update.
type fakeVMDB struct {
	forced, requested int
}

func (f *fakeVMDB) RequestConfigurationUpdate() { f.requested++ }

func (f *fakeVMDB) ForceConfigurationUpdate(context.Context) error {
	f.forced++
	return nil
}

func (f *fakeVMDB) BuildScrapeConfigForVMAgent(context.Context, string) ([]byte, error) {
	return nil, nil
}

// TestStateChanged drives stateChanged through a sequence of reports, so the checks are tested
// where they run: they must read the Agent as stored before updateAgentStatus replaces it, or a
// restart on a new port is never forced (PMM-14267).
func TestStateChanged(t *testing.T) {
	t.Parallel()

	agentColumns := []string{
		"agent_id", "agent_type", "runs_on_node_id", "service_id", "node_id",
		"pmm_agent_id", "custom_labels", "environment_variables", "created_at", "updated_at",
		"disabled", "status", "listen_port", "version", "process_exec_path", "is_connected",
		"username", "password", "agent_password", "tls", "tls_skip_verify",
		"log_level", "exporter_options", "qan_options", "rta_options",
		"aws_options", "azure_options", "mongo_options", "mysql_options", "postgresql_options", "valkey_options",
	}
	const agentID = "test-exporter"
	const selectAgent = `SELECT .+ FROM "agents" WHERE .+ LIMIT 1`

	type stored struct {
		status          inventoryv1.AgentStatus
		listenPort      int
		exporterOptions string
		agentType       models.AgentType // node_exporter when empty
	}
	type report struct {
		status     inventoryv1.AgentStatus
		listenPort uint32
		stored     stored // the row as stateChanged finds it for this report
	}

	// The exporter has no pmm_agent_id, so stateChanged returns before asking for a state update.
	agentRow := func(s stored) *sqlmock.Rows {
		agentType := s.agentType
		if agentType == "" {
			agentType = models.NodeExporterType
		}
		return sqlmock.NewRows(agentColumns).AddRow(
			agentID, string(agentType), nil, nil, "test-node", nil, nil, nil,
			time.Now(), time.Now(), false, s.status.String(), s.listenPort, nil, nil, false,
			nil, nil, nil, false, false, nil,
			s.exporterOptions, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`, `{}`,
		)
	}

	// updateArgs expects updateAgentStatus to store the reported port; the other columns are not
	// what this test is about.
	updateArgs := func(listenPort uint32) []driver.Value {
		args := make([]driver.Value, 0, len(agentColumns))
		for _, column := range agentColumns[1:] { // agent_id is the WHERE argument, last
			if column == "listen_port" {
				args = append(args, listenPort)
				continue
			}
			args = append(args, sqlmock.AnyArg())
		}

		return append(args, agentID)
	}

	starting, running := inventoryv1.AgentStatus_AGENT_STATUS_STARTING, inventoryv1.AgentStatus_AGENT_STATUS_RUNNING

	for _, tc := range []struct {
		name            string
		reports         []report
		forced, request int
	}{
		{
			name: "a restart on a new port is forced once, when it becomes RUNNING",
			reports: []report{
				{starting, 42001, stored{running, 42000, `{}`, ""}},
				{running, 42001, stored{starting, 42001, `{}`, ""}},
			},
			forced: 1, request: 1,
		},
		{
			name: "a running Agent whose port moved is forced (a reconnect)",
			reports: []report{
				{running, 42001, stored{running, 42000, `{}`, ""}},
			},
			forced: 1, request: 0,
		},
		{
			name: "an Agent that cannot start is never forced",
			reports: []report{
				{starting, 42002, stored{starting, 42001, `{}`, ""}},
				{inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, 42002, stored{starting, 42002, `{}`, ""}},
				{inventoryv1.AgentStatus_AGENT_STATUS_DONE, 42002, stored{inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, 42002, `{}`, ""}},
			},
			forced: 0, request: 3,
		},
		{
			name: "a push-mode Agent becoming RUNNING is not forced",
			reports: []report{
				{running, 42001, stored{starting, 42001, `{"push_metrics": true}`, ""}},
			},
			forced: 0, request: 1,
		},
		{
			name: "a built-in Agent becoming RUNNING is not forced",
			reports: []report{
				{running, 0, stored{starting, 0, `{}`, models.QANMySQLPerfSchemaAgentType}},
			},
			forced: 0, request: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = mock.ExpectClose()
				assert.NoError(t, sqlDB.Close())
			})
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			vmdb := &fakeVMDB{}
			h := &Handler{db: db, r: &Registry{roster: newRoster(db)}, vmdb: vmdb}
			ctx := logger.SetEntry(t.Context(), logrus.WithField("test", t.Name()))

			for _, r := range tc.reports {
				mock.ExpectBegin()
				mock.ExpectQuery(selectAgent).WithArgs(agentID).WillReturnRows(agentRow(r.stored)) // checkPortChanged
				if r.status == running {
					mock.ExpectQuery(selectAgent).WithArgs(agentID).WillReturnRows(agentRow(r.stored)) // checkEnteredRunning
				}
				mock.ExpectQuery(selectAgent).WithArgs(agentID).WillReturnRows(agentRow(r.stored)) // updateAgentStatus
				mock.ExpectExec(`UPDATE "agents" SET`).WithArgs(updateArgs(r.listenPort)...).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				// The Agent again, for its pmm_agent_id: roster.get returns the Agent's own ID.
				mock.ExpectQuery(selectAgent).WithArgs(agentID).WillReturnRows(agentRow(r.stored))

				err := h.stateChanged(ctx, &agentv1.StateChangedRequest{
					AgentId:    agentID,
					Status:     r.status,
					ListenPort: r.listenPort,
				})
				require.NoError(t, err)
			}

			assert.Equal(t, tc.forced, vmdb.forced, "forced updates")
			assert.Equal(t, tc.request, vmdb.requested, "batched updates")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
