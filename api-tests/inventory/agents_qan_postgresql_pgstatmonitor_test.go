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

func TestPGStatMonitorQanAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for Qan PostgreSQL Agent pg_stat_monitor")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QanAgent test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
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
		})
		agentID := res.QANPostgresqlPgstatmonitorAgent.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
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
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANPostgreSQLPgStatMonitorAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:      agentID,
					ServiceID:    serviceID,
					Username:     "username",
					PMMAgentID:   pmmAgentID,
					Disabled:     true,
					Status:       &AgentStatusDone,
					CustomLabels: map[string]string{},
					LogLevel:     new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANPostgreSQLPgStatMonitorAgentOK)

		changeQANPostgreSQLPgStatMonitorAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCustomLabels{
							Values: map[string]string{
								"new_label": "QANPostgreSQLPgStatMonitorAgent",
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
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentOKBodyQANPostgresqlPgstatmonitorAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANPostgreSQLPgStatMonitorAgent",
					},
					Status:   &AgentStatusDone,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANPostgreSQLPgStatMonitorAgentOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL PgStatMonitor password rotation")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QAN PgStatMonitor password rotation test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN PostgreSQL PgStatMonitor agent with initial credentials
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
				ServiceID:           serviceID,
				Username:            "initial-postgres-monitor-user",
				Password:            "initial-postgres-monitor-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		agentID := res.QANPostgresqlPgstatmonitorAgent.AgentID

		// Test password rotation
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
					Password: new("new-rotated-postgres-monitor-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-postgres-monitor-user", changeQANAgentOK.Payload.QANPostgresqlPgstatmonitorAgent.Username)
		assert.False(t, changeQANAgentOK.Payload.QANPostgresqlPgstatmonitorAgent.Disabled)

		// Verify password change with username change
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
					Username: new("new-postgres-monitor-user"),
					Password: new("another-new-postgres-monitor-password"),
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
		assert.Equal(t, "new-postgres-monitor-user", getAgentRes.Payload.QANPostgresqlPgstatmonitorAgent.Username)
		assert.False(t, getAgentRes.Payload.QANPostgresqlPgstatmonitorAgent.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL PgStatMonitor partial update")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QAN PgStatMonitor partial update test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN PostgreSQL PgStatMonitor agent with comprehensive initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
				ServiceID:            serviceID,
				Username:             "initial-pgstatmonitor-user",
				Password:             "initial-pgstatmonitor-password",
				PMMAgentID:           pmmAgentID,
				MaxQueryLength:       2048,
				TLS:                  true,
				TLSSkipVerify:        false,
				DisableQueryExamples: true,
				CustomLabels: map[string]string{
					"environment": "test",
					"team":        "dev",
				},
				LogLevel:               new("LOG_LEVEL_INFO"),
				SkipConnectionCheck:    true,
				DisableCommentsParsing: true,
			},
		})
		agentID := res.QANPostgresqlPgstatmonitorAgent.AgentID

		// Change only username, verify all other fields remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
					Username: new("updated-pgstatmonitor-user"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify only username changed, all other fields remain the same
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		agent := getAgentRes.Payload.QANPostgresqlPgstatmonitorAgent
		assert.Equal(t, "updated-pgstatmonitor-user", agent.Username) // Changed
		assert.Equal(t, int32(2048), agent.MaxQueryLength)            // Unchanged
		assert.True(t, agent.TLS)                                     // Unchanged
		assert.False(t, agent.TLSSkipVerify)                          // Unchanged
		assert.True(t, agent.QueryExamplesDisabled)                   // Unchanged
		assert.True(t, agent.DisableCommentsParsing)                  // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "test",
			"team":        "dev",
		}, agent.CustomLabels) // Unchanged
		assert.Equal(t, new("LOG_LEVEL_INFO"), agent.LogLevel) // Unchanged
		assert.False(t, agent.Disabled)                        // Unchanged
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL PgStatMonitor change all fields")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QAN PgStatMonitor change all fields test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN PostgreSQL PgStatMonitor agent with initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
				ServiceID:            serviceID,
				Username:             "initial-pgstatmonitor-user",
				Password:             "initial-pgstatmonitor-password",
				PMMAgentID:           pmmAgentID,
				MaxQueryLength:       1024,
				TLS:                  false,
				TLSSkipVerify:        true,
				DisableQueryExamples: false,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				LogLevel:               new("LOG_LEVEL_WARN"),
				SkipConnectionCheck:    true,
				DisableCommentsParsing: false,
			},
		})
		agentID := res.QANPostgresqlPgstatmonitorAgent.AgentID

		// Change ALL available fields at once
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatmonitorAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
					Username:             new("changed-pgstatmonitor-user"),
					Password:             new("changed-pgstatmonitor-password"),
					MaxQueryLength:       new(int32(4096)),
					TLS:                  new(true),
					TLSSkipVerify:        new(false),
					DisableQueryExamples: new(true),
					CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "backend",
						},
					},
					LogLevel:               new("LOG_LEVEL_DEBUG"),
					DisableCommentsParsing: new(true),
					Enable:                 new(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyQANPostgresqlPgstatmonitorAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-pgstatmonitor-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         4096,
			TLS:                    true,
			TLSSkipVerify:          false,
			QueryExamplesDisabled:  true,
			DisableCommentsParsing: true,
			Disabled:               true, // agent was disabled
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			Status:   &AgentStatusDone,
			LogLevel: new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedAgent, changeQANAgentOK.Payload.QANPostgresqlPgstatmonitorAgent)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyQANPostgresqlPgstatmonitorAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-pgstatmonitor-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         4096,
			TLS:                    true,
			TLSSkipVerify:          false,
			QueryExamplesDisabled:  true,
			DisableCommentsParsing: true,
			Disabled:               true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			Status:   &AgentStatusDone,
			LogLevel: new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.QANPostgresqlPgstatmonitorAgent)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatMonitorAgentParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
			},
		})
		serviceID := service.Postgresql.ServiceID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANPostgreSQLPgStatMonitorAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANPostgresqlPgstatmonitorAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
			},
		})
		serviceID := service.Postgresql.ServiceID

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
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID)
		}
	})
}
