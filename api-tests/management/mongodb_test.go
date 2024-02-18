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
	"github.com/percona/pmm/api/management/v1/json/client"
	"github.com/percona/pmm/api/management/v1/json/client/service"
)

func TestAddMongoDB(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
				DisableCollectors:   []string{"database"},
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         27017,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mongodb exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				DisabledCollectors: []string{"database"},
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				StatsCollections:   make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With agents", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name-for-all-fields")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-all-fields")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:             nodeID,
				PMMAgentID:         pmmAgentID,
				ServiceName:        serviceName,
				Address:            "10.10.10.10",
				Port:               27017,
				Username:           "username",
				Password:           "password",
				QANMongodbProfiler: true,

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         27017,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that exporters are added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		require.NotNil(t, listAgents)
		defer removeAllAgentsInList(t, listAgents)

		require.Len(t, listAgents.Payload.MongodbExporter, 1)
		require.Len(t, listAgents.Payload.QANMongodbProfilerAgent, 1)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Username:           "username",
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				StatsCollections:   make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		assert.Equal(t, []*agents.ListAgentsOKBodyQANMongodbProfilerAgentItems0{
			{
				AgentID:      listAgents.Payload.QANMongodbProfilerAgent[0].AgentID,
				ServiceID:    serviceID,
				PMMAgentID:   pmmAgentID,
				Username:     "username",
				Status:       &AgentStatusUnknown,
				CustomLabels: map[string]string{},
				LogLevel:     pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.QANMongodbProfilerAgent)
	})

	t.Run("With labels", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name-for-all-fields")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-all-fields")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:         nodeID,
				PMMAgentID:     pmmAgentID,
				ServiceName:    serviceName,
				Address:        "10.10.10.10",
				Port:           27017,
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        "10.10.10.10",
				Port:           27017,
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},
			},
		}, *serviceOK.Payload)
	})

	t.Run("With the same name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-the-same-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		params = &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "11.11.11.11",
				Port:        27017,
			},
		}
		addMongoDBOK, err = client.Default.Service.AddMongoDB(params)
		require.Nil(t, addMongoDBOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("With add_node block", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		nodeNameAddNode := pmmapitests.TestString(t, "node-for-add-node-name")
		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				AddNode: &service.AddMongoDBParamsBodyAddNode{
					NodeType: pointer.ToString(service.AddMongoDBParamsBodyAddNodeNodeTypeNODETYPEGENERICNODE),
					NodeName: nodeNameAddNode,
				},
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
			},
		}
		_, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "add_node structure can be used only for remote nodes")

		params = &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				AddNode: &service.AddMongoDBParamsBodyAddNode{
					NodeType: pointer.ToString(service.AddMongoDBParamsBodyAddNodeNodeTypeNODETYPEREMOTERDSNODE),
					NodeName: nodeNameAddNode,
				},
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
			},
		}
		_, err = client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "add_node structure can be used only for remote nodes")

		params = &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				AddNode: &service.AddMongoDBParamsBodyAddNode{
					NodeType: pointer.ToString(service.AddMongoDBParamsBodyAddNodeNodeTypeNODETYPEREMOTENODE),
					NodeName: nodeNameAddNode,
				},
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID

		newNodeID := addMongoDBOK.Payload.Service.NodeID
		require.NotEqual(t, nodeID, newNodeID)
		defer pmmapitests.RemoveNodes(t, newNodeID)
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
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       newNodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         27017,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mongodb exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				StatsCollections:   make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With Wrong Node Type", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "generic-node-for-wrong-node-type")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		remoteNodeOKBody := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote Node for wrong type test"))
		remoteNodeID := remoteNodeOKBody.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, remoteNodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      remoteNodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				PMMAgentID:  pmmAgentID,

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "node_id or node_name can be used only for generic nodes or container nodes")
		assert.Nil(t, addMongoDBOK)
	})

	t.Run("Empty Service Name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body:    service.AddMongoDBBody{NodeID: nodeID},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMongoDBRequest.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addMongoDBOK)
	})

	t.Run("Empty Address", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
				PMMAgentID:  pmmAgentID,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		assert.Nil(t, addMongoDBOK)
	})

	t.Run("Empty Port", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				PMMAgentID:  pmmAgentID,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port are expected to be passed with address.")
		assert.Nil(t, addMongoDBOK)
	})

	t.Run("Empty Pmm Agent ID", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMongoDBRequest.PmmAgentId: value length must be at least 1 runes")
		assert.Nil(t, addMongoDBOK)
	})

	t.Run("Address And Socket Conflict.", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				PMMAgentID:  pmmAgentID,
				Username:    "username",
				Password:    "password",
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,
				Socket:      "/tmp/mongodb-27017.sock",
			},
		}
		addProxySQLOK, err := client.Default.Service.AddMongoDB(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		assert.Nil(t, addProxySQLOK)
	})

	t.Run("Socket", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-mongo-socket-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-mongo-socket-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Socket:      "/tmp/mongodb-27017.sock",

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Socket:       "/tmp/mongodb-27017.sock",
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mongodb exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				StatsCollections:   make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With MetricsModePush", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
				MetricsMode:         pointer.ToString("METRICS_MODE_PUSH"),
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         27017,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mongodb exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				StatsCollections:   make([]string, 0),
				DisabledCollectors: make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With MetricsModePull", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
				MetricsMode:         pointer.ToString("METRICS_MODE_PULL"),
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         27017,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mongodb exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				StatsCollections:   make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With MetricsModeAuto", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,

				SkipConnectionCheck: true,
				MetricsMode:         pointer.ToString("METRICS_MODE_UNSPECIFIED"),
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID := addMongoDBOK.Payload.Service.ServiceID
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
			Mongodb: &services.GetServiceOKBodyMongodb{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				Address:      "10.10.10.10",
				Port:         27017,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mongodb exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMongodbExporterItems0{
			{
				AgentID:            listAgents.Payload.MongodbExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				StatsCollections:   make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MongodbExporter)
		defer removeAllAgentsInList(t, listAgents)
	})
}

func TestRemoveMongoDB(t *testing.T) {
	addMongoDB := func(t *testing.T, serviceName, nodeName string, withAgents bool) (nodeID string, pmmAgentID string, serviceID string) {
		t.Helper()
		nodeID, pmmAgentID = RegisterGenericNode(t, service.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(service.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		params := &service.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: service.AddMongoDBBody{
				NodeID:             nodeID,
				PMMAgentID:         pmmAgentID,
				ServiceName:        serviceName,
				Address:            "10.10.10.10",
				Port:               27017,
				Username:           "username",
				Password:           "password",
				QANMongodbProfiler: withAgents,

				SkipConnectionCheck: true,
			},
		}
		addMongoDBOK, err := client.Default.Service.AddMongoDB(params)
		require.NoError(t, err)
		require.NotNil(t, addMongoDBOK)
		require.NotNil(t, addMongoDBOK.Payload.Service)
		serviceID = addMongoDBOK.Payload.Service.ServiceID
		return
	}

	t.Run("By name", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		nodeID, pmmAgentID, serviceID := addMongoDB(t, serviceName, nodeName, true)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceName: serviceName,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeSERVICETYPEMONGODBSERVICE),
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
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgents)
	})

	t.Run("By ID", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-id")
		nodeName := pmmapitests.TestString(t, "node-remove-by-id")
		nodeID, pmmAgentID, serviceID := addMongoDB(t, serviceName, nodeName, true)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeSERVICETYPEMONGODBSERVICE),
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
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgents)
	})

	t.Run("Both params", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-both-params")
		nodeName := pmmapitests.TestString(t, "node-remove-both-params")
		nodeID, pmmAgentID, serviceID := addMongoDB(t, serviceName, nodeName, false)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceName: serviceName,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeSERVICETYPEMYSQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected; not both")
	})

	t.Run("Wrong type", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-wrong-type")
		nodeName := pmmapitests.TestString(t, "node-remove-wrong-type")
		nodeID, pmmAgentID, serviceID := addMongoDB(t, serviceName, nodeName, false)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeSERVICETYPEPOSTGRESQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "wrong service type")
	})
}
