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

package inventory

import (
	"context"
	"net/http"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

// AgentStatusUnknown means agent is not connected and we don't know anything about its status.
var AgentStatusUnknown = inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN)]

func TestAgents(t *testing.T) {
	t.Parallel()
	t.Run("List", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for agents list")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for agents list"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		mySqldExporter := addAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:           serviceID,
				Username:            "username",
				Password:            "password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, mySqldExporterID)

		res, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{Context: pmmapitests.Context})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotZerof(t, len(res.Payload.MysqldExporter), "There should be at least one service")

		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertPMMAgentExists(t, res, pmmAgentID)
	})

	t.Run("FilterList", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for agents filters")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for agents filters"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for filter test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		mySqldExporter := addAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,

				SkipConnectionCheck: true,
			},
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, mySqldExporterID)

		nodeExporter, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
						PMMAgentID: pmmAgentID,
						CustomLabels: map[string]string{
							"custom_label_node_exporter": "node_exporter",
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		require.NotNil(t, nodeExporter)
		nodeExporterID := nodeExporter.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, nodeExporterID)

		// Filter by pmm agent ID.
		res, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				Body:    agents.ListAgentsBody{PMMAgentID: pmmAgentID},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotZerof(t, len(res.Payload.MysqldExporter), "There should be at least one agent")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertNodeExporterExists(t, res, nodeExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)

		// Filter by node ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				Body:    agents.ListAgentsBody{NodeID: nodeID},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotZerof(t, len(res.Payload.NodeExporter), "There should be at least one node exporter")
		assertMySQLExporterNotExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterExists(t, res, nodeExporterID)

		// Filter by service ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				Body:    agents.ListAgentsBody{ServiceID: serviceID},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotZerof(t, len(res.Payload.MysqldExporter), "There should be at least one mysql exporter")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterNotExists(t, res, nodeExporterID)

		// Filter by service ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				Body:    agents.ListAgentsBody{AgentType: pointer.ToString(agents.ListAgentsBodyAgentTypeAGENTTYPEMYSQLDEXPORTER)},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotZerof(t, len(res.Payload.MysqldExporter), "There should be at least one mysql exporter")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterNotExists(t, res, nodeExporterID)
	})

	t.Run("TwoOrMoreFilters", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				Body: agents.ListAgentsBody{
					PMMAgentID: pmmAgentID,
					NodeID:     genericNodeID,
					ServiceID:  "some-service-id",
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected at most one param: pmm_agent_id, node_id or service_id")
		assert.Nil(t, res)
	})

	t.Run("AddWithInvalidType", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, nodeID)
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		serviceID := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      nodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, ""),
			},
		}).Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		_, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
						ServiceID:           serviceID,
						Username:            "username",
						Password:            "password",
						PMMAgentID:          pmmAgentID,
						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})

		pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.FailedPrecondition, "invalid combination of service type mysql and agent type mongodb_exporter")
	})
}

func TestPMMAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for PMM-agent"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		res := pmmapitests.AddPMMAgent(t, nodeID)
		require.Equal(t, nodeID, res.PMMAgent.RunsOnNodeID)
		agentID := res.PMMAgent.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				Body:    agents.GetAgentBody{AgentID: agentID},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				PMMAgent: &agents.GetAgentOKBodyPMMAgent{
					AgentID:      agentID,
					RunsOnNodeID: nodeID,
					CustomLabels: map[string]string{},
				},
			},
		}, getAgentRes)

		params := &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: agentID,
			},
			Context: context.Background(),
		}
		removeAgentOK, err := client.Default.AgentsService.RemoveAgent(params)
		assert.NoError(t, err)
		assert.NotNil(t, removeAgentOK)
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					PMMAgent: &agents.AddAgentParamsBodyPMMAgent{
						RunsOnNodeID: "",
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPMMAgentParams.RunsOnNodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.PMMAgent.AgentID)
		}
	})

	t.Run("Remove pmm-agent with agents", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for PMM-agent"))
		nodeID := node.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      nodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for remove pmm-agent test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgentOKBody := pmmapitests.AddPMMAgent(t, nodeID)
		require.Equal(t, nodeID, pmmAgentOKBody.PMMAgent.RunsOnNodeID)
		pmmAgentID := pmmAgentOKBody.PMMAgent.AgentID

		nodeExporterOK := addNodeExporter(t, pmmAgentID, make(map[string]string))
		nodeExporterID := nodeExporterOK.Payload.NodeExporter.AgentID

		mySqldExporter := addAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},

				SkipConnectionCheck: true,
			},
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID

		params := &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: pmmAgentID,
			},
			Context: context.Background(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		assert.Nil(t, res)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `pmm-agent with ID %q has agents.`, pmmAgentID)

		// Check that agents aren't removed.
		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				Body:    agents.GetAgentBody{AgentID: pmmAgentID},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				PMMAgent: &agents.GetAgentOKBodyPMMAgent{
					AgentID:      pmmAgentID,
					RunsOnNodeID: nodeID,
					CustomLabels: map[string]string{},
				},
			},
		}, getAgentRes)

		listAgentsOK, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				Body: agents.ListAgentsBody{
					PMMAgentID: pmmAgentID,
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyNodeExporterItems0{
			{
				PMMAgentID:         pmmAgentID,
				AgentID:            nodeExporterID,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		},
			listAgentsOK.Payload.NodeExporter)
		assert.Equal(t, []*agents.ListAgentsOKBodyMysqldExporterItems0{
			{
				PMMAgentID: pmmAgentID,
				AgentID:    mySqldExporterID,
				ServiceID:  serviceID,
				Username:   "username",
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},
				Status:             &AgentStatusUnknown,
				LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				DisabledCollectors: make([]string, 0),
			},
		}, listAgentsOK.Payload.MysqldExporter)

		// Remove with force flag.
		params = &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: pmmAgentID,
				Force:   true,
			},
			Context: context.Background(),
		}
		res, err = client.Default.AgentsService.RemoveAgent(params)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Check that agents are removed.
		getAgentRes, err = client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				Body:    agents.GetAgentBody{AgentID: pmmAgentID},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID %q not found.", pmmAgentID)
		assert.Nil(t, getAgentRes)

		listAgentsOK, err = client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			Body: agents.ListAgentsBody{
				PMMAgentID: pmmAgentID,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID %q not found.", pmmAgentID)
		assert.Nil(t, listAgentsOK)
	})

	t.Run("Remove not-exist agent", func(t *testing.T) {
		t.Parallel()

		agentID := "not-exist-pmm-agent"
		params := &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: agentID,
			},
			Context: context.Background(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		assert.Nil(t, res)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Agent with ID %q not found.`, agentID)
	})

	t.Run("Remove with empty params", func(t *testing.T) {
		t.Parallel()

		removeResp, err := client.Default.AgentsService.RemoveAgent(&agents.RemoveAgentParams{
			Body:    agents.RemoveAgentBody{},
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RemoveAgentRequest.AgentId: value length must be at least 1 runes")
		assert.Nil(t, removeResp)
	})

	t.Run("Remove pmm-agent on PMM Server", func(t *testing.T) {
		t.Parallel()

		removeResp, err := client.Default.AgentsService.RemoveAgent(
			&agents.RemoveAgentParams{
				Body: agents.RemoveAgentBody{
					AgentID: "pmm-server",
					Force:   true,
				},
				Context: context.Background(),
			})
		pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "pmm-agent on PMM Server can't be removed.")
		assert.Nil(t, removeResp)
	})
}

func TestQanAgentExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QanAgent test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
						ServiceID:  serviceID,
						Username:   "username",
						Password:   "password",
						PMMAgentID: pmmAgentID,
						CustomLabels: map[string]string{
							"new_label": "QANMysqlPerfschemaAgent",
						},

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANMysqlPerfschemaAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				QANMysqlPerfschemaAgent: &agents.GetAgentOKBodyQANMysqlPerfschemaAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlPerfschemaAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANMySQLPerfSchemaAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				Body: agents.ChangeAgentBody{
					QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
						AgentID: agentID,
						Common: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCommon{
							Enable:       pointer.ToBool(false),
							CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCommonCustomLabels{},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentOKBodyQANMysqlPerfschemaAgent{
					AgentID:      agentID,
					ServiceID:    serviceID,
					Username:     "username",
					PMMAgentID:   pmmAgentID,
					Disabled:     true,
					Status:       &AgentStatusUnknown,
					CustomLabels: map[string]string{},
					LogLevel:     pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANMySQLPerfSchemaAgentOK)

		changeQANMySQLPerfSchemaAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				Body: agents.ChangeAgentBody{
					QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
						AgentID: agentID,
						Common: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCommon{
							Enable: pointer.ToBool(true),
							CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCommonCustomLabels{
								Values: map[string]string{
									"new_label": "QANMysqlPerfschemaAgent",
								},
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentOKBodyQANMysqlPerfschemaAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlPerfschemaAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANMySQLPerfSchemaAgentOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
						ServiceID:  "",
						PMMAgentID: pmmAgentID,
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLPerfSchemaAgentParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
						ServiceID:  serviceID,
						PMMAgentID: "",
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLPerfSchemaAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
						ServiceID:  "pmm-service-id",
						PMMAgentID: pmmAgentID,
						Username:   "username",
						Password:   "password",
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for not exists node ID"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
						ServiceID:  serviceID,
						PMMAgentID: "pmm-not-exist-server",
						Username:   "username",
						Password:   "password",
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})
}

func TestPGStatStatementsQanAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan PostgreSQL Agent pg_stat_statements")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
						ServiceID:  serviceID,
						Username:   "username",
						Password:   "password",
						PMMAgentID: pmmAgentID,
						CustomLabels: map[string]string{
							"new_label": "QANPostgreSQLPgStatementsAgent",
						},

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatementsAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				Body:    agents.GetAgentBody{AgentID: agentID},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				QANPostgresqlPgstatementsAgent: &agents.GetAgentOKBodyQANPostgresqlPgstatementsAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatementsAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANPostgreSQLPgStatementsAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						AgentID: agentID,
						Common: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCommon{
							Enable:       pointer.ToBool(false),
							CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCommonCustomLabels{},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentOKBodyQANPostgresqlPgstatementsAgent{
					AgentID:      agentID,
					ServiceID:    serviceID,
					Username:     "username",
					PMMAgentID:   pmmAgentID,
					Disabled:     true,
					Status:       &AgentStatusUnknown,
					CustomLabels: map[string]string{},
					LogLevel:     pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANPostgreSQLPgStatementsAgentOK)

		changeQANPostgreSQLPgStatementsAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						AgentID: agentID,
						Common: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCommon{
							Enable: pointer.ToBool(true),
							CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCommonCustomLabels{
								Values: map[string]string{
									"new_label": "QANPostgreSQLPgStatementsAgent",
								},
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentOKBodyQANPostgresqlPgstatementsAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatementsAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANPostgreSQLPgStatementsAgentOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
						ServiceID:  "",
						PMMAgentID: pmmAgentID,
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatementsAgentParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatementsAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
						ServiceID:  serviceID,
						PMMAgentID: "",
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatementsAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatementsAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
						ServiceID:  "pmm-service-id",
						PMMAgentID: pmmAgentID,
						Username:   "username",
						Password:   "password",
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatementsAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
						ServiceID:  serviceID,
						PMMAgentID: "pmm-not-exist-server",
						Username:   "username",
						Password:   "password",
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatementsAgent.AgentID)
		}
	})
}

func TestPGStatMonitorQanAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan PostgreSQL Agent pg_stat_monitor")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						ServiceID:  serviceID,
						Username:   "username",
						Password:   "password",
						PMMAgentID: pmmAgentID,
						CustomLabels: map[string]string{
							"new_label": "QANPostgreSQLPgStatMonitorAgent",
						},

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.GetAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:               agentID,
					ServiceID:             serviceID,
					Username:              "username",
					PMMAgentID:            pmmAgentID,
					QueryExamplesDisabled: false,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANPostgreSQLPgStatMonitorAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						AgentID: agentID,
						Common: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCommon{
							Enable:       pointer.ToBool(false),
							CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCommonCustomLabels{},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:      agentID,
					ServiceID:    serviceID,
					Username:     "username",
					PMMAgentID:   pmmAgentID,
					Disabled:     true,
					Status:       &AgentStatusUnknown,
					CustomLabels: map[string]string{},
					LogLevel:     pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANPostgreSQLPgStatMonitorAgentOK)

		changeQANPostgreSQLPgStatMonitorAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						AgentID: agentID,
						Common: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCommon{
							Enable: pointer.ToBool(true),
							CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCommonCustomLabels{
								Values: map[string]string{
									"new_label": "QANPostgreSQLPgStatMonitorAgent",
								},
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANPostgreSQLPgStatMonitorAgentOK)
	})

	t.Run("BasicWithDisabledExamples", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan PostgreSQL Agent pg_stat_monitor")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						ServiceID:            serviceID,
						Username:             "username",
						Password:             "password",
						PMMAgentID:           pmmAgentID,
						DisableQueryExamples: true,
						CustomLabels: map[string]string{
							"new_label": "QANPostgreSQLPgStatMonitorAgent",
						},

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				Body:    agents.GetAgentBody{AgentID: agentID},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.GetAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:               agentID,
					ServiceID:             serviceID,
					Username:              "username",
					PMMAgentID:            pmmAgentID,
					QueryExamplesDisabled: true,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						ServiceID:  "",
						PMMAgentID: pmmAgentID,
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatMonitorAgentParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						ServiceID:  serviceID,
						PMMAgentID: "",
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatMonitorAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
					ServiceID:  "pmm-service-id",
					PMMAgentID: pmmAgentID,
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						ServiceID:  serviceID,
						PMMAgentID: "pmm-not-exist-server",
						Username:   "username",
						Password:   "password",
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})
}
