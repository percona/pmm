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

func TestQANMySQLSlowlogAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL Slowlog Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN Slowlog Agent test"),
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
					QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
						ServiceID:  serviceID,
						Username:   "username",
						Password:   "password",
						PMMAgentID: pmmAgentID,
						CustomLabels: map[string]string{
							"new_label": "QANMysqlSlowlogAgent",
						},

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		agentID := res.Payload.QANMysqlSlowlogAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				QANMysqlSlowlogAgent: &agents.GetAgentOKBodyQANMysqlSlowlogAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlSlowlogAgent",
					},
					Status:             &AgentStatusUnknown,
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxSlowlogFileSize: "0",
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANMySQLSlowlogAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANMysqlSlowlogAgent: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANMysqlSlowlogAgent: &agents.ChangeAgentOKBodyQANMysqlSlowlogAgent{
					AgentID:            agentID,
					ServiceID:          serviceID,
					Username:           "username",
					PMMAgentID:         pmmAgentID,
					Disabled:           true,
					Status:             &AgentStatusUnknown,
					CustomLabels:       map[string]string{},
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxSlowlogFileSize: "0",
				},
			},
		}, changeQANMySQLSlowlogAgentOK)

		changeQANMySQLSlowlogAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANMysqlSlowlogAgent: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgentCustomLabels{
							Values: map[string]string{
								"new_label": "QANMysqlSlowlogAgent",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANMysqlSlowlogAgent: &agents.ChangeAgentOKBodyQANMysqlSlowlogAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANMysqlSlowlogAgent",
					},
					Status:             &AgentStatusUnknown,
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxSlowlogFileSize: "0",
				},
			},
		}, changeQANMySQLSlowlogAgentOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL Slowlog password rotation")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN Slowlog password rotation test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create QAN MySQL Slowlog agent with initial credentials
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
					ServiceID:           serviceID,
					Username:            "initial-mysql-slowlog-user",
					Password:            "initial-mysql-slowlog-password",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.QANMysqlSlowlogAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test password rotation
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlSlowlogAgent: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
					Password: pointer.ToString("new-rotated-mysql-slowlog-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-mysql-slowlog-user", changeQANAgentOK.Payload.QANMysqlSlowlogAgent.Username)
		assert.False(t, changeQANAgentOK.Payload.QANMysqlSlowlogAgent.Disabled)

		// Verify password change with username change
		changeQANAgentOK, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlSlowlogAgent: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
					Username: pointer.ToString("new-mysql-slowlog-user"),
					Password: pointer.ToString("another-new-mysql-slowlog-password"),
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
		assert.Equal(t, "new-mysql-slowlog-user", getAgentRes.Payload.QANMysqlSlowlogAgent.Username)
		assert.False(t, getAgentRes.Payload.QANMysqlSlowlogAgent.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL Slowlog partial update")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN Slowlog partial update test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create QAN MySQL Slowlog agent with comprehensive initial configuration
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
					ServiceID:            serviceID,
					Username:             "initial-slowlog-user",
					Password:             "initial-slowlog-password",
					PMMAgentID:           pmmAgentID,
					MaxQueryLength:       1024,
					DisableQueryExamples: true,
					TLS:                  true,
					TLSSkipVerify:        false,
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
		agentID := res.Payload.QANMysqlSlowlogAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change only username, verify all other fields remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlSlowlogAgent: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
					Username: pointer.ToString("updated-slowlog-user"),
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

		agent := getAgentRes.Payload.QANMysqlSlowlogAgent
		assert.Equal(t, "updated-slowlog-user", agent.Username) // Changed
		assert.Equal(t, int32(1024), agent.MaxQueryLength)      // Unchanged
		assert.True(t, agent.QueryExamplesDisabled)             // Unchanged
		assert.True(t, agent.TLS)                               // Unchanged
		assert.False(t, agent.TLSSkipVerify)                    // Unchanged
		assert.True(t, agent.DisableCommentsParsing)            // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "test",
			"team":        "dev",
		}, agent.CustomLabels) // Unchanged
		assert.Equal(t, pointer.ToString("LOG_LEVEL_INFO"), agent.LogLevel) // Unchanged
		assert.False(t, agent.Disabled)                                     // Unchanged
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL Slowlog change all fields")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN Slowlog change all fields test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create QAN MySQL Slowlog agent with initial configuration
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
					ServiceID:            serviceID,
					Username:             "initial-slowlog-user",
					Password:             "initial-slowlog-password",
					PMMAgentID:           pmmAgentID,
					MaxQueryLength:       512,
					DisableQueryExamples: false,
					TLS:                  false,
					TLSSkipVerify:        true,
					MaxSlowlogFileSize:   "1024",
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
		agentID := res.Payload.QANMysqlSlowlogAgent.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change ALL available fields at once
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlSlowlogAgent: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
					Username:             pointer.ToString("changed-slowlog-user"),
					Password:             pointer.ToString("changed-slowlog-password"),
					MaxQueryLength:       pointer.ToInt32(2048),
					DisableQueryExamples: pointer.ToBool(true),
					TLS:                  pointer.ToBool(true),
					TLSSkipVerify:        pointer.ToBool(false),
					MaxSlowlogFileSize:   pointer.ToString("4096"),
					CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgentCustomLabels{
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
		expectedAgent := &agents.ChangeAgentOKBodyQANMysqlSlowlogAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-slowlog-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         2048,
			QueryExamplesDisabled:  true,
			TLS:                    true,
			TLSSkipVerify:          false,
			MaxSlowlogFileSize:     "4096",
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

		assert.Equal(t, expectedAgent, changeQANAgentOK.Payload.QANMysqlSlowlogAgent)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyQANMysqlSlowlogAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-slowlog-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         2048,
			QueryExamplesDisabled:  true,
			TLS:                    true,
			TLSSkipVerify:          false,
			MaxSlowlogFileSize:     "4096",
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

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.QANMysqlSlowlogAgent)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Slowlog Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
						ServiceID:  "",
						PMMAgentID: pmmAgentID,
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLSlowlogAgentParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlSlowlogAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Slowlog Agent")).NodeID
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
					QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
						ServiceID:  serviceID,
						PMMAgentID: "",
						Username:   "username",
						Password:   "password",

						SkipConnectionCheck: true,
					},
				},
				Context: pmmapitests.Context,
			})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMySQLSlowlogAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlSlowlogAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Slowlog Agent")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlSlowlogAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Slowlog Agent")).NodeID
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
					QANMysqlSlowlogAgent: &agents.AddAgentParamsBodyQANMysqlSlowlogAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.QANMysqlSlowlogAgent.AgentID)
		}
	})
}
