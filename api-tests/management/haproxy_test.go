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
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func TestAddHAProxy(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "genericNode-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				ServiceName:         serviceName,
				ListenPort:          8404,
				NodeID:              nodeID,
				SkipConnectionCheck: true,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		require.NoError(t, err)
		require.NotNil(t, addHAProxyOK)
		require.NotNil(t, addHAProxyOK.Payload.Service)
		serviceID := addHAProxyOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Haproxy: &services.GetServiceOKBodyHaproxy{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
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
				AgentID:            listAgents.Payload.ExternalExporter[0].AgentID,
				ServiceID:          serviceID,
				ListenPort:         8404,
				RunsOnNodeID:       nodeID,
				Scheme:             "http",
				MetricsPath:        "/metrics",
				PushMetricsEnabled: true,
				CustomLabels:       map[string]string{},
			},
		}, listAgents.Payload.ExternalExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With labels", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "genericNode-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-all-fields-name")

		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
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
				SkipConnectionCheck: true,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		require.NoError(t, err)
		require.NotNil(t, addHAProxyOK)
		require.NotNil(t, addHAProxyOK.Payload.Service)
		serviceID := addHAProxyOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Haproxy: &services.GetServiceOKBodyHaproxy{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},
			},
		}, *serviceOK.Payload)
	})

	t.Run("OnRemoteNode", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "genericNode-for-basic-name")

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				AddNode: &mservice.AddHAProxyParamsBodyAddNode{
					NodeType:     pointer.ToString(mservice.AddHAProxyParamsBodyAddNodeNodeTypeNODETYPEREMOTENODE),
					NodeName:     nodeName,
					MachineID:    "/machine-id/",
					Distro:       "linux",
					Region:       "us-west2",
					CustomLabels: map[string]string{"foo": "bar-for-node"},
				},
				Address:             "localhost",
				ServiceName:         serviceName,
				ListenPort:          8404,
				SkipConnectionCheck: true,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		require.NoError(t, err)
		require.NotNil(t, addHAProxyOK)
		require.NotNil(t, addHAProxyOK.Payload.Service)
		nodeID := addHAProxyOK.Payload.Service.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)
		serviceID := addHAProxyOK.Payload.Service.ServiceID
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
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Haproxy: &services.GetServiceOKBodyHaproxy{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
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
				ListenPort:   8404,
				RunsOnNodeID: nodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				CustomLabels: map[string]string{},
			},
		}, listAgents.Payload.ExternalExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With the same name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "genericNode-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")

		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				NodeID:              nodeID,
				ServiceName:         serviceName,
				ListenPort:          9250,
				SkipConnectionCheck: true,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		require.NoError(t, err)
		require.NotNil(t, addHAProxyOK)
		require.NotNil(t, addHAProxyOK.Payload.Service)
		serviceID := addHAProxyOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		params = &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
				ListenPort:  9260,
			},
		}
		addHAProxyOK, err = client.Default.ManagementService.AddHAProxy(params)
		require.Nil(t, addHAProxyOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("Empty Service Name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				NodeID: nodeID,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddHAProxyRequest.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addHAProxyOK)
	})

	t.Run("Empty ListenPort", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddHAProxyRequest.ListenPort: value must be inside range (0, 65536)")
		assert.Nil(t, addHAProxyOK)
	})

	t.Run("Empty Node ID", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				ServiceName: serviceName,
				ListenPort:  12345,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected only one param; node id, node name or register node params")
		assert.Nil(t, addHAProxyOK)
	})

	t.Run("Empty Address for Add Node", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID := genericNode.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				AddNode: &mservice.AddHAProxyParamsBodyAddNode{
					NodeType: pointer.ToString(mservice.AddHAProxyParamsBodyAddNodeNodeTypeNODETYPEREMOTENODE),
					NodeName: "haproxy-serverless",
				},
				ServiceName: serviceName,
				ListenPort:  12345,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "address can't be empty for add node request.")
		assert.Nil(t, addHAProxyOK)
	})
}

func TestRemoveHAProxy(t *testing.T) {
	addHAProxy := func(t *testing.T, serviceName, nodeName string) (nodeID string, serviceID string) {
		t.Helper()
		genericNode := pmmapitests.AddGenericNode(t, nodeName)
		nodeID = genericNode.NodeID

		params := &mservice.AddHAProxyParams{
			Context: pmmapitests.Context,
			Body: mservice.AddHAProxyBody{
				NodeID:              nodeID,
				ServiceName:         serviceName,
				Username:            "username",
				Password:            "password",
				ListenPort:          12345,
				SkipConnectionCheck: true,
			},
		}
		addHAProxyOK, err := client.Default.ManagementService.AddHAProxy(params)
		require.NoError(t, err)
		require.NotNil(t, addHAProxyOK)
		require.NotNil(t, addHAProxyOK.Payload.Service)
		serviceID = addHAProxyOK.Payload.Service.ServiceID
		return
	}

	t.Run("By name", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		nodeID, serviceID := addHAProxy(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Body: mservice.RemoveServiceBody{
				ServiceName: serviceName,
				ServiceType: pointer.ToString(mservice.RemoveServiceBodyServiceTypeSERVICETYPEHAPROXYSERVICE),
			},
			Context: pmmapitests.Context,
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
		nodeID, serviceID := addHAProxy(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Body: mservice.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceType: pointer.ToString(mservice.RemoveServiceBodyServiceTypeSERVICETYPEHAPROXYSERVICE),
			},
			Context: pmmapitests.Context,
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

	t.Run("Both params", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-both-params")
		nodeName := pmmapitests.TestString(t, "node-remove-both-params")
		nodeID, serviceID := addHAProxy(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Body: mservice.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceName: serviceName,
				ServiceType: pointer.ToString(mservice.RemoveServiceBodyServiceTypeSERVICETYPEHAPROXYSERVICE),
			},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected; not both")
	})

	t.Run("Wrong type", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-wrong-type")
		nodeName := pmmapitests.TestString(t, "node-remove-wrong-type")
		nodeID, serviceID := addHAProxy(t, serviceName, nodeName)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Body: mservice.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceType: pointer.ToString(mservice.RemoveServiceBodyServiceTypeSERVICETYPEPOSTGRESQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "wrong service type")
	})

	t.Run("No params", func(t *testing.T) {
		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Body:    mservice.RemoveServiceBody{},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected")
	})
}
