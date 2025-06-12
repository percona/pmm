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

func TestPostgresExporter(t *testing.T) {
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

		PostgresExporter := addAgent(t, agents.AddAgentBody{
			PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				SkipConnectionCheck:    true,
				MaxExporterConnections: 10,
			},
		})
		agentID := PostgresExporter.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
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
					DisabledCollectors:     make([]string, 0),
					LogLevel:               pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					Status:                 &AgentStatusUnknown,
					MaxExporterConnections: 10,
				},
			},
		}, getAgentRes)

		// Test change API.
		changePostgresExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyPostgresExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:                agentID,
					ServiceID:              serviceID,
					Username:               "username",
					PMMAgentID:             pmmAgentID,
					Disabled:               true,
					Status:                 &AgentStatusUnknown,
					CustomLabels:           map[string]string{},
					DisabledCollectors:     make([]string, 0),
					LogLevel:               pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxExporterConnections: 10,
				},
			},
		}, changePostgresExporterOK)

		changePostgresExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyPostgresExporterCustomLabels{
							Values: map[string]string{
								"new_label": "postgres_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "postgres_exporter",
					},
					Status:                 &AgentStatusUnknown,
					DisabledCollectors:     make([]string, 0),
					LogLevel:               pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxExporterConnections: 10,
				},
			},
		}, changePostgresExporterOK)
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
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:  "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgresExporterParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.PostgresExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
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

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:  serviceID,
					PMMAgentID: "",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgresExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.PostgresExporter.AgentID)
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
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
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
			pmmapitests.RemoveAgents(t, res.Payload.PostgresExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
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

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
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
			pmmapitests.RemoveAgents(t, res.Payload.PostgresExporter.AgentID)
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

		PostgresExporter := addAgent(t, agents.AddAgentBody{
			PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				SkipConnectionCheck: true,
				PushMetrics:         true,
			},
		})
		agentID := PostgresExporter.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
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
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changePostgresExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						EnablePushMetrics: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changePostgresExporterOK)

		changePostgresExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changePostgresExporterOK)
	})

	t.Run("ChangeTLSAndAgentConfiguration", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for postgres exporter"))
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

		// Add agent with skip connection check
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
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
		agentID := res.Payload.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing TLS fields, LogLevel, and AgentPassword
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					TLS:               pointer.ToBool(true),
					TLSSkipVerify:     pointer.ToBool(false),
					AgentPassword:     pointer.ToString("new-agent-password"),
					LogLevel:          pointer.ToString(agents.ChangeAgentParamsBodyPostgresExporterLogLevelLOGLEVELWARN),
					DisableCollectors: []string{"collector1", "collector2"},
					ExposeExporter:    pointer.ToBool(true),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify the TLS and other new fields were applied
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.PostgresExporter)

		postgresExporter := getAgentRes.Payload.PostgresExporter
		assert.True(t, postgresExporter.TLS)
		assert.False(t, postgresExporter.TLSSkipVerify)
		assert.Equal(t, pointer.ToString("LOG_LEVEL_WARN"), postgresExporter.LogLevel)
		assert.ElementsMatch(t, []string{"collector1", "collector2"}, postgresExporter.DisabledCollectors)
		assert.True(t, postgresExporter.ExposeExporter)
		// Note: TLS cert/key and agent_password are not returned in GetAgent for security reasons
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for postgres exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for password rotation test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Add agent with initial password
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:           serviceID,
					Username:            "postgres-user",
					Password:            "initial-postgres-password-123",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing password (simulating password rotation)
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					Password: pointer.ToString("rotated-postgres-password-456"),
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
		require.NotNil(t, getAgentRes.Payload.PostgresExporter)

		postgresExporter := getAgentRes.Payload.PostgresExporter
		assert.Equal(t, "postgres-user", postgresExporter.Username) // Username unchanged
		assert.False(t, postgresExporter.Disabled)                  // Agent still enabled

		// Note: Password is not returned in GetAgent response for security reasons
		// This test verifies that the password change operation completes successfully
		// without returning the actual password value

		// Test changing username, password, and TLS settings together
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					Username:      pointer.ToString("new-postgres-user"),
					Password:      pointer.ToString("final-postgres-password-789"),
					TLS:           pointer.ToBool(true),
					TLSSkipVerify: pointer.ToBool(false),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify username and TLS changes completed
		getAgentRes, err = client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.PostgresExporter)

		assert.Equal(t, "new-postgres-user", getAgentRes.Payload.PostgresExporter.Username)
		assert.True(t, getAgentRes.Payload.PostgresExporter.TLS)
		assert.False(t, getAgentRes.Payload.PostgresExporter.TLSSkipVerify)
		assert.False(t, getAgentRes.Payload.PostgresExporter.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for postgres exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for partial change test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Add agent with specific initial values for multiple fields
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:              serviceID,
					Username:               "initial-postgres-user",
					Password:               "initial-postgres-password",
					PMMAgentID:             pmmAgentID,
					SkipConnectionCheck:    true,
					MaxExporterConnections: 25,
					CustomLabels: map[string]string{
						"env":        "development",
						"database":   "postgresql",
						"monitoring": "enabled",
					},
					TLS:            true,
					TLSSkipVerify:  false,
					TLSCa:          "initial-ca-cert",
					TLSCert:        "initial-client-cert",
					TLSKey:         "initial-client-key",
					PushMetrics:    true,
					LogLevel:       pointer.ToString("LOG_LEVEL_ERROR"),
					ExposeExporter: true,
					DisableCollectors: []string{
						"pg_stat_bgwriter",
						"pg_stat_database",
					},
					AutoDiscoveryLimit: 100,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Get initial state to capture all original values
		initialAgent, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, initialAgent.Payload.PostgresExporter)

		initialExporter := initialAgent.Payload.PostgresExporter

		// Change ONLY the password - all other fields should remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					Password: pointer.ToString("new-password-only"),
					// All other fields are intentionally NOT set (nil)
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify that ONLY the password-related behavior changed, all other fields preserved
		updatedAgent, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedAgent.Payload.PostgresExporter)

		updatedExporter := updatedAgent.Payload.PostgresExporter

		// Verify all original fields are preserved (password can't be checked as it's not returned)
		assert.Equal(t, initialExporter.Username, updatedExporter.Username, "Username should remain unchanged")
		assert.Equal(t, initialExporter.MaxExporterConnections, updatedExporter.MaxExporterConnections, "MaxExporterConnections should remain unchanged")
		assert.Equal(t, initialExporter.CustomLabels, updatedExporter.CustomLabels, "CustomLabels should remain unchanged")
		assert.Equal(t, initialExporter.TLS, updatedExporter.TLS, "TLS should remain unchanged")
		assert.Equal(t, initialExporter.TLSSkipVerify, updatedExporter.TLSSkipVerify, "TLSSkipVerify should remain unchanged")
		assert.Equal(t, initialExporter.PushMetricsEnabled, updatedExporter.PushMetricsEnabled, "PushMetricsEnabled should remain unchanged")
		assert.Equal(t, initialExporter.LogLevel, updatedExporter.LogLevel, "LogLevel should remain unchanged")
		assert.Equal(t, initialExporter.ExposeExporter, updatedExporter.ExposeExporter, "ExposeExporter should remain unchanged")
		assert.Equal(t, initialExporter.DisabledCollectors, updatedExporter.DisabledCollectors, "DisabledCollectors should remain unchanged")
		assert.Equal(t, initialExporter.Disabled, updatedExporter.Disabled, "Disabled status should remain unchanged")
		assert.Equal(t, initialExporter.AutoDiscoveryLimit, updatedExporter.AutoDiscoveryLimit, "AutoDiscoveryLimit should remain unchanged")

		// Now change ONLY the max connections - all other fields should remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
					MaxExporterConnections: pointer.ToInt32(50),
					// All other fields are intentionally NOT set (nil)
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify that ONLY the max connections changed
		finalAgent, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, finalAgent.Payload.PostgresExporter)

		finalExporter := finalAgent.Payload.PostgresExporter

		// Max connections should be changed
		assert.Equal(t, int32(50), finalExporter.MaxExporterConnections, "MaxExporterConnections should be changed")

		// All other fields should still match the initial values
		assert.Equal(t, initialExporter.Username, finalExporter.Username, "Username should remain unchanged")
		assert.Equal(t, initialExporter.CustomLabels, finalExporter.CustomLabels, "CustomLabels should remain unchanged")
		assert.Equal(t, initialExporter.TLS, finalExporter.TLS, "TLS should remain unchanged")
		assert.Equal(t, initialExporter.TLSSkipVerify, finalExporter.TLSSkipVerify, "TLSSkipVerify should remain unchanged")
		assert.Equal(t, initialExporter.PushMetricsEnabled, finalExporter.PushMetricsEnabled, "PushMetricsEnabled should remain unchanged")
		assert.Equal(t, initialExporter.LogLevel, finalExporter.LogLevel, "LogLevel should remain unchanged")
		assert.Equal(t, initialExporter.ExposeExporter, finalExporter.ExposeExporter, "ExposeExporter should remain unchanged")
		assert.Equal(t, initialExporter.DisabledCollectors, finalExporter.DisabledCollectors, "DisabledCollectors should remain unchanged")
		assert.Equal(t, initialExporter.Disabled, finalExporter.Disabled, "Disabled status should remain unchanged")
		assert.Equal(t, initialExporter.AutoDiscoveryLimit, finalExporter.AutoDiscoveryLimit, "AutoDiscoveryLimit should remain unchanged")
	})
}
