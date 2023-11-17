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

package inventory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventorypb "github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestNodes(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		actualNodes, err := ns.List(ctx, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMM Server Node

		addNodeResponse, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test-bm"})
		require.NoError(t, err)
		expectedNode := &inventorypb.GenericNode{
			NodeId:   "/node_id/00000000-0000-4000-8000-000000000005",
			NodeName: "test-bm",
		}
		assert.Equal(t, expectedNode, addNodeResponse)

		getNodeResponse, err := ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: "/node_id/00000000-0000-4000-8000-000000000005"})
		require.NoError(t, err)
		assert.Equal(t, expectedNode, getNodeResponse)

		nodesResponse, err := ns.List(ctx, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, nodesResponse, 2)
		assert.Equal(t, expectedNode, nodesResponse[0])

		err = ns.Remove(ctx, "/node_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		getNodeResponse, err = ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: "/node_id/00000000-0000-4000-8000-000000000005"})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "/node_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, getNodeResponse)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		getNodeResponse, err := ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: ""})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node ID.`), err)
		assert.Nil(t, getNodeResponse)
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: ""})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node name.`), err)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test", Address: "test"})
		require.NoError(t, err)

		_, err = ns.AddRemoteNode(ctx, &inventorypb.AddRemoteNodeRequest{NodeName: "test"})
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test" already exists.`), err)
	})

	t.Run("AddHostnameNotUnique", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test1", Address: "test"})
		require.NoError(t, err)

		_, err = ns.AddGenericNode(ctx, &inventorypb.AddGenericNodeRequest{NodeName: "test2", Address: "test"})
		require.NoError(t, err)
	})

	t.Run("AddRemoteRDSNodeNotUnique", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddRemoteRDSNode(ctx, &inventorypb.AddRemoteRDSNodeRequest{NodeName: "test1", Region: "test-region", Address: "test"})
		require.NoError(t, err)

		_, err = ns.AddRemoteRDSNode(ctx, &inventorypb.AddRemoteRDSNodeRequest{NodeName: "test2", Region: "test-region", Address: "test"})
		expected := status.New(codes.AlreadyExists, `Node with instance "test" and region "test-region" already exists.`)
		tests.AssertGRPCError(t, expected, err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		defer teardown(t)

		err := ns.Remove(ctx, "no-such-id", false)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})
}

func TestAddNode(t *testing.T) {
	t.Run("BasicGeneric", func(t *testing.T) {
		const nodeID = "/node_id/00000000-0000-4000-8000-000000000005"
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		actualNodes, err := ns.List(ctx, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMM Server Node

		addNodeResponse, err := ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Generic{
				Generic: &inventorypb.AddGenericNodeRequest{NodeName: "test-bm", Region: "test-region", Address: "test"},
			},
		})
		require.NoError(t, err)

		expectedNode := &inventorypb.GenericNode{
			NodeId:   nodeID,
			NodeName: "test-bm",
			Region:   "test-region",
			Address:  "test",
		}
		assert.Equal(t, expectedNode, addNodeResponse.GetGeneric())

		getNodeResponse, err := ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: nodeID})
		require.NoError(t, err)
		assert.Equal(t, expectedNode, getNodeResponse)

		nodesResponse, err := ns.List(ctx, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, nodesResponse, 2)
		assert.Equal(t, expectedNode, nodesResponse[0])

		err = ns.Remove(ctx, nodeID, false)
		require.NoError(t, err)
		getNodeResponse, err = ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: nodeID})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Node with ID %q not found.", nodeID)), err)
		assert.Nil(t, getNodeResponse)
	})

	t.Run("AddAllNodeTypes", func(t *testing.T) {
		const (
			nodeID1 = "/node_id/00000000-0000-4000-8000-000000000005"
			nodeID2 = "/node_id/00000000-0000-4000-8000-000000000006"
			nodeID3 = "/node_id/00000000-0000-4000-8000-000000000007"
			nodeID4 = "/node_id/00000000-0000-4000-8000-000000000008"
			nodeID5 = "/node_id/00000000-0000-4000-8000-000000000009"
		)
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		actualNodes, err := ns.List(ctx, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, actualNodes, 1) // PMM Server Node

		expectedNode1 := &inventorypb.GenericNode{
			NodeId:   nodeID1,
			NodeName: "test-name1",
			Region:   "test-region",
			Address:  "test1",
		}
		addNodeResponse, err := ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Generic{
				Generic: &inventorypb.AddGenericNodeRequest{
					NodeName: "test-name1",
					Region:   "test-region",
					Address:  "test1",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, expectedNode1, addNodeResponse.GetGeneric())

		expectedNode2 := &inventorypb.ContainerNode{
			NodeId:   nodeID2,
			NodeName: "test-name2",
			Region:   "test-region",
			Address:  "test2",
		}
		addNodeResponse, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Container{
				Container: &inventorypb.AddContainerNodeRequest{
					NodeName: "test-name2",
					Region:   "test-region",
					Address:  "test2",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, expectedNode2, addNodeResponse.GetContainer())

		expectedNode3 := &inventorypb.RemoteNode{
			NodeId:   nodeID3,
			NodeName: "test-name3",
			Region:   "test-region",
			Address:  "test3",
			CustomLabels: map[string]string{
				"testkey": "test-value",
				"region":  "test-region",
			},
		}
		addNodeResponse, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Remote{
				Remote: &inventorypb.AddRemoteNodeRequest{
					NodeName: "test-name3",
					Region:   "test-region",
					Address:  "test3",
					CustomLabels: map[string]string{
						"testkey": "test-value",
						"region":  "test-region",
					},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, expectedNode3, addNodeResponse.GetRemote())

		expectedNode4 := &inventorypb.RemoteAzureDatabaseNode{
			NodeId:   nodeID4,
			NodeName: "test-name4",
			Region:   "test-region",
			Az:       "test-region-az",
			Address:  "test4",
		}
		addNodeResponse, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_RemoteAzure{
				RemoteAzure: &inventorypb.AddRemoteAzureDatabaseNodeRequest{
					NodeName: "test-name4",
					Region:   "test-region",
					Az:       "test-region-az",
					Address:  "test4",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, expectedNode4, addNodeResponse.GetRemoteAzureDatabase())

		expectedNode5 := &inventorypb.RemoteRDSNode{
			NodeId:   nodeID5,
			NodeName: "test-name5",
			Region:   "test-region",
			Az:       "test-region-az",
			Address:  "test5",
		}
		addNodeResponse, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_RemoteRds{
				RemoteRds: &inventorypb.AddRemoteRDSNodeRequest{
					NodeName: "test-name5",
					Region:   "test-region",
					Az:       "test-region-az",
					Address:  "test5",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, expectedNode5, addNodeResponse.GetRemoteRds())

		getNodeResponse, err := ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: nodeID1})
		require.NoError(t, err)
		assert.Equal(t, expectedNode1, getNodeResponse)

		nodesResponse, err := ns.List(ctx, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, nodesResponse, 6)
		assert.Equal(t, expectedNode1, nodesResponse[0])

		err = ns.Remove(ctx, nodeID1, false)
		require.NoError(t, err)
		getNodeResponse, err = ns.Get(ctx, &inventorypb.GetNodeRequest{NodeId: nodeID1})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Node with ID %q not found.", nodeID1)), err)
		assert.Nil(t, getNodeResponse)
	})

	t.Run("AddRemoteRDSNodeNonUnique", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_RemoteRds{
				RemoteRds: &inventorypb.AddRemoteRDSNodeRequest{NodeName: "test1", Region: "test-region", Address: "test"},
			},
		})
		require.NoError(t, err)

		_, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_RemoteRds{
				RemoteRds: &inventorypb.AddRemoteRDSNodeRequest{NodeName: "test2", Region: "test-region", Address: "test"},
			},
		})
		expected := status.New(codes.AlreadyExists, `Node with instance "test" and region "test-region" already exists.`)
		tests.AssertGRPCError(t, expected, err)
	})

	t.Run("AddHostnameNotUnique", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Generic{
				Generic: &inventorypb.AddGenericNodeRequest{NodeName: "test1", Address: "test"},
			},
		})
		require.NoError(t, err)

		_, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Generic{
				Generic: &inventorypb.AddGenericNodeRequest{NodeName: "test2", Address: "test"},
			},
		})
		require.NoError(t, err)
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddNode(ctx,
			&inventorypb.AddNodeRequest{
				Request: &inventorypb.AddNodeRequest_Generic{
					Generic: &inventorypb.AddGenericNodeRequest{NodeName: ""},
				},
			},
		)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node name.`), err)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		_, _, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Generic{
				Generic: &inventorypb.AddGenericNodeRequest{NodeName: "test", Address: "test"},
			},
		})
		require.NoError(t, err)

		_, err = ns.AddNode(ctx, &inventorypb.AddNodeRequest{
			Request: &inventorypb.AddNodeRequest_Remote{
				Remote: &inventorypb.AddRemoteNodeRequest{NodeName: "test"},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test" already exists.`), err)
	})
}
