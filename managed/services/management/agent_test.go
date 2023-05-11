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

package management

import (
	"context"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1beta1 "github.com/percona/pmm/api/managementpb/agent"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func setup(t *testing.T) (context.Context, *AgentService, func(t *testing.T), *mockPrometheusService) { //nolint:unparam
	t.Helper()

	ctx := logger.Set(context.Background(), t.Name())
	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	vmdb := &mockPrometheusService{}
	vmdb.Test(t)

	state := &mockAgentsStateUpdater{}
	state.Test(t)

	ar := &mockAgentsRegistry{}
	ar.Test(t)

	teardown := func(t *testing.T) {
		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())
		vmdb.AssertExpectations(t)
		state.AssertExpectations(t)
		ar.AssertExpectations(t)
	}
	s := NewAgentService(db, ar)

	return ctx, s, teardown, vmdb
}

func TestAgentService(t *testing.T) {
	t.Run("List of agents", func(t *testing.T) {
		const (
			pgExporterID      = "/agent_id/00000000-0000-4000-8000-000000000003"
			pgStatStatementID = "/agent_id/00000000-0000-4000-8000-000000000004"
		)

		t.Run("should output a list of agents provisioned by default", func(t *testing.T) {
			ctx, s, teardown, _ := setup(t)
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
			assert.Len(t, response.Agents, 3) // 2 exporters + 1 agent
		})

		t.Run("should output a list of agents provisioned for RDS service", func(t *testing.T) {
			ctx, s, teardown, _ := setup(t)
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

			s.r.(*mockAgentsRegistry).On("IsConnected", rdsExporter.AgentID).Return(false).Once() // RDS exporter

			response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{
				ServiceId: service.ServiceID,
			})

			require.NoError(t, err)
			assert.Len(t, response.Agents, 1)
		})

		t.Run("should output a list of agents provisioned for Azure service", func(t *testing.T) {
			ctx, s, teardown, _ := setup(t)
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

			s.r.(*mockAgentsRegistry).On("IsConnected", azureExporter.AgentID).Return(false).Once() // Azure exporter

			response, err := s.ListAgents(ctx, &agentv1beta1.ListAgentRequest{
				ServiceId: service.ServiceID,
			})

			require.NoError(t, err)
			assert.Len(t, response.Agents, 1)
		})
	})
}
