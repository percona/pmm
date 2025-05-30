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

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Mysqld exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for mysqld exporter"),
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

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mysqld exporter"))
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

		// Add agent with skip connection check
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:           serviceID,
					Username:            "username",
					Password:            "password",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing ALL available MySQLdExporter fields
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
					// Core agent settings
					Enable:            pointer.ToBool(true),
					EnablePushMetrics: pointer.ToBool(true),
					Username:          pointer.ToString("new-mysql-user"),

					// TLS configuration
					TLS:           pointer.ToBool(true),
					TLSSkipVerify: pointer.ToBool(false),
					TLSCa:         pointer.ToString("test-ca-cert"),
					TLSCert:       pointer.ToString("test-client-cert"),
					TLSKey:        pointer.ToString("test-client-key"),

					// Tablestats configuration
					TablestatsGroupTableLimit: pointer.ToInt32(1000),

					// Connection and monitoring settings
					SkipConnectionCheck: pointer.ToBool(true),
					DisableCollectors:   []string{"info_schema.innodb_metrics", "info_schema.processlist"},

					// Agent configuration
					AgentPassword:  pointer.ToString("new-agent-password"),
					LogLevel:       pointer.ToString(agents.ChangeAgentParamsBodyMysqldExporterLogLevelLOGLEVELDEBUG),
					ExposeExporter: pointer.ToBool(true),

					// Metrics configuration
					MetricsResolutions: &agents.ChangeAgentParamsBodyMysqldExporterMetricsResolutions{
						Hr: "5s",
						Mr: "10s",
						Lr: "60s",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify ALL the fields were applied correctly
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MysqldExporter)

		mysqldExporter := getAgentRes.Payload.MysqldExporter

		// Core agent settings
		assert.False(t, mysqldExporter.Disabled) // Enable: true means Disabled: false
		assert.True(t, mysqldExporter.PushMetricsEnabled)
		assert.Equal(t, "new-mysql-user", mysqldExporter.Username)

		// TLS configuration
		assert.True(t, mysqldExporter.TLS)
		assert.False(t, mysqldExporter.TLSSkipVerify)

		// Tablestats configuration
		assert.Equal(t, int32(1000), mysqldExporter.TablestatsGroupTableLimit)

		// Collectors configuration
		assert.ElementsMatch(t, []string{"info_schema.innodb_metrics", "info_schema.processlist"}, mysqldExporter.DisabledCollectors)

		// Agent configuration
		assert.Equal(t, pointer.ToString("LOG_LEVEL_DEBUG"), mysqldExporter.LogLevel)
		assert.True(t, mysqldExporter.ExposeExporter)

		// Metrics configuration
		assert.Equal(t, "5s", mysqldExporter.MetricsResolutions.Hr)
		assert.Equal(t, "10s", mysqldExporter.MetricsResolutions.Mr)
		assert.Equal(t, "60s", mysqldExporter.MetricsResolutions.Lr)

		// Note: TLS cert/key/ca and agent_password are not returned in GetAgent for security reasons
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mysqld exporter partial update"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for MySQLdExporter partial update test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create MySQLd Exporter with comprehensive initial configuration
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:                 serviceID,
					Username:                  "initial-mysql-user",
					Password:                  "initial-mysql-password",
					PMMAgentID:                pmmAgentID,
					TLS:                       true,
					TLSSkipVerify:             false,
					TablestatsGroupTableLimit: 1500,
					DisableCollectors:         []string{"info_schema.innodb_metrics", "performance_schema.file_events"},
					AgentPassword:             "initial-agent-password",
					LogLevel:                  pointer.ToString("LOG_LEVEL_INFO"),
					ExposeExporter:            true,
					CustomLabels: map[string]string{
						"environment": "staging",
						"team":        "database",
						"region":      "us-west-2",
					},
					SkipConnectionCheck: true,
					PushMetrics:         true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change only the log level, verify all other fields remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
					LogLevel: pointer.ToString("LOG_LEVEL_ERROR"),
					// Note: All other fields are intentionally NOT specified
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify only the specified field changed
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		agent := getAgentRes.Payload.MysqldExporter
		// Log level should be changed
		assert.Equal(t, pointer.ToString("LOG_LEVEL_ERROR"), agent.LogLevel)

		// Everything else should remain unchanged
		assert.Equal(t, "initial-mysql-user", agent.Username)
		assert.True(t, agent.TLS)
		assert.False(t, agent.TLSSkipVerify)
		assert.Equal(t, int32(1500), agent.TablestatsGroupTableLimit)
		assert.ElementsMatch(t, []string{"info_schema.innodb_metrics", "performance_schema.file_events"}, agent.DisabledCollectors)
		assert.True(t, agent.ExposeExporter)
		assert.True(t, agent.PushMetricsEnabled)
		assert.Equal(t, map[string]string{
			"environment": "staging",
			"team":        "database",
			"region":      "us-west-2",
		}, agent.CustomLabels)
		assert.False(t, agent.Disabled)
	})

	t.Run("ChangeCustomLabels", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mysqld exporter"))
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

		// Add agent with initial custom labels
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:           serviceID,
					Username:            "username",
					Password:            "password",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
					CustomLabels: map[string]string{
						"initial_label": "initial_value",
						"env":           "test",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing custom labels to new set
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
					CustomLabels: &agents.ChangeAgentParamsBodyMysqldExporterCustomLabels{
						Values: map[string]string{
							"new_label":   "new_value",
							"environment": "production",
							"team":        "database",
						},
					},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify custom labels were updated correctly
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MysqldExporter)

		expectedLabels := map[string]string{
			"new_label":   "new_value",
			"environment": "production",
			"team":        "database",
		}
		assert.Equal(t, expectedLabels, getAgentRes.Payload.MysqldExporter.CustomLabels)

		// Test clearing all custom labels
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
					CustomLabels: &agents.ChangeAgentParamsBodyMysqldExporterCustomLabels{
						Values: map[string]string{},
					},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all custom labels were cleared
		getAgentRes, err = client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MysqldExporter)

		assert.Equal(t, map[string]string{}, getAgentRes.Payload.MysqldExporter.CustomLabels)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mysqld exporter"))
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

		// Add agent with initial password
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
					ServiceID:           serviceID,
					Username:            "mysql-user",
					Password:            "initial-password-123",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MysqldExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing password (simulating password rotation)
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
					Password: pointer.ToString("rotated-password-456"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify agent still works after password change
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MysqldExporter)

		mysqldExporter := getAgentRes.Payload.MysqldExporter
		assert.Equal(t, "mysql-user", mysqldExporter.Username) // Username unchanged
		assert.False(t, mysqldExporter.Disabled)               // Agent still enabled

		// Note: Password is not returned in GetAgent response for security reasons
		// This test verifies that the password change operation completes successfully
		// without returning the actual password value

		// Test changing username and password together
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MysqldExporter: &agents.ChangeAgentParamsBodyMysqldExporter{
					Username: pointer.ToString("new-mysql-user"),
					Password: pointer.ToString("final-password-789"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify both username and password change completed
		getAgentRes, err = client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MysqldExporter)

		assert.Equal(t, "new-mysql-user", getAgentRes.Payload.MysqldExporter.Username)
		assert.False(t, getAgentRes.Payload.MysqldExporter.Disabled)
	})
}
