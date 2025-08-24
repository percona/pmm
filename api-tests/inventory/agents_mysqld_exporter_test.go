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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

func TestMySQLdExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for MySQLdExporter test"),
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
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},
				SkipConnectionCheck:       true,
				TablestatsGroupTableLimit: 2000,
			},
		})
		assert.EqualValues(t, 0, mySqldExporter.MysqldExporter.TableCount)
		assert.EqualValues(t, 2000, mySqldExporter.MysqldExporter.TablestatsGroupTableLimit)
		agentID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOKBody{
			MysqldExporter: &agents.GetAgentOKBodyMysqldExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},
				TablestatsGroupTableLimit: 2000,
				Status:                    &AgentStatusUnknown,
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, getAgentRes.Payload)

		// Test change API.
		changeMySQLdExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyMysqldExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			MysqldExporter: &agents.ChangeAgentOKBodyMysqldExporter{
				AgentID:                   agentID,
				ServiceID:                 serviceID,
				Username:                  "username",
				PMMAgentID:                pmmAgentID,
				Disabled:                  true,
				TablestatsGroupTableLimit: 2000,
				Status:                    &AgentStatusUnknown,
				DisabledCollectors:        make([]string, 0),
				CustomLabels:              map[string]string{},
				LogLevel:                  pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, changeMySQLdExporterOK.Payload)

		changeMySQLdExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyMysqldExporterCustomLabels{
							Values: map[string]string{
								"new_label": "mysql_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			MysqldExporter: &agents.ChangeAgentOKBodyMysqldExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				Disabled:   false,
				CustomLabels: map[string]string{
					"new_label": "mysql_exporter",
				},
				TablestatsGroupTableLimit: 2000,
				Status:                    &AgentStatusUnknown,
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, changeMySQLdExporterOK.Payload)
	})

	t.Run("WithRealPMMAgent", func(t *testing.T) {
		t.Skip("Skipping until we know there are connected agents in the new environment")
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for MySQLdExporter test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{Context: pmmapitests.Context})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Payload.PMMAgent, "There should be at least one service")

		pmmAgentID := ""
		for _, agent := range res.Payload.PMMAgent {
			if agent.Connected {
				pmmAgentID = agent.AgentID
				break
			}
		}
		if pmmAgentID == "" {
			t.Skip("There are no connected agents")
		}

		mySqldExporter := addAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:  serviceID,
				Username:   "pmm-agent",          // from pmm-agent docker-compose.yml
				Password:   "pmm-agent-password", // from pmm-agent docker-compose.yml
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},

				TablestatsGroupTableLimit: 2000,
			},
		})
		assert.Positive(t, mySqldExporter.MysqldExporter.TableCount)
		assert.EqualValues(t, 2000, mySqldExporter.MysqldExporter.TablestatsGroupTableLimit)
		agentID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:  "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLdExporterParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.MysqldExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
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

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:  serviceID,
					PMMAgentID: "",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLdExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.MysqldExporter.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
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
			pmmapitests.RemoveAgents(t, res.Payload.MysqldExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
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

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:  serviceID,
					PMMAgentID: "pmm-not-exist-server",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.MysqldExporter.AgentID)
		}
	})

	t.Run("With PushMetrics", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for MySQLdExporter test"),
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
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},

				SkipConnectionCheck:       true,
				TablestatsGroupTableLimit: 2000,
				PushMetrics:               true,
			},
		})
		assert.EqualValues(t, 0, mySqldExporter.MysqldExporter.TableCount)
		assert.EqualValues(t, 2000, mySqldExporter.MysqldExporter.TablestatsGroupTableLimit)
		agentID := mySqldExporter.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOKBody{
			MysqldExporter: &agents.GetAgentOKBodyMysqldExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},
				TablestatsGroupTableLimit: 2000,
				PushMetricsEnabled:        true,
				Status:                    &AgentStatusUnknown,
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, getAgentRes.Payload)

		// Test change API.
		changeMySQLdExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
						EnablePushMetrics: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			MysqldExporter: &agents.ChangeAgentOKBodyMysqldExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},
				TablestatsGroupTableLimit: 2000,
				Status:                    &AgentStatusUnknown,
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, changeMySQLdExporterOK.Payload)

		changeMySQLdExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			MysqldExporter: &agents.ChangeAgentOKBodyMysqldExporter{
				AgentID:    agentID,
				ServiceID:  serviceID,
				Username:   "username",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_mysql_exporter": "mysql_exporter",
				},
				TablestatsGroupTableLimit: 2000,
				PushMetricsEnabled:        true,
				Status:                    &AgentStatusUnknown,
				DisabledCollectors:        make([]string, 0),
				LogLevel:                  pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
			},
		}, changeMySQLdExporterOK.Payload)
	})
}
