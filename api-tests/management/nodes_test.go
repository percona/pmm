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

	"github.com/AlekSi/pointer"
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
	t.Run("Generic Node", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
			})
			defer pmmapitests.UnregisterNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)
			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					CustomLabels: map[string]string{},
				},
			})

			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			assertNodeExporterCreated(t, pmmAgentID)
		})

		t.Run("Reregister with same node name (no re-register - should fail)", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-all")
			nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  "node-address-1",
				Region:   "region-1",
			})
			defer pmmapitests.UnregisterNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

			body := mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  "node-address-1",
				Region:   "region-1",
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			_, err := client.Default.ManagementService.RegisterNode(&params)
			wantErr := fmt.Sprintf("Node with name %q already exists.", nodeName)
			pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, wantErr)
		})

		t.Run("Reregister with same node name (re-register)", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-all")
			nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  "node-address-2",
				Region:   "region-2",
			})
			assert.NotEmpty(t, nodeID)
			assert.NotEmpty(t, pmmAgentID)

			body := mservice.RegisterNodeBody{
				NodeName:   nodeName,
				NodeType:   pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:    "node-address-2",
				Region:     "region-3",
				Reregister: true,
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			node, err := client.Default.ManagementService.RegisterNode(&params)
			assert.NoError(t, err)

			defer pmmapitests.UnregisterNodes(t, node.Payload.GenericNode.NodeID)
			defer RemovePMMAgentWithSubAgents(t, node.Payload.PMMAgent.AgentID)

			assertNodeExporterCreated(t, node.Payload.PMMAgent.AgentID)
		})

		t.Run("Reregister with different node name (no re-register - should fail)", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-all")
			nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  "node-address-3",
				Region:   "region-3",
			})
			defer pmmapitests.UnregisterNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

			body := mservice.RegisterNodeBody{
				NodeName: nodeName + "_new",
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  "node-address-3",
				Region:   "region-3",
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
			nodeName := pmmapitests.TestString(t, "node-all")
			nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:  "node-address-4",
				Region:   "region-4",
			})
			assert.NotEmpty(t, nodeID)
			assert.NotEmpty(t, pmmAgentID)

			body := mservice.RegisterNodeBody{
				NodeName:   nodeName + "_new",
				NodeType:   pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				Address:    "node-address-4",
				Region:     "region-4",
				Reregister: true,
			}
			params := mservice.RegisterNodeParams{
				Context: pmmapitests.Context,
				Body:    body,
			}
			node, err := client.Default.ManagementService.RegisterNode(&params)
			assert.NoError(t, err)

			defer pmmapitests.UnregisterNodes(t, node.Payload.GenericNode.NodeID)
			_, ok := assertNodeExporterCreated(t, node.Payload.PMMAgent.AgentID)
			if ok {
				defer RemovePMMAgentWithSubAgents(t, node.Payload.PMMAgent.AgentID)
			}
		})

		t.Run("With all fields", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-name")
			machineID := pmmapitests.TestString(t, "machine-id")
			nodeModel := pmmapitests.TestString(t, "node-model")
			body := mservice.RegisterNodeBody{
				NodeName:          nodeName,
				NodeType:          pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				MachineID:         machineID,
				NodeModel:         nodeModel,
				Az:                "eu",
				Region:            "us-west",
				Address:           "10.10.10.10",
				Distro:            "Linux",
				CustomLabels:      map[string]string{"foo": "bar"},
				DisableCollectors: []string{"diskstats", "filesystem", "standard.process"},
			}
			nodeID, pmmAgentID := RegisterGenericNode(t, body)
			defer pmmapitests.UnregisterNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					MachineID:    machineID,
					NodeModel:    nodeModel,
					Az:           "eu",
					Region:       "us-west",
					Address:      "10.10.10.10",
					Distro:       "Linux",
					CustomLabels: map[string]string{"foo": "bar"},
				},
			})

			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
				PMMAgentID: pointer.ToString(pmmAgentID),
				Context:    pmmapitests.Context,
			})
			assert.NoError(t, err)
			require.Len(t, listAgentsOK.Payload.NodeExporter, 1)
			nodeExporterAgentID := listAgentsOK.Payload.NodeExporter[0].AgentID
			ok := assert.Equal(t, agents.ListAgentsOKBodyNodeExporterItems0{
				PMMAgentID:         pmmAgentID,
				AgentID:            nodeExporterAgentID,
				DisabledCollectors: []string{"diskstats", "filesystem", "standard.process"},
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			}, *listAgentsOK.Payload.NodeExporter[0])

			if ok {
				defer pmmapitests.RemoveAgents(t, nodeExporterAgentID)
			}
		})

		t.Run("Re-register", func(t *testing.T) {
			t.Skip("Re-register logic is not defined yet. https://jira.percona.com/browse/PMM-3717")

			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
			})
			defer pmmapitests.RemoveNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

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
			newNodeID, newPMMAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
				NodeName:     nodeName,
				NodeType:     pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
				MachineID:    machineID,
				NodeModel:    nodeModel,
				Az:           "eu",
				Region:       "us-west",
				Address:      "10.10.10.10",
				Distro:       "Linux",
				CustomLabels: map[string]string{"foo": "bar"},
			})
			if !assert.Equal(t, nodeID, newNodeID) {
				defer pmmapitests.RemoveNodes(t, newNodeID)
			}
			if !assert.Equal(t, pmmAgentID, newPMMAgentID) {
				defer pmmapitests.RemoveAgents(t, newPMMAgentID)
			}

			// Check Node fields is updated
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Generic: &nodes.GetNodeOKBodyGeneric{
					NodeID:       nodeID,
					NodeName:     nodeName,
					MachineID:    machineID,
					NodeModel:    nodeModel,
					Az:           "eu",
					Region:       "us-west",
					Address:      "10.10.10.10",
					Distro:       "Linux",
					CustomLabels: map[string]string{"foo": "bar"},
				},
			})
		})
	})

	t.Run("Container Node", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, pmmAgentID := registerContainerNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
			})
			defer pmmapitests.UnregisterNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

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
				defer pmmapitests.RemoveAgents(t, nodeExporterAgentID)
			}
		})

		t.Run("With all fields", func(t *testing.T) {
			nodeName := pmmapitests.TestString(t, "node-name")
			nodeModel := pmmapitests.TestString(t, "node-model")
			containerID := pmmapitests.TestString(t, "container-id")
			containerName := pmmapitests.TestString(t, "container-name")
			body := mservice.RegisterNodeBody{
				NodeName:      nodeName,
				NodeType:      pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
				NodeModel:     nodeModel,
				ContainerID:   containerID,
				ContainerName: containerName,
				Az:            "eu",
				Region:        "us-west",
				Address:       "10.10.10.10",
				CustomLabels:  map[string]string{"foo": "bar"},
			}
			nodeID, pmmAgentID := registerContainerNode(t, body)
			defer pmmapitests.UnregisterNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

			// Check Node is created
			assertNodeCreated(t, nodeID, nodes.GetNodeOKBody{
				Container: &nodes.GetNodeOKBodyContainer{
					NodeID:        nodeID,
					NodeName:      nodeName,
					NodeModel:     nodeModel,
					ContainerID:   containerID,
					ContainerName: containerName,
					Az:            "eu",
					Region:        "us-west",
					Address:       "10.10.10.10",
					CustomLabels:  map[string]string{"foo": "bar"},
				},
			})

			// Check PMM Agent is created
			assertPMMAgentCreated(t, nodeID, pmmAgentID)

			// Check Node Exporter is created
			nodeExporterAgentID, ok := assertNodeExporterCreated(t, pmmAgentID)
			if ok {
				defer pmmapitests.RemoveAgents(t, nodeExporterAgentID)
			}
		})

		t.Run("Re-register", func(t *testing.T) {
			t.Skip("Re-register logic is not defined yet. https://jira.percona.com/browse/PMM-3717")

			nodeName := pmmapitests.TestString(t, "node-name")
			nodeID, pmmAgentID := registerContainerNode(t, mservice.RegisterNodeBody{
				NodeName: nodeName,
				NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
			})
			defer pmmapitests.RemoveNodes(t, nodeID)
			defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

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
			newNodeID, newPMMAgentID := registerContainerNode(t, mservice.RegisterNodeBody{
				NodeName:      nodeName,
				NodeType:      pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE),
				ContainerID:   containerID,
				ContainerName: containerName,
				NodeModel:     nodeModel,
				Az:            "eu",
				Region:        "us-west",
				Address:       "10.10.10.10",
				CustomLabels:  map[string]string{"foo": "bar"},
			})
			if !assert.Equal(t, nodeID, newNodeID) {
				defer pmmapitests.RemoveNodes(t, newNodeID)
			}
			if !assert.Equal(t, pmmAgentID, newPMMAgentID) {
				defer pmmapitests.RemoveAgents(t, newPMMAgentID)
			}

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
					Address:       "10.10.10.10",
					CustomLabels:  map[string]string{"foo": "bar"},
				},
			})
		})
	})

	t.Run("Empty node name", func(t *testing.T) {
		params := mservice.RegisterNodeParams{
			Context: pmmapitests.Context,
			Body:    mservice.RegisterNodeBody{},
		}
		registerOK, err := client.Default.ManagementService.RegisterNode(&params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RegisterNodeRequest.NodeName: value length must be at least 1 runes")
		require.Nil(t, registerOK)
	})

	t.Run("Unsupported node type", func(t *testing.T) {
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
