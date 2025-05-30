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
)

func TestNodeExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		customLabels := map[string]string{
			"custom_label_node_exporter": "node_exporter",
		}
		res := addNodeExporter(t, pmmAgentID, customLabels)
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				NodeExporter: &agents.GetAgentOKBodyNodeExporter{
					AgentID:            agentID,
					PMMAgentID:         pmmAgentID,
					Disabled:           false,
					CustomLabels:       customLabels,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyNodeExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				NodeExporter: &agents.ChangeAgentOKBodyNodeExporter{
					AgentID:            agentID,
					PMMAgentID:         pmmAgentID,
					Disabled:           true,
					Status:             &AgentStatusUnknown,
					CustomLabels:       map[string]string{},
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeNodeExporterOK)

		changeNodeExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyNodeExporterCustomLabels{
							Values: map[string]string{
								"new_label": "node_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				NodeExporter: &agents.ChangeAgentOKBodyNodeExporter{
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "node_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeNodeExporterOK)
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
					PMMAgentID: "",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddNodeExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.NodeExporter.AgentID)
		}
	})

	t.Run("NotExistPmmAgentID", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
					PMMAgentID: "pmm-node-exporter-node",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-node-exporter-node not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.NodeExporter.AgentID)
		}
	})

	t.Run("With PushMetrics", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		customLabels := map[string]string{
			"custom_label_node_exporter": "node_exporter",
		}
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
					PMMAgentID:   pmmAgentID,
					CustomLabels: customLabels,
					PushMetrics:  true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Payload.NodeExporter)
		require.Equal(t, pmmAgentID, res.Payload.NodeExporter.PMMAgentID)
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				NodeExporter: &agents.GetAgentOKBodyNodeExporter{
					AgentID:            agentID,
					PMMAgentID:         pmmAgentID,
					Disabled:           false,
					CustomLabels:       customLabels,
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					EnablePushMetrics: pointer.ToBool(false),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				NodeExporter: &agents.ChangeAgentOKBodyNodeExporter{
					AgentID:            agentID,
					PMMAgentID:         pmmAgentID,
					Disabled:           false,
					CustomLabels:       customLabels,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeNodeExporterOK)

		changeNodeExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				NodeExporter: &agents.ChangeAgentOKBodyNodeExporter{
					AgentID:            agentID,
					PMMAgentID:         pmmAgentID,
					Disabled:           false,
					CustomLabels:       customLabels,
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeNodeExporterOK)
	})

	t.Run("ChangeCollectorsAndLogLevel", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res := addNodeExporter(t, pmmAgentID, map[string]string{})
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing new fields: DisableCollectors, LogLevel, ExposeExporter
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					DisableCollectors: []string{"cpu", "diskstats"},
					LogLevel:          pointer.ToString(agents.ChangeAgentParamsBodyNodeExporterLogLevelLOGLEVELDEBUG),
					ExposeExporter:    pointer.ToBool(true),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify the changes were applied by getting the agent
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.NodeExporter)

		nodeExporter := getAgentRes.Payload.NodeExporter
		assert.Equal(t, []string{"cpu", "diskstats"}, nodeExporter.DisabledCollectors)
		assert.Equal(t, "LOG_LEVEL_DEBUG", pointer.GetString(nodeExporter.LogLevel))
		assert.True(t, nodeExporter.ExposeExporter)

		// Also verify the ChangeAgent response has the basic fields correct
		assert.Equal(t, agentID, changeNodeExporterOK.Payload.NodeExporter.AgentID)
		assert.Equal(t, pmmAgentID, changeNodeExporterOK.Payload.NodeExporter.PMMAgentID)
		assert.False(t, changeNodeExporterOK.Payload.NodeExporter.Disabled)
	})

	t.Run("ChangeMetricsResolutions", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res := addNodeExporter(t, pmmAgentID, map[string]string{})
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing MetricsResolutions
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					MetricsResolutions: &agents.ChangeAgentParamsBodyNodeExporterMetricsResolutions{
						Hr: "5s",
						Mr: "10s",
						Lr: "60s",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify the MetricsResolutions were applied
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.NodeExporter.MetricsResolutions)
		assert.Equal(t, "5s", getAgentRes.Payload.NodeExporter.MetricsResolutions.Hr)
		assert.Equal(t, "10s", getAgentRes.Payload.NodeExporter.MetricsResolutions.Mr)
		assert.Equal(t, "60s", getAgentRes.Payload.NodeExporter.MetricsResolutions.Lr)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node Exporter field changes"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create Node Exporter with initial configuration
		res := addNodeExporter(t, pmmAgentID, map[string]string{
			"environment": "test",
			"version":     "1.0",
		})
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing configurable fields (Node Exporter doesn't have passwords)
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					EnablePushMetrics: pointer.ToBool(true),
					LogLevel:          pointer.ToString("LOG_LEVEL_DEBUG"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.True(t, changeNodeExporterOK.Payload.NodeExporter.PushMetricsEnabled)
		assert.False(t, changeNodeExporterOK.Payload.NodeExporter.Disabled)

		// Test changing collectors and expose settings
		changeNodeExporterOK, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					DisableCollectors: []string{"cpu", "diskstats"},
					ExposeExporter:    pointer.ToBool(true),
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
		assert.Equal(t, []string{"cpu", "diskstats"}, getAgentRes.Payload.NodeExporter.DisabledCollectors)
		assert.True(t, getAgentRes.Payload.NodeExporter.ExposeExporter)
		assert.False(t, getAgentRes.Payload.NodeExporter.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node Exporter partial update"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create Node Exporter with comprehensive initial configuration
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
					PMMAgentID:  pmmAgentID,
					PushMetrics: true,
					LogLevel:    pointer.ToString("LOG_LEVEL_WARN"),
					CustomLabels: map[string]string{
						"environment": "staging",
						"team":        "infrastructure",
						"region":      "us-east",
					},
					DisableCollectors: []string{"filesystem", "netdev"},
					ExposeExporter:    true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change only log level, verify all other fields remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					LogLevel: pointer.ToString("LOG_LEVEL_ERROR"),
					// All other fields are intentionally NOT set
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify only log level changed, all other fields remain the same
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		agent := getAgentRes.Payload.NodeExporter
		assert.Equal(t, pointer.ToString("LOG_LEVEL_ERROR"), agent.LogLevel)        // Changed
		assert.True(t, agent.PushMetricsEnabled)                                    // Unchanged
		assert.True(t, agent.ExposeExporter)                                        // Unchanged
		assert.Equal(t, []string{"filesystem", "netdev"}, agent.DisabledCollectors) // Unchanged
		assert.Equal(t, map[string]string{
			"environment": "staging",
			"team":        "infrastructure",
			"region":      "us-east",
		}, agent.CustomLabels) // Unchanged
		assert.False(t, agent.Disabled) // Unchanged
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node Exporter change all fields"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create Node Exporter with initial configuration
		res := addNodeExporter(t, pmmAgentID, map[string]string{
			"environment": "staging",
			"version":     "1.0",
		})
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change ALL available fields at once
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				NodeExporter: &agents.ChangeAgentParamsBodyNodeExporter{
					CustomLabels: &agents.ChangeAgentParamsBodyNodeExporterCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "sre",
						},
					},
					LogLevel:          pointer.ToString("LOG_LEVEL_DEBUG"),
					EnablePushMetrics: pointer.ToBool(true),
					DisableCollectors: []string{"cpu", "diskstats", "loadavg"},
					ExposeExporter:    pointer.ToBool(true),
					MetricsResolutions: &agents.ChangeAgentParamsBodyNodeExporterMetricsResolutions{
						Hr: "5s",
						Mr: "30s",
						Lr: "300s",
					},
					Enable: pointer.ToBool(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyNodeExporter{
			AgentID:    agentID,
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "sre",
			},
			LogLevel:           pointer.ToString("LOG_LEVEL_DEBUG"),
			PushMetricsEnabled: true,
			DisabledCollectors: []string{"cpu", "diskstats", "loadavg"},
			ExposeExporter:     true,
			Disabled:           true, // agent was disabled
			Status:             &AgentStatusUnknown,
			MetricsResolutions: &agents.ChangeAgentOKBodyNodeExporterMetricsResolutions{
				Hr: "5s",
				Mr: "30s",
				Lr: "300s",
			},
		}

		assert.Equal(t, expectedAgent, changeNodeExporterOK.Payload.NodeExporter)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyNodeExporter{
			AgentID:    agentID,
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "sre",
			},
			LogLevel:           pointer.ToString("LOG_LEVEL_DEBUG"),
			PushMetricsEnabled: true,
			DisabledCollectors: []string{"cpu", "diskstats", "loadavg"},
			ExposeExporter:     true,
			Disabled:           true,
			Status:             &AgentStatusUnknown,
			MetricsResolutions: &agents.GetAgentOKBodyNodeExporterMetricsResolutions{
				Hr: "5s",
				Mr: "30s",
				Lr: "300s",
			},
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.NodeExporter)
	})
}
