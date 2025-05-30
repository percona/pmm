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

func TestPGStatStatementsQanAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL Agent pg_stat_statements")).NodeID
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
				AgentID: agentID,
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
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
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
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCustomLabels{
							Values: map[string]string{
								"new_label": "QANPostgreSQLPgStatementsAgent",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
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

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL PgStatements password rotation")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QAN PgStatements password rotation test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create QAN PostgreSQL PgStatements agent with initial credentials
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
					ServiceID:           serviceID,
					Username:            "initial-postgres-qan-user",
					Password:            "initial-postgres-qan-password",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatementsAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test password rotation
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
					Password: pointer.ToString("new-rotated-postgres-qan-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-postgres-qan-user", changeQANAgentOK.Payload.QANPostgresqlPgstatementsAgent.Username)
		assert.False(t, changeQANAgentOK.Payload.QANPostgresqlPgstatementsAgent.Disabled)

		// Verify password change with username change
		changeQANAgentOK, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
					Username: pointer.ToString("new-postgres-qan-user"),
					Password: pointer.ToString("another-new-postgres-qan-password"),
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
		assert.Equal(t, "new-postgres-qan-user", getAgentRes.Payload.QANPostgresqlPgstatementsAgent.Username)
		assert.False(t, getAgentRes.Payload.QANPostgresqlPgstatementsAgent.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL PgStatements partial update")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QAN PgStatements partial update test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create QAN PostgreSQL PgStatements agent with comprehensive initial configuration
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
					ServiceID:      serviceID,
					Username:       "initial-pgstatements-user",
					Password:       "initial-pgstatements-password",
					PMMAgentID:     pmmAgentID,
					MaxQueryLength: 2048,
					TLS:            true,
					TLSSkipVerify:  false,
					CustomLabels: map[string]string{
						"environment": "test",
						"team":        "dev",
					},
					LogLevel:               pointer.ToString("LOG_LEVEL_INFO"),
					SkipConnectionCheck:    true,
					DisableCommentsParsing: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatementsAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change only username, verify all other fields remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
					Username: pointer.ToString("updated-pgstatements-user"),
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

		agent := getAgentRes.Payload.QANPostgresqlPgstatementsAgent
		assert.Equal(t, "updated-pgstatements-user", agent.Username) // Changed
		assert.Equal(t, int32(2048), agent.MaxQueryLength)           // Unchanged
		assert.True(t, agent.TLS)                                    // Unchanged
		assert.False(t, agent.TLSSkipVerify)                         // Unchanged
		assert.True(t, agent.DisableCommentsParsing)                 // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "test",
			"team":        "dev",
		}, agent.CustomLabels) // Unchanged
		assert.Equal(t, pointer.ToString("LOG_LEVEL_INFO"), agent.LogLevel) // Unchanged
		assert.False(t, agent.Disabled)                                     // Unchanged
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN PostgreSQL PgStatements change all fields")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for QAN PgStatements change all fields test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create QAN PostgreSQL PgStatements agent with initial configuration
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.AddAgentParamsBodyQANPostgresqlPgstatementsAgent{
					ServiceID:      serviceID,
					Username:       "initial-pgstatements-user",
					Password:       "initial-pgstatements-password",
					PMMAgentID:     pmmAgentID,
					MaxQueryLength: 1024,
					TLS:            false,
					TLSSkipVerify:  true,
					CustomLabels: map[string]string{
						"environment": "staging",
						"version":     "1.0",
					},
					LogLevel:               pointer.ToString("LOG_LEVEL_WARN"),
					SkipConnectionCheck:    true,
					DisableCommentsParsing: false,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.QANPostgresqlPgstatementsAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change ALL available fields at once
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
					Username:       pointer.ToString("changed-pgstatements-user"),
					Password:       pointer.ToString("changed-pgstatements-password"),
					MaxQueryLength: pointer.ToInt32(4096),
					TLS:            pointer.ToBool(true),
					TLSSkipVerify:  pointer.ToBool(false),
					CustomLabels: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgentCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "backend",
						},
					},
					LogLevel:               pointer.ToString("LOG_LEVEL_DEBUG"),
					DisableCommentsParsing: pointer.ToBool(true),
					Enable:                 pointer.ToBool(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyQANPostgresqlPgstatementsAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-pgstatements-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         4096,
			TLS:                    true,
			TLSSkipVerify:          false,
			DisableCommentsParsing: true,
			Disabled:               true, // agent was disabled
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			Status:   &AgentStatusUnknown,
			LogLevel: pointer.ToString("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedAgent, changeQANAgentOK.Payload.QANPostgresqlPgstatementsAgent)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyQANPostgresqlPgstatementsAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-pgstatements-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         4096,
			TLS:                    true,
			TLSSkipVerify:          false,
			DisableCommentsParsing: true,
			Disabled:               true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			Status:   &AgentStatusUnknown,
			LogLevel: pointer.ToString("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.QANPostgresqlPgstatementsAgent)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
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

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
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

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
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

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
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
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANPostgresqlPgstatementsAgent.AgentID)
		}
	})
}
