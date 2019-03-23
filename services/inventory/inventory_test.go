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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestNodes(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := logger.Set(context.Background(), t.Name())

	setup := func(t *testing.T) (q *reform.Querier, ns *NodesService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		q = tx.Querier
		require.NoError(t, err)

		r := new(mockRegistry)
		r.Test(t)
		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
			r.AssertExpectations(t)
		}
		ns = NewNodesService(r)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		actualNodes, err := ns.List(ctx, q)
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMMServerNodeType

		actualNode, err := ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.GenericNodeType,
			NodeName: "test-bm",
		})
		require.NoError(t, err)
		expectedNode := &inventorypb.GenericNode{
			NodeId:   "/node_id/00000000-0000-4000-8000-000000000001",
			NodeName: "test-bm",
		}
		assert.Equal(t, expectedNode, actualNode)

		actualNode, err = ns.Get(ctx, q, "/node_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		assert.Equal(t, expectedNode, actualNode)

		actualNodes, err = ns.List(ctx, q)
		require.NoError(t, err)
		require.Len(t, actualNodes, 2)
		assert.Equal(t, expectedNode, actualNodes[0])

		err = ns.Remove(ctx, q, "/node_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		actualNode, err = ns.Get(ctx, q, "/node_id/00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "/node_id/00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		actualNode, err := ns.Get(ctx, q, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.GenericNodeType,
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node name.`), err)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.GenericNodeType,
			NodeName: "test",
			Address:  pointer.ToString("test"),
		})
		require.NoError(t, err)

		_, err = ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.RemoteNodeType,
			NodeName: "test",
		})
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test" already exists.`), err)
	})

	t.Run("AddHostnameNotUnique", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.GenericNodeType,
			NodeName: "test1",
			Address:  pointer.ToString("test"),
		})
		require.NoError(t, err)

		_, err = ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.GenericNodeType,
			NodeName: "test2",
			Address:  pointer.ToString("test"),
		})
		require.NoError(t, err)
	})

	t.Run("AddInstanceRegionNotUnique", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.RemoteAmazonRDSNodeType,
			NodeName: "test1",
			Address:  pointer.ToString("test-instance"),
			Region:   pointer.ToString("test-region"),
		})
		require.NoError(t, err)

		_, err = ns.Add(ctx, q, &AddNodeParams{
			NodeType: models.RemoteAmazonRDSNodeType,
			NodeName: "test2",
			Address:  pointer.ToString("test-instance"),
			Region:   pointer.ToString("test-region"),
		})
		expected := status.New(codes.AlreadyExists, `Node with instance "test-instance" and region "test-region" already exists.`)
		tests.AssertGRPCError(t, expected, err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		q, ns, teardown := setup(t)
		defer teardown(t)

		err := ns.Remove(ctx, q, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})
}

func TestServices(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := logger.Set(context.Background(), t.Name())

	setup := func(t *testing.T) (q *reform.Querier, ss *ServicesService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q = tx.Querier

		r := new(mockRegistry)
		r.Test(t)
		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
			r.AssertExpectations(t)
		}
		ns := NewNodesService(r)
		ss = NewServicesService(r, ns)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		q, ss, teardown := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, q)
		require.NoError(t, err)
		require.Len(t, actualServices, 0)

		actualMySQLService, err := ss.AddMySQL(ctx, q, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306))
		require.NoError(t, err)
		expectedService := &inventorypb.MySQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000001",
			ServiceName: "test-mysql",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        3306,
		}
		assert.Equal(t, expectedService, actualMySQLService)

		actualService, err := ss.Get(ctx, q, "/service_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, q)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedService, actualServices[0])

		err = ss.Remove(ctx, q, "/service_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, q, "/service_id/00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualService)

		actualService, err = ss.AddMongoDB(ctx, q, "test-mongo", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(27017))
		require.NoError(t, err)
		expectedMdbService := &inventorypb.MongoDBService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000002",
			ServiceName: "test-mongo",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        27017,
		}
		assert.Equal(t, expectedMdbService, actualService)

		actualService, err = ss.Get(ctx, q, "/service_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		assert.Equal(t, expectedMdbService, actualService)

		actualServices, err = ss.List(ctx, q)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedMdbService, actualServices[0])

		err = ss.Remove(ctx, q, "/service_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, q, "/service_id/00000000-0000-4000-8000-000000000002")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000002" not found.`), err)
		assert.Nil(t, actualService)

		actualService, err = ss.AddPostgreSQL(ctx, q, "test-postgres", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(5432))
		require.NoError(t, err)
		expectedPostgreSQLService := &inventorypb.PostgreSQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000003",
			ServiceName: "test-postgres",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        5432,
		}
		assert.Equal(t, expectedPostgreSQLService, actualService)

		actualService, err = ss.Get(ctx, q, "/service_id/00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		assert.Equal(t, expectedPostgreSQLService, actualService)

		actualServices, err = ss.List(ctx, q)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedPostgreSQLService, actualServices[0])

		err = ss.Remove(ctx, q, "/service_id/00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, q, "/service_id/00000000-0000-4000-8000-000000000003")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000003" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		q, ss, teardown := setup(t)
		defer teardown(t)

		actualNode, err := ss.Get(ctx, q, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Service ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		q, ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, q, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306))
		require.NoError(t, err)

		_, err = ss.AddMySQL(ctx, q, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306))
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Service with name "test-mysql" already exists.`), err)
	})

	t.Run("AddNodeNotFound", func(t *testing.T) {
		q, ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, q, "test-mysql", "no-such-id", pointer.ToString("127.0.0.1"), pointer.ToUint16(3306))
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		q, ss, teardown := setup(t)
		defer teardown(t)

		err := ss.Remove(ctx, q, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})
}

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

		db = reform.NewDB(tests.OpenTestDB(t), postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		r := new(mockRegistry)
		r.Test(t)

		ns := NewNodesService(r)
		ss = NewServicesService(r, ns)
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
		pmmAgent, err := as.AddPMMAgent(ctx, models.PMMServerNodeID)
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
			ChangeDisabled: &inventorypb.ChangeNodeExporterRequest_Disabled{
				Disabled: true,
			},
		})
		require.NoError(t, err)
		expectedNodeExporter = &inventorypb.NodeExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000002",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000001",
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter)

		actualAgent, err := as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		assert.Equal(t, expectedNodeExporter, actualAgent)

		s, err := ss.AddMySQL(ctx, db.Querier, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306))
		require.NoError(t, err)

		actualAgent, err = as.AddMySQLdExporter(ctx, db.Querier, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ID(),
			Username:   "username",
		})
		require.NoError(t, err)
		expectedMySQLdExporter := &inventorypb.MySQLdExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000004",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000001",
			ServiceId:  s.ID(),
			Username:   "username",
		}
		assert.Equal(t, expectedMySQLdExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000004")
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporter, actualAgent)

		ms, err := ss.AddMongoDB(ctx, db.Querier, "test-mongo", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(27017))
		require.NoError(t, err)

		actualAgent, err = as.AddMongoDBExporter(ctx, db.Querier, &inventorypb.AddMongoDBExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ms.ID(),
			Username:   "username",
		})
		require.NoError(t, err)
		expectedMongoDBExporter := &inventorypb.MongoDBExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ms.ID(),
			Username:   "username",
		}
		assert.Equal(t, expectedMongoDBExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedMongoDBExporter, actualAgent)

		ps, err := ss.AddPostgreSQL(ctx, db.Querier, "test-postgres", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(5432))
		require.NoError(t, err)

		actualAgent, err = as.AddPostgresExporter(ctx, db.Querier, &inventorypb.AddPostgresExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ps.ID(),
			Username:   "username",
		})
		require.NoError(t, err)
		expectedPostgresExporter := &inventorypb.PostgresExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000008",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ps.ID(),
			Username:   "username",
		}
		assert.Equal(t, expectedPostgresExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000008")
		require.NoError(t, err)
		assert.Equal(t, expectedPostgresExporter, actualAgent)

		// err = as.SetDisabled(ctx, db, "/agent_id/00000000-0000-4000-8000-000000000001", true)
		// require.NoError(t, err)
		// expectedMySQLdExporter.Disabled = true
		// actualAgent, err = as.Get(ctx, db, "/agent_id/00000000-0000-4000-8000-000000000001")
		// require.NoError(t, err)
		// assert.Equal(t, expectedMySQLdExporter, actualAgent)

		actualAgents, err = as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 5)
		assert.Equal(t, pmmAgent, actualAgents[0])
		assert.Equal(t, expectedNodeExporter, actualAgents[1])
		assert.Equal(t, expectedMySQLdExporter, actualAgents[2])
		assert.Equal(t, expectedMongoDBExporter, actualAgents[3])
		assert.Equal(t, expectedPostgresExporter, actualAgents[4])

		actualAgents, err = as.List(ctx, AgentFilters{ServiceID: s.ID()})
		require.NoError(t, err)
		require.Len(t, actualAgents, 1)
		assert.Equal(t, expectedMySQLdExporter, actualAgents[0])

		actualAgents, err = as.List(ctx, AgentFilters{PMMAgentID: pmmAgent.AgentId})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4)
		assert.Equal(t, expectedNodeExporter, actualAgents[0])
		assert.Equal(t, expectedMySQLdExporter, actualAgents[1])
		assert.Equal(t, expectedMongoDBExporter, actualAgents[2])
		assert.Equal(t, expectedPostgresExporter, actualAgents[3])

		actualAgents, err = as.List(ctx, AgentFilters{PMMAgentID: pmmAgent.AgentId, NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4)
		assert.Equal(t, expectedNodeExporter, actualAgents[0])
		assert.Equal(t, expectedMySQLdExporter, actualAgents[1])
		assert.Equal(t, expectedMongoDBExporter, actualAgents[2])
		assert.Equal(t, expectedPostgresExporter, actualAgents[3])

		actualAgents, err = as.List(ctx, AgentFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualAgents, 1)
		assert.Equal(t, expectedNodeExporter, actualAgents[0])

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000002")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000002" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000004")
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000004")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000004" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000006" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000008")
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000008")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000008" not found.`), err)
		assert.Nil(t, actualAgent)

		as.r.(*mockRegistry).On("Kick", ctx, "/agent_id/00000000-0000-4000-8000-000000000001").Return(true)
		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000001")
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
		actualAgent, err := as.AddPMMAgent(ctx, models.PMMServerNodeID)
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000001",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    false,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent)

		as.r.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000002").Return(true)
		actualAgent, err = as.AddPMMAgent(ctx, models.PMMServerNodeID)
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
		pmmAgent, err := as.AddPMMAgent(ctx, models.PMMServerNodeID)
		require.NoError(t, err)

		_, err = as.AddMySQLdExporter(ctx, db.Querier, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	// t.Run("DisableNotFound", func(t *testing.T) {
	// setup(t)
	// defer teardown(t)

	// 	err := as.SetDisabled(ctx, db, "no-such-id", true)
	// 	tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	// })

	t.Run("RemoveNotFound", func(t *testing.T) {
		setup(t)
		defer teardown(t)

		err := as.Remove(ctx, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})
}
