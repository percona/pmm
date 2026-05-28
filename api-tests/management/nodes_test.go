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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func TestNodeRegister(t *testing.T) {
	t.Parallel()

	t.Run("Generic Node", func(t *testing.T) {
		t.Parallel()

		t.Run("Basic", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-name")
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body: mservice.RegisterNodeBody{
					NodeName: nodeName,
					NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				},
			}
			registerOK, err := client.Default.ManagementService.RegisterNode(&params)
			require.NoError(t, err)
			require.NotNil(t, registerOK)
			require.NotNil(t, registerOK.Payload.PMMAgent)
			require.NotNil(t, registerOK.Payload.PMMAgent.AgentID)
			require.NotNil(t, registerOK.Payload.GenericNode)
			require.NotNil(t, registerOK.Payload.GenericNode.NodeID)
			nodeID := registerOK.Payload.GenericNode.NodeID
			t.Cleanup(func() {
				pmmapitests.UnregisterNodes(t, nodeID)
			})

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					CustomLabels: map[string]string{},
				},
			})

			pmmAgentID := registerOK.Payload.PMMAgent.AgentID
			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			assertNodeExporterCreated(t, pmmAgentID)
		})

		t.Run("Reregister with same node name (no re-register - should fail)", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-all")
			nodeAddress := pmmapitests.TestString(t, "node-address-1")
			nodeRegion := pmmapitests.TestString(t, "region-1")
			RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  nodeAddress,
				Region:   nodeRegion,
			})

			body := mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  nodeAddress,
				Region:   nodeRegion,
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			_, err := client.Default.ManagementService.RegisterNode(&params)
			// Historically, this test asserted on the full error string
			// "Node with name %s already exists.". However, the JSON client serializes gRPC errors
			// and may add quoting/escaping around the node name, making exact string comparison brittle.
			// We therefore assert here on the HTTP status and gRPC code, and only require that the
			// message contains the stable prefix "Node with name". The checks below additionally verify
			// that the message includes the specific node name and the phrase "already exists", so the
			// test still covers the intended behavior even if the exact formatting changes.
			pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Node with name")
			require.Error(t, err)
			require.Contains(t, err.Error(), nodeName, "Error message should contain the node name")
			require.Contains(t, err.Error(), "already exists", "Error message should indicate node already exists")
		})

		t.Run("Reregister with same node name (re-register)", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-all")
			nodeAddress := pmmapitests.TestString(t, "node-address-2")
			RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  nodeAddress,
				Region:   pmmapitests.TestString(t, "region-2"),
			})

			body := mservice.RegisterNodeBody{
				NodeName:   nodeName,
				NodeType:   new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:    nodeAddress,
				Region:     pmmapitests.TestString(t, "region-3"),
				Reregister: true,
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			node, err := client.Default.ManagementService.RegisterNode(&params)
			require.NoError(t, err)
			nodeID := node.Payload.GenericNode.NodeID
			t.Cleanup(func() {
				pmmapitests.UnregisterNodes(t, nodeID)
			})

			assertNodeExporterCreated(t, node.Payload.PMMAgent.AgentID)
		})

		t.Run("Reregister with different node name (no re-register - should fail)", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-all")
			nodeAddress := pmmapitests.TestString(t, "node-address-2")
			nodeRegion := pmmapitests.TestString(t, "region-3")
			RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  nodeAddress,
				Region:   nodeRegion,
			})

			body := mservice.RegisterNodeBody{
				NodeName: nodeName + "_new",
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  nodeAddress,
				Region:   nodeRegion,
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			_, err := client.Default.ManagementService.RegisterNode(&params)
			wantErr := fmt.Sprintf("Node with address %q and region %q already exists.", body.Address, body.Region)
			pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, wantErr)
		})

		t.Run("Reregister with different node name (re-register)", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-all")
			nodeAddress := pmmapitests.TestString(t, "node-address-4")
			nodeRegion := pmmapitests.TestString(t, "region-4")
			RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  nodeAddress,
				Region:   nodeRegion,
			})

			body := mservice.RegisterNodeBody{
				NodeName:   nodeName + "_new",
				NodeType:   new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:    nodeAddress,
				Region:     nodeRegion,
				Reregister: true,
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			node, err := client.Default.ManagementService.RegisterNode(&params)
			require.NoError(t, err)

			nodeID := node.Payload.GenericNode.NodeID
			t.Cleanup(func() {
				pmmapitests.UnregisterNodes(t, nodeID)
			})

			assertNodeExporterCreated(t, node.Payload.PMMAgent.AgentID)
		})

		t.Run("With all fields", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-name-1")
			machineID := pmmapitests.TestString(t, "machine-id-1")
			nodeModel := pmmapitests.TestString(t, "node-model-1")
			nodeAddress := pmmapitests.TestString(t, "node-address-3")
			body := mservice.RegisterNodeBody{
				NodeName:          nodeName,
				NodeType:          new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				MachineID:         machineID,
				NodeModel:         nodeModel,
				Az:                "eu",
				Region:            "us-west",
				Address:           nodeAddress,
				Distro:            "Linux",
				CustomLabels:      map[string]string{"foo": "bar"},
				DisableCollectors: []string{"diskstats", "filesystem", "standard.process"},
			}
			nodeID, pmmAgentID := RegisterNode(t, body)

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					MachineID:    machineID,
					NodeModel:    nodeModel,
					Az:           "eu",
					Region:       "us-west",
					Address:      nodeAddress,
					Distro:       "Linux",
					CustomLabels: map[string]string{"foo": "bar"},
				},
			})

			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
				PMMAgentID: new(pmmAgentID),
				Context:    pmmapitests.Context,
			})
			require.NoError(t, err)
			require.Len(t, listAgentsOK.Payload.NodeExporter, 1)
			nodeExporterAgentID := listAgentsOK.Payload.NodeExporter[0].AgentID
			assert.Equal(t, agents.ListAgentsOKBodyNodeExporterItems0{
				PMMAgentID:         pmmAgentID,
				AgentID:            nodeExporterAgentID,
				DisabledCollectors: []string{"diskstats", "filesystem", "standard.process"},
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
			}, *listAgentsOK.Payload.NodeExporter[0])
		})

		t.Run("Re-register", func(t *testing.T) {
			t.Skip("Re-register logic is not defined yet. https://jira.percona.com/browse/PMM-3717")

			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, _ := RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
			})

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:   nodeID,
					NodeName: nodeName,
				},
			})

			// Re-register node
			machineID := pmmapitests.TestString(t, "machine-id")
			nodeModel := pmmapitests.TestString(t, "node-model")
			nodeAddress := pmmapitests.TestString(t, "10.10.10.10")
			RegisterNode(t, mservice.RegisterNodeBody{
				NodeName:     nodeName,
				NodeType:     new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				MachineID:    machineID,
				NodeModel:    nodeModel,
				Az:           "eu",
				Region:       "us-west",
				Address:      nodeAddress,
				Distro:       "Linux",
				CustomLabels: map[string]string{"foo": "bar"},
			})

			// Check Node fields is updated
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					MachineID:    machineID,
					NodeModel:    nodeModel,
					Az:           "eu",
					Region:       "us-west",
					Address:      nodeAddress,
					Distro:       "Linux",
					CustomLabels: map[string]string{"foo": "bar"},
				},
			})
		})
	})

	t.Run("Container Node", func(t *testing.T) {
		t.Parallel()

		t.Run("Basic", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
			})
			assert.NotEmpty(t, nodeID)
			assert.NotEmpty(t, pmmAgentID)
			t.Cleanup(func() { pmmapitests.UnregisterNodes(t, nodeID) })
			t.Cleanup(func() { RemovePMMAgentWithSubAgents(t, pmmAgentID) })

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Container: &nodes.GetNodeOKBodyContainer{
					NodeID:       nodeID,
					NodeName:     nodeName,
					CustomLabels: map[string]string{},
				},
			})

			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			nodeExporterAgentID, ok := assertNodeExporterCreated(t, pmmAgentID)
			if ok {
				t.Cleanup(func() { pmmapitests.RemoveAgents(t, nodeExporterAgentID) })
			}
		})

		t.Run("With all fields", func(t *testing.T) {
			t.Parallel()

			nodeName := pmmapitests.TestString(t, "node-name")
			nodeModel := pmmapitests.TestString(t, "node-model")
			containerID := pmmapitests.TestString(t, "container-id")
			containerName := pmmapitests.TestString(t, "container-name")
			nodeRegion := pmmapitests.TestString(t, "us-west")
			nodeAddress := pmmapitests.TestString(t, "10.10.10.10")
			body := mservice.RegisterNodeBody{
				NodeName:      nodeName,
				NodeType:      new(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
				NodeModel:     nodeModel,
				ContainerID:   containerID,
				ContainerName: containerName,
				Az:            "eu",
				Region:        nodeRegion,
				Address:       nodeAddress,
				CustomLabels:  map[string]string{"foo": "bar"},
			}
			nodeID, pmmAgentID := RegisterNode(t, body)

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Container: &nodes.GetNodeOKBodyContainer{
					NodeID:        nodeID,
					NodeName:      nodeName,
					NodeModel:     nodeModel,
					ContainerID:   containerID,
					ContainerName: containerName,
					Az:            "eu",
					Region:        nodeRegion,
					Address:       nodeAddress,
					CustomLabels:  map[string]string{"foo": "bar"},
				},
			})

			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			assertNodeExporterCreated(t, pmmAgentID)
		})

		t.Run("Re-register", func(t *testing.T) {
			t.Skip("Re-register logic is not defined yet. https://jira.percona.com/browse/PMM-3717")

			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, _ := RegisterNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
			})

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:   nodeID,
					NodeName: nodeName,
				},
			})

			// Re-register node
			nodeModel := pmmapitests.TestString(t, "node-model")
			containerID := pmmapitests.TestString(t, "container-id")
			containerName := pmmapitests.TestString(t, "container-name")
			nodeAddress := pmmapitests.TestString(t, "10.10.10.10")
			RegisterNode(t, mservice.RegisterNodeBody{
				NodeName:      nodeName,
				NodeType:      new(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
				ContainerID:   containerID,
				ContainerName: containerName,
				NodeModel:     nodeModel,
				Az:            "eu",
				Region:        "us-west",
				Address:       nodeAddress,
				CustomLabels:  map[string]string{"foo": "bar"},
			})

			// Check Node fields is updated
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Container: &nodes.GetNodeOKBodyContainer{
					NodeID:        nodeID,
					NodeName:      nodeName,
					ContainerID:   containerID,
					ContainerName: containerName,
					NodeModel:     nodeModel,
					Az:            "eu",
					Region:        "us-west",
					Address:       nodeAddress,
					CustomLabels:  map[string]string{"foo": "bar"},
				},
			})
		})
	})

	t.Run("Empty node name", func(t *testing.T) {
		t.Parallel()

		params := mservice.RegisterNodeParams{
			Context: pmmapitests.Context,
			Body:    mservice.RegisterNodeBody{},
		}
		registerOK, err := client.Default.ManagementService.RegisterNode(&params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RegisterNodeRequest.NodeName: value length must be at least 1 runes")
		require.Nil(t, registerOK)
	})

	t.Run("Unsupported node type", func(t *testing.T) {
		t.Parallel()

		params := mservice.RegisterNodeParams{
			Context: pmmapitests.Context,
			Body: mservice.RegisterNodeBody{
				NodeName: pmmapitests.TestString(t, "node-name"),
			},
		}
		registerOK, err := client.Default.ManagementService.RegisterNode(&params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `Unsupported Node type "NODE_TYPE_UNSPECIFIED".`)
		require.Nil(t, registerOK)
	})
}
