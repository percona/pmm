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

		var actualNodes []inventory.Node
		actualNodes, err := ns.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMMServerNodeType

		var actualNode, expectedNode inventory.Node
		actualNode, err = ns.Add(ctx, models.BareMetalNodeType, "test-bm", pointer.ToString("test-bm"), nil)
		require.NoError(t, err)
		expectedNode = &inventory.BareMetalNode{
			Id:       2,
			Name:     "test-bm",
			Hostname: "test-bm",
		}
		assert.Equal(t, expectedNode, actualNode)

		actualNode, err = ns.Get(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, expectedNode, actualNode)

		err = ns.Change(ctx, 2, "test-bm-new")
		require.NoError(t, err)
		actualNodes, err = ns.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualNodes, 2)
		expectedNode = &inventory.BareMetalNode{
			Id:       2,
			Name:     "test-bm-new",
			Hostname: "test-bm",
		}
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

		err := ns.Change(ctx, 2, "test-bm-new")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 2 not found.`), err)
	})

	t.Run("ChangeNameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.Add(ctx, models.RemoteNodeType, "test-remote", nil, nil)
		require.NoError(t, err)

		rdsNode, err := ns.Add(ctx, models.AWSRDSNodeType, "test-rds", nil, nil)
		require.NoError(t, err)

		err = ns.Change(ctx, rdsNode.(*inventory.AWSRDSNode).Id, "test-remote")
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test-remote" already exists.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		err := ns.Remove(ctx, 2)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID 2 not found.`), err)
	})
}
