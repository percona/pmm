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
	inventoryClient "github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	"github.com/percona/pmm/api/managementpb/json/client"
	mysql "github.com/percona/pmm/api/managementpb/json/client/my_sql"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/percona/pmm/api/managementpb/json/client/service"
)

func TestAddMySQL(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				Username:    "username",

				SkipConnectionCheck: true,
				DisableCollectors:   []string{"global_status", "perf_schema.tablelocks"},
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:   serviceID,
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, agents.ListAgentsOKBody{
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: 1000,
					DisabledCollectors:        []string{"global_status", "perf_schema.tablelocks"},
					PushMetricsEnabled:        true,
					Status:                    &AgentStatusUnknown,
				},
			},
		}, *listAgents.Payload)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With agents", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-all-fields-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-all-fields-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:             nodeID,
				PMMAgentID:         pmmAgentID,
				ServiceName:        serviceName,
				Address:            "10.10.10.10",
				Port:               3306,
				Username:           "username",
				Password:           "password",
				QANMysqlSlowlog:    true,
				QANMysqlPerfschema: true,

				SkipConnectionCheck:       true,
				TablestatsGroupTableLimit: -1,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:   serviceID,
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}, *serviceOK.Payload)

		// Check that exporters are added.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		require.NotNil(t, listAgents)
		defer removeAllAgentsInList(t, listAgents)
		require.Len(t, listAgents.Payload.MysqldExporter, 1)
		require.Len(t, listAgents.Payload.QANMysqlSlowlogAgent, 1)
		require.Len(t, listAgents.Payload.QANMysqlPerfschemaAgent, 1)
		assert.Equal(t, agents.ListAgentsOKBody{
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: -1,
					TablestatsGroupDisabled:   true,
					PushMetricsEnabled:        true,
					Status:                    &AgentStatusUnknown,
				},
			},
			QANMysqlSlowlogAgent: []*agents.ListAgentsOKBodyQANMysqlSlowlogAgentItems0{
				{
					AgentID:            listAgents.Payload.QANMysqlSlowlogAgent[0].AgentID,
					ServiceID:          serviceID,
					PMMAgentID:         pmmAgentID,
					Username:           "username",
					MaxSlowlogFileSize: "1073741824",
					Status:             &AgentStatusUnknown,
				},
			},
			QANMysqlPerfschemaAgent: []*agents.ListAgentsOKBodyQANMysqlPerfschemaAgentItems0{
				{
					AgentID:    listAgents.Payload.QANMysqlPerfschemaAgent[0].AgentID,
					ServiceID:  serviceID,
					PMMAgentID: pmmAgentID,
					Username:   "username",
					Status:     &AgentStatusUnknown,
				},
			},
		}, *listAgents.Payload)
	})

	t.Run("With labels", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-all-fields-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-all-fields-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:         nodeID,
				PMMAgentID:     pmmAgentID,
				ServiceName:    serviceName,
				Address:        "10.10.10.10",
				Port:           3306,
				Username:       "username",
				Password:       "password",
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},

				SkipConnectionCheck: true,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        "10.10.10.10",
				Port:           3306,
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},
			},
		}, *serviceOK.Payload)
	})

	t.Run("With the same name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-the-same-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				Username:    "username",

				SkipConnectionCheck: true,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		params = &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "11.11.11.11",
				Port:        3307,
				Username:    "username",
			},
		}
		addMySQLOK, err = client.Default.MySQL.AddMySQL(params)
		require.Nil(t, addMySQLOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("With add_node block", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		nodeNameAddNode := pmmapitests.TestString(t, "node-for-add-node-name")
		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				AddNode: &mysql.AddMySQLParamsBodyAddNode{
					NodeType: pointer.ToString(mysql.AddMySQLParamsBodyAddNodeNodeTypeGENERICNODE),
					NodeName: nodeNameAddNode,
				},
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,
				Username:    "username",

				SkipConnectionCheck: true,
			},
		}
		_, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "add_node structure can be used only for remote nodes")

		params = &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				AddNode: &mysql.AddMySQLParamsBodyAddNode{
					NodeType: pointer.ToString(mysql.AddMySQLParamsBodyAddNodeNodeTypeREMOTERDSNODE),
					NodeName: nodeNameAddNode,
				},
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,
				Username:    "username",

				SkipConnectionCheck: true,
			},
		}
		_, err = client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "add_node structure can be used only for remote nodes")

		params = &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				AddNode: &mysql.AddMySQLParamsBodyAddNode{
					NodeType: pointer.ToString(mysql.AddMySQLParamsBodyAddNodeNodeTypeREMOTENODE),
					NodeName: nodeNameAddNode,
				},
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,
				Username:    "username",

				SkipConnectionCheck: true,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID

		newNodeID := addMySQLOK.Payload.Service.NodeID
		require.NotEqual(t, nodeID, newNodeID)
		defer pmmapitests.RemoveNodes(t, newNodeID)
		defer pmmapitests.RemoveServices(t, serviceID)
		defer removeServiceAgents(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:   serviceID,
				NodeID:      newNodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        27017,
			},
		}, *serviceOK.Payload)

		// Check that mysql exporter is added by default.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, agents.ListAgentsOKBody{
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: 1000,
					PushMetricsEnabled:        true,
					Status:                    &AgentStatusUnknown,
				},
			},
		}, *listAgents.Payload)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With Wrong Node Type", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "generic-node-for-wrong-node-type")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		remoteNodeOKBody := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote Node for wrong type test"))
		remoteNodeID := remoteNodeOKBody.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, remoteNodeID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      remoteNodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				PMMAgentID:  pmmAgentID,
				Username:    "username",

				SkipConnectionCheck: true,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "node_id or node_name can be used only for generic nodes or container nodes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty Service Name", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body:    mysql.AddMySQLBody{NodeID: nodeID},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLRequest.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty Address And Socket", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				PMMAgentID:  pmmAgentID,
				Username:    "username",
				Password:    "password",
				NodeID:      nodeID,
				ServiceName: serviceName,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty Port", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				PMMAgentID:  pmmAgentID,
				Username:    "username",
				Password:    "password",
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port are expected to be passed with address.")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Address And Socket Conflict.", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				PMMAgentID:  pmmAgentID,
				Username:    "username",
				Password:    "password",
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				Socket:      "/var/run/mysqld/mysqld.sock",
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty Pmm Agent ID", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLRequest.PmmAgentId: value length must be at least 1 runes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty username", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-name")
		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				PMMAgentID:  pmmAgentID,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLRequest.Username: value length must be at least 1 runes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("With MetricsModePush", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				Username:    "username",

				SkipConnectionCheck: true,
				MetricsMode:         pointer.ToString("PUSH"),
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:   serviceID,
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, agents.ListAgentsOKBody{
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: 1000,
					PushMetricsEnabled:        true,
					Status:                    &AgentStatusUnknown,
				},
			},
		}, *listAgents.Payload)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With MetricsModePull", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				Username:    "username",

				SkipConnectionCheck: true,
				MetricsMode:         pointer.ToString("PULL"),
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:   serviceID,
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, agents.ListAgentsOKBody{
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: 1000,
					Status:                    &AgentStatusUnknown,
				},
			},
		}, *listAgents.Payload)
		defer removeAllAgentsInList(t, listAgents)
	})

	t.Run("With MetricsModeAuto", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		serviceName := pmmapitests.TestString(t, "service-for-basic-name")

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:      nodeID,
				PMMAgentID:  pmmAgentID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
				Username:    "username",

				SkipConnectionCheck: true,
				MetricsMode:         pointer.ToString("AUTO"),
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID := addMySQLOK.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.Services.GetService(&services.GetServiceParams{
			Body: services.GetServiceBody{
				ServiceID: serviceID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:   serviceID,
				NodeID:      nodeID,
				ServiceName: serviceName,
				Address:     "10.10.10.10",
				Port:        3306,
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Context: pmmapitests.Context,
			Body: agents.ListAgentsBody{
				ServiceID: serviceID,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, agents.ListAgentsOKBody{
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: 1000,
					PushMetricsEnabled:        true,
					Status:                    &AgentStatusUnknown,
				},
			},
		}, *listAgents.Payload)
		defer removeAllAgentsInList(t, listAgents)
	})
}

func TestRemoveMySQL(t *testing.T) {
	addMySQL := func(t *testing.T, serviceName, nodeName string, withAgents bool) (nodeID string, pmmAgentID string, serviceID string) {
		t.Helper()
		nodeID, pmmAgentID = RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
		})

		params := &mysql.AddMySQLParams{
			Context: pmmapitests.Context,
			Body: mysql.AddMySQLBody{
				NodeID:             nodeID,
				PMMAgentID:         pmmAgentID,
				ServiceName:        serviceName,
				Address:            "10.10.10.10",
				Port:               3306,
				Username:           "username",
				Password:           "password",
				QANMysqlSlowlog:    withAgents,
				QANMysqlPerfschema: withAgents,

				SkipConnectionCheck: true,
			},
		}
		addMySQLOK, err := client.Default.MySQL.AddMySQL(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Service)
		serviceID = addMySQLOK.Payload.Service.ServiceID
		return
	}

	t.Run("By name", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		nodeID, pmmAgentID, serviceID := addMySQL(t, serviceName, nodeName, true)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceName: serviceName,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeMYSQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		noError := assert.NoError(t, err)
		notNil := assert.NotNil(t, removeServiceOK)
		if !noError || !notNil {
			defer pmmapitests.RemoveServices(t, serviceID)
		}

		// Check that the service removed with agents.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
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
		nodeID, pmmAgentID, serviceID := addMySQL(t, serviceName, nodeName, true)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeMYSQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		noError := assert.NoError(t, err)
		notNil := assert.NotNil(t, removeServiceOK)
		if !noError || !notNil {
			defer pmmapitests.RemoveServices(t, serviceID)
		}

		// Check that the service removed with agents.
		listAgents, err := inventoryClient.Default.Agents.ListAgents(&agents.ListAgentsParams{
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
		nodeID, pmmAgentID, serviceID := addMySQL(t, serviceName, nodeName, false)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceName: serviceName,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypeMYSQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected; not both")
	})

	t.Run("Wrong type", func(t *testing.T) {
		serviceName := pmmapitests.TestString(t, "service-remove-wrong-type")
		nodeName := pmmapitests.TestString(t, "node-remove-wrong-type")
		nodeID, pmmAgentID, serviceID := addMySQL(t, serviceName, nodeName, false)
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer pmmapitests.RemoveServices(t, serviceID)
		defer RemovePMMAgentWithSubAgents(t, pmmAgentID)

		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body: service.RemoveServiceBody{
				ServiceID:   serviceID,
				ServiceType: pointer.ToString(service.RemoveServiceBodyServiceTypePOSTGRESQLSERVICE),
			},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "wrong service type")
	})

	t.Run("No params", func(t *testing.T) {
		removeServiceOK, err := client.Default.Service.RemoveService(&service.RemoveServiceParams{
			Body:    service.RemoveServiceBody{},
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected")
	})
}
