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

func TestAddMySQL(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        3306,
					Username:    "username",

					SkipConnectionCheck: true,
					DisableCollectors:   []string{"global_status", "perf_schema.tablelocks"},
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        address,
				Port:           3306,
				CustomLabels:   map[string]string{},
				ExtraDsnParams: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
			{
				AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
				ServiceID:                 serviceID,
				PMMAgentID:                pmmAgentID,
				Username:                  "username",
				TablestatsGroupTableLimit: 1000,
				DisabledCollectors:        []string{"global_status", "perf_schema.tablelocks"},
				PushMetricsEnabled:        true,
				Status:                    &AgentStatusUnknown,
				CustomLabels:              map[string]string{},
				ExtraDsnParams:            map[string]string{},
				LogLevel:                  new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MysqldExporter)
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:             nodeID,
					PMMAgentID:         pmmAgentID,
					ServiceName:        serviceName,
					Address:            address,
					Port:               3306,
					Username:           "username",
					Password:           "password",
					QANMysqlSlowlog:    true,
					QANMysqlPerfschema: true,

					SkipConnectionCheck:       true,
					TablestatsGroupTableLimit: -1,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        address,
				Port:           3306,
				CustomLabels:   map[string]string{},
				ExtraDsnParams: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that exporters are added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		require.NotNil(t, listAgents)
		require.Len(t, listAgents.Payload.MysqldExporter, 1)
		require.Len(t, listAgents.Payload.QANMysqlSlowlogAgent, 1)
		require.Len(t, listAgents.Payload.QANMysqlPerfschemaAgent, 1)
		assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
			{
				AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
				ServiceID:                 serviceID,
				PMMAgentID:                pmmAgentID,
				Username:                  "username",
				TablestatsGroupTableLimit: -1,
				TablestatsGroupDisabled:   true,
				PushMetricsEnabled:        true,
				Status:                    &AgentStatusUnknown,
				CustomLabels:              map[string]string{},
				ExtraDsnParams:            map[string]string{},
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  new("LOG_LEVEL_UNSPECIFIED"),
			},
		},
			listAgents.Payload.MysqldExporter)

		assert.Equal(t, []*agents.ListAgentsOKBodyQANMysqlSlowlogAgentItems0{
			{
				AgentID:            listAgents.Payload.QANMysqlSlowlogAgent[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Username:           "username",
				MaxSlowlogFileSize: "1073741824",
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				ExtraDsnParams:     map[string]string{},
				LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.QANMysqlSlowlogAgent)

		assert.Equal(t, []*agents.ListAgentsOKBodyQANMysqlPerfschemaAgentItems0{
			{
				AgentID:        listAgents.Payload.QANMysqlPerfschemaAgent[0].AgentID,
				ServiceID:      serviceID,
				PMMAgentID:     pmmAgentID,
				Username:       "username",
				Status:         &AgentStatusUnknown,
				CustomLabels:   map[string]string{},
				ExtraDsnParams: map[string]string{},
				LogLevel:       new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.QANMysqlPerfschemaAgent)
	})

	t.Run("With labels", func(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:         nodeID,
					PMMAgentID:     pmmAgentID,
					ServiceName:    serviceName,
					Address:        address,
					Port:           3306,
					Username:       "username",
					Password:       "password",
					Environment:    "some-environment",
					Cluster:        "cluster-name",
					ReplicationSet: "replication-set",
					CustomLabels:   map[string]string{"bar": "foo"},

					SkipConnectionCheck: true,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        address,
				Port:           3306,
				Environment:    "some-environment",
				Cluster:        "cluster-name",
				ReplicationSet: "replication-set",
				CustomLabels:   map[string]string{"bar": "foo"},
				ExtraDsnParams: map[string]string{},
			},
		}, *serviceOK.Payload)
	})

	t.Run("With the same name", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-for-the-same-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		serviceName := pmmapitests.TestString(t, "service-for-the-same-name")
		address := pmmapitests.TestString(t, "10.10.10.10")

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        3306,
					Username:    "username",

					SkipConnectionCheck: true,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		params = &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "11.11.11.11"),
					Port:        3307,
					Username:    "username",
				},
			},
		}
		addMySQLOK, err = client.Default.ManagementService.AddService(params)
		require.Nil(t, addMySQLOK)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, `Service with name %q already exists.`, serviceName)
	})

	t.Run("With add_node block", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		nodeNameAddNode := pmmapitests.TestString(t, "node-for-add-node-name")
		serviceName := pmmapitests.TestString(t, "service-name-for-basic-name")

		t.Run("generic node", func(t *testing.T) {
			t.Parallel()
			params := &mservice.AddServiceParams{
				Context: pmmapitests.Context,
				Body: mservice.AddServiceBody{
					Mysql: &mservice.AddServiceParamsBodyMysql{
						AddNode: &mservice.AddServiceParamsBodyMysqlAddNode{
							NodeType: new(mservice.AddServiceParamsBodyMysqlAddNodeNodeTypeNODETYPEGENERICNODE),
							NodeName: nodeNameAddNode,
						},
						PMMAgentID:  pmmAgentID,
						ServiceName: serviceName,
						Address:     pmmapitests.TestString(t, "10.10.10.10"),
						Port:        27017,
						Username:    "username",

						SkipConnectionCheck: true,
					},
				},
			}
			_, err := client.Default.ManagementService.AddService(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "add_node structure can be used only for remote nodes")
		})

		t.Run("rds node", func(t *testing.T) {
			t.Parallel()

			params := &mservice.AddServiceParams{
				Context: pmmapitests.Context,
				Body: mservice.AddServiceBody{
					Mysql: &mservice.AddServiceParamsBodyMysql{
						AddNode: &mservice.AddServiceParamsBodyMysqlAddNode{
							NodeType: new(mservice.AddServiceParamsBodyMysqlAddNodeNodeTypeNODETYPEREMOTERDSNODE),
							NodeName: nodeNameAddNode,
						},
						PMMAgentID:  pmmAgentID,
						ServiceName: serviceName,
						Address:     pmmapitests.TestString(t, "10.10.10.10"),
						Port:        27017,
						Username:    "username",

						SkipConnectionCheck: true,
					},
				},
			}
			_, err := client.Default.ManagementService.AddService(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "add_node structure can be used only for remote nodes")
		})

		t.Run("remote node", func(t *testing.T) {
			t.Parallel()

			address := pmmapitests.TestString(t, "10.10.10.10")

			params := &mservice.AddServiceParams{
				Context: pmmapitests.Context,
				Body: mservice.AddServiceBody{
					Mysql: &mservice.AddServiceParamsBodyMysql{
						AddNode: &mservice.AddServiceParamsBodyMysqlAddNode{
							NodeType: new(mservice.AddServiceParamsBodyMysqlAddNodeNodeTypeNODETYPEREMOTENODE),
							NodeName: nodeNameAddNode,
						},
						PMMAgentID:  pmmAgentID,
						ServiceName: serviceName,
						Address:     address,
						Port:        27017,
						Username:    "username",

						SkipConnectionCheck: true,
					},
				},
			}
			addMySQLOK, err := client.Default.ManagementService.AddService(params)
			require.NoError(t, err)
			require.NotNil(t, addMySQLOK)
			require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
			newNodeID := addMySQLOK.Payload.Mysql.Service.NodeID
			require.NotEqual(t, nodeID, newNodeID)
			t.Cleanup(func() {
				pmmapitests.RemoveNodes(t, newNodeID)
			})

			serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
			t.Cleanup(func() {
				pmmapitests.RemoveServices(t, serviceID)
			})

			// Check that service is created and its fields.
			serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
				ServiceID: serviceID,
				Context:   pmmapitests.Context,
			})
			require.NoError(t, err)
			require.NotNil(t, serviceOK)
			assert.Equal(t, services.GetServiceOKBody{
				Mysql: &services.GetServiceOKBodyMysql{
					ServiceID:      serviceID,
					NodeID:         newNodeID,
					ServiceName:    serviceName,
					Address:        address,
					Port:           27017,
					CustomLabels:   map[string]string{},
					ExtraDsnParams: map[string]string{},
				},
			}, *serviceOK.Payload)

			// Check that mysql exporter is added by default.
			listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
				Context:   pmmapitests.Context,
				ServiceID: new(serviceID),
			})
			require.NoError(t, err)
			assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
					ServiceID:                 serviceID,
					PMMAgentID:                pmmAgentID,
					Username:                  "username",
					TablestatsGroupTableLimit: 1000,
					PushMetricsEnabled:        true,
					Status:                    &AgentStatusUnknown,
					CustomLabels:              map[string]string{},
					ExtraDsnParams:            map[string]string{},
					LogLevel:                  new("LOG_LEVEL_UNSPECIFIED"),
					DisabledCollectors:        make([]string, 0),
				},
			}, listAgents.Payload.MysqldExporter)
		})
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      remoteNodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        3306,
					PMMAgentID:  pmmAgentID,
					Username:    "username",

					SkipConnectionCheck: true,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "node_id or node_name can be used only for generic nodes or container nodes")
		assert.Nil(t, addMySQLOK)
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID: nodeID,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLServiceParams.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty Address And Socket", func(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					PMMAgentID:  pmmAgentID,
					Username:    "username",
					Password:    "password",
					NodeID:      nodeID,
					ServiceName: serviceName,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		assert.Nil(t, addMySQLOK)
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					PMMAgentID:  pmmAgentID,
					Username:    "username",
					Password:    "password",
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		assert.Nil(t, addMySQLOK)
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					PMMAgentID:  pmmAgentID,
					Username:    "username",
					Password:    "password",
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        3306,
					Socket:      "/var/run/mysqld/mysqld.sock",
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		assert.Nil(t, addMySQLOK)
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        3306,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLServiceParams.PmmAgentId: value length must be at least 1 runes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("Empty username", func(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        3306,
					PMMAgentID:  pmmAgentID,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLServiceParams.Username: value length must be at least 1 runes")
		assert.Nil(t, addMySQLOK)
	})

	t.Run("With MetricsModePush", func(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        3306,
					Username:    "username",

					SkipConnectionCheck: true,
					MetricsMode:         new("METRICS_MODE_PUSH"),
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        address,
				Port:           3306,
				CustomLabels:   map[string]string{},
				ExtraDsnParams: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
			{
				AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
				ServiceID:                 serviceID,
				PMMAgentID:                pmmAgentID,
				Username:                  "username",
				TablestatsGroupTableLimit: 1000,
				PushMetricsEnabled:        true,
				Status:                    &AgentStatusUnknown,
				CustomLabels:              map[string]string{},
				ExtraDsnParams:            map[string]string{},
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MysqldExporter)
	})

	t.Run("With MetricsModePull", func(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        3306,
					Username:    "username",

					SkipConnectionCheck: true,
					MetricsMode:         new("METRICS_MODE_PULL"),
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        address,
				Port:           3306,
				CustomLabels:   map[string]string{},
				ExtraDsnParams: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
			{
				AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
				ServiceID:                 serviceID,
				PMMAgentID:                pmmAgentID,
				Username:                  "username",
				TablestatsGroupTableLimit: 1000,
				Status:                    &AgentStatusUnknown,
				CustomLabels:              map[string]string{},
				ExtraDsnParams:            map[string]string{},
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MysqldExporter)
	})

	t.Run("With MetricsModeAuto", func(t *testing.T) {
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
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        3306,
					Username:    "username",

					SkipConnectionCheck: true,
					MetricsMode:         new("METRICS_MODE_UNSPECIFIED"),
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID := addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		// Check that service is created and its fields.
		serviceOK, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, serviceOK)
		assert.Equal(t, services.GetServiceOKBody{
			Mysql: &services.GetServiceOKBodyMysql{
				ServiceID:      serviceID,
				NodeID:         nodeID,
				ServiceName:    serviceName,
				Address:        address,
				Port:           3306,
				CustomLabels:   map[string]string{},
				ExtraDsnParams: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that mysqld exporter is added by default.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
			{
				AgentID:                   listAgents.Payload.MysqldExporter[0].AgentID,
				ServiceID:                 serviceID,
				PMMAgentID:                pmmAgentID,
				Username:                  "username",
				TablestatsGroupTableLimit: 1000,
				PushMetricsEnabled:        true,
				Status:                    &AgentStatusUnknown,
				CustomLabels:              map[string]string{},
				ExtraDsnParams:            map[string]string{},
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.MysqldExporter)
	})
}

func TestRemoveMySQL(t *testing.T) {
	t.Parallel()

	addMySQL := func(t *testing.T, serviceName, nodeName string, withAgents bool) (serviceID string) {
		t.Helper()
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Mysql: &mservice.AddServiceParamsBodyMysql{
					NodeID:             nodeID,
					PMMAgentID:         pmmAgentID,
					ServiceName:        serviceName,
					Address:            pmmapitests.TestString(t, "10.10.10.10"),
					Port:               3306,
					Username:           "username",
					Password:           "password",
					QANMysqlSlowlog:    withAgents,
					QANMysqlPerfschema: withAgents,

					SkipConnectionCheck: true,
				},
			},
		}
		addMySQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addMySQLOK)
		require.NotNil(t, addMySQLOK.Payload.Mysql.Service)
		serviceID = addMySQLOK.Payload.Mysql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})
		return serviceID
	}

	t.Run("By name", func(t *testing.T) {
		t.Parallel()

		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		serviceID := addMySQL(t, serviceName, nodeName, true)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceName,
			ServiceType: new(types.ServiceTypeMySQLService),
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
		serviceID := addMySQL(t, serviceName, nodeName, true)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: new(types.ServiceTypeMySQLService),
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

	t.Run("Wrong type", func(t *testing.T) {
		t.Parallel()

		serviceName := pmmapitests.TestString(t, "service-remove-wrong-type")
		nodeName := pmmapitests.TestString(t, "node-remove-wrong-type")
		serviceID := addMySQL(t, serviceName, nodeName, false)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: new(types.ServiceTypePostgreSQLService),
			Context:     pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "wrong service type")
	})

	t.Run("No params", func(t *testing.T) {
		t.Parallel()

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			Context: pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "service_id or service_name expected")
	})
}
