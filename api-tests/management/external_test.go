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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func TestAddExternal(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "genericNode-for-basic-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					RunsOnNodeID:        nodeID,
					ServiceName:         serviceName,
					ListenPort:          9104,
					NodeID:              nodeID,
					Group:               "", // empty group - pmm-admin does not support groups.
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addExternalOK)
		require.NotNil(t, addExternalOK.Payload.External.Service)
		serviceID := addExternalOK.Payload.External.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			External: &services.GetServiceOKBodyExternal{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Group:        "external",
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that external exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: pointer.ToString(serviceID),
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyExternalExporterItems0{
			{
				AgentID:      listAgents.Payload.ExternalExporter[0].AgentID,
				ServiceID:    serviceID,
				ListenPort:   9104,
				RunsOnNodeID: nodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				CustomLabels: map[string]string{},
			},
		}, listAgents.Payload.ExternalExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With labels", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-all-fields-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-for-all-fields-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					RunsOnNodeID:        nodeID,
					ServiceName:         serviceName,
					Username:            "username",
					Password:            "password",
					Scheme:              "https",
					MetricsPath:         "/metrics-path",
					ListenPort:          9250,
					NodeID:              nodeID,
					Environment:         "some-environment",
					Cluster:             "cluster-name",
					ReplicationSet:      "replication-set",
					CustomLabels:        map[string]string{"bar": "foo"},
					Group:               "redis",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addExternalOK)
		require.NotNil(t, addExternalOK.Payload.External.Service)
		serviceID := addExternalOK.Payload.External.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			External: &services.GetServiceOKBodyExternal{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},

				Group: "redis",
			},
		}, *serviceOK.Payload)
	})

	t.Run("OnRemoteNode", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "genericNode-for-basic-name")

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					AddNode: &mservice.AddServiceParamsBodyExternalAddNode{
						NodeType:     pointer.ToString(mservice.AddServiceParamsBodyExternalAddNodeNodeTypeNODETYPEREMOTENODE),
						NodeName:     nodeName,
						MachineID:    "/machine-id/",
						Distro:       "linux",
						Region:       "us-west2",
						CustomLabels: map[string]string{"foo": "bar-for-node"},
					},
					Address:             "localhost",
					ServiceName:         serviceName,
					ListenPort:          9104,
					Group:               "", // empty group - pmm-admin does not support group.
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addExternalOK)
		require.NotNil(t, addExternalOK.Payload.External.Service)
		nodeID := addExternalOK.Payload.External.Service.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)
		serviceID := addExternalOK.Payload.External.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that node is created and its fields.
		node, err := inventoryClient.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  nodeID,
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, node)
		assert.Equal(t, nodes.GetNodeOKBody{
			Remote: &nodes.GetNodeOKBodyRemote{
				NodeID:       nodeID,
				NodeName:     nodeName,
				Address:      "localhost",
				Region:       "us-west2",
				CustomLabels: map[string]string{"foo": "bar-for-node"},
			},
		}, *node.Payload)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			External: &services.GetServiceOKBodyExternal{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Group:        "external",
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that external exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: pointer.ToString(serviceID),
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyExternalExporterItems0{
			{
				AgentID:      listAgents.Payload.ExternalExporter[0].AgentID,
				ServiceID:    serviceID,
				ListenPort:   9104,
				RunsOnNodeID: nodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				CustomLabels: map[string]string{},
			},
		}, listAgents.Payload.ExternalExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With the same name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-the-same-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					NodeID:              nodeID,
					RunsOnNodeID:        nodeID,
					ServiceName:         serviceName,
					ListenPort:          9250,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addExternalOK)
		require.NotNil(t, addExternalOK.Payload.External.Service)
		serviceID := addExternalOK.Payload.External.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		params = &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					NodeID:              nodeID,
					RunsOnNodeID:        nodeID,
					ServiceName:         serviceName,
					ListenPort:          9260,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err = client.Default.ManagementService.AddService(params)
		require.Nil(t, addExternalOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("Empty Service Name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					NodeID:              nodeID,
					RunsOnNodeID:        nodeID,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddExternalRequest.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addExternalOK)
	})

	t.Run("Empty ListenPort", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					NodeID:              nodeID,
					ServiceName:         serviceName,
					RunsOnNodeID:        nodeID,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddExternalRequest.ListenPort: value must be inside range (0, 65536)")
		assert.Nil(t, addExternalOK)
	})

	t.Run("Empty Node ID", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					RunsOnNodeID:        nodeID,
					ServiceName:         serviceName,
					ListenPort:          12345,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "runs_on_node_id and node_id should be specified together.")
		assert.Nil(t, addExternalOK)
	})

	t.Run("Empty Runs On Node ID", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					NodeID:              nodeID,
					ServiceName:         serviceName,
					ListenPort:          12345,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "runs_on_node_id and node_id should be specified together.")
		assert.Nil(t, addExternalOK)
	})

	t.Run("Empty Address for Add Node", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					AddNode: &mservice.AddServiceParamsBodyExternalAddNode{
						NodeType: pointer.ToString(mservice.AddServiceParamsBodyExternalAddNodeNodeTypeNODETYPEREMOTENODE),
						NodeName: "external-serverless",
					},
					ServiceName:         serviceName,
					ListenPort:          12345,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "address can't be empty for add node request.")
		assert.Nil(t, addExternalOK)
	})
}

func TestRemoveExternal(t *testing.T) {
	addExternal := func(t *testing.T, serviceName, nodeName string) (nodeID string, serviceID string) {
		t.Helper()
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID = genericNode.NodeID

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				External: &mservice.AddServiceParamsBodyExternal{
					NodeID:              nodeID,
					RunsOnNodeID:        nodeID,
					ServiceName:         serviceName,
					Username:            "username",
					Password:            "password",
					ListenPort:          12345,
					Group:               "external",
					SkipConnectionCheck: true,
				},
			},
		}
		addExternalOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addExternalOK)
		require.NotNil(t, addExternalOK.Payload.External.Service)
		serviceID = addExternalOK.Payload.External.Service.ServiceID
		return
	}

	t.Run("By name", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		nodeID, serviceID := addExternal(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceName,
			ServiceType: pointer.ToString(types.ServiceTypeExternalService),
			Context:     pmmapitests.Context,
		})
		noError := assert.NoError(t, err)
		notNil := assert.NotNil(t, removeServiceOK)
		if !noError || !notNil {
			defer pmmapitests.RemoveServices(t, serviceID)
		}

		// Check that the service removed with agents.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: pointer.ToString(serviceID),
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgents)
	})

	t.Run("By ID", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-id")
		nodeName := pmmapitests.TestString(t, "node-remove-by-id")
		nodeID, serviceID := addExternal(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: pointer.ToString(types.ServiceTypeExternalService),
			Context:     pmmapitests.Context,
		})
		noError := assert.NoError(t, err)
		notNil := assert.NotNil(t, removeServiceOK)
		if !noError || !notNil {
			defer pmmapitests.RemoveServices(t, serviceID)
		}

		// Check that the service removed with agents.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: pointer.ToString(serviceID),
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgents)
	})

	t.Run("Wrong type", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-wrong-type")
		nodeName := pmmapitests.TestString(t, "node-remove-wrong-type")
		nodeID, serviceID := addExternal(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: pointer.ToString(types.ServiceTypePostgreSQLService),
			Context:     pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "wrong service type")
	})

	t.Run("No params", func(t *testing.T) {
		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected")
	})
}
