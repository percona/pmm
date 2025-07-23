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

func TestExternalExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for External Exporter test"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		ExternalExporter := addAgent(t, agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID: genericNodeID,
				ServiceID:    serviceID,
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_for_external_exporter": "external_exporter",
				},
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOKBody{
			ExternalExporter: &agents.GetAgentOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_for_external_exporter": "external_exporter",
				},
			},
		}, getAgentRes.Payload)
	})

	t.Run("Advanced", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for external exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      nodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for External Exporter test"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		ExternalExporter := addAgent(t, agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID: genericNodeID,
				ServiceID:    serviceID,
				Username:     "username",
				Password:     "password",
				Scheme:       "https",
				MetricsPath:  "/metrics-hr",
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_external_exporter": "external_exporter",
				},
				TLSSkipVerify: true,
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOKBody{
			ExternalExporter: &agents.GetAgentOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Username:     "username",
				Scheme:       "https",
				MetricsPath:  "/metrics-hr",
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_external_exporter": "external_exporter",
				},
				TLSSkipVerify: true,
			},
		}, getAgentRes.Payload)

		// Test change API.
		changeExternalExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyExternalExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			ExternalExporter: &agents.ChangeAgentOKBodyExternalExporter{
				AgentID:       agentID,
				ServiceID:     serviceID,
				RunsOnNodeID:  genericNodeID,
				Username:      "username",
				Scheme:        "https",
				MetricsPath:   "/metrics-hr",
				ListenPort:    12345,
				Disabled:      true,
				CustomLabels:  map[string]string{},
				TLSSkipVerify: true,
			},
		}, changeExternalExporterOK.Payload)

		changeExternalExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyExternalExporterCustomLabels{
							Values: map[string]string{
								"new_label": "external_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			ExternalExporter: &agents.ChangeAgentOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Username:     "username",
				Scheme:       "https",
				MetricsPath:  "/metrics-hr",
				ListenPort:   12345,
				Disabled:     false,
				CustomLabels: map[string]string{
					"new_label": "external_exporter",
				},
				TLSSkipVerify: true,
			},
		}, changeExternalExporterOK.Payload)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
					ServiceID:    "",
					RunsOnNodeID: genericNodeID,
					ListenPort:   12345,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Empty Service ID.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("AddListenPortEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for agent"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
					ServiceID:    serviceID,
					RunsOnNodeID: genericNodeID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddExternalExporterParams.ListenPort: value must be inside range (0, 65536)")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("AddRunsOnNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for agent"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
					ServiceID:    serviceID,
					RunsOnNodeID: "",
					ListenPort:   12345,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddExternalExporterParams.RunsOnNodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
					ServiceID:    "pmm-service-id",
					RunsOnNodeID: genericNodeID,
					ListenPort:   12345,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("NotExistNodeID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for not exists node ID"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
					ServiceID:    serviceID,
					RunsOnNodeID: "pmm-not-exist-server",
					ListenPort:   12345,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("WithPushMetrics", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)
		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for External Exporter test"),
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		ExternalExporter := addAgent(t, agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID: genericNodeID,
				ServiceID:    serviceID,
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_for_external_exporter": "external_exporter",
				},
				PushMetrics: true,
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOKBody{
			ExternalExporter: &agents.GetAgentOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_for_external_exporter": "external_exporter",
				},
				PushMetricsEnabled: true,
			},
		}, getAgentRes.Payload)

		// Test change API.
		changeExternalExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
						EnablePushMetrics: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			ExternalExporter: &agents.ChangeAgentOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_for_external_exporter": "external_exporter",
				},
				PushMetricsEnabled: false,
			},
		}, changeExternalExporterOK.Payload)

		changeExternalExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOKBody{
			ExternalExporter: &agents.ChangeAgentOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Scheme:       "http",
				MetricsPath:  "/metrics",
				ListenPort:   12345,
				CustomLabels: map[string]string{
					"custom_label_for_external_exporter": "external_exporter",
				},
				PushMetricsEnabled: true,
			},
		}, changeExternalExporterOK.Payload)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for External Exporter field changes test"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		// Create External Exporter with initial configuration (Note: External Exporter ChangeAgent API doesn't support password changes)
		ExternalExporter := addAgent(t, agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID: genericNodeID,
				ServiceID:    serviceID,
				Username:     "initial-external-user",
				Password:     "initial-external-password",
				Scheme:       "https",
				MetricsPath:  "/metrics",
				ListenPort:   9090,
				CustomLabels: map[string]string{
					"environment": "test",
				},
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing username (External Exporter ChangeAgent doesn't support password changes)
		changeExternalExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
					Username: pointer.ToString("rotated-external-user"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "rotated-external-user", changeExternalExporterOK.Payload.ExternalExporter.Username)
		assert.False(t, changeExternalExporterOK.Payload.ExternalExporter.Disabled)

		// Test changing scheme and metrics path
		changeExternalExporterOK, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
					Scheme:      pointer.ToString("http"),
					MetricsPath: pointer.ToString("/new-metrics"),
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
		assert.Equal(t, "rotated-external-user", getAgentRes.Payload.ExternalExporter.Username)
		assert.Equal(t, "http", getAgentRes.Payload.ExternalExporter.Scheme)
		assert.Equal(t, "/new-metrics", getAgentRes.Payload.ExternalExporter.MetricsPath)
		assert.False(t, getAgentRes.Payload.ExternalExporter.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for External Exporter partial update test"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create External Exporter with comprehensive initial configuration
		ExternalExporter := addAgent(t, agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID: genericNodeID,
				ServiceID:    serviceID,
				Username:     "initial-external-user",
				Password:     "initial-external-password",
				Scheme:       "https",
				MetricsPath:  "/custom-metrics",
				ListenPort:   8080,
				CustomLabels: map[string]string{
					"environment": "staging",
					"team":        "monitoring",
					"region":      "us-west",
				},
				PushMetrics: true,
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change only username, verify all other fields remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
					Username: pointer.ToString("changed-external-user"),
					// Note: custom labels, scheme, metrics path, push metrics are NOT specified
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

		agent := getAgentRes.Payload.ExternalExporter
		// Username should be changed
		assert.Equal(t, "changed-external-user", agent.Username)

		// Everything else should remain unchanged
		assert.Equal(t, "https", agent.Scheme)
		assert.Equal(t, "/custom-metrics", agent.MetricsPath)
		assert.Equal(t, int64(8080), agent.ListenPort)
		assert.True(t, agent.PushMetricsEnabled)
		assert.Equal(t, map[string]string{
			"environment": "staging",
			"team":        "monitoring",
			"region":      "us-west",
		}, agent.CustomLabels)
		assert.False(t, agent.Disabled)
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "External Service for External Exporter change all fields test"),
				Group:       "external",
			},
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create External Exporter with initial configuration
		ExternalExporter := addAgent(t, agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID: genericNodeID,
				ServiceID:    serviceID,
				Username:     "initial-external-user",
				Password:     "initial-external-password",
				Scheme:       "http",
				MetricsPath:  "/metrics",
				ListenPort:   9090,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				PushMetrics: false,
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change ALL available fields at once (External Exporter ChangeAgent doesn't support password changes)
		changeExternalExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				ExternalExporter: &agents.ChangeAgentParamsBodyExternalExporter{
					Username:    pointer.ToString("changed-external-user"),
					Scheme:      pointer.ToString("https"),
					MetricsPath: pointer.ToString("/new-metrics"),
					ListenPort:  pointer.ToInt64(8080),
					CustomLabels: &agents.ChangeAgentParamsBodyExternalExporterCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "infrastructure",
						},
					},
					EnablePushMetrics: pointer.ToBool(true),
					Enable:            pointer.ToBool(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyExternalExporter{
			AgentID:            agentID,
			ServiceID:          serviceID,
			RunsOnNodeID:       genericNodeID,
			Username:           "changed-external-user",
			Scheme:             "https",
			MetricsPath:        "/new-metrics",
			ListenPort:         8080,
			Disabled:           true, // agent was disabled
			PushMetricsEnabled: true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "infrastructure",
			},
		}

		assert.Equal(t, expectedAgent, changeExternalExporterOK.Payload.ExternalExporter)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyExternalExporter{
			AgentID:            agentID,
			ServiceID:          serviceID,
			RunsOnNodeID:       genericNodeID,
			Username:           "changed-external-user",
			Scheme:             "https",
			MetricsPath:        "/new-metrics",
			ListenPort:         8080,
			Disabled:           true,
			PushMetricsEnabled: true,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "infrastructure",
			},
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.ExternalExporter)
	})
}
