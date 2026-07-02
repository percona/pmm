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

func TestRTAMongoDBAgent(t *testing.T) {
	t.Parallel()

	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MongoDB Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for RTA Profiler Agent test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		t.Cleanup(func() {
			pmmapitests.RemoveAgents(t, pmmAgentID)
		})

		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"new_label": "RTAMongodbAgent",
				},
				RtaOptions: &agents.AddAgentParamsBodyRtaMongodbAgentRtaOptions{
					CollectInterval: "5s",
				},

				SkipConnectionCheck: true,
			},
		})
		agentID := res.RtaMongodbAgent.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				RtaMongodbAgent: &agents.GetAgentOKBodyRtaMongodbAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "RTAMongodbAgent",
					},
					RtaOptions: &agents.GetAgentOKBodyRtaMongodbAgentRtaOptions{
						CollectInterval: "5s",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeRTAMongoDBAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RtaMongodbAgent: &agents.ChangeAgentParamsBodyRtaMongodbAgent{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyRtaMongodbAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				RtaMongodbAgent: &agents.ChangeAgentOKBodyRtaMongodbAgent{
					AgentID:      agentID,
					ServiceID:    serviceID,
					Username:     "username",
					PMMAgentID:   pmmAgentID,
					Disabled:     true,
					Status:       &AgentStatusDone,
					CustomLabels: map[string]string{},
					RtaOptions: &agents.ChangeAgentOKBodyRtaMongodbAgentRtaOptions{
						CollectInterval: "5s",
					},
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRTAMongoDBAgentOK)

		changeRTAMongoDBAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RtaMongodbAgent: &agents.ChangeAgentParamsBodyRtaMongodbAgent{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyRtaMongodbAgentCustomLabels{
							Values: map[string]string{
								"new_label": "RTAMongodbAgent",
							},
						},
						RtaOptions: &agents.ChangeAgentParamsBodyRtaMongodbAgentRtaOptions{
							CollectInterval: "10s",
						},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				RtaMongodbAgent: &agents.ChangeAgentOKBodyRtaMongodbAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "RTAMongodbAgent",
					},
					RtaOptions: &agents.ChangeAgentOKBodyRtaMongodbAgentRtaOptions{
						CollectInterval: "10s",
					},
					Status:   &AgentStatusDone,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRTAMongoDBAgentOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MongoDB password rotation")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for RTA password rotation test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create RTA MongoDB agent with initial credentials
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
				ServiceID:           serviceID,
				Username:            "initial-rta-mongodb-user",
				Password:            "initial-rta-mongodb-password",
				PMMAgentID:          pmmAgentID,
				SkipConnectionCheck: true,
			},
		})
		agentID := res.RtaMongodbAgent.AgentID

		// Test password rotation
		changeRTAAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RtaMongodbAgent: &agents.ChangeAgentParamsBodyRtaMongodbAgent{
					Password: new("new-rotated-rta-mongodb-password"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "initial-rta-mongodb-user", changeRTAAgentOK.Payload.RtaMongodbAgent.Username)
		assert.False(t, changeRTAAgentOK.Payload.RtaMongodbAgent.Disabled)

		// Verify password change with username change
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RtaMongodbAgent: &agents.ChangeAgentParamsBodyRtaMongodbAgent{
					Username: new("new-rta-mongodb-user"),
					Password: new("another-new-rta-mongodb-password"),
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
		assert.Equal(t, "new-rta-mongodb-user", getAgentRes.Payload.RtaMongodbAgent.Username)
		assert.False(t, getAgentRes.Payload.RtaMongodbAgent.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MongoDB partial update")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for RTA partial update test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create RTA MongoDB Profiler agent with comprehensive initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
				ServiceID:               serviceID,
				Username:                "initial-rta-user",
				Password:                "initial-rta-password",
				PMMAgentID:              pmmAgentID,
				TLS:                     true,
				TLSSkipVerify:           false,
				AuthenticationMechanism: "MONGODB-CR",
				CustomLabels: map[string]string{
					"environment": "test",
					"team":        "dev",
				},
				RtaOptions: &agents.AddAgentParamsBodyRtaMongodbAgentRtaOptions{
					CollectInterval: "6s",
				},
				LogLevel:            new("LOG_LEVEL_DEBUG"),
				SkipConnectionCheck: true,
			},
		})
		agentID := res.RtaMongodbAgent.AgentID

		// Change only username, verify all other fields remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RtaMongodbAgent: &agents.ChangeAgentParamsBodyRtaMongodbAgent{
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

		agent := getAgentRes.Payload.RtaMongodbAgent
		assert.Equal(t, "updated-profiler-user", agent.Username) // Changed
		assert.True(t, agent.TLS)                                // Unchanged
		assert.False(t, agent.TLSSkipVerify)                     // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "test",
			"team":        "dev",
		}, agent.CustomLabels) // Unchanged
		assert.Equal(t, "6s", agent.RtaOptions.CollectInterval) // Unchanged
		assert.Equal(t, new("LOG_LEVEL_DEBUG"), agent.LogLevel) // Unchanged
		assert.False(t, agent.Disabled)                         // Unchanged
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MongoDB change all fields")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for RTA Profiler change all fields test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create RTA MongoDB Profiler agent with initial configuration
		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
				ServiceID:     serviceID,
				Username:      "initial-mongodb-user",
				Password:      "initial-mongodb-password",
				PMMAgentID:    pmmAgentID,
				TLS:           false,
				TLSSkipVerify: true,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				RtaOptions: &agents.AddAgentParamsBodyRtaMongodbAgentRtaOptions{
					CollectInterval: "6s",
				},
				LogLevel:            new("LOG_LEVEL_WARN"),
				SkipConnectionCheck: true,
			},
		})
		agentID := res.RtaMongodbAgent.AgentID

		// Change ALL available fields at once
		changeRTAAgentOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RtaMongodbAgent: &agents.ChangeAgentParamsBodyRtaMongodbAgent{
					Username:      new("changed-mongodb-user"),
					Password:      new("changed-mongodb-password"),
					TLS:           new(true),
					TLSSkipVerify: new(false),
					CustomLabels: &agents.ChangeAgentParamsBodyRtaMongodbAgentCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "backend",
						},
					},
					RtaOptions: &agents.ChangeAgentParamsBodyRtaMongodbAgentRtaOptions{
						CollectInterval: "10s",
					},
					LogLevel: new("LOG_LEVEL_DEBUG"),
					Enable:   new(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyRtaMongodbAgent{
			AgentID:       agentID,
			ServiceID:     serviceID,
			Username:      "changed-mongodb-user",
			PMMAgentID:    pmmAgentID,
			TLS:           true,
			TLSSkipVerify: false,
			Disabled:      true, // agent was disabled
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			RtaOptions: &agents.ChangeAgentOKBodyRtaMongodbAgentRtaOptions{
				CollectInterval: "10s",
			},
			Status:   &AgentStatusDone,
			LogLevel: new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedAgent, changeRTAAgentOK.Payload.RtaMongodbAgent)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyRtaMongodbAgent{
			AgentID:       agentID,
			ServiceID:     serviceID,
			Username:      "changed-mongodb-user",
			PMMAgentID:    pmmAgentID,
			TLS:           true,
			TLSSkipVerify: false,
			Disabled:      true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "backend",
			},
			RtaOptions: &agents.GetAgentOKBodyRtaMongodbAgentRtaOptions{
				CollectInterval: "10s",
			},
			Status:   &AgentStatusDone,
			LogLevel: new("LOG_LEVEL_DEBUG"),
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.RtaMongodbAgent)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
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
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRTAMongoDBAgentParams.ServiceId: value length must be at least 1 runes")

		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.RtaMongodbAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for RTA agent"),
			},
		})
		serviceID := service.Mongodb.ServiceID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
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
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRTAMongoDBAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.RtaMongodbAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.RtaMongodbAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA Agent")).NodeID

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
					RtaMongodbAgent: &agents.AddAgentParamsBodyRtaMongodbAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.RtaMongodbAgent.AgentID)
		}
	})
}
