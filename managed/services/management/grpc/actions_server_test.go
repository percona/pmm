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

package grpc

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	actionsv1 "github.com/percona/pmm/api/actions/v1"
	"github.com/percona/pmm/managed/models"
)

const (
	testServiceID = "service-id"
	testNodeID    = "node-id"
)

var remoteNodeTypes = []models.NodeType{
	models.RemoteNodeType,
	models.RemoteRDSNodeType,
	models.RemoteAzureDatabaseNodeType,
}

func rowValues(t *testing.T, r reform.Record) []driver.Value {
	t.Helper()
	values := make([]driver.Value, 0, len(r.Values()))
	for _, v := range r.Values() {
		dv, err := driver.DefaultParameterConverter.ConvertValue(v)
		require.NoError(t, err)
		values = append(values, dv)
	}
	return values
}

// setupActionsServer returns an actions server whose database holds one Service on a Node of the given type.
func setupActionsServer(t *testing.T, serviceType models.ServiceType, nodeType models.NodeType) (*actionsServer, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	service := &models.Service{ServiceID: testServiceID, ServiceType: serviceType, ServiceName: "service", NodeID: testNodeID}
	node := &models.Node{NodeID: testNodeID, NodeType: nodeType, NodeName: "node"}
	mock.ExpectQuery(`SELECT .+ FROM "services" WHERE .+ LIMIT 1`).
		WithArgs(testServiceID).
		WillReturnRows(sqlmock.NewRows(models.ServiceTable.Columns()).AddRow(rowValues(t, service)...))
	mock.ExpectQuery(`SELECT .+ FROM "nodes" WHERE .+ LIMIT 1`).
		WithArgs(testNodeID).
		WillReturnRows(sqlmock.NewRows(models.NodeTable.Columns()).AddRow(rowValues(t, node)...))

	return &actionsServer{db: db, l: logrus.WithField("test", t.Name())}, mock
}

func expectNoPMMAgentsOnNode(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE runs_on_node_id = \$1 AND agent_type = \$2`).
		WithArgs(testNodeID, models.PMMAgentType).
		WillReturnRows(sqlmock.NewRows(models.AgentTable.Columns()))
}

// expectPostgresExporter mocks the lookup of the postgres_exporter of the test Service, run by the given pmm-agent.
func expectPostgresExporter(t *testing.T, mock sqlmock.Sqlmock, pmmAgentID *string) {
	t.Helper()

	service := &models.Service{ServiceID: testServiceID, ServiceType: models.PostgreSQLServiceType, ServiceName: "service", NodeID: testNodeID}
	exporter := &models.Agent{
		AgentID:    "exporter-id",
		AgentType:  models.PostgresExporterType,
		PMMAgentID: pmmAgentID,
		ServiceID:  new(testServiceID),
	}
	mock.ExpectQuery(`SELECT .+ FROM "services" WHERE .+ LIMIT 1`).
		WithArgs(testServiceID).
		WillReturnRows(sqlmock.NewRows(models.ServiceTable.Columns()).AddRow(rowValues(t, service)...))
	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE service_id = \$1 AND agent_type = \$2`).
		WithArgs(testServiceID, models.PostgresExporterType).
		WillReturnRows(sqlmock.NewRows(models.AgentTable.Columns()).AddRow(rowValues(t, exporter)...))
}

func assertFailedPrecondition(t *testing.T, err error, message string) {
	t.Helper()
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Equal(t, message, status.Convert(err).Message())
}

func TestStartPTMySQLSummaryAction(t *testing.T) {
	t.Parallel()

	req := &actionsv1.StartPTMySQLSummaryActionParams{ServiceId: testServiceID}

	for _, nodeType := range remoteNodeTypes {
		t.Run(string(nodeType), func(t *testing.T) {
			t.Parallel()

			s, mock := setupActionsServer(t, models.MySQLServiceType, nodeType)

			_, err := s.StartPTMySQLSummaryAction(t.Context(), req)
			assertFailedPrecondition(t, err, "PTMySQL Summary doesn't work with remote instances yet")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("no pmm-agent on node", func(t *testing.T) {
		t.Parallel()

		s, mock := setupActionsServer(t, models.MySQLServiceType, models.GenericNodeType)
		expectNoPMMAgentsOnNode(mock)

		_, err := s.StartPTMySQLSummaryAction(t.Context(), req)
		assertFailedPrecondition(t, err, "No pmm-agent running on node node-id")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStartPTPgSummaryAction(t *testing.T) {
	t.Parallel()

	req := &actionsv1.StartPTPgSummaryActionParams{ServiceId: testServiceID}

	// expectActionResult mocks the action result insert and fails it, so the test stops before the action is sent to a pmm-agent.
	expectActionResult := func(mock sqlmock.Sqlmock, pmmAgentID string) error {
		errStop := errors.New("stop after the pmm-agent is chosen")
		mock.ExpectQuery(`INSERT INTO "action_results"`).
			WithArgs(sqlmock.AnyArg(), pmmAgentID, false, "", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errStop)
		return errStop
	}

	for _, nodeType := range remoteNodeTypes {
		for _, pmmAgentID := range []string{models.PMMServerAgentID, "pmm-agent-id"} {
			t.Run(string(nodeType)+"/"+pmmAgentID, func(t *testing.T) {
				t.Parallel()

				s, mock := setupActionsServer(t, models.PostgreSQLServiceType, nodeType)
				expectPostgresExporter(t, mock, new(pmmAgentID))
				errStop := expectActionResult(mock, pmmAgentID)

				_, err := s.StartPTPgSummaryAction(t.Context(), req)
				require.ErrorIs(t, err, errStop)
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}

	t.Run("requested pmm-agent runs the exporter", func(t *testing.T) {
		t.Parallel()

		s, mock := setupActionsServer(t, models.PostgreSQLServiceType, models.RemoteRDSNodeType)
		expectPostgresExporter(t, mock, new("pmm-agent-id"))
		errStop := expectActionResult(mock, "pmm-agent-id")

		_, err := s.StartPTPgSummaryAction(t.Context(), &actionsv1.StartPTPgSummaryActionParams{ServiceId: testServiceID, PmmAgentId: "pmm-agent-id"})
		require.ErrorIs(t, err, errStop)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("requested pmm-agent does not run the exporter", func(t *testing.T) {
		t.Parallel()

		s, mock := setupActionsServer(t, models.PostgreSQLServiceType, models.RemoteRDSNodeType)
		expectPostgresExporter(t, mock, new(models.PMMServerAgentID))

		_, err := s.StartPTPgSummaryAction(t.Context(), &actionsv1.StartPTPgSummaryActionParams{ServiceId: testServiceID, PmmAgentId: "pmm-agent-id"})
		assertFailedPrecondition(t, err, "Couldn't find pmm-agent-id to run action")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no pmm-agent for remote service", func(t *testing.T) {
		t.Parallel()

		s, mock := setupActionsServer(t, models.PostgreSQLServiceType, models.RemoteRDSNodeType)
		expectPostgresExporter(t, mock, nil)

		_, err := s.StartPTPgSummaryAction(t.Context(), req)
		assertFailedPrecondition(t, err, "No pmm-agent found for service service-id")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exporter lookup fails", func(t *testing.T) {
		t.Parallel()

		s, mock := setupActionsServer(t, models.PostgreSQLServiceType, models.RemoteRDSNodeType)
		errLookup := errors.New("lookup failed")
		mock.ExpectQuery(`SELECT .+ FROM "services" WHERE .+ LIMIT 1`).
			WithArgs(testServiceID).
			WillReturnError(errLookup)

		_, err := s.StartPTPgSummaryAction(t.Context(), req)
		require.ErrorIs(t, err, errLookup)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no pmm-agent on node", func(t *testing.T) {
		t.Parallel()

		s, mock := setupActionsServer(t, models.PostgreSQLServiceType, models.GenericNodeType)
		expectPostgresExporter(t, mock, new("pmm-agent-id"))
		expectNoPMMAgentsOnNode(mock)

		_, err := s.StartPTPgSummaryAction(t.Context(), req)
		assertFailedPrecondition(t, err, "No pmm-agent running on node node-id")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
