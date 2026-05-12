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
)

func TestRDSExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteRDSNode(t, pmmapitests.TestString(t, "Remote node for RDS exporter")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		rdsExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
				NodeID:     nodeID,
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_rds_exporter": "rds_exporter",
				},
				SkipConnectionCheck:    true,
				DisableBasicMetrics:    true,
				DisableEnhancedMetrics: true,
			},
		})
		agentID := rdsExporter.RDSExporter.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				RDSExporter: &agents.GetAgentOKBodyRDSExporter{
					NodeID:     nodeID,
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_rds_exporter": "rds_exporter",
					},
					BasicMetricsDisabled:    true,
					EnhancedMetricsDisabled: true,
					Status:                  &AgentStatusUnknown,
					LogLevel:                new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeRDSExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
						Enable:       new(false),
						CustomLabels: &agents.ChangeAgentParamsBodyRDSExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				RDSExporter: &agents.ChangeAgentOKBodyRDSExporter{
					NodeID:                  nodeID,
					AgentID:                 agentID,
					PMMAgentID:              pmmAgentID,
					Disabled:                true,
					BasicMetricsDisabled:    true,
					EnhancedMetricsDisabled: true,
					Status:                  &AgentStatusDone,
					CustomLabels:            map[string]string{},
					LogLevel:                new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRDSExporterOK)

		changeRDSExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
						Enable: new(true),
						CustomLabels: &agents.ChangeAgentParamsBodyRDSExporterCustomLabels{
							Values: map[string]string{
								"new_label": "rds_exporter",
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
				RDSExporter: &agents.ChangeAgentOKBodyRDSExporter{
					NodeID:     nodeID,
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "rds_exporter",
					},
					BasicMetricsDisabled:    true,
					EnhancedMetricsDisabled: true,
					Status:                  &AgentStatusDone,
					LogLevel:                new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRDSExporterOK)
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
					NodeID:     "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRDSExporterParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.RDSExporter.AgentID)
		}
	})

	t.Run("NotExistNodeID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
					NodeID:     "pmm-node-id",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID \"pmm-node-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.RDSExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
					NodeID:     "nodeID",
					PMMAgentID: "pmm-not-exist-server",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.RDSExporter.AgentID)
		}
	})

	t.Run("With PushMetrics", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteRDSNode(t, pmmapitests.TestString(t, "Remote node for RDS exporter")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		rdsExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
				NodeID:     nodeID,
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_rds_exporter": "rds_exporter",
				},
				SkipConnectionCheck:    true,
				DisableBasicMetrics:    true,
				DisableEnhancedMetrics: true,
			},
		})
		agentID := rdsExporter.RDSExporter.AgentID

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)

		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				RDSExporter: &agents.GetAgentOKBodyRDSExporter{
					NodeID:     nodeID,
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_rds_exporter": "rds_exporter",
					},
					BasicMetricsDisabled:    true,
					EnhancedMetricsDisabled: true,
					Status:                  &AgentStatusUnknown,
					LogLevel:                new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeRDSExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
						EnablePushMetrics: new(true),
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				RDSExporter: &agents.ChangeAgentOKBodyRDSExporter{
					NodeID:     nodeID,
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_rds_exporter": "rds_exporter",
					},
					BasicMetricsDisabled:    true,
					EnhancedMetricsDisabled: true,
					PushMetricsEnabled:      true,
					Status:                  &AgentStatusUnknown,
					LogLevel:                new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRDSExporterOK)

		changeRDSExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
						EnablePushMetrics: new(false),
					},
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				RDSExporter: &agents.ChangeAgentOKBodyRDSExporter{
					NodeID:     nodeID,
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_rds_exporter": "rds_exporter",
					},
					BasicMetricsDisabled:    true,
					EnhancedMetricsDisabled: true,
					Status:                  &AgentStatusUnknown,
					LogLevel:                new("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeRDSExporterOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteRDSNode(t, pmmapitests.TestString(t, "Remote RDS node for credential rotation test")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create RDS Exporter with initial AWS credentials
		rdsExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
				NodeID:       nodeID,
				PMMAgentID:   pmmAgentID,
				AWSAccessKey: "initial-access-key",
				AWSSecretKey: "initial-secret-key",
				LogLevel:     new("LOG_LEVEL_WARN"),
				CustomLabels: map[string]string{
					"environment": "test",
				},
				SkipConnectionCheck: true,
			},
		})
		agentID := rdsExporter.RDSExporter.AgentID

		// Test AWS secret key rotation
		changeRDSExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
					AWSSecretKey: new("rotated-secret-key"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.False(t, changeRDSExporterOK.Payload.RDSExporter.Disabled)

		// Test AWS access key and secret key rotation together
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
					AWSAccessKey: new("new-access-key"),
					AWSSecretKey: new("new-secret-key"),
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
		assert.Equal(t, "new-access-key", getAgentRes.Payload.RDSExporter.AWSAccessKey)
		assert.False(t, getAgentRes.Payload.RDSExporter.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteRDSNode(t, pmmapitests.TestString(t, "Remote RDS node for partial update test")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create RDS Exporter with comprehensive initial configuration
		rdsExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
				NodeID:                 nodeID,
				PMMAgentID:             pmmAgentID,
				AWSAccessKey:           "initial-access-key",
				AWSSecretKey:           "initial-secret-key",
				LogLevel:               new("LOG_LEVEL_INFO"),
				DisableBasicMetrics:    true,
				DisableEnhancedMetrics: false,
				CustomLabels: map[string]string{
					"environment": "staging",
					"team":        "sre",
					"region":      "us-east-1",
				},
				SkipConnectionCheck: true,
				PushMetrics:         true,
			},
		})
		agentID := rdsExporter.RDSExporter.AgentID

		// Change only log level, verify all other fields remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
					LogLevel: new("LOG_LEVEL_DEBUG"),
					// Note: AWS keys, custom labels, metrics settings are NOT specified
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

		agent := getAgentRes.Payload.RDSExporter
		// Log level should be changed
		assert.Equal(t, new("LOG_LEVEL_DEBUG"), agent.LogLevel)

		// Everything else should remain unchanged
		assert.Equal(t, "initial-access-key", agent.AWSAccessKey)
		assert.True(t, agent.BasicMetricsDisabled)
		assert.False(t, agent.EnhancedMetricsDisabled)
		assert.True(t, agent.PushMetricsEnabled)
		assert.Equal(t, map[string]string{
			"environment": "staging",
			"team":        "sre",
			"region":      "us-east-1",
		}, agent.CustomLabels)
		assert.False(t, agent.Disabled)
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		nodeID := pmmapitests.AddRemoteRDSNode(t, pmmapitests.TestString(t, "Remote RDS node for change all fields test")).NodeID
		pmmAgentID := pmmapitests.AddPMMAgent(t, genericNodeID).AgentID

		// Create RDS Exporter with initial configuration
		rdsExporter := pmmapitests.AddAgent(t, agents.AddAgentBody{
			RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
				NodeID:                 nodeID,
				PMMAgentID:             pmmAgentID,
				AWSAccessKey:           "initial-access-key",
				AWSSecretKey:           "initial-secret-key",
				LogLevel:               new("LOG_LEVEL_WARN"),
				DisableBasicMetrics:    false,
				DisableEnhancedMetrics: false,
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				SkipConnectionCheck: true,
				PushMetrics:         false,
			},
		})
		agentID := rdsExporter.RDSExporter.AgentID

		// Change ALL available fields at once
		changeRDSExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				RDSExporter: &agents.ChangeAgentParamsBodyRDSExporter{
					AWSAccessKey:           new("new-access-key"),
					AWSSecretKey:           new("new-secret-key"),
					LogLevel:               new("LOG_LEVEL_ERROR"),
					DisableBasicMetrics:    new(true),
					DisableEnhancedMetrics: new(true),
					EnablePushMetrics:      new(true),
					CustomLabels: &agents.ChangeAgentParamsBodyRDSExporterCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "platform",
						},
					},
					Enable: new(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyRDSExporter{
			NodeID:                  nodeID,
			AgentID:                 agentID,
			PMMAgentID:              pmmAgentID,
			AWSAccessKey:            "new-access-key",
			LogLevel:                new("LOG_LEVEL_ERROR"),
			BasicMetricsDisabled:    true,
			EnhancedMetricsDisabled: true,
			PushMetricsEnabled:      true,
			Disabled:                true, // agent was disabled
			Status:                  &AgentStatusDone,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "platform",
			},
		}

		assert.Equal(t, expectedAgent, changeRDSExporterOK.Payload.RDSExporter)

		// Also verify by getting the agent independently
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyRDSExporter{
			NodeID:                  nodeID,
			AgentID:                 agentID,
			PMMAgentID:              pmmAgentID,
			AWSAccessKey:            "new-access-key",
			LogLevel:                new("LOG_LEVEL_ERROR"),
			BasicMetricsDisabled:    true,
			EnhancedMetricsDisabled: true,
			PushMetricsEnabled:      true,
			Disabled:                true,
			Status:                  &AgentStatusDone,
			CustomLabels: map[string]string{
				"environment": "production",
				"version":     "2.0",
				"team":        "platform",
			},
		}

		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.RDSExporter)
	})
}
