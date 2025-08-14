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
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func TestAddValkey(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Address:             "10.10.10.10",
					Port:                6379,
					Username:            "default",
					SkipConnectionCheck: true,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addValkeyOK)
		require.NotNil(t, addValkeyOK.Payload.Valkey.Service)
		serviceID := addValkeyOK.Payload.Valkey.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that the service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Valkey: &services.GetServiceOKBodyValkey{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         6379,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that no one exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: pointer.ToString(serviceID),
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyValkeyExporterItems0{
			{
				AgentID:            listAgents.Payload.ValkeyExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Username:           "default",
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       make(map[string]string),
				DisabledCollectors: []string{},
			},
		}, listAgents.Payload.ValkeyExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With agents", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-all-fields-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-all-fields-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Address:             "10.10.10.10",
					Port:                6379,
					Username:            "default",
					Password:            "password",
					SkipConnectionCheck: true,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addValkeyOK)
		require.NotNil(t, addValkeyOK.Payload.Valkey.Service)
		serviceID := addValkeyOK.Payload.Valkey.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that the service was created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Valkey: &services.GetServiceOKBodyValkey{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         6379,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that exporters are added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: pointer.ToString(serviceID),
		})
		assert.NoError(t, err)
		require.NotNil(t, listAgents)
		defer removeAllAgentsInList(t, listAgents)
		require.Len(t, listAgents.Payload.ValkeyExporter, 1)
		assert.Equal(t, []*agents.ListAgentsOKBodyValkeyExporterItems0{
			{
				AgentID:            listAgents.Payload.ValkeyExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Username:           "default",
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				DisabledCollectors: []string{},
				CustomLabels:       make(map[string]string),
			},
		}, listAgents.Payload.ValkeyExporter)
	})

	t.Run("With the same name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-the-same-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Username:            "default",
					Address:             "10.10.10.10",
					Port:                6379,
					SkipConnectionCheck: true,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addValkeyOK)
		require.NotNil(t, addValkeyOK.Payload.Valkey.Service)
		serviceID := addValkeyOK.Payload.Valkey.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		params = &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Username:    "default",
					Address:     "11.11.11.11",
					Port:        6380,
				},
			},
		}
		addValkeyOK, err = client.Default.ManagementService.AddService(params)
		require.Nil(t, addValkeyOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("With Wrong Node Type", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "generic-node-for-wrong-node-type")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		remoteNodeOKBody := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote Node for wrong type test"))
		remoteNodeID := remoteNodeOKBody.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, remoteNodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              remoteNodeID,
					ServiceName:         serviceName,
					Address:             "10.10.10.10",
					Port:                3306,
					PMMAgentID:          pmmAgentID,
					Username:            "default",
					SkipConnectionCheck: true,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "node_id or node_name can be used only for generic nodes or container nodes")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Empty Service Name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID: nodeID,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddValkeyServiceParams.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Empty Address", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					PMMAgentID:  pmmAgentID,
					NodeID:      nodeID,
					ServiceName: serviceName,
					Username:    "default",
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Empty Port", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:      nodeID,
					ServiceName: serviceName,
					PMMAgentID:  pmmAgentID,
					Username:    "default",
					Address:     "10.10.10.10",
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Empty Pmm Agent ID", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     "10.10.10.10",
					Port:        6371,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddValkeyServiceParams.PmmAgentId: value length must be at least 1 runes")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Address And Socket Conflict.", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					PMMAgentID:  pmmAgentID,
					Username:    "default",
					Password:    "password",
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     "10.10.10.10",
					Port:        6371,
					Socket:      "/var/run/valkey.sock",
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		assert.Nil(t, addValkeyOK)
	})
}

func TestRemoveValkey(t *testing.T) {
	addValkey := func(t *testing.T, serviceName, nodeName string) (nodeID string, pmmAgentID string, serviceID string) {
		t.Helper()
		nodeID, pmmAgentID = RegisterGenericNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Address:             "10.10.10.10",
					Port:                6379,
					Username:            "default",
					Password:            "password",
					SkipConnectionCheck: true,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addValkeyOK)
		require.NotNil(t, addValkeyOK.Payload.Valkey.Service)
		serviceID = addValkeyOK.Payload.Valkey.Service.ServiceID
		return
	}

	t.Run("By name", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		nodeID, pmmAgentID, serviceID := addValkey(t, serviceName, nodeName)
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceName,
			ServiceType: pointer.ToString(types.ServiceTypeValkeyService),
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
		nodeID, pmmAgentID, serviceID := addValkey(t, serviceName, nodeName)
		defer pmmapitests.UnregisterNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: pointer.ToString(types.ServiceTypeValkeyService),
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
}
