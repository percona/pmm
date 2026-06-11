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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

// AgentStatusUnknown means agent is not connected and we don't know anything about its status.
var AgentStatusUnknown = inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN)]

// AgentStatusDone means the agent has either been stopped or disabled.
var AgentStatusDone = inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_DONE)]

func TestAgents(t *testing.T) {
	t.Parallel()
	t.Run("List", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for agents list")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for agents list")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		mySqldExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:           serviceID,
				Username:            "username",
				Password:            "password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID

		// Use filtered calls to avoid a TOCTOU race: an unfiltered ListAgents iterates over
		// all agents in the DB and converts them one by one; between the query and the
		// conversion loop a parallel test may delete a pmm_agent that an external exporter
		// (created with push_metrics) still references, causing a spurious 404.
		resByAgent, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			PMMAgentID: new(pmmAgentID),
			Context:    pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, resByAgent)
		require.NotEmpty(t, resByAgent.Payload.MysqldExporter, "There should be at least one service")
		assertMySQLExporterExists(t, resByAgent, mySqldExporterID)

		// pmmAgents use runs_on_node_id (not node_id), so no NodeID filter returns them.
		// Filter by agent type instead: pmmAgent conversion has no secondary DB lookups,
		// so it is immune to the TOCTOU race that affects external exporters.
		resByType, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			AgentType: new(types.AgentTypePMMAgent),
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, resByType)
		assertPMMAgentExists(t, resByType, pmmAgentID)
	})

	t.Run("FilterList", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for agents filters")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for agents filters")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for filter test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		mySqldExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,

				SkipConnectionCheck: true,
			},
		})
		mySqldExporterID := mySqldExporter.MysqldExporter.AgentID

		nodeExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_node_exporter": "node_exporter",
				},
			},
		})
		nodeExporterID := nodeExporter.NodeExporter.AgentID

		// Filter by pmm agent ID.
		res, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				PMMAgentID: new(pmmAgentID),
				Context:    pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.MysqldExporter, "There should be at least one agent")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertNodeExporterExists(t, res, nodeExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)

		// Filter by node ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				NodeID:  new(nodeID),
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.NodeExporter, "There should be at least one node exporter")
		assertMySQLExporterNotExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterExists(t, res, nodeExporterID)

		// Filter by service ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				ServiceID: new(serviceID),
				Context:   pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.MysqldExporter, "There should be at least one mysql exporter")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterNotExists(t, res, nodeExporterID)

		// Filter by agent type, scoped to this test's pmm-agent to avoid the 404
		// race an unscoped type filter hits (see the List subtest).
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				PMMAgentID: new(pmmAgentID),
				AgentType:  new(types.AgentTypeMySQLdExporter),
				Context:    pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.MysqldExporter, "There should be at least one mysql exporter")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterNotExists(t, res, nodeExporterID)
	})

	t.Run("TwoOrMoreFilters", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				PMMAgentID: new(pmmAgentID),
				NodeID:     new(genericNodeID),
				ServiceID:  new("some-service-id"),
				Context:    pmmapitests.Context,
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected at most one param: pmm_agent_id, node_id or service_id")
		assert.Nil(t, res)
	})

	t.Run("AddWithInvalidType", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		serviceID := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      nodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, ""),
			},
		}).Mysql.ServiceID

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
			},
		)

		pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.FailedPrecondition, "invalid combination of service type mysql and agent type mongodb_exporter")
	})
}

func TestPMMAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for PMM-agent")).NodeID
		agentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
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
			AgentID: agentID,
			Context: context.Background(),
		}
		removeAgentOK, err := client.Default.AgentsService.RemoveAgent(params)
		require.NoError(t, err)
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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPMMAgentParams.RunsOnNodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.PMMAgent.AgentID)
		}
	})

	t.Run("Remove pmm-agent with agents", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node for PMM-agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      nodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for remove pmm-agent test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID
		nodeExporterID := pmmapitests.AddNodeExporter(t, pmmAgentID, make(map[string]string)).AgentID

		mySqldExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
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
			AgentID: pmmAgentID,
			Context: context.Background(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		assert.Nil(t, res)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `pmm-agent with ID %s has agents.`, pmmAgentID)

		// Check that agents aren't removed.
		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: pmmAgentID,
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
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
				PMMAgentID: new(pmmAgentID),
				Context:    pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, []*agents.ListAgentsOKBodyNodeExporterItems0{
			{
				PMMAgentID:         pmmAgentID,
				AgentID:            nodeExporterID,
				Status:             &AgentStatusUnknown,
				CustomLabels:       map[string]string{},
				DisabledCollectors: make([]string, 0),
				LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
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
				LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				ExtraDsnParams:     map[string]string{},
				DisabledCollectors: make([]string, 0),
			},
		}, listAgentsOK.Payload.MysqldExporter)

		// Remove with force flag.
		params = &agents.RemoveAgentParams{
			AgentID: pmmAgentID,
			Force:   new(true),
			Context: context.Background(),
		}
		res, err = client.Default.AgentsService.RemoveAgent(params)
		require.NoError(t, err)
		assert.NotNil(t, res)

		// Check that agents are removed.
		getAgentRes, err = client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: pmmAgentID,
				Context: pmmapitests.Context,
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID %s not found.", pmmAgentID)
		assert.Nil(t, getAgentRes)

		listAgentsOK, err = client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			PMMAgentID: new(pmmAgentID),
			Context:    pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID %s not found.", pmmAgentID)
		assert.Nil(t, listAgentsOK)
	})

	t.Run("Remove not-exist agent", func(t *testing.T) {
		t.Parallel()

		agentID := "not-exist-pmm-agent"
		params := &agents.RemoveAgentParams{
			AgentID: agentID,
			Context: context.Background(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		assert.Nil(t, res)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Agent with ID %s not found.`, agentID)
	})

	t.Run("Remove with empty params", func(t *testing.T) {
		t.Parallel()

		removeResp, err := client.Default.AgentsService.RemoveAgent(&agents.RemoveAgentParams{
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RemoveAgentRequest.AgentId: value length must be at least 1 runes")
		assert.Nil(t, removeResp)
	})

	t.Run("Remove pmm-agent on PMM Server", func(t *testing.T) {
		t.Parallel()

		removeResp, err := client.Default.AgentsService.RemoveAgent(
			&agents.RemoveAgentParams{
				AgentID: "pmm-server",
				Force:   new(true),
				Context: context.Background(),
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "pmm-agent on PMM Server can't be removed.")
		assert.Nil(t, removeResp)
	})
}

func TestQanAgentExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QanAgent test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
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
		})
		agentID := res.QANMysqlPerfschemaAgent.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
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
					Status:         &AgentStatusUnknown,
					LogLevel:       new("LOG_LEVEL_UNSPECIFIED"),
					ExtraDsnParams: map[string]string{},
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANMySQLPerfSchemaAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentOKBodyQANMysqlPerfschemaAgent{
					AgentID:        agentID,
					ServiceID:      serviceID,
					Username:       "username",
					PMMAgentID:     pmmAgentID,
					Disabled:       true,
					Status:         &AgentStatusDone,
					CustomLabels:   map[string]string{},
					ExtraDsnParams: map[string]string{},
					LogLevel:       new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANMySQLPerfSchemaAgentOK)

		changeQANMySQLPerfSchemaAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCustomLabels{
							Values: map[string]string{
								"new_label": "QANMysqlPerfschemaAgent",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
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
					Status:         &AgentStatusDone,
					LogLevel:       new("LOG_LEVEL_UNSPECIFIED"),
					ExtraDsnParams: map[string]string{},
				},
			},
		}, changeQANMySQLPerfSchemaAgentOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLPerfSchemaAgentParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
			},
		})
		serviceID := service.Mysql.ServiceID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLPerfSchemaAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for not exists node ID"),
			},
		})
		serviceID := service.Mysql.ServiceID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlPerfschemaAgent.AgentID)
		}
	})
}

func TestMetricsResolutionsChange(t *testing.T) {
	t.Parallel()

	genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node")).NodeID
	nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter")).NodeID

	service := pmmapitests.AddService(t, services.AddServiceBody{
		Postgresql: &services.AddServiceParamsBodyPostgresql{
			NodeID:      genericNodeID,
			Address:     pmmapitests.TestString(t, "localhost"),
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
		},
	})
	serviceID := service.Postgresql.ServiceID
	pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

	res := pmmapitests.AddAgent(t, agents.AddAgentBody{
		PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"custom_label_postgres_exporter": "postgres_exporter",
			},
			SkipConnectionCheck: true,
		},
	})
	agentID := res.PostgresExporter.AgentID

	getAgentRes, err := client.Default.AgentsService.GetAgent(
		&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &agents.GetAgentOKBodyPostgresExporter{
		AgentID:    agentID,
		ServiceID:  serviceID,
		Username:   "username",
		PMMAgentID: pmmAgentID,
		CustomLabels: map[string]string{
			"custom_label_postgres_exporter": "postgres_exporter",
		},
		Status:             &AgentStatusUnknown,
		LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
		DisabledCollectors: []string{},
	}, getAgentRes.Payload.PostgresExporter)

	// Change metrics resolutions
	changePostgresExporterOK, err := client.Default.AgentsService.ChangeAgent(
		&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					MetricsResolutions: &agents.ChangeAgentParamsBodyPostgresExporterMetricsResolutions{
						Hr: "600s",
						Mr: "300s",
						Lr: "100s",
					},
				},
			},
			Context: pmmapitests.Context,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &agents.ChangeAgentOKBodyPostgresExporter{
		AgentID:    agentID,
		ServiceID:  serviceID,
		Username:   "username",
		PMMAgentID: pmmAgentID,
		CustomLabels: map[string]string{
			"custom_label_postgres_exporter": "postgres_exporter",
		},
		Status:             &AgentStatusUnknown,
		LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
		DisabledCollectors: []string{},
		MetricsResolutions: &agents.ChangeAgentOKBodyPostgresExporterMetricsResolutions{
			Hr: "600s",
			Mr: "300s",
			Lr: "100s",
		},
	}, changePostgresExporterOK.Payload.PostgresExporter)

	// Reset part of metrics resolutions
	changePostgresExporterOK, err = client.Default.AgentsService.ChangeAgent(
		&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					MetricsResolutions: &agents.ChangeAgentParamsBodyPostgresExporterMetricsResolutions{
						Hr: "600s",
						Mr: "300s",
						Lr: "0s",
					},
				},
			},
			Context: pmmapitests.Context,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &agents.ChangeAgentOKBodyPostgresExporter{
		AgentID:    agentID,
		ServiceID:  serviceID,
		Username:   "username",
		PMMAgentID: pmmAgentID,
		CustomLabels: map[string]string{
			"custom_label_postgres_exporter": "postgres_exporter",
		},
		Status:             &AgentStatusUnknown,
		LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
		DisabledCollectors: []string{},
		MetricsResolutions: &agents.ChangeAgentOKBodyPostgresExporterMetricsResolutions{
			Hr: "600s",
			Mr: "300s",
		},
	}, changePostgresExporterOK.Payload.PostgresExporter)

	// Change part of metrics resolutions
	changePostgresExporterOK, err = client.Default.AgentsService.ChangeAgent(
		&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					MetricsResolutions: &agents.ChangeAgentParamsBodyPostgresExporterMetricsResolutions{
						Hr: "500s",
					},
				},
			},
			Context: pmmapitests.Context,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &agents.ChangeAgentOKBodyPostgresExporter{
		AgentID:    agentID,
		ServiceID:  serviceID,
		Username:   "username",
		PMMAgentID: pmmAgentID,
		CustomLabels: map[string]string{
			"custom_label_postgres_exporter": "postgres_exporter",
		},
		Status:             &AgentStatusUnknown,
		LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
		DisabledCollectors: []string{},
		MetricsResolutions: &agents.ChangeAgentOKBodyPostgresExporterMetricsResolutions{
			Hr: "500s",
			Mr: "300s",
		},
	}, changePostgresExporterOK.Payload.PostgresExporter)
}
