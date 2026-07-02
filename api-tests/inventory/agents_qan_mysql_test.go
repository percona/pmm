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

func TestQANMySQLPerfSchemaAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN Agent test"),
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
					ExtraDsnParams: map[string]string{},
					Status:         &AgentStatusUnknown,
					LogLevel:       new("LOG_LEVEL_UNSPECIFIED"),
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
					ExtraDsnParams: map[string]string{},
					Status:         &AgentStatusDone,
					LogLevel:       new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANMySQLPerfSchemaAgentOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL password rotation")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN password rotation test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN MySQL PerfSchema agent with initial credentials
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
				ServiceID:           serviceID,
				Username:            "initial-mysql-qan-user",
				Password:            "initial-mysql-qan-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		agentID := res.QANMysqlPerfschemaAgent.AgentID

		// Test password rotation
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
					Password: new("new-rotated-mysql-qan-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-mysql-qan-user", changeQANAgentOK.Payload.QANMysqlPerfschemaAgent.Username)
		assert.False(t, changeQANAgentOK.Payload.QANMysqlPerfschemaAgent.Disabled)

		// Verify password change with username change
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
					Username: new("new-mysql-qan-user"),
					Password: new("another-new-mysql-qan-password"),
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
		assert.Equal(t, "new-mysql-qan-user", getAgentRes.Payload.QANMysqlPerfschemaAgent.Username)
		assert.False(t, getAgentRes.Payload.QANMysqlPerfschemaAgent.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL partial update")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN partial update test"),
			},
		})
		serviceID := service.Mysql.ServiceID

		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN agent with comprehensive initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
				ServiceID:           serviceID,
				Username:            "original-mysql-qan-user",
				Password:            "original-mysql-qan-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
				CustomLabels: map[string]string{
					"env":     "production",
					"team":    "analytics",
					"service": "mysql-qan",
				},
				LogLevel: new("LOG_LEVEL_DEBUG"),
			},
		})
		agentID := res.QANMysqlPerfschemaAgent.AgentID

		// Change only one field (username), others should remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
					Username: new("changed-mysql-qan-user"),
					// Note: password, custom labels, and log level are NOT specified
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
		assert.Equal(t, "changed-mysql-qan-user", getAgentRes.Payload.QANMysqlPerfschemaAgent.Username)

		// Everything else should remain unchanged
		assert.Equal(t, map[string]string{
			"env":     "production",
			"team":    "analytics",
			"service": "mysql-qan",
		}, getAgentRes.Payload.QANMysqlPerfschemaAgent.CustomLabels)
		assert.Equal(t, new("LOG_LEVEL_DEBUG"), getAgentRes.Payload.QANMysqlPerfschemaAgent.LogLevel)
		assert.False(t, getAgentRes.Payload.QANMysqlPerfschemaAgent.Disabled)
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MySQL PerfSchema change all fields")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for QAN PerfSchema change all fields test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN MySQL PerfSchema agent with initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMysqlPerfschemaAgent: &agents.AddAgentParamsBodyQANMysqlPerfschemaAgent{
				ServiceID:            serviceID,
				Username:             "initial-perfschema-user",
				Password:             "initial-perfschema-password",
				PMMAgentID:           pmmAgentID,
				MaxQueryLength:       768,
				DisableQueryExamples: false,
				TLS:                  false,
				TLSSkipVerify:        true,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				LogLevel:               new("LOG_LEVEL_WARN"),
				SkipConnectionCheck:    true,
				DisableCommentsParsing: false,
			},
		})
		agentID := res.QANMysqlPerfschemaAgent.AgentID

		// Change ALL available fields at once
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMysqlPerfschemaAgent: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgent{
					Username:             new("changed-perfschema-user"),
					Password:             new("changed-perfschema-password"),
					MaxQueryLength:       new(int32(1536)),
					DisableQueryExamples: new(true),
					TLS:                  new(true),
					TLSSkipVerify:        new(false),
					CustomLabels: &agents.ChangeAgentParamsBodyQANMysqlPerfschemaAgentCustomLabels{
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
		expectedAgent := &agents.ChangeAgentOKBodyQANMysqlPerfschemaAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-perfschema-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         1536,
			QueryExamplesDisabled:  true,
			TLS:                    true,
			TLSSkipVerify:          false,
			DisableCommentsParsing: true,
			Disabled:               true, // agent was disabled
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			ExtraDsnParams: map[string]string{},
			Status:         &AgentStatusDone,
			LogLevel:       new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedAgent, changeQANAgentOK.Payload.QANMysqlPerfschemaAgent)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyQANMysqlPerfschemaAgent{
			AgentID:                agentID,
			ServiceID:              serviceID,
			Username:               "changed-perfschema-user",
			PMMAgentID:             pmmAgentID,
			MaxQueryLength:         1536,
			QueryExamplesDisabled:  true,
			TLS:                    true,
			TLSSkipVerify:          false,
			DisableCommentsParsing: true,
			Disabled:               true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			ExtraDsnParams: map[string]string{},
			Status:         &AgentStatusDone,
			LogLevel:       new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.QANMysqlPerfschemaAgent)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
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

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID

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

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID
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

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Agent")).NodeID

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
