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
	"github.com/percona/pmm/api/inventory/v1/types"
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
		require.NotEmpty(t, res.Payload.MysqldExporter, "There should be at least one service")

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
		require.NoError(t, err)
		require.NotNil(t, nodeExporter)
		nodeExporterID := nodeExporter.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, nodeExporterID)

		// Filter by pmm agent ID.
		res, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				PMMAgentID: pointer.ToString(pmmAgentID),
				Context:    pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.MysqldExporter, "There should be at least one agent")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertNodeExporterExists(t, res, nodeExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)

		// Filter by node ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				NodeID:  pointer.ToString(nodeID),
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.NodeExporter, "There should be at least one node exporter")
		assertMySQLExporterNotExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterExists(t, res, nodeExporterID)

		// Filter by service ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				ServiceID: pointer.ToString(serviceID),
				Context:   pmmapitests.Context,
			})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.MysqldExporter, "There should be at least one mysql exporter")
		assertMySQLExporterExists(t, res, mySqldExporterID)
		assertPMMAgentNotExists(t, res, pmmAgentID)
		assertNodeExporterNotExists(t, res, nodeExporterID)

		// Filter by service ID.
		res, err = client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				AgentType: pointer.ToString(types.AgentTypeMySQLdExporter),
				Context:   pmmapitests.Context,
			})
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
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.ListAgents(
			&agents.ListAgentsParams{
				PMMAgentID: pointer.ToString(pmmAgentID),
				NodeID:     pointer.ToString(genericNodeID),
				ServiceID:  pointer.ToString("some-service-id"),
				Context:    pmmapitests.Context,
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
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
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
			})
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
				PMMAgentID: pointer.ToString(pmmAgentID),
				Context:    pmmapitests.Context,
			})
		require.NoError(t, err)
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
			AgentID: pmmAgentID,
			Force:   pointer.ToBool(true),
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
			})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID %s not found.", pmmAgentID)
		assert.Nil(t, getAgentRes)

		listAgentsOK, err = client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			PMMAgentID: pointer.ToString(pmmAgentID),
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
				Force:   pointer.ToBool(true),
				Context: context.Background(),
			})
		pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "pmm-agent on PMM Server can't be removed.")
		assert.Nil(t, removeResp)
	})
}

func TestMetricsResolutionsChange(t *testing.T) {
	t.Parallel()

	genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Generic node")).NodeID
	require.NotEmpty(t, genericNodeID)
	defer pmmapitests.RemoveNodes(t, genericNodeID)

	node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
	nodeID := node.Remote.NodeID
	defer pmmapitests.RemoveNodes(t, nodeID)

	service := addService(t, services.AddServiceBody{
		Postgresql: &services.AddServiceParamsBodyPostgresql{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
		},
	})
	serviceID := service.Postgresql.ServiceID
	defer pmmapitests.RemoveServices(t, serviceID)

	pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
	pmmAgentID := pmmAgent.PMMAgent.AgentID
	defer pmmapitests.RemoveAgents(t, pmmAgentID)

	res, err := client.Default.AgentsService.AddAgent(
		&agents.AddAgentParams{
			Body: agents.AddAgentBody{
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
			},
			Context: pmmapitests.Context,
		})
	require.NoError(t, err)
	agentID := res.Payload.PostgresExporter.AgentID
	defer pmmapitests.RemoveAgents(t, agentID)

	getAgentRes, err := client.Default.AgentsService.GetAgent(
		&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
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
		LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
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
		})
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
		LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
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
		})
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
		LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
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
		})
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
		LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
		DisabledCollectors: []string{},
		MetricsResolutions: &agents.ChangeAgentOKBodyPostgresExporterMetricsResolutions{
			Hr: "500s",
			Mr: "300s",
		},
	}, changePostgresExporterOK.Payload.PostgresExporter)
}
