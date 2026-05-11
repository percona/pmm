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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

func TestProxySQLExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for ProxySQLExporter test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		ProxySQLExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_proxysql_exporter": "proxysql_exporter",
				},

				SkipConnectionCheck: true,
			},
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				ProxysqlExporter: &agents.GetAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeProxySQLExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyProxysqlExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				ProxysqlExporter: &agents.ChangeAgentOKBodyProxysqlExporter{
					AgentID:            agentID,
					ServiceID:          serviceID,
					Username:           "username",
					PMMAgentID:         pmmAgentID,
					Disabled:           true,
					Status:             &AgentStatusDone,
					CustomLabels:       map[string]string{},
					DisabledCollectors: make([]string, 0),
					LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)

		changeProxySQLExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyProxysqlExporterCustomLabels{
							Values: map[string]string{
								"new_label": "proxysql_exporter",
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
				ProxysqlExporter: &agents.ChangeAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "proxysql_exporter",
					},
					Status:             &AgentStatusDone,
					DisabledCollectors: make([]string, 0),
					LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
					ServiceID:  "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddProxySQLExporterParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for agent"),
			},
		})
		serviceID := service.Proxysql.ServiceID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
					ServiceID:  serviceID,
					PMMAgentID: "",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddProxySQLExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
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
			pmmapitests.RemoveAgents(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for not exists node ID"),
			},
		})
		serviceID := service.Proxysql.ServiceID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
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
			pmmapitests.RemoveAgents(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})
	t.Run("With PushMetrics", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for ProxySQLExporter test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		ProxySQLExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_proxysql_exporter": "proxysql_exporter",
				},

				SkipConnectionCheck: true,
			},
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				ProxysqlExporter: &agents.GetAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeProxySQLExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
					EnablePushMetrics: new(true),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				ProxysqlExporter: &agents.ChangeAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)

		changeProxySQLExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
						EnablePushMetrics: new(false),
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				ProxysqlExporter: &agents.ChangeAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for proxysql exporter")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6033,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for password rotation test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		// Create agent with initial credentials
		ProxySQLExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
				ServiceID:           serviceID,
				Username:            "initial-user",
				Password:            "initial-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID

		// Test password rotation
		changeProxySQLExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
					Password: new("new-rotated-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-user", changeProxySQLExporterOK.Payload.ProxysqlExporter.Username)
		assert.False(t, changeProxySQLExporterOK.Payload.ProxysqlExporter.Disabled)

		// Verify password change with username change
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
					Username: new("new-proxysql-user"),
					Password: new("another-new-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Get agent to verify changes took effect
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "new-proxysql-user", getAgentRes.Payload.ProxysqlExporter.Username)
		assert.False(t, getAgentRes.Payload.ProxysqlExporter.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for proxysql exporter")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6033,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for partial field test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		// Create agent with comprehensive initial configuration
		ProxySQLExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
				ServiceID:           serviceID,
				Username:            "original-user",
				Password:            "original-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
				CustomLabels: map[string]string{
					"env":    "staging",
					"team":   "database",
					"region": "us-west",
				},
				PushMetrics: true,
				LogLevel:    new("LOG_LEVEL_INFO"),
			},
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID

		// Change only one field (username), others should remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
					Username: new("changed-user"),
					// Note: password, custom labels, push metrics, and log level are NOT specified
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

		// Username should be changed
		assert.Equal(t, "changed-user", getAgentRes.Payload.ProxysqlExporter.Username)

		// Everything else should remain unchanged
		assert.Equal(t, map[string]string{
			"env":    "staging",
			"team":   "database",
			"region": "us-west",
		}, getAgentRes.Payload.ProxysqlExporter.CustomLabels)
		assert.True(t, getAgentRes.Payload.ProxysqlExporter.PushMetricsEnabled)
		assert.Equal(t, new("LOG_LEVEL_INFO"), getAgentRes.Payload.ProxysqlExporter.LogLevel)
		assert.False(t, getAgentRes.Payload.ProxysqlExporter.Disabled)
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for proxysql exporter change all fields")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6033,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for change all fields test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		// Create ProxySQL Exporter with initial configuration
		ProxySQLExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
				ServiceID:           serviceID,
				Username:            "initial-user",
				Password:            "initial-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				PushMetrics: false,
				LogLevel:    new("LOG_LEVEL_WARN"),
			},
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID

		// Change ALL available fields at once
		changeProxySQLExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ProxysqlExporter: &agents.ChangeAgentParamsBodyProxysqlExporter{
					Username:          new("new-proxysql-user"),
					Password:          new("new-proxysql-password"),
					LogLevel:          new("LOG_LEVEL_ERROR"),
					EnablePushMetrics: new(true),
					DisableCollectors: []string{"mysql_connection_pool", "mysql_connection_list"},
					CustomLabels: &agents.ChangeAgentParamsBodyProxysqlExporterCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "platform",
						},
					},
					Enable: new(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyProxysqlExporter{
			AgentID:            agentID,
			ServiceID:          serviceID,
			PMMAgentID:         pmmAgentID,
			Username:           "new-proxysql-user",
			LogLevel:           new("LOG_LEVEL_ERROR"),
			PushMetricsEnabled: true,
			DisabledCollectors: []string{"mysql_connection_pool", "mysql_connection_list"},
			Disabled:           true, // agent was disabled
			Status:             &AgentStatusDone,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "platform",
			},
		}

		assert.Equal(t, expectedAgent, changeProxySQLExporterOK.Payload.ProxysqlExporter)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyProxysqlExporter{
			AgentID:            agentID,
			ServiceID:          serviceID,
			PMMAgentID:         pmmAgentID,
			Username:           "new-proxysql-user",
			LogLevel:           new("LOG_LEVEL_ERROR"),
			PushMetricsEnabled: true,
			DisabledCollectors: []string{"mysql_connection_pool", "mysql_connection_list"},
			Disabled:           true,
			Status:             &AgentStatusDone,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "platform",
			},
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.ProxysqlExporter)
	})
}
