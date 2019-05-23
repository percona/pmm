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

package inventory

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestAgents(t *testing.T) {
	var (
		ctx context.Context
		db  *reform.DB
		ss  *ServicesService
		as  *AgentsService
	)
	setup := func(t *testing.T) {
		ctx = logger.Set(context.Background(), t.Name())

		uuid.SetRand(new(tests.IDReader))

		db = reform.NewDB(testdb.Open(t), postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		r := new(mockRegistry)
		r.Test(t)

		ss = NewServicesService(db, r)
		as = NewAgentsService(db, r)
	}

	teardown := func(t *testing.T) {
		assert.NoError(t, db.DBInterface().(*sql.DB).Close())
		as.r.(*mockRegistry).AssertExpectations(t)
	}

	t.Run("Basic", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		actualAgents, err := as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 0)

		as.r.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000001").Return(true)
		as.r.(*mockRegistry).On("SendSetStateRequest", ctx, "/agent_id/00000000-0000-4000-8000-000000000001")
		as.r.(*mockRegistry).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		actualNodeExporter, err := as.AddNodeExporter(ctx, &inventorypb.AddNodeExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
		})
		require.NoError(t, err)
		expectedNodeExporter := &inventorypb.NodeExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000002",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000001",
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter)

		actualNodeExporter, err = as.ChangeNodeExporter(ctx, &inventorypb.ChangeNodeExporterRequest{
			AgentId: "/agent_id/00000000-0000-4000-8000-000000000002",
			Common: &inventorypb.ChangeCommonAgentParams{
				ChangeDisabled: &inventorypb.ChangeCommonAgentParams_Disabled{
					Disabled: true,
				},
			},
		})
		require.NoError(t, err)
		expectedNodeExporter = &inventorypb.NodeExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000002",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000001",
			Disabled:   true,
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter)

		actualAgent, err := as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		assert.Equal(t, expectedNodeExporter, actualAgent)

		s, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		actualAgent, err = as.AddMySQLdExporter(ctx, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedMySQLdExporter := &inventorypb.MySQLdExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000004",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000001",
			ServiceId:  s.ServiceId,
			Username:   "username",
		}
		assert.Equal(t, expectedMySQLdExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000004")
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporter, actualAgent)

		ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
		})
		require.NoError(t, err)

		actualAgent, err = as.AddMongoDBExporter(ctx, &inventorypb.AddMongoDBExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ms.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedMongoDBExporter := &inventorypb.MongoDBExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ms.ServiceId,
			Username:   "username",
		}
		assert.Equal(t, expectedMongoDBExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedMongoDBExporter, actualAgent)

		actualAgent, err = as.AddQANMySQLSlowlogAgent(ctx, &inventorypb.AddQANMySQLSlowlogAgentRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedQANMySQLSlowlogAgent := &inventorypb.QANMySQLSlowlogAgent{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000007",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ServiceId,
			Username:   "username",
		}
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000007")
		require.NoError(t, err)
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgent)

		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(5432),
		})
		require.NoError(t, err)

		actualAgent, err = as.AddPostgresExporter(ctx, &inventorypb.AddPostgresExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ps.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedPostgresExporter := &inventorypb.PostgresExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000009",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ps.ServiceId,
			Username:   "username",
		}
		assert.Equal(t, expectedPostgresExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000009")
		require.NoError(t, err)
		assert.Equal(t, expectedPostgresExporter, actualAgent)

		actualAgents, err = as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 6)
		assert.Equal(t, pmmAgent, actualAgents[0])
		assert.Equal(t, expectedNodeExporter, actualAgents[1])
		assert.Equal(t, expectedMySQLdExporter, actualAgents[2])
		assert.Equal(t, expectedMongoDBExporter, actualAgents[3])
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[4])
		assert.Equal(t, expectedPostgresExporter, actualAgents[5])

		// filter by service ID
		actualAgents, err = as.List(ctx, AgentFilters{ServiceID: s.ServiceId})
		require.NoError(t, err)
		require.Len(t, actualAgents, 2)
		assert.Equal(t, expectedMySQLdExporter, actualAgents[0])
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[1])

		actualAgents, err = as.List(ctx, AgentFilters{PMMAgentID: pmmAgent.AgentId})
		require.NoError(t, err)
		require.Len(t, actualAgents, 5)
		assert.Equal(t, expectedNodeExporter, actualAgents[0])
		assert.Equal(t, expectedMySQLdExporter, actualAgents[1])
		assert.Equal(t, expectedMongoDBExporter, actualAgents[2])
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[3])
		assert.Equal(t, expectedPostgresExporter, actualAgents[4])

		actualAgents, err = as.List(ctx, AgentFilters{PMMAgentID: pmmAgent.AgentId, NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualAgents, 5)
		assert.Equal(t, expectedNodeExporter, actualAgents[0])
		assert.Equal(t, expectedMySQLdExporter, actualAgents[1])
		assert.Equal(t, expectedMongoDBExporter, actualAgents[2])
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[3])
		assert.Equal(t, expectedPostgresExporter, actualAgents[4])

		actualAgents, err = as.List(ctx, AgentFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualAgents, 1)
		assert.Equal(t, expectedNodeExporter, actualAgents[0])

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000002", false)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000002")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000002" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000004", false)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000004")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000004" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000006", false)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000006" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000007", false)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000007")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000007" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000009", false)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000009")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000009" not found.`), err)
		assert.Nil(t, actualAgent)

		as.r.(*mockRegistry).On("Kick", ctx, "/agent_id/00000000-0000-4000-8000-000000000001").Return(true)
		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000001", false)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualAgent)

		actualAgents, err = as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 0)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		actualNode, err := as.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Agent ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddPMMAgent", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		as.r.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000001").Return(false)
		actualAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000001",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    false,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent)

		as.r.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000002").Return(true)
		actualAgent, err = as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent = &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000002",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent)
	})

	t.Run("AddPmmAgentNotFound", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		_, err := as.AddNodeExporter(ctx, &inventorypb.AddNodeExporterRequest{
			PmmAgentId: "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})

	t.Run("AddServiceNotFound", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		as.r.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000001").Return(true)
		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		_, err = as.AddMySQLdExporter(ctx, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		err := as.Remove(ctx, "no-such-id", false)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})
}
