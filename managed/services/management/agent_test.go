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

package management

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1beta1 "github.com/percona/pmm/api/managementpb/agent"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

var now time.Time

func setup(t *testing.T) (context.Context, *AgentService, func(t *testing.T)) {
	t.Helper()

	now = models.Now()
	origNowF := models.Now
	models.Now = func() time.Time {
		return now
	}

	ctx := logger.Set(context.Background(), t.Name())
	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	state := &mockAgentsStateUpdater{}
	state.Test(t)

	ar := &mockAgentsRegistry{}
	ar.Test(t)

	teardown := func(t *testing.T) {
		t.Helper()
		models.Now = origNowF
		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())
		state.AssertExpectations(t)
		ar.AssertExpectations(t)
	}
	s := NewAgentService(db, ar)

	return ctx, s, teardown
}

func TestAgentService(t *testing.T) {
	t.Run("Should return a validation error when no params passed", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		defer teardown(t)

		response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{})
		assert.Nil(t, response)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Either service_id or node_id is expected."), err)
	})

	t.Run("Should return a validation error when both params passed", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		defer teardown(t)

		response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{ServiceId: "foo-id", NodeId: "bar-id"})
		assert.Nil(t, response)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Either service_id or node_id is expected, not both."), err)
	})

	t.Run("ListAgents", func(t *testing.T) {
		const (
			pgExporterID      = "/agent_id/00000000-0000-4000-8000-000000000003"
			pgStatStatementID = "/agent_id/00000000-0000-4000-8000-000000000004"
		)

		t.Run("should output a list of agents provisioned by default", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			services, err := models.FindServices(s.db.Querier, models.ServiceFilters{
				NodeID: models.PMMServerNodeID,
			})

			require.NoError(t, err)
			assert.Len(t, services, 1)
			service := services[0]

			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once() // PMM Server Agent
			s.r.(*mockAgentsRegistry).On("IsConnected", pgExporterID).Return(false).Once()           // PMM Server PostgreSQL exporter
			s.r.(*mockAgentsRegistry).On("IsConnected", pgStatStatementID).Return(false).Once()      // PMM Server PG Stat Statements agent
			response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{
				ServiceId: service.ServiceID,
			})
			require.NoError(t, err)

			expected := []*agentv1beta1.UniversalAgent{
				{
					AgentId:     pgExporterID,
					AgentType:   "postgres_exporter",
					PmmAgentId:  models.PMMServerAgentID,
					IsConnected: false,
					CreatedAt:   timestamppb.New(now),
					UpdatedAt:   timestamppb.New(now),
					Username:    "postgres",
					PostgresqlOptions: &agentv1beta1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet: false,
					},
					ServiceId:               "/service_id/00000000-0000-4000-8000-000000000002",
					Status:                  "UNKNOWN",
					Tls:                     true,
					CommentsParsingDisabled: true,
				},
				{
					AgentId:     pgStatStatementID,
					AgentType:   "qan-postgresql-pgstatements-agent",
					PmmAgentId:  models.PMMServerAgentID,
					IsConnected: false,
					CreatedAt:   timestamppb.New(now),
					UpdatedAt:   timestamppb.New(now),
					Username:    "postgres",
					PostgresqlOptions: &agentv1beta1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet: false,
					},
					ServiceId:               "/service_id/00000000-0000-4000-8000-000000000002",
					Status:                  "UNKNOWN",
					Tls:                     true,
					CommentsParsingDisabled: true,
				},
				{
					AgentId:      models.PMMServerAgentID,
					AgentType:    "pmm-agent",
					RunsOnNodeId: models.PMMServerAgentID,
					IsConnected:  true,
					CreatedAt:    timestamppb.New(now),
					UpdatedAt:    timestamppb.New(now),
				},
			}

			assert.Equal(t, expected, response.Agents)
		})

		t.Run("should output a list of agents provisioned for RDS service", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			node, err := models.CreateNode(s.db.Querier, models.RemoteRDSNodeType, &models.CreateNodeParams{
				NodeName: "test",
				Address:  "test-address",
				Region:   pointer.ToString("test-region"),
			})
			require.NoError(t, err)

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      node.NodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			pmmAgent, err := models.CreatePMMAgent(s.db.Querier, models.PMMServerNodeID, nil)
			require.NoError(t, err)

			rdsExporter, err := models.CreateAgent(s.db.Querier, models.RDSExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				ServiceID:  service.ServiceID,
			})
			require.NoError(t, err)

			s.r.(*mockAgentsRegistry).On("IsConnected", rdsExporter.AgentID).Return(false).Once()

			response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{
				ServiceId: service.ServiceID,
			})
			require.NoError(t, err)

			expected := []*agentv1beta1.UniversalAgent{
				{
					AgentId:     rdsExporter.AgentID,
					AgentType:   "rds_exporter",
					PmmAgentId:  "/agent_id/00000000-0000-4000-8000-000000000007",
					IsConnected: false,
					CreatedAt:   timestamppb.New(now),
					UpdatedAt:   timestamppb.New(now),
					ServiceId:   "/service_id/00000000-0000-4000-8000-000000000006",
					Status:      "UNKNOWN",
				},
			}
			assert.Equal(t, expected, response.Agents)
		})

		t.Run("should output a list of agents provisioned for Azure service", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			node, err := models.CreateNode(s.db.Querier, models.RemoteAzureDatabaseNodeType, &models.CreateNodeParams{
				NodeName: "test",
				Address:  "test-address",
				Region:   pointer.ToString("test-region"),
			})
			require.NoError(t, err)

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      node.NodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			pmmAgent, err := models.CreatePMMAgent(s.db.Querier, models.PMMServerNodeID, nil)
			require.NoError(t, err)

			azureExporter, err := models.CreateAgent(s.db.Querier, models.AzureDatabaseExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				ServiceID:  service.ServiceID,
			})
			require.NoError(t, err)

			s.r.(*mockAgentsRegistry).On("IsConnected", azureExporter.AgentID).Return(false).Once()

			response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{
				ServiceId: service.ServiceID,
			})
			require.NoError(t, err)

			expected := []*agentv1beta1.UniversalAgent{
				{
					AgentId:     azureExporter.AgentID,
					AgentType:   "azure_database_exporter",
					PmmAgentId:  "/agent_id/00000000-0000-4000-8000-000000000007",
					IsConnected: false,
					CreatedAt:   timestamppb.New(now),
					UpdatedAt:   timestamppb.New(now),
					ServiceId:   "/service_id/00000000-0000-4000-8000-000000000006",
					Status:      "UNKNOWN",
				},
			}
			assert.Equal(t, expected, response.Agents)
		})
	})
}
