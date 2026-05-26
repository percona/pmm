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

func TestAddPostgreSQL(t *testing.T) {
	t.Parallel()

	const defaultPostgresDBName = "postgres"

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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        5432,
					Username:    "username",

					SkipConnectionCheck:    true,
					DisableCollectors:      []string{"custom_query.ml", "custom_query.mr.directory"},
					AutoDiscoveryLimit:     0,
					MaxExporterConnections: 0,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql.Service)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
			Postgresql: &services.GetServiceOKBodyPostgresql{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				DatabaseName: defaultPostgresDBName,
				Address:      address,
				Port:         5432,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that no one exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyPostgresExporterItems0{
			{
				AgentID:                listAgents.Payload.PostgresExporter[0].AgentID,
				ServiceID:              serviceID,
				PMMAgentID:             pmmAgentID,
				Username:               "username",
				DisabledCollectors:     []string{"custom_query.ml", "custom_query.mr.directory"},
				PushMetricsEnabled:     true,
				Status:                 &AgentStatusUnknown,
				AutoDiscoveryLimit:     0,
				CustomLabels:           make(map[string]string),
				LogLevel:               new("LOG_LEVEL_UNSPECIFIED"),
				MaxExporterConnections: 0,
			},
		}, listAgents.Payload.PostgresExporter)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:                          nodeID,
					PMMAgentID:                      pmmAgentID,
					ServiceName:                     serviceName,
					Address:                         address,
					Port:                            5432,
					Username:                        "username",
					Password:                        "password",
					QANPostgresqlPgstatementsAgent:  true,
					QANPostgresqlPgstatmonitorAgent: true,
					DisableQueryExamples:            true,

					SkipConnectionCheck:    true,
					AutoDiscoveryLimit:     15,
					MaxExporterConnections: 10,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql.Service)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
			Postgresql: &services.GetServiceOKBodyPostgresql{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				DatabaseName: defaultPostgresDBName,
				Address:      address,
				Port:         5432,
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
		require.Len(t, listAgents.Payload.PostgresExporter, 1)
		require.Len(t, listAgents.Payload.QANPostgresqlPgstatementsAgent, 1)
		require.Len(t, listAgents.Payload.QANPostgresqlPgstatmonitorAgent, 1)
		assert.Equal(t, []*agents.ListAgentsOKBodyPostgresExporterItems0{
			{
				AgentID:                listAgents.Payload.PostgresExporter[0].AgentID,
				ServiceID:              serviceID,
				PMMAgentID:             pmmAgentID,
				Username:               "username",
				PushMetricsEnabled:     true,
				Status:                 &AgentStatusUnknown,
				AutoDiscoveryLimit:     15,
				CustomLabels:           make(map[string]string),
				DisabledCollectors:     make([]string, 0),
				LogLevel:               new("LOG_LEVEL_UNSPECIFIED"),
				MaxExporterConnections: 10,
			},
		}, listAgents.Payload.PostgresExporter)

		assert.Equal(t, []*agents.ListAgentsOKBodyQANPostgresqlPgstatementsAgentItems0{
			{
				AgentID:      listAgents.Payload.QANPostgresqlPgstatementsAgent[0].AgentID,
				ServiceID:    serviceID,
				PMMAgentID:   pmmAgentID,
				Username:     "username",
				Status:       &AgentStatusUnknown,
				CustomLabels: map[string]string{},
				LogLevel:     new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.QANPostgresqlPgstatementsAgent)

		assert.Equal(t, []*agents.ListAgentsOKBodyQANPostgresqlPgstatmonitorAgentItems0{
			{
				AgentID:               listAgents.Payload.QANPostgresqlPgstatmonitorAgent[0].AgentID,
				ServiceID:             serviceID,
				PMMAgentID:            pmmAgentID,
				Username:              "username",
				QueryExamplesDisabled: true,
				Status:                &AgentStatusUnknown,
				CustomLabels:          map[string]string{},
				LogLevel:              new("LOG_LEVEL_UNSPECIFIED"),
			},
		}, listAgents.Payload.QANPostgresqlPgstatmonitorAgent)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:       nodeID,
					PMMAgentID:   pmmAgentID,
					ServiceName:  serviceName,
					Address:      address,
					Port:         5432,
					Username:     "username",
					Environment:  "some-environment",
					CustomLabels: map[string]string{"bar": "foo"},

					SkipConnectionCheck: true,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql.Service)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
			Postgresql: &services.GetServiceOKBodyPostgresql{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				DatabaseName: defaultPostgresDBName,
				Address:      address,
				Port:         5432,
				Environment:  "some-environment",
				CustomLabels: map[string]string{"bar": "foo"},
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Username:    "username",
					Address:     address,
					Port:        5432,

					SkipConnectionCheck: true,
					AutoDiscoveryLimit:  -2,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql.Service)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		params = &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Username:    "username",
					Address:     pmmapitests.TestString(t, "11.11.11.11"),
					Port:        5433,
				},
			},
		}
		addPostgreSQLOK, err = client.Default.ManagementService.AddService(params)
		require.Nil(t, addPostgreSQLOK)
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
					Postgresql: &mservice.AddServiceParamsBodyPostgresql{
						AddNode: &mservice.AddServiceParamsBodyPostgresqlAddNode{
							NodeType: new(mservice.AddServiceParamsBodyPostgresqlAddNodeNodeTypeNODETYPEGENERICNODE),
							NodeName: nodeNameAddNode,
						},
						PMMAgentID:  pmmAgentID,
						ServiceName: serviceName,
						Address:     pmmapitests.TestString(t, "10.10.10.10"),
						Port:        27017,
						Username:    "username",

						SkipConnectionCheck: true,
						AutoDiscoveryLimit:  -1,
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
					Postgresql: &mservice.AddServiceParamsBodyPostgresql{
						AddNode: &mservice.AddServiceParamsBodyPostgresqlAddNode{
							NodeType: new(mservice.AddServiceParamsBodyPostgresqlAddNodeNodeTypeNODETYPEREMOTERDSNODE),
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

			serviceAddress := pmmapitests.TestString(t, "10.10.10.10")
			params := &mservice.AddServiceParams{
				Context: pmmapitests.Context,
				Body: mservice.AddServiceBody{
					Postgresql: &mservice.AddServiceParamsBodyPostgresql{
						AddNode: &mservice.AddServiceParamsBodyPostgresqlAddNode{
							NodeType: new(mservice.AddServiceParamsBodyPostgresqlAddNodeNodeTypeNODETYPEREMOTENODE),
							NodeName: nodeNameAddNode,
						},
						PMMAgentID:  pmmAgentID,
						ServiceName: serviceName,
						Address:     serviceAddress,
						Port:        27017,
						Username:    "username",

						SkipConnectionCheck: true,
						AutoDiscoveryLimit:  5,
					},
				},
			}
			addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
			require.NoError(t, err)
			require.NotNil(t, addPostgreSQLOK)
			require.NotNil(t, addPostgreSQLOK.Payload.Postgresql)
			newNodeID := addPostgreSQLOK.Payload.Postgresql.Service.NodeID
			require.NotEqual(t, nodeID, newNodeID)
			t.Cleanup(func() {
				pmmapitests.RemoveNodes(t, newNodeID)
			})
			serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
				Postgresql: &services.GetServiceOKBodyPostgresql{
					ServiceID:    serviceID,
					NodeID:       newNodeID,
					ServiceName:  serviceName,
					DatabaseName: defaultPostgresDBName,
					Address:      serviceAddress,
					Port:         27017,
					CustomLabels: map[string]string{},
				},
			}, *serviceOK.Payload)

			// Check that postgresql exporter is added by default.
			listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
				Context:   pmmapitests.Context,
				ServiceID: new(serviceID),
			})
			require.NoError(t, err)
			assert.Equal(t, []*agents.ListAgentsOKBodyPostgresExporterItems0{
				{
					AgentID:            listAgents.Payload.PostgresExporter[0].AgentID,
					ServiceID:          serviceID,
					PMMAgentID:         pmmAgentID,
					Username:           "username",
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					AutoDiscoveryLimit: 5,
					CustomLabels:       make(map[string]string),
					DisabledCollectors: make([]string, 0),
					LogLevel:           new(agents.ListAgentsOKBodyPostgresExporterItems0LogLevelLOGLEVELUNSPECIFIED),
				},
			}, listAgents.Payload.PostgresExporter)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
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
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "node_id or node_name can be used only for generic nodes or container nodes")
		assert.Nil(t, addPostgreSQLOK)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID: nodeID,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgreSQLServiceParams.ServiceName: value length must be at least 1 runes")
		assert.Nil(t, addPostgreSQLOK)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					PMMAgentID:  pmmAgentID,
					NodeID:      nodeID,
					ServiceName: serviceName,
					Username:    "username",
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		assert.Nil(t, addPostgreSQLOK)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					ServiceName: serviceName,
					PMMAgentID:  pmmAgentID,
					Username:    "username",
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		assert.Nil(t, addPostgreSQLOK)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        5432,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgreSQLServiceParams.PmmAgentId: value length must be at least 1 runes")
		assert.Nil(t, addPostgreSQLOK)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					PMMAgentID:  pmmAgentID,
					Username:    "username",
					Password:    "password",
					NodeID:      nodeID,
					ServiceName: serviceName,
					Address:     pmmapitests.TestString(t, "10.10.10.10"),
					Port:        5432,
					Socket:      "/var/run/postgresql",
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		assert.Nil(t, addPostgreSQLOK)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        5432,
					Username:    "username",

					SkipConnectionCheck: true,
					MetricsMode:         new("METRICS_MODE_PUSH"),
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
			Postgresql: &services.GetServiceOKBodyPostgresql{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				DatabaseName: defaultPostgresDBName,
				Address:      address,
				Port:         5432,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that no one exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyPostgresExporterItems0{
			{
				AgentID:            listAgents.Payload.PostgresExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Username:           "username",
				PushMetricsEnabled: true,
				Status:             &AgentStatusUnknown,
				AutoDiscoveryLimit: 0,
				CustomLabels:       make(map[string]string),
				DisabledCollectors: make([]string, 0),
				LogLevel:           new(agents.ListAgentsOKBodyPostgresExporterItems0LogLevelLOGLEVELUNSPECIFIED),
			},
		}, listAgents.Payload.PostgresExporter)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        5432,
					Username:    "username",

					SkipConnectionCheck: true,
					MetricsMode:         new("METRICS_MODE_PULL"),
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
			Postgresql: &services.GetServiceOKBodyPostgresql{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				DatabaseName: defaultPostgresDBName,
				Address:      address,
				Port:         5432,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that no one exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyPostgresExporterItems0{
			{
				AgentID:            listAgents.Payload.PostgresExporter[0].AgentID,
				ServiceID:          serviceID,
				PMMAgentID:         pmmAgentID,
				Username:           "username",
				Status:             &AgentStatusUnknown,
				AutoDiscoveryLimit: 0,
				CustomLabels:       make(map[string]string),
				DisabledCollectors: make([]string, 0),
				LogLevel:           new(agents.ListAgentsOKBodyPostgresExporterItems0LogLevelLOGLEVELUNSPECIFIED),
			},
		}, listAgents.Payload.PostgresExporter)
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
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:      nodeID,
					PMMAgentID:  pmmAgentID,
					ServiceName: serviceName,
					Address:     address,
					Port:        5432,
					Username:    "username",

					SkipConnectionCheck: true,
					MetricsMode:         new("METRICS_MODE_UNSPECIFIED"),
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql.Service)
		serviceID := addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
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
			Postgresql: &services.GetServiceOKBodyPostgresql{
				ServiceID:    serviceID,
				NodeID:       nodeID,
				ServiceName:  serviceName,
				DatabaseName: defaultPostgresDBName,
				Address:      address,
				Port:         5432,
				CustomLabels: map[string]string{},
			},
		}, *serviceOK.Payload)

		// Check that no one exporter is added.
		listAgents, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Context:   pmmapitests.Context,
			ServiceID: new(serviceID),
		})
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyPostgresExporterItems0{
			{
				AgentID:                listAgents.Payload.PostgresExporter[0].AgentID,
				ServiceID:              serviceID,
				PMMAgentID:             pmmAgentID,
				Username:               "username",
				PushMetricsEnabled:     true,
				Status:                 &AgentStatusUnknown,
				AutoDiscoveryLimit:     0,
				CustomLabels:           make(map[string]string),
				DisabledCollectors:     make([]string, 0),
				LogLevel:               new(agents.ListAgentsOKBodyPostgresExporterItems0LogLevelLOGLEVELUNSPECIFIED),
				MaxExporterConnections: 0,
			},
		}, listAgents.Payload.PostgresExporter)
	})
}

func TestRemovePostgreSQL(t *testing.T) {
	t.Parallel()

	addPostgreSQL := func(t *testing.T, serviceName, nodeName string, withAgents bool) (serviceID string) {
		t.Helper()
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		params := &mservice.AddServiceParams{
			Context: pmmapitests.Context,
			Body: mservice.AddServiceBody{
				Postgresql: &mservice.AddServiceParamsBodyPostgresql{
					NodeID:                          nodeID,
					PMMAgentID:                      pmmAgentID,
					ServiceName:                     serviceName,
					Address:                         pmmapitests.TestString(t, "10.10.10.10"),
					Port:                            5432,
					Username:                        "username",
					Password:                        "password",
					QANPostgresqlPgstatementsAgent:  withAgents,
					QANPostgresqlPgstatmonitorAgent: withAgents,
					SkipConnectionCheck:             true,
				},
			},
		}
		addPostgreSQLOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addPostgreSQLOK)
		require.NotNil(t, addPostgreSQLOK.Payload.Postgresql.Service)
		serviceID = addPostgreSQLOK.Payload.Postgresql.Service.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})
		return serviceID
	}

	t.Run("By name", func(t *testing.T) {
		t.Parallel()

		serviceName := pmmapitests.TestString(t, "service-remove-by-name")
		nodeName := pmmapitests.TestString(t, "node-remove-by-name")
		serviceID := addPostgreSQL(t, serviceName, nodeName, true)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceName,
			ServiceType: new(types.ServiceTypePostgreSQLService),
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
		serviceID := addPostgreSQL(t, serviceName, nodeName, true)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: new(types.ServiceTypePostgreSQLService),
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
		serviceID := addPostgreSQL(t, serviceName, nodeName, false)

		removeServiceOK, err := client.Default.ManagementService.RemoveService(&mservice.RemoveServiceParams{
			ServiceID:   serviceID,
			ServiceType: new(types.ServiceTypeMySQLService),
			Context:     pmmapitests.Context,
		})
		assert.Nil(t, removeServiceOK)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "wrong service type")
	})
}
