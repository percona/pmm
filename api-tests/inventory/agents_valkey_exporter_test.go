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

func TestValkeyExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Valkey: &services.AddServiceParamsBodyValkey{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6379,
				ServiceName: pmmapitests.TestString(t, "Valkey Service for ValkeyExporter test"),
			},
		})
		serviceID := service.Valkey.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		valkeyExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
				ServiceID:  serviceID,
				Username:   "default",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label": "test",
				},
				SkipConnectionCheck: true,
			},
		})
		agentID := valkeyExporter.ValkeyExporter.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				ValkeyExporter: &agents.GetAgentOKBodyValkeyExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "default",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label": "test",
					},
					DisabledCollectors: make([]string, 0),
					Status:             &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeValkeyExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ValkeyExporter: &agents.ChangeAgentParamsBodyValkeyExporter{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyValkeyExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				ValkeyExporter: &agents.ChangeAgentOKBodyValkeyExporter{
					AgentID:            agentID,
					ServiceID:          serviceID,
					Username:           "default",
					PMMAgentID:         pmmAgentID,
					Disabled:           true,
					Status:             &AgentStatusDone,
					CustomLabels:       map[string]string{},
					DisabledCollectors: make([]string, 0),
				},
			},
		}, changeValkeyExporterOK)

		changeValkeyExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ValkeyExporter: &agents.ChangeAgentParamsBodyValkeyExporter{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyValkeyExporterCustomLabels{
							Values: map[string]string{
								"new_label": "valkey_exporter",
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
				ValkeyExporter: &agents.ChangeAgentOKBodyValkeyExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "default",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "valkey_exporter",
					},
					Status:             &AgentStatusDone,
					DisabledCollectors: make([]string, 0),
				},
			},
		}, changeValkeyExporterOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
					ServiceID:  "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddValkeyExporterParams.ServiceId: value must be a valid UUID")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.ValkeyExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Valkey: &services.AddServiceParamsBodyValkey{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6379,
				ServiceName: pmmapitests.TestString(t, "Valkey Service for agent"),
			},
		})
		serviceID := service.Valkey.ServiceID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
					ServiceID:  serviceID,
					PMMAgentID: "",
					Username:   "default",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddValkeyExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ValkeyExporter.AgentID)
		}
	})

	t.Run("NonExistentServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
					ServiceID:  "00000000-0000-0000-0000-000000000000",
					PMMAgentID: pmmAgentID,
					Username:   "default",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"00000000-0000-0000-0000-000000000000\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ValkeyExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Valkey: &services.AddServiceParamsBodyValkey{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6379,
				ServiceName: pmmapitests.TestString(t, "Valkey Service for not exists node ID"),
			},
		})
		serviceID := service.Valkey.ServiceID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
					ServiceID:  serviceID,
					PMMAgentID: "pmm-not-exist-server",
					Username:   "default",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ValkeyExporter.AgentID)
		}
	})

	t.Run("With PushMetrics", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter")).NodeID

		service := pmmapitests.AddService(t, services.AddServiceBody{
			Valkey: &services.AddServiceParamsBodyValkey{
				NodeID:      genericNodeID,
				Address:     pmmapitests.TestString(t, "localhost"),
				Port:        6379,
				ServiceName: pmmapitests.TestString(t, "Valkey Service for ValkeyExporter test"),
			},
		})
		serviceID := service.Valkey.ServiceID
		pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

		valkeyExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
				ServiceID:  serviceID,
				Username:   "default",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_valkey_exporter": "valkey_exporter",
				},
				SkipConnectionCheck: true,
				PushMetrics:         true,
			},
		})
		agentID := valkeyExporter.ValkeyExporter.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				ValkeyExporter: &agents.GetAgentOKBodyValkeyExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "default",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_valkey_exporter": "valkey_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeValkeyExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ValkeyExporter: &agents.ChangeAgentParamsBodyValkeyExporter{
						EnablePushMetrics: new(false),
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				ValkeyExporter: &agents.ChangeAgentOKBodyValkeyExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "default",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_valkey_exporter": "valkey_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
				},
			},
		}, changeValkeyExporterOK)

		changeValkeyExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ValkeyExporter: &agents.ChangeAgentParamsBodyValkeyExporter{
						EnablePushMetrics: new(true),
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				ValkeyExporter: &agents.ChangeAgentOKBodyValkeyExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "default",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_valkey_exporter": "valkey_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
				},
			},
		}, changeValkeyExporterOK)
	})
}
