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

	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestNodes(t *testing.T) {
	ctx := logger.Set(context.Background(), t.Name())

	//nolint:unparam
	setup := func(t *testing.T) (ns *NodesService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		r := new(mockRegistry)
		r.Test(t)

		teardown = func(t *testing.T) {
			r.AssertExpectations(t)
			require.NoError(t, sqlDB.Close())
		}
		ns = NewNodesService(db)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		actualNodes, err := ns.List(ctx, nil)
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMMServerNodeType

		addNodeResponse, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test-bm"})
		require.NoError(t, err)
		expectedNode := &inventorypb.GenericNode{
			NodeId:   "/node_id/00000000-0000-4000-8000-000000000001",
			NodeName: "test-bm",
		}
		assert.Equal(t, expectedNode, addNodeResponse)

		getNodeResponse, err := ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: "/node_id/00000000-0000-4000-8000-000000000001"})
		require.NoError(t, err)
		assert.Equal(t, expectedNode, getNodeResponse)

		nodesResponse, err := ns.List(ctx, nil)
		require.NoError(t, err)
		require.Len(t, nodesResponse, 2)
		assert.Equal(t, expectedNode, nodesResponse[0])

		err = ns.Remove(ctx, "/node_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		getNodeResponse, err = ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: "/node_id/00000000-0000-4000-8000-000000000001"})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "/node_id/00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, getNodeResponse)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		getNodeResponse, err := ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: ""})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node ID.`), err)
		assert.Nil(t, getNodeResponse)
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: ""})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node name.`), err)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test", Address: "test"})
		require.NoError(t, err)

		_, err = ns.AddRemoteNode(ctx, &inventorypb.AddRemoteNodeRequest{NodeName: "test"})
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test" already exists.`), err)
	})

	t.Run("AddHostnameNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test1", Address: "test"})
		require.NoError(t, err)

		_, err = ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test2", Address: "test"})
		require.NoError(t, err)
	})

	t.Run("AddInstanceRegionNotUnique", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		_, err := ns.AddRemoteAmazonRDSNode(ctx, &inventorypb.AddRemoteAmazonRDSNodeRequest{NodeName: "test1", Instance: "test-instance", Region: "test-region"})
		require.NoError(t, err)

		_, err = ns.AddRemoteAmazonRDSNode(ctx, &inventorypb.AddRemoteAmazonRDSNodeRequest{NodeName: "test2", Instance: "test-instance", Region: "test-region"})
		expected := status.New(codes.AlreadyExists, `Node with instance "test-instance" and region "test-region" already exists.`)
		tests.AssertGRPCError(t, expected, err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ns, teardown := setup(t)
		defer teardown(t)

		err := ns.Remove(ctx, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})
}
