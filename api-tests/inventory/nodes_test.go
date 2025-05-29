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
	"context"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

func TestNodes(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		remoteNode := pmmapitests.AddNode(t, &nodes.AddNodeBody{
			Remote: &nodes.AddNodeParamsBodyRemote{
				NodeName: pmmapitests.TestString(t, "Test Remote Node for List"),
				Address:  "10.10.10.1",
			},
		})
		remoteNodeID := remoteNode.Remote.NodeID
		t.Cleanup(func() { pmmapitests.RemoveNodes(t, remoteNodeID) })

		genericNode := pmmapitests.AddNode(t, &nodes.AddNodeBody{
			Generic: &nodes.AddNodeParamsBodyGeneric{
				NodeName: pmmapitests.TestString(t, "Test Remote Node for List"),
				Address:  "10.10.10.2",
			},
		})
		genericNodeID := genericNode.Generic.NodeID
		require.NotEmpty(t, genericNodeID)
		t.Cleanup(func() { pmmapitests.RemoveNodes(t, genericNodeID) })

		res, err := client.Default.NodesService.ListNodes(nil)
		require.NoError(t, err)
		require.NotEmptyf(t, res.Payload.Generic, "There should be at least one node")
		require.Conditionf(t, func() (success bool) {
			for _, v := range res.Payload.Generic {
				if v.NodeID == genericNodeID {
					return true
				}
			}
			return false
		}, "There should be a generic node with id `%s`", genericNodeID)
		require.NotEmptyf(t, res.Payload.Remote, "There should be at least one node")
		require.Conditionf(t, func() (success bool) {
			for _, v := range res.Payload.Remote {
				if v.NodeID == remoteNodeID {
					return true
				}
			}
			return false
		}, "There should be a remote node with id `%s`", remoteNodeID)

		res, err = client.Default.NodesService.ListNodes(&nodes.ListNodesParams{
			NodeType: pointer.ToString(types.NodeTypeGenericNode),
			Context:  pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotEmptyf(t, res.Payload.Generic, "There should be at least one generic node")
		require.Conditionf(t, func() (success bool) {
			for _, v := range res.Payload.Generic {
				if v.NodeID == genericNodeID {
					return true
				}
			}
			return false
		}, "There should be a generic node with id `%s`", genericNodeID)
		require.Conditionf(t, func() (success bool) {
			for _, v := range res.Payload.Remote {
				if v.NodeID == remoteNodeID {
					return false
				}
			}
			return true
		}, "There shouldn't be a remote node with id `%s`", remoteNodeID)
	})
}

func TestGetNode(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "TestGenericNode")
		nodeID := pmmapitests.AddGenericNode(t, nodeName).NodeID
		require.NotEmpty(t, nodeID)
		defer pmmapitests.RemoveNodes(t, nodeID)

		expectedResponse := nodes.GetNodeOK{
			Payload: &nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					Address:      "10.10.10.10",
					CustomLabels: map[string]string{},
				},
			},
		}

		params := &nodes.GetNodeParams{
			NodeID:  nodeID,
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.GetNode(params)
		require.NoError(t, err)
		assert.Equal(t, expectedResponse.Payload, res.Payload)
	})

	t.Run("NotFound", func(t *testing.T) {
		params := &nodes.GetNodeParams{
			NodeID:  "pmm-not-found",
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.GetNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID \"pmm-not-found\" not found.")
		assert.Nil(t, res)
	})

	t.Run("EmptyNodeID", func(t *testing.T) {
		params := &nodes.GetNodeParams{
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.GetNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid GetNodeRequest.NodeId: value length must be at least 1 runes")
		assert.Nil(t, res)
	})
}

func TestGenericNode(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "Test Generic Node")
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  "10.10.10.10",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.AddNode(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Payload.Generic)
		nodeID := res.Payload.Generic.NodeID
		t.Cleanup(func() { pmmapitests.RemoveNodes(t, nodeID) })

		// Check that the node exists in DB.
		getNodeRes, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  nodeID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		expectedResponse := &nodes.GetNodeOK{
			Payload: &nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       res.Payload.Generic.NodeID,
					NodeName:     nodeName,
					Address:      "10.10.10.10",
					CustomLabels: map[string]string{},
				},
			},
		}
		require.Equal(t, expectedResponse, getNodeRes)

		// Check for duplicates.
		res, err = client.Default.NodesService.AddNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Node with name %q already exists.", nodeName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.Generic.NodeID)
		}
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{NodeName: ""},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.AddNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddGenericNodeParams.NodeName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.Generic.NodeID)
		}
	})
}

func TestContainerNode(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "Test Container Node")
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Container: &nodes.AddNodeParamsBodyContainer{
					NodeName:      nodeName,
					ContainerID:   "docker-id",
					ContainerName: "docker-name",
					MachineID:     "machine-id",
					Address:       "10.10.1.10",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.AddNode(params)
		require.NoError(t, err)
		require.NotNil(t, res.Payload.Container)
		nodeID := res.Payload.Container.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		// Check that the node exists in DB.
		getNodeRes, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  nodeID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		expectedResponse := &nodes.GetNodeOK{
			Payload: &nodes.GetNodeOKBody{
				Container: &nodes.GetNodeOKBodyContainer{
					NodeID:        res.Payload.Container.NodeID,
					NodeName:      nodeName,
					ContainerID:   "docker-id",
					ContainerName: "docker-name",
					MachineID:     "machine-id",
					Address:       "10.10.1.10",
					CustomLabels:  map[string]string{},
				},
			},
		}
		require.Equal(t, expectedResponse, getNodeRes)

		// Check for duplicates.
		res, err = client.Default.NodesService.AddNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Node with name %q already exists.", nodeName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.Container.NodeID)
		}
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Container: &nodes.AddNodeParamsBodyContainer{NodeName: ""},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.AddNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddContainerNodeParams.NodeName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.Container.NodeID)
		}
	})
}

func TestRemoteNode(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "Test Remote Node")
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Remote: &nodes.AddNodeParamsBodyRemote{
					NodeName:     nodeName,
					Az:           "eu",
					Region:       "us-west",
					Address:      "10.10.10.11",
					CustomLabels: map[string]string{"foo": "bar"},
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.AddNode(params)
		require.NoError(t, err)
		require.NotNil(t, res.Payload.Remote)
		nodeID := res.Payload.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		// Check node exists in DB.
		getNodeRes, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  nodeID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		expectedResponse := &nodes.GetNodeOK{
			Payload: &nodes.GetNodeOKBody{
				Remote: &nodes.GetNodeOKBodyRemote{
					NodeID:       res.Payload.Remote.NodeID,
					NodeName:     nodeName,
					Az:           "eu",
					Region:       "us-west",
					Address:      "10.10.10.11",
					CustomLabels: map[string]string{"foo": "bar"},
				},
			},
		}
		require.Equal(t, expectedResponse, getNodeRes)

		// Check duplicates.
		res, err = client.Default.NodesService.AddNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Node with name %q already exists.", nodeName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.Remote.NodeID)
		}
	})

	t.Run("AddNameEmpty", func(t *testing.T) {
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Remote: &nodes.AddNodeParamsBodyRemote{NodeName: ""},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.AddNode(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRemoteNodeParams.NodeName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.Remote.NodeID)
		}
	})
}

func TestRemoveNode(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "Generic Node for basic remove test")
		node := pmmapitests.AddNode(t,
			&nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  "10.10.10.1",
				},
			})
		nodeID := node.Generic.NodeID

		removeResp, err := client.Default.NodesService.RemoveNode(&nodes.RemoveNodeParams{
			NodeID:  nodeID,
			Context: context.Background(),
		})
		require.NoError(t, err)
		assert.NotNil(t, removeResp)
	})

	t.Run("With service", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "Generic Node for remove test")
		node := pmmapitests.AddNode(t,
			&nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  "10.10.10.1",
				},
			},
		)

		serviceName := pmmapitests.TestString(t, "MySQL Service for agent")
		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      node.Generic.NodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: serviceName,
			},
		})
		serviceID := service.Mysql.ServiceID

		removeResp, err := client.Default.NodesService.RemoveNode(&nodes.RemoveNodeParams{
			NodeID:  node.Generic.NodeID,
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `Node with ID %q has services.`, node.Generic.NodeID)
		assert.Nil(t, removeResp)

		// Check that node and service isn't removed.
		getServiceResp, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  node.Generic.NodeID,
			Context: pmmapitests.Context,
		})
		assert.NotNil(t, getServiceResp)
		require.NoError(t, err)

		listAgentsOK, err := client.Default.ServicesService.ListServices(&services.ListServicesParams{
			NodeID:  pointer.ToString(node.Generic.NodeID),
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, []*services.ListServicesOKBodyMysqlItems0{
			{
				NodeID:       node.Generic.NodeID,
				ServiceID:    serviceID,
				Address:      "localhost",
				Port:         3306,
				ServiceName:  serviceName,
				CustomLabels: map[string]string{},
			},
		}, listAgentsOK.Payload.Mysql)

		// Remove with force flag.
		params := &nodes.RemoveNodeParams{
			NodeID:  node.Generic.NodeID,
			Force:   pointer.ToBool(true),
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.RemoveNode(params)
		require.NoError(t, err)
		assert.NotNil(t, res)

		// Check that the node and agents are removed.
		getServiceResp, err = client.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  node.Generic.NodeID,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID %q not found.", node.Generic.NodeID)
		assert.Nil(t, getServiceResp)

		listAgentsOK, err = client.Default.ServicesService.ListServices(&services.ListServicesParams{
			NodeID:  pointer.ToString(node.Generic.NodeID),
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &services.ListServicesOKBody{
			Mysql:      make([]*services.ListServicesOKBodyMysqlItems0, 0),
			Mongodb:    make([]*services.ListServicesOKBodyMongodbItems0, 0),
			Postgresql: make([]*services.ListServicesOKBodyPostgresqlItems0, 0),
			Valkey:     make([]*services.ListServicesOKBodyValkeyItems0, 0),
			Proxysql:   make([]*services.ListServicesOKBodyProxysqlItems0, 0),
			Haproxy:    make([]*services.ListServicesOKBodyHaproxyItems0, 0),
			External:   make([]*services.ListServicesOKBodyExternalItems0, 0),
			Valkey:     make([]*services.ListServicesOKBodyValkeyItems0, 0),
		}, listAgentsOK.Payload)
	})

	t.Run("With pmm-agent", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "Generic Node for remove test")
		node := pmmapitests.AddNode(t,
			&nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  "10.10.10.1",
				},
			},
		)

		_ = pmmapitests.AddPMMAgent(t, node.Generic.NodeID)

		removeResp, err := client.Default.NodesService.RemoveNode(&nodes.RemoveNodeParams{
			NodeID:  node.Generic.NodeID,
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `Node with ID %q has pmm-agent.`, node.Generic.NodeID)
		assert.Nil(t, removeResp)

		// Remove with force flag.
		params := &nodes.RemoveNodeParams{
			NodeID:  node.Generic.NodeID,
			Force:   pointer.ToBool(true),
			Context: pmmapitests.Context,
		}
		res, err := client.Default.NodesService.RemoveNode(params)
		require.NoError(t, err)
		assert.NotNil(t, res)

		// Check that the node and agents are removed.
		getServiceResp, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  node.Generic.NodeID,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID %q not found.", node.Generic.NodeID)
		assert.Nil(t, getServiceResp)

		listAgentsOK, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			NodeID:  pointer.ToString(node.Generic.NodeID),
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID %q not found.", node.Generic.NodeID)
		assert.Nil(t, listAgentsOK)
	})

	t.Run("Not-exist node", func(t *testing.T) {
		nodeID := "not-exist-node-id"
		removeResp, err := client.Default.NodesService.RemoveNode(&nodes.RemoveNodeParams{
			NodeID:  nodeID,
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Node with ID %q not found.`, nodeID)
		assert.Nil(t, removeResp)
	})

	t.Run("Empty params", func(t *testing.T) {
		removeResp, err := client.Default.NodesService.RemoveNode(&nodes.RemoveNodeParams{
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RemoveNodeRequest.NodeId: value length must be at least 1 runes")
		assert.Nil(t, removeResp)
	})

	t.Run("PMM Server", func(t *testing.T) {
		removeResp, err := client.Default.NodesService.RemoveNode(&nodes.RemoveNodeParams{
			NodeID:  "pmm-server",
			Force:   pointer.ToBool(true),
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "PMM Server node can't be removed.")
		assert.Nil(t, removeResp)
	})
}
