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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestNodes(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := context.Background()

	setup := func(t *testing.T) (ns *NodesService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		ns = NewNodesService(tx.Querier)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		actualNodes, err := ns.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMMServerNodeType

		actualNode, err := ns.Add(ctx, "", models.GenericNodeType, "test-bm", pointer.ToString("test-bm"), nil)
		require.NoError(t, err)
		expectedNode := &inventory.GenericNode{
			Id:       "gen:00000000-0000-4000-8000-000000000001",
			Name:     "test-bm",
			Hostname: "test-bm",
		}
		assert.Equal(t, expectedNode, actualNode)

		actualNode, err = ns.Get(ctx, "gen:00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		assert.Equal(t, expectedNode, actualNode)

		actualNode, err = ns.Change(ctx, "gen:00000000-0000-4000-8000-000000000001", "test-bm-new")
		require.NoError(t, err)
		expectedNode = &inventory.GenericNode{
			Id:       "gen:00000000-0000-4000-8000-000000000001",
			Name:     "test-bm-new",
			Hostname: "test-bm",
		}
		assert.Equal(t, expectedNode, actualNode)

		actualNodes, err = ns.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualNodes, 2)
		assert.Equal(t, expectedNode, actualNodes[0])

		err = ns.Remove(ctx, "gen:00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		actualNode, err = ns.Get(ctx, "gen:00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "gen:00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		actualNode, err := ns.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, "", models.GenericNodeType, "", nil, nil)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node name.`), err)
	})

	t.Run("AddIDNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, "test-id", models.GenericNodeType, "test", nil, nil)
		require.NoError(t, err)

		_, err = ns.Add(ctx, "test-id", models.GenericNodeType, "test", nil, nil)
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with ID "test-id" already exists.`), err)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, "", models.GenericNodeType, "test", pointer.ToString("test"), nil)
		require.NoError(t, err)

		_, err = ns.Add(ctx, "", models.RemoteNodeType, "test", nil, nil)
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test" already exists.`), err)
	})

	t.Run("AddHostnameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, "", models.GenericNodeType, "test1", pointer.ToString("test"), nil)
		require.NoError(t, err)

		_, err = ns.Add(ctx, "", models.GenericNodeType, "test2", pointer.ToString("test"), nil)
		require.NoError(t, err)
	})

	t.Run("AddHostnameRegionNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, "", models.AmazonRDSRemoteNodeType, "test1", pointer.ToString("test-hostname"), pointer.ToString("test-region"))
		require.NoError(t, err)

		_, err = ns.Add(ctx, "", models.AmazonRDSRemoteNodeType, "test2", pointer.ToString("test-hostname"), pointer.ToString("test-region"))
		expected := status.New(codes.AlreadyExists, `Node with hostname "test-hostname" and region "test-region" already exists.`)
		tests.AssertGRPCError(t, expected, err)
	})

	t.Run("ChangeNotFound", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Change(ctx, "no-such-id", "test-bm-new")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})

	t.Run("ChangeNameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, "", models.RemoteNodeType, "test-remote", nil, nil)
		require.NoError(t, err)

		rdsNode, err := ns.Add(ctx, "", models.AmazonRDSRemoteNodeType, "test-rds", nil, nil)
		require.NoError(t, err)

		_, err = ns.Change(ctx, rdsNode.(*inventory.AmazonRDSRemoteNode).Id, "test-remote")
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test-remote" already exists.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		err := ns.Remove(ctx, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})
}

func TestServices(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := context.Background()

	setup := func(t *testing.T) (ss *ServicesService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		ss = NewServicesService(tx.Querier)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 0)

		actualService, err := ss.AddMySQL(ctx, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)
		expectedService := &inventory.MySQLService{
			Id:   "gen:00000000-0000-4000-8000-000000000001",
			Name: "test-mysql",
			HostNodeInfo: &inventory.HostNodeInfo{
				NodeId:            models.PMMServerNodeID,
				ContainerId:       "TODO",
				ContainerName:     "TODO",
				KubernetesPodUid:  "TODO",
				KubernetesPodName: "TODO",
			},
			Address: "127.0.0.1",
			Port:    3306,
		}
		assert.Equal(t, expectedService, actualService)

		actualService, err = ss.Get(ctx, "gen:00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualService, err = ss.Change(ctx, "gen:00000000-0000-4000-8000-000000000001", "test-mysql-new")
		require.NoError(t, err)
		expectedService = &inventory.MySQLService{
			Id:   "gen:00000000-0000-4000-8000-000000000001",
			Name: "test-mysql-new",
			HostNodeInfo: &inventory.HostNodeInfo{
				NodeId:            models.PMMServerNodeID,
				ContainerId:       "TODO",
				ContainerName:     "TODO",
				KubernetesPodUid:  "TODO",
				KubernetesPodName: "TODO",
			},
			Address: "127.0.0.1",
			Port:    3306,
		}
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedService, actualServices[0])

		err = ss.Remove(ctx, "gen:00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "gen:00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "gen:00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualNode, err := ss.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Service ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		_, err = ss.AddMySQL(ctx, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Service with name "test-mysql" already exists.`), err)
	})

	t.Run("AddNodeNotFound", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, "test-mysql", "no-such-id", pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})

	t.Run("ChangeNotFound", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.Change(ctx, "no-such-id", "test-mysql-new")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("ChangeNameNotUnique", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		s, err := ss.AddMySQL(ctx, "test-mysql-2", models.PMMServerNodeID, pointer.ToString("127.0.0.2"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		_, err = ss.Change(ctx, s.(*inventory.MySQLService).Id, "test-mysql")
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Service with name "test-mysql" already exists.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		err := ss.Remove(ctx, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})
}

func TestAgents(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := context.Background()

	setup := func(t *testing.T) (ns *NodesService, ss *ServicesService, as *AgentsService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		ns = NewNodesService(tx.Querier)
		ss = NewServicesService(tx.Querier)
		as = NewAgentsService(tx.Querier, nil)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ns, ss, as, teardown := setup(t)
		defer teardown(t)

		actualAgents, err := as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 0)

		actualAgent, err := as.AddNodeExporter(ctx, models.PMMServerNodeID, true)
		require.NoError(t, err)
		expectedNodeExporterAgent := &inventory.NodeExporter{
			Id: "gen:00000000-0000-4000-8000-000000000001",
			HostNodeInfo: &inventory.HostNodeInfo{
				NodeId:            models.PMMServerNodeID,
				ContainerId:       "TODO",
				ContainerName:     "TODO",
				KubernetesPodUid:  "TODO",
				KubernetesPodName: "TODO",
			},
			Disabled: true,
		}
		assert.Equal(t, expectedNodeExporterAgent, actualAgent)

		actualAgent, err = as.Get(ctx, "gen:00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		assert.Equal(t, expectedNodeExporterAgent, actualAgent)

		_, err = ss.AddMySQL(ctx, "test-mysql", models.PMMServerNodeID, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		_, err = ns.Add(ctx, "some-node-id", models.GenericNodeType, "new node name", pointer.ToString("127.0.0.1"), pointer.ToString(models.RemoteNodeRegion))
		require.NoError(t, err)

		actualAgent, err = as.AddMySQLdExporter(ctx, "some-node-id", false, "gen:00000000-0000-4000-8000-000000000002", pointer.ToString("username"), nil)
		require.NoError(t, err)
		expectedMySQLdExporterAgent := &inventory.MySQLdExporter{
			Id: "gen:00000000-0000-4000-8000-000000000003",
			HostNodeInfo: &inventory.HostNodeInfo{
				NodeId:            "some-node-id",
				ContainerId:       "TODO",
				ContainerName:     "TODO",
				KubernetesPodUid:  "TODO",
				KubernetesPodName: "TODO",
			},
			ServiceId: "gen:00000000-0000-4000-8000-000000000002",
			Username:  "username",
		}
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgent)

		actualAgent, err = as.Get(ctx, "gen:00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgent)

		err = as.SetDisabled(ctx, "gen:00000000-0000-4000-8000-000000000003", true)
		require.NoError(t, err)
		expectedMySQLdExporterAgent.Disabled = true
		actualAgent, err = as.Get(ctx, "gen:00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgent)

		actualAgents, err = as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 2)
		assert.Equal(t, expectedNodeExporterAgent, actualAgents[0])
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgents[1])

		actualAgents, err = as.List(ctx, AgentFilters{ServiceID: "gen:00000000-0000-4000-8000-000000000002"})
		require.NoError(t, err)
		require.Len(t, actualAgents, 1)
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgents[0])

		actualAgents, err = as.List(ctx, AgentFilters{RunsOnNodeID: "some-node-id"})
		require.NoError(t, err)
		require.Len(t, actualAgents, 1)
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgents[0])

		actualAgents, err = as.List(ctx, AgentFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualAgents, 1)
		assert.Equal(t, expectedNodeExporterAgent, actualAgents[0])

		err = as.Remove(ctx, "gen:00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "gen:00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "gen:00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, "gen:00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "gen:00000000-0000-4000-8000-000000000003")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "gen:00000000-0000-4000-8000-000000000003" not found.`), err)
		assert.Nil(t, actualAgent)

		actualAgents, err = as.List(ctx, AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 0)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		_, _, as, teardown := setup(t)
		defer teardown(t)

		actualNode, err := as.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Agent ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddPMMAgent", func(t *testing.T) {
		_, _, as, teardown := setup(t)
		defer teardown(t)

		actualAgent, err := as.AddPMMAgent(ctx, models.PMMServerNodeID)
		require.NoError(t, err)
		expectedPMMAgent := &inventory.PMMAgent{
			Id: "gen:00000000-0000-4000-8000-000000000001",
			HostNodeInfo: &inventory.HostNodeInfo{
				NodeId:            models.PMMServerNodeID,
				ContainerId:       "TODO",
				ContainerName:     "TODO",
				KubernetesPodUid:  "TODO",
				KubernetesPodName: "TODO",
			},
		}
		assert.Equal(t, expectedPMMAgent, actualAgent)
	})

	t.Run("AddNodeNotFound", func(t *testing.T) {
		_, _, as, teardown := setup(t)
		defer teardown(t)

		_, err := as.AddNodeExporter(ctx, "no-such-id", true)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})

	t.Run("AddServiceNotFound", func(t *testing.T) {
		_, _, as, teardown := setup(t)
		defer teardown(t)

		_, err := as.AddMySQLdExporter(ctx, models.PMMServerNodeID, false, "no-such-id", pointer.ToString("username"), nil)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("DisableNotFound", func(t *testing.T) {
		_, _, as, teardown := setup(t)
		defer teardown(t)

		err := as.SetDisabled(ctx, "no-such-id", true)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		_, _, as, teardown := setup(t)
		defer teardown(t)

		err := as.Remove(ctx, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})
}
