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
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

// AgentStatusUnknown means agent is not connected and we don't know anything about its status.
var AgentStatusUnknown = inventorypb.AgentStatus_name[int32(inventorypb.AgentStatus_UNKNOWN)]

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

		service := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		mySqldExporter := addMySQLdExporter(t, agents.AddMySQLdExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,

			SkipConnectionCheck: true,
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, mySqldExporterID)

		res, err := client.Default.Agents.ListAgents(&agents.ListAgentsParams{Context: pmmapitests.Context})
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

		service := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, "MySQL Service for filter test"),
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		mySqldExporter := addMySQLdExporter(t, agents.AddMySQLdExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,

			SkipConnectionCheck: true,
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, mySqldExporterID)

		nodeExporter, err := client.Default.Agents.AddNodeExporter(&agents.AddNodeExporterParams{
			Body: agents.AddNodeExporterBody{
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_node_exporter": "node_exporter",
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, nodeExporter)
		nodeExporterID := nodeExporter.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, nodeExporterID)

		// Filter by pmm agent ID.
		res, err := client.Default.Agents.ListAgents(&agents.ListAgentsParams{
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
		res, err = client.Default.Agents.ListAgents(&agents.ListAgentsParams{
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
		res, err = client.Default.Agents.ListAgents(&agents.ListAgentsParams{
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
		res, err = client.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Body:    agents.ListAgentsBody{AgentType: pointer.ToString(agents.ListAgentsBodyAgentTypeMYSQLDEXPORTER)},
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

		res, err := client.Default.Agents.ListAgents(&agents.ListAgentsParams{
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

		serviceID := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      nodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, ""),
		}).Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		_, err := client.Default.Agents.AddMongoDBExporter(&agents.AddMongoDBExporterParams{
			Body: agents.AddMongoDBExporterBody{
				ServiceID:           serviceID,
				Username:            "username",
				Password:            "password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
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

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				PMMAgent: &agents.GetAgentOKBodyPMMAgent{
					AgentID:      agentID,
					RunsOnNodeID: nodeID,
				},
			},
		}, getAgentRes)

		params := &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: agentID,
			},
			Context: context.Background(),
		}
		removeAgentOK, err := client.Default.Agents.RemoveAgent(params)
		assert.NoError(t, err)
		assert.NotNil(t, removeAgentOK)
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.Agents.AddPMMAgent(&agents.AddPMMAgentParams{
			Body:    agents.AddPMMAgentBody{RunsOnNodeID: ""},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPMMAgentRequest.RunsOnNodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.PMMAgent.AgentID)
		}
	})

	t.Run("Remove pmm-agent with agents", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for PMM-agent"))
		nodeID := node.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      nodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, "MySQL Service for remove pmm-agent test"),
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgentOKBody := pmmapitests.AddPMMAgent(t, nodeID)
		require.Equal(t, nodeID, pmmAgentOKBody.PMMAgent.RunsOnNodeID)
		pmmAgentID := pmmAgentOKBody.PMMAgent.AgentID

		nodeExporterOK := addNodeExporter(t, pmmAgentID, make(map[string]string))
		nodeExporterID := nodeExporterOK.Payload.NodeExporter.AgentID

		mySqldExporter := addMySQLdExporter(t, agents.AddMySQLdExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"custom_label_mysql_exporter": "mysql_exporter",
			},

			SkipConnectionCheck: true,
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID

		params := &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: pmmAgentID,
			},
			Context: context.Background(),
		}
		res, err := client.Default.Agents.RemoveAgent(params)
		assert.Nil(t, res)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `pmm-agent with ID %q has agents.`, pmmAgentID)

		// Check that agents aren't removed.
		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: pmmAgentID},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				PMMAgent: &agents.GetAgentOKBodyPMMAgent{
					AgentID:      pmmAgentID,
					RunsOnNodeID: nodeID,
				},
			},
		}, getAgentRes)

		listAgentsOK, err := client.Default.Agents.ListAgents(&agents.ListAgentsParams{
			Body: agents.ListAgentsBody{
				PMMAgentID: pmmAgentID,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ListAgentsOKBody{
			NodeExporter: []*agents.ListAgentsOKBodyNodeExporterItems0{
				{
					PMMAgentID: pmmAgentID,
					AgentID:    nodeExporterID,
					Status:     &AgentStatusUnknown,
				},
			},
			MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
				{
					PMMAgentID: pmmAgentID,
					AgentID:    mySqldExporterID,
					ServiceID:  serviceID,
					Username:   "username",
					CustomLabels: map[string]string{
						"custom_label_mysql_exporter": "mysql_exporter",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, listAgentsOK.Payload)

		// Remove with force flag.
		params = &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: pmmAgentID,
				Force:   true,
			},
			Context: context.Background(),
		}
		res, err = client.Default.Agents.RemoveAgent(params)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Check that agents are removed.
		getAgentRes, err = client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: pmmAgentID},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID %q not found.", pmmAgentID)
		assert.Nil(t, getAgentRes)

		listAgentsOK, err = client.Default.Agents.ListAgents(&agents.ListAgentsParams{
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
		res, err := client.Default.Agents.RemoveAgent(params)
		assert.Nil(t, res)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Agent with ID %q not found.`, agentID)
	})

	t.Run("Remove with empty params", func(t *testing.T) {
		t.Parallel()

		removeResp, err := client.Default.Agents.RemoveAgent(&agents.RemoveAgentParams{
			Body:    agents.RemoveAgentBody{},
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RemoveAgentRequest.AgentId: value length must be at least 1 runes")
		assert.Nil(t, removeResp)
	})

	t.Run("Remove pmm-agent on PMM Server", func(t *testing.T) {
		t.Parallel()

		removeResp, err := client.Default.Agents.RemoveAgent(&agents.RemoveAgentParams{
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

		service := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, "MySQL Service for QanAgent test"),
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.Agents.AddQANMySQLPerfSchemaAgent(
			&agents.AddQANMySQLPerfSchemaAgentParams{
				Body: agents.AddQANMySQLPerfSchemaAgentBody{
					ServiceID:  serviceID,
					Username:   "username",
					Password:   "password",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlPerfschemaAgent",
					},

					SkipConnectionCheck: true,
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANMysqlPerfschemaAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
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
					Status: &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANMySQLPerfSchemaAgentOK, err := client.Default.Agents.ChangeQANMySQLPerfSchemaAgent(&agents.ChangeQANMySQLPerfSchemaAgentParams{
			Body: agents.ChangeQANMySQLPerfSchemaAgentBody{
				AgentID: agentID,
				Common: &agents.ChangeQANMySQLPerfSchemaAgentParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeQANMySQLPerfSchemaAgentOK{
			Payload: &agents.ChangeQANMySQLPerfSchemaAgentOKBody{
				QANMysqlPerfschemaAgent: &agents.ChangeQANMySQLPerfSchemaAgentOKBodyQANMysqlPerfschemaAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   true,
					Status:     &AgentStatusUnknown,
				},
			},
		}, changeQANMySQLPerfSchemaAgentOK)

		changeQANMySQLPerfSchemaAgentOK, err = client.Default.Agents.ChangeQANMySQLPerfSchemaAgent(&agents.ChangeQANMySQLPerfSchemaAgentParams{
			Body: agents.ChangeQANMySQLPerfSchemaAgentBody{
				AgentID: agentID,
				Common: &agents.ChangeQANMySQLPerfSchemaAgentParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlPerfschemaAgent",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeQANMySQLPerfSchemaAgentOK{
			Payload: &agents.ChangeQANMySQLPerfSchemaAgentOKBody{
				QANMysqlPerfschemaAgent: &agents.ChangeQANMySQLPerfSchemaAgentOKBodyQANMysqlPerfschemaAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlPerfschemaAgent",
					},
					Status: &AgentStatusUnknown,
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

		res, err := client.Default.Agents.AddQANMySQLPerfSchemaAgent(&agents.AddQANMySQLPerfSchemaAgentParams{
			Body: agents.AddQANMySQLPerfSchemaAgentBody{
				ServiceID:  "",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",

				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLPerfSchemaAgentRequest.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddQANMySQLPerfSchemaAgent(&agents.AddQANMySQLPerfSchemaAgentParams{
			Body: agents.AddQANMySQLPerfSchemaAgentBody{
				ServiceID:  serviceID,
				PMMAgentID: "",
				Username:   "username",
				Password:   "password",

				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLPerfSchemaAgentRequest.PmmAgentId: value length must be at least 1 runes")
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

		res, err := client.Default.Agents.AddQANMySQLPerfSchemaAgent(&agents.AddQANMySQLPerfSchemaAgentParams{
			Body: agents.AddQANMySQLPerfSchemaAgentBody{
				ServiceID:  "pmm-service-id",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",
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

		service := addMySQLService(t, services.AddMySQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        3306,
			ServiceName: pmmapitests.TestString(t, "MySQL Service for not exists node ID"),
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddQANMySQLPerfSchemaAgent(&agents.AddQANMySQLPerfSchemaAgentParams{
			Body: agents.AddQANMySQLPerfSchemaAgentBody{
				ServiceID:  serviceID,
				PMMAgentID: "pmm-not-exist-server",
				Username:   "username",
				Password:   "password",
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatementsAgent(
			&agents.AddQANPostgreSQLPgStatementsAgentParams{
				Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
					ServiceID:  serviceID,
					Username:   "username",
					Password:   "password",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatementsAgent",
					},

					SkipConnectionCheck: true,
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatementsAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
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
					Status: &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANPostgreSQLPgStatementsAgentOK, err := client.Default.Agents.ChangeQANPostgreSQLPgStatementsAgent(&agents.ChangeQANPostgreSQLPgStatementsAgentParams{
			Body: agents.ChangeQANPostgreSQLPgStatementsAgentBody{
				AgentID: agentID,
				Common: &agents.ChangeQANPostgreSQLPgStatementsAgentParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeQANPostgreSQLPgStatementsAgentOK{
			Payload: &agents.ChangeQANPostgreSQLPgStatementsAgentOKBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeQANPostgreSQLPgStatementsAgentOKBodyQANPostgresqlPgstatementsAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   true,
					Status:     &AgentStatusUnknown,
				},
			},
		}, changeQANPostgreSQLPgStatementsAgentOK)

		changeQANPostgreSQLPgStatementsAgentOK, err = client.Default.Agents.ChangeQANPostgreSQLPgStatementsAgent(&agents.ChangeQANPostgreSQLPgStatementsAgentParams{
			Body: agents.ChangeQANPostgreSQLPgStatementsAgentBody{
				AgentID: agentID,
				Common: &agents.ChangeQANPostgreSQLPgStatementsAgentParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatementsAgent",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeQANPostgreSQLPgStatementsAgentOK{
			Payload: &agents.ChangeQANPostgreSQLPgStatementsAgentOKBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeQANPostgreSQLPgStatementsAgentOKBodyQANPostgresqlPgstatementsAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatementsAgent",
					},
					Status: &AgentStatusUnknown,
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

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatementsAgent(&agents.AddQANPostgreSQLPgStatementsAgentParams{
			Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
				ServiceID:  "",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",

				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatementsAgentRequest.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatementsAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatementsAgent(&agents.AddQANPostgreSQLPgStatementsAgentParams{
			Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
				ServiceID:  serviceID,
				PMMAgentID: "",
				Username:   "username",
				Password:   "password",

				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatementsAgentRequest.PmmAgentId: value length must be at least 1 runes")
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

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatementsAgent(&agents.AddQANPostgreSQLPgStatementsAgentParams{
			Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
				ServiceID:  "pmm-service-id",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatementsAgent(&agents.AddQANPostgreSQLPgStatementsAgentParams{
			Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
				ServiceID:  serviceID,
				PMMAgentID: "pmm-not-exist-server",
				Username:   "username",
				Password:   "password",
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(
			&agents.AddQANPostgreSQLPgStatMonitorAgentParams{
				Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
					ServiceID:  serviceID,
					Username:   "username",
					Password:   "password",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},

					SkipConnectionCheck: true,
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
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
					Status: &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANPostgreSQLPgStatMonitorAgentOK, err := client.Default.Agents.ChangeQANPostgreSQLPgStatMonitorAgent(&agents.ChangeQANPostgreSQLPgStatMonitorAgentParams{
			Body: agents.ChangeQANPostgreSQLPgStatMonitorAgentBody{
				AgentID: agentID,
				Common: &agents.ChangeQANPostgreSQLPgStatMonitorAgentParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeQANPostgreSQLPgStatMonitorAgentOK{
			Payload: &agents.ChangeQANPostgreSQLPgStatMonitorAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeQANPostgreSQLPgStatMonitorAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   true,
					Status:     &AgentStatusUnknown,
				},
			},
		}, changeQANPostgreSQLPgStatMonitorAgentOK)

		changeQANPostgreSQLPgStatMonitorAgentOK, err = client.Default.Agents.ChangeQANPostgreSQLPgStatMonitorAgent(&agents.ChangeQANPostgreSQLPgStatMonitorAgentParams{
			Body: agents.ChangeQANPostgreSQLPgStatMonitorAgentBody{
				AgentID: agentID,
				Common: &agents.ChangeQANPostgreSQLPgStatMonitorAgentParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeQANPostgreSQLPgStatMonitorAgentOK{
			Payload: &agents.ChangeQANPostgreSQLPgStatMonitorAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeQANPostgreSQLPgStatMonitorAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, changeQANPostgreSQLPgStatMonitorAgentOK)
	})

	t.Run("BasicWithDisabledExamples", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan PostgreSQL Agent pg_stat_monitor")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(
			&agents.AddQANPostgreSQLPgStatMonitorAgentParams{
				Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
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
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
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
					Status: &AgentStatusUnknown,
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

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(&agents.AddQANPostgreSQLPgStatMonitorAgentParams{
			Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
				ServiceID:  "",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",

				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatMonitorAgentRequest.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(&agents.AddQANPostgreSQLPgStatMonitorAgentParams{
			Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
				ServiceID:  serviceID,
				PMMAgentID: "",
				Username:   "username",
				Password:   "password",

				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatMonitorAgentRequest.PmmAgentId: value length must be at least 1 runes")
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

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(&agents.AddQANPostgreSQLPgStatMonitorAgentParams{
			Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
				ServiceID:  "pmm-service-id",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(&agents.AddQANPostgreSQLPgStatMonitorAgentParams{
			Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
				ServiceID:  serviceID,
				PMMAgentID: "pmm-not-exist-server",
				Username:   "username",
				Password:   "password",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})
}

func TestMetricsResolutionsChange(t *testing.T) {
	t.Parallel()

	genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
	require.NotEmpty(t, genericNodeID)
	defer pmmapitests.RemoveNodes(t, genericNodeID)

	node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
	nodeID := node.Remote.NodeID
	defer pmmapitests.RemoveNodes(t, nodeID)

	service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
		NodeID:      genericNodeID,
		Address:     "localhost",
		Port:        5432,
		ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
	})
	serviceID := service.Postgresql.ServiceID
	defer pmmapitests.RemoveServices(t, serviceID)

	pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
	pmmAgentID := pmmAgent.PMMAgent.AgentID
	defer pmmapitests.RemoveAgents(t, pmmAgentID)

	PostgresExporter := addPostgresExporter(t, agents.AddPostgresExporterBody{
		ServiceID:  serviceID,
		Username:   "username",
		Password:   "password",
		PMMAgentID: pmmAgentID,
		CustomLabels: map[string]string{
			"custom_label_postgres_exporter": "postgres_exporter",
		},

		SkipConnectionCheck: true,
	})
	agentID := PostgresExporter.PostgresExporter.AgentID
	defer pmmapitests.RemoveAgents(t, agentID)

	getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
		Body:    agents.GetAgentBody{AgentID: agentID},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, &agents.GetAgentOK{
		Payload: &agents.GetAgentOKBody{
			PostgresExporter: &agents.GetAgentOKBodyPostgresExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				Status: &AgentStatusUnknown,
			},
		},
	}, getAgentRes)

	// Change metrics resolutions
	changePostgresExporterOK, err := client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
		Body: agents.ChangePostgresExporterBody{
			AgentID: agentID,
			Common: &agents.ChangePostgresExporterParamsBodyCommon{
				MetricsResolutions: &agents.ChangePostgresExporterParamsBodyCommonMetricsResolutions{
					Hr: "600s",
					Mr: "300s",
					Lr: "100s",
				},
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, &agents.ChangePostgresExporterOK{
		Payload: &agents.ChangePostgresExporterOKBody{
			PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				Status: &AgentStatusUnknown,
				MetricsResolutions: &agents.ChangePostgresExporterOKBodyPostgresExporterMetricsResolutions{
					Hr: "600s",
					Mr: "300s",
					Lr: "100s",
				},
			},
		},
	}, changePostgresExporterOK)

	// Reset part of metrics resolutions
	changePostgresExporterOK, err = client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
		Body: agents.ChangePostgresExporterBody{
			AgentID: agentID,
			Common: &agents.ChangePostgresExporterParamsBodyCommon{
				MetricsResolutions: &agents.ChangePostgresExporterParamsBodyCommonMetricsResolutions{
					Hr: "600s",
					Mr: "300s",
					Lr: "0s",
				},
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, &agents.ChangePostgresExporterOK{
		Payload: &agents.ChangePostgresExporterOKBody{
			PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				Status: &AgentStatusUnknown,
				MetricsResolutions: &agents.ChangePostgresExporterOKBodyPostgresExporterMetricsResolutions{
					Hr: "600s",
					Mr: "300s",
				},
			},
		},
	}, changePostgresExporterOK)

	// Change part of metrics resolutions
	changePostgresExporterOK, err = client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
		Body: agents.ChangePostgresExporterBody{
			AgentID: agentID,
			Common: &agents.ChangePostgresExporterParamsBodyCommon{
				MetricsResolutions: &agents.ChangePostgresExporterParamsBodyCommonMetricsResolutions{
					Hr: "500s",
				},
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, &agents.ChangePostgresExporterOK{
		Payload: &agents.ChangePostgresExporterOKBody{
			PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				Status: &AgentStatusUnknown,
				MetricsResolutions: &agents.ChangePostgresExporterOKBodyPostgresExporterMetricsResolutions{
					Hr: "500s",
					Mr: "300s",
				},
			},
		},
	}, changePostgresExporterOK)
}
