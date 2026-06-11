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

func TestQANMongoDBProfilerAgent(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MongoDB Profiler Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for QAN Profiler Agent test"),
			},
		})
		serviceID := service.Mongodb.ServiceID

		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"new_label": "QANMongodbProfilerAgent",
				},

				SkipConnectionCheck: true,
			},
		})
		agentID := res.QANMongodbProfilerAgent.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				QANMongodbProfilerAgent: &agents.GetAgentOKBodyQANMongodbProfilerAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "QANMongodbProfilerAgent",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeQANMongoDBProfilerAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANMongodbProfilerAgent: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				QANMongodbProfilerAgent: &agents.ChangeAgentOKBodyQANMongodbProfilerAgent{
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
		}, changeQANMongoDBProfilerAgentOK)

		changeQANMongoDBProfilerAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANMongodbProfilerAgent: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgentCustomLabels{
							Values: map[string]string{
								"new_label": "QANMongodbProfilerAgent",
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
				QANMongodbProfilerAgent: &agents.ChangeAgentOKBodyQANMongodbProfilerAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "QANMongodbProfilerAgent",
					},
					Status:   &AgentStatusDone,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeQANMongoDBProfilerAgentOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MongoDB Profiler password rotation")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for QAN Profiler password rotation test"),
			},
		})
		serviceID := service.Mongodb.ServiceID

		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN MongoDB Profiler agent with initial credentials
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
				ServiceID:           serviceID,
				Username:            "initial-mongodb-profiler-user",
				Password:            "initial-mongodb-profiler-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		agentID := res.QANMongodbProfilerAgent.AgentID

		// Test password rotation
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMongodbProfilerAgent: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
					Password: new("new-rotated-mongodb-profiler-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-mongodb-profiler-user", changeQANAgentOK.Payload.QANMongodbProfilerAgent.Username)
		assert.False(t, changeQANAgentOK.Payload.QANMongodbProfilerAgent.Disabled)

		// Verify password change with username change
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMongodbProfilerAgent: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
					Username: new("new-mongodb-profiler-user"),
					Password: new("another-new-mongodb-profiler-password"),
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
		assert.Equal(t, "new-mongodb-profiler-user", getAgentRes.Payload.QANMongodbProfilerAgent.Username)
		assert.False(t, getAgentRes.Payload.QANMongodbProfilerAgent.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MongoDB Profiler partial update")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for QAN Profiler partial update test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN MongoDB Profiler agent with comprehensive initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
				ServiceID:               serviceID,
				Username:                "initial-profiler-user",
				Password:                "initial-profiler-password",
				PMMAgentID:              pmmAgentID,
				MaxQueryLength:          2048,
				TLS:                     true,
				TLSSkipVerify:           false,
				AuthenticationMechanism: "MONGODB-CR",
				AuthenticationDatabase:  "admin",
				CustomLabels: map[string]string{
					"environment": "test",
					"team":        "dev",
				},
				LogLevel:            new("LOG_LEVEL_DEBUG"),
				SkipConnectionCheck: true,
			},
		})
		agentID := res.QANMongodbProfilerAgent.AgentID

		// Change only username, verify all other fields remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMongodbProfilerAgent: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
					Username: new("updated-profiler-user"),
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

		agent := getAgentRes.Payload.QANMongodbProfilerAgent
		assert.Equal(t, "updated-profiler-user", agent.Username) // Changed
		assert.Equal(t, int32(2048), agent.MaxQueryLength)       // Unchanged
		assert.True(t, agent.TLS)                                // Unchanged
		assert.False(t, agent.TLSSkipVerify)                     // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "test",
			"team":        "dev",
		}, agent.CustomLabels) // Unchanged
		assert.Equal(t, new("LOG_LEVEL_DEBUG"), agent.LogLevel) // Unchanged
		assert.False(t, agent.Disabled)                         // Unchanged
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN MongoDB Profiler change all fields")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for QAN Profiler change all fields test"),
			},
		})
		serviceID := service.Mongodb.ServiceID

		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create QAN MongoDB Profiler agent with initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
				ServiceID:      serviceID,
				Username:       "initial-mongodb-user",
				Password:       "initial-mongodb-password",
				PMMAgentID:     pmmAgentID,
				MaxQueryLength: 1024,
				TLS:            false,
				TLSSkipVerify:  true,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				LogLevel:            new("LOG_LEVEL_WARN"),
				SkipConnectionCheck: true,
			},
		})
		agentID := res.QANMongodbProfilerAgent.AgentID

		// Change ALL available fields at once
		changeQANAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANMongodbProfilerAgent: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
					Username:       new("changed-mongodb-user"),
					Password:       new("changed-mongodb-password"),
					MaxQueryLength: new(int32(4096)),
					TLS:            new(true),
					TLSSkipVerify:  new(false),
					CustomLabels: &agents.ChangeAgentParamsBodyQANMongodbProfilerAgentCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "backend",
						},
					},
					LogLevel: new("LOG_LEVEL_DEBUG"),
					Enable:   new(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyQANMongodbProfilerAgent{
			AgentID:        agentID,
			ServiceID:      serviceID,
			Username:       "changed-mongodb-user",
			PMMAgentID:     pmmAgentID,
			MaxQueryLength: 4096,
			TLS:            true,
			TLSSkipVerify:  false,
			Disabled:       true, // agent was disabled
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			Status:   &AgentStatusDone,
			LogLevel: new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedAgent, changeQANAgentOK.Payload.QANMongodbProfilerAgent)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyQANMongodbProfilerAgent{
			AgentID:        agentID,
			ServiceID:      serviceID,
			Username:       "changed-mongodb-user",
			PMMAgentID:     pmmAgentID,
			MaxQueryLength: 4096,
			TLS:            true,
			TLSSkipVerify:  false,
			Disabled:       true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			Status:   &AgentStatusDone,
			LogLevel: new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.QANMongodbProfilerAgent)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Profiler Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
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
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMongoDBProfilerAgentParams.ServiceId: value length must be at least 1 runes")

		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMongodbProfilerAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Profiler Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for agent"),
			},
		})
		serviceID := service.Mongodb.ServiceID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
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
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddQANMongoDBProfilerAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.QANMongodbProfilerAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Profiler Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.QANMongodbProfilerAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for QAN Profiler Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for not exists node ID"),
			},
		})
		serviceID := service.Mongodb.ServiceID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					QANMongodbProfilerAgent: &agents.AddAgentParamsBodyQANMongodbProfilerAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.QANMongodbProfilerAgent.AgentID)
		}
	})
}
