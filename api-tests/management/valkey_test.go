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
	t.Parallel()

	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")
		address := pmmapitests.TestString(t, "10.10.10.10")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Address:             address,
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
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that the service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Valkey: &services.GetServiceOKBodyValkey{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      address,
				Port:         6379,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that no one exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
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
	})

	t.Run("With agents", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-for-all-fields-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		serviceName := pmmapitests.TestString(t, "service-for-all-fields-name")
		address := pmmapitests.TestString(t, "10.10.10.10")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Address:             address,
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
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that the service was created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Valkey: &services.GetServiceOKBodyValkey{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      address,
				Port:         6379,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that exporters are added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		require.NotNil(t, listAgents)
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
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-for-the-same-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Username:            "default",
					Address:             pmmapitests.TestString(t, "10.10.10.10"),
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
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		params = &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Username:    "default",
					Address:     pmmapitests.TestString(t, "11.11.11.11"),
					Port:        6380,
				},
			},
		}
		addValkeyOK, err = client.Default.ManagementService.AddService(params)
		require.Nil(t, addValkeyOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("With Wrong Node Type", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "generic-node-for-wrong-node-type")
		_, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		remoteNodeOKBody := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote Node for wrong type test"))
		remoteNodeID := remoteNodeOKBody.NodeID

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              remoteNodeID,
					ServiceName:         serviceName,
					Address:             pmmapitests.TestString(t, "10.10.10.10"),
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
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, _ := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

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
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

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
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:      nodeID,
					ServiceName: serviceName,
					PMMAgentID:  pmmAgentID,
					Username:    "default",
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Empty Pmm Agent ID", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, _ := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        6371,
				},
			},
		}
		addValkeyOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddValkeyServiceParams.PmmAgentId: value length must be at least 1 runes")
		assert.Nil(t, addValkeyOK)
	})

	t.Run("Address And Socket Conflict.", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

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
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
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
	t.Parallel()

	addValkey := func(t *testing.T, serviceName, nodeName string) (serviceID string) {
		t.Helper()
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Valkey: &mservice.AddServiceParamsBodyValkey{
					NodeID:              nodeID,
					PMMAgentID:          pmmAgentID,
					ServiceName:         serviceName,
					Address:             pmmapitests.TestString(t, "10.10.10.10"),
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
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})
		return serviceID
	}

	t.Run("By name", func(t *testing.T) {
		t.Parallel()

		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		serviceID := addValkey(t, serviceName, nodeName)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceName,
			ServiceType: new(types.ServiceTypeValkeyService),
			Context:     pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, removeServiceOK)

		// Check that the service removed with agents.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgents)
	})

	t.Run("By ID", func(t *testing.T) {
		t.Parallel()

		serviceName := pmmapitests.TestString(t, "service-remove-by-id")
		nodeName := pmmapitests.TestString(t, "node-remove-by-id")
		serviceID := addValkey(t, serviceName, nodeName)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: new(types.ServiceTypeValkeyService),
			Context:     pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, removeServiceOK)

		// Check that the service removed with agents.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgents)
	})
}
