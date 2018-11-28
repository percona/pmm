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
		db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		ns = &NodesService{
			Q: tx.Querier,
		}
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		actualNodes, err := ns.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMMServerNodeType

		actualNode, err := ns.Add(ctx, models.BareMetalNodeType, "test-bm", pointer.ToString("test-bm"), nil)
		require.NoError(t, err)
		expectedNode := &inventory.BareMetalNode{
			Id:       2,
			Name:     "test-bm",
			Hostname: "test-bm",
		}
		assert.Equal(t, expectedNode, actualNode)

		actualNode, err = ns.Get(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, expectedNode, actualNode)

		actualNode, err = ns.Change(ctx, 2, "test-bm-new")
		require.NoError(t, err)
		expectedNode = &inventory.BareMetalNode{
			Id:       2,
			Name:     "test-bm-new",
			Hostname: "test-bm",
		}
		assert.Equal(t, expectedNode, actualNode)

		actualNodes, err = ns.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualNodes, 2)
		assert.Equal(t, expectedNode, actualNodes[1])

		err = ns.Remove(ctx, 2)
		require.NoError(t, err)
		actualNode, err = ns.Get(ctx, 2)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 2 not found.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, models.VirtualMachineNodeType, "test", pointer.ToString("test"), nil)
		require.NoError(t, err)

		_, err = ns.Add(ctx, models.ContainerNodeType, "test", nil, nil)
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test" already exists.`), err)
	})

	t.Run("AddHostnameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, models.BareMetalNodeType, "test1", pointer.ToString("test"), nil)
		require.NoError(t, err)

		_, err = ns.Add(ctx, models.BareMetalNodeType, "test2", pointer.ToString("test"), nil)
		require.NoError(t, err)
	})

	t.Run("AddHostnameRegionNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, models.AWSRDSNodeType, "test1", pointer.ToString("test-hostname"), pointer.ToString("test-region"))
		require.NoError(t, err)

		_, err = ns.Add(ctx, models.AWSRDSNodeType, "test2", pointer.ToString("test-hostname"), pointer.ToString("test-region"))
		expected := status.New(codes.AlreadyExists, `Node with hostname "test-hostname" and region "test-region" already exists.`)
		tests.AssertGRPCError(t, expected, err)
	})

	t.Run("ChangeNotFound", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Change(ctx, 2, "test-bm-new")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 2 not found.`), err)
	})

	t.Run("ChangeNameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, models.RemoteNodeType, "test-remote", nil, nil)
		require.NoError(t, err)

		rdsNode, err := ns.Add(ctx, models.AWSRDSNodeType, "test-rds", nil, nil)
		require.NoError(t, err)

		_, err = ns.Change(ctx, rdsNode.(*inventory.AWSRDSNode).Id, "test-remote")
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test-remote" already exists.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		err := ns.Remove(ctx, 2)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 2 not found.`), err)
	})
}

func TestServices(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := context.Background()

	setup := func(t *testing.T) (ss *ServicesService, teardown func(t *testing.T)) {
		db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		ss = &ServicesService{
			Q: tx.Querier,
		}
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 0)

		actualService, err := ss.AddMySQL(ctx, "test-mysql", 1, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)
		expectedService := &inventory.MySQLService{
			Id:      1000,
			Name:    "test-mysql",
			NodeId:  1,
			Address: "127.0.0.1",
			Port:    3306,
		}
		assert.Equal(t, expectedService, actualService)

		actualService, err = ss.Get(ctx, 1000)
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualService, err = ss.Change(ctx, 1000, "test-mysql-new")
		require.NoError(t, err)
		expectedService = &inventory.MySQLService{
			Id:      1000,
			Name:    "test-mysql-new",
			NodeId:  1,
			Address: "127.0.0.1",
			Port:    3306,
		}
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedService, actualServices[0])

		err = ss.Remove(ctx, 1000)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, 1000)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID 1000 not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, "test-mysql", 1, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		_, err = ss.AddMySQL(ctx, "test-mysql", 1, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Service with name "test-mysql" already exists.`), err)
	})

	t.Run("AddNodeNotFound", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, "test-mysql", 2, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 2 not found.`), err)
	})

	t.Run("ChangeNotFound", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.Change(ctx, 1, "test-mysql-new")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID 1 not found.`), err)
	})

	t.Run("ChangeNameNotUnique", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, "test-mysql", 1, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		s, err := ss.AddMySQL(ctx, "test-mysql-2", 1, pointer.ToString("127.0.0.2"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		_, err = ss.Change(ctx, s.(*inventory.MySQLService).Id, "test-mysql")
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Service with name "test-mysql" already exists.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		err := ss.Remove(ctx, 1)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID 1 not found.`), err)
	})
}

func TestAgents(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	ctx := context.Background()

	setup := func(t *testing.T) (ss *ServicesService, as *AgentsService, teardown func(t *testing.T)) {
		db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		ss = &ServicesService{
			Q: tx.Querier,
		}
		as = &AgentsService{
			Q: tx.Querier,
		}
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ss, as, teardown := setup(t)
		defer teardown(t)

		actualAgents, err := as.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualAgents, 0)

		actualAgent, err := as.AddNodeExporter(ctx, 1, true)
		require.NoError(t, err)
		expectedNodeExporterAgent := &inventory.NodeExporter{
			Id:           1000000,
			RunsOnNodeId: 1,
			Disabled:     true,
		}
		assert.Equal(t, expectedNodeExporterAgent, actualAgent)

		actualAgent, err = as.Get(ctx, 1000000)
		require.NoError(t, err)
		assert.Equal(t, expectedNodeExporterAgent, actualAgent)

		_, err = ss.AddMySQL(ctx, "test-mysql", 1, pointer.ToString("127.0.0.1"), pointer.ToUint16(3306), nil)
		require.NoError(t, err)

		actualAgent, err = as.AddMySQLdExporter(ctx, 1, false, 1000, pointer.ToString("username"), nil)
		require.NoError(t, err)
		expectedMySQLdExporterAgent := &inventory.MySQLdExporter{
			Id:           1000001,
			RunsOnNodeId: 1,
			ServiceId:    1000,
			Username:     "username",
		}
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgent)

		actualAgent, err = as.Get(ctx, 1000001)
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgent)

		err = as.SetDisabled(ctx, 1000001, true)
		require.NoError(t, err)
		expectedMySQLdExporterAgent.Disabled = true
		actualAgent, err = as.Get(ctx, 1000001)
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgent)

		actualAgents, err = as.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualAgents, 2)
		assert.Equal(t, expectedNodeExporterAgent, actualAgents[0])
		assert.Equal(t, expectedMySQLdExporterAgent, actualAgents[1])

		err = as.Remove(ctx, 1000000)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, 1000000)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID 1000000 not found.`), err)
		assert.Nil(t, actualAgent)

		err = as.Remove(ctx, 1000001)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, 1000001)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID 1000001 not found.`), err)
		assert.Nil(t, actualAgent)

		actualAgents, err = as.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualAgents, 0)
	})

	t.Run("AddNodeNotFound", func(t *testing.T) {
		_, as, teardown := setup(t)
		defer teardown(t)

		_, err := as.AddNodeExporter(ctx, 1000, true)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 1000 not found.`), err)
	})

	t.Run("AddServiceNotFound", func(t *testing.T) {
		_, as, teardown := setup(t)
		defer teardown(t)

		_, err := as.AddMySQLdExporter(ctx, 1, false, 1000, pointer.ToString("username"), nil)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID 1000 not found.`), err)
	})

	t.Run("DisableNotFound", func(t *testing.T) {
		_, as, teardown := setup(t)
		defer teardown(t)

		err := as.SetDisabled(ctx, 1, true)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID 1 not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		_, as, teardown := setup(t)
		defer teardown(t)

		err := as.Remove(ctx, 999999999)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID 999999999 not found.`), err)
	})
}
