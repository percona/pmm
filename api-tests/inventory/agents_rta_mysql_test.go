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

func TestRTAMySQLAgent(t *testing.T) {
	t.Parallel()

	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MySQL Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for RTA Agent test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		t.Cleanup(func() {
			pmmapitests.RemoveAgents(t, pmmAgentID)
		})

		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"new_label": "RTAMysqlAgent",
				},
				RtaOptions: &agents.AddAgentParamsBodyRtaMysqlAgentRtaOptions{
					CollectInterval: "5s",
				},

				SkipConnectionCheck: true,
			},
		})
		agentID := res.RtaMysqlAgent.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				RtaMysqlAgent: &agents.GetAgentOKBodyRtaMysqlAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "RTAMysqlAgent",
					},
					RtaOptions: &agents.GetAgentOKBodyRtaMysqlAgentRtaOptions{
						CollectInterval: "5s",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API: disable, then re-enable with new labels and interval.
		changeRTAMySQLAgentOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RtaMysqlAgent: &agents.ChangeAgentParamsBodyRtaMysqlAgent{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyRtaMysqlAgentCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				RtaMysqlAgent: &agents.ChangeAgentOKBodyRtaMysqlAgent{
					AgentID:      agentID,
					ServiceID:    serviceID,
					Username:     "username",
					PMMAgentID:   pmmAgentID,
					Disabled:     true,
					Status:       &AgentStatusDone,
					CustomLabels: map[string]string{},
					RtaOptions: &agents.ChangeAgentOKBodyRtaMysqlAgentRtaOptions{
						CollectInterval: "5s",
					},
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRTAMySQLAgentOK)

		changeRTAMySQLAgentOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RtaMysqlAgent: &agents.ChangeAgentParamsBodyRtaMysqlAgent{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyRtaMysqlAgentCustomLabels{
							Values: map[string]string{
								"new_label": "RTAMysqlAgent",
							},
						},
						RtaOptions: &agents.ChangeAgentParamsBodyRtaMysqlAgentRtaOptions{
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
				RtaMysqlAgent: &agents.ChangeAgentOKBodyRtaMysqlAgent{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "RTAMysqlAgent",
					},
					RtaOptions: &agents.ChangeAgentOKBodyRtaMysqlAgentRtaOptions{
						CollectInterval: "10s",
					},
					Status:   &AgentStatusDone,
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRTAMySQLAgentOK)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MySQL partial update")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for RTA partial update test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
				ServiceID:     serviceID,
				Username:      "initial-rta-user",
				Password:      "initial-rta-password",
				PMMAgentID:    pmmAgentID,
				TLS:           true,
				TLSSkipVerify: false,
				CustomLabels: map[string]string{
					"environment": "test",
					"team":        "dev",
				},
				RtaOptions: &agents.AddAgentParamsBodyRtaMysqlAgentRtaOptions{
					CollectInterval: "6s",
				},
				LogLevel:            new("LOG_LEVEL_DEBUG"),
				SkipConnectionCheck: true,
			},
		})
		agentID := res.RtaMysqlAgent.AgentID

		// Change only username; everything else must remain unchanged.
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RtaMysqlAgent: &agents.ChangeAgentParamsBodyRtaMysqlAgent{
					Username: new("updated-user"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		agent := getAgentRes.Payload.RtaMysqlAgent
		assert.Equal(t, "updated-user", agent.Username) // Changed
		assert.True(t, agent.TLS)                       // Unchanged
		assert.False(t, agent.TLSSkipVerify)            // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "test",
			"team":        "dev",
		}, agent.CustomLabels) // Unchanged
		assert.Equal(t, "6s", agent.RtaOptions.CollectInterval) // Unchanged
		assert.Equal(t, new("LOG_LEVEL_DEBUG"), agent.LogLevel) // Unchanged
		assert.False(t, agent.Disabled)                         // Unchanged
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MySQL Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
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
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRTAMySQLAgentParams.ServiceId: value length must be at least 1 runes")

		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.RtaMysqlAgent.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MySQL Agent")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for RTA agent"),
			},
		})
		serviceID := service.Mysql.ServiceID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
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
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRTAMySQLAgentParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.RtaMysqlAgent.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MySQL Agent")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(
			&agents.AddAgentParams{
				Body: agents.AddAgentBody{
					RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.RtaMysqlAgent.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Test Generic Node for RTA MySQL Agent")).NodeID

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
					RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
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
			pmmapitests.RemoveAgents(t, res.Payload.RtaMysqlAgent.AgentID)
		}
	})
}
