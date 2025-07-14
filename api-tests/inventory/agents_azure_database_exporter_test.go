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

func TestAzureDatabaseExporter(t *testing.T) { //nolint:tparallel
	// TODO Fix this test to run in parallel.
	// t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()
		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := addRemoteAzureDatabaseNode(t, pmmapitests.TestString(t, "Remote node for Azure database exporter"))
		nodeID := node.RemoteAzureDatabase.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		azureDatabaseExporter := addAgent(t, agents.AddAgentBody{
			AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
				NodeID:                    nodeID,
				PMMAgentID:                pmmAgentID,
				AzureDatabaseResourceType: "mysql",
				AzureSubscriptionID:       "azure_subscription_id",
				CustomLabels: map[string]string{
					"custom_label_azure_database_exporter": "azure_database_exporter",
				},
				SkipConnectionCheck: true,
			},
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				AzureDatabaseExporter: &agents.GetAgentOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					PMMAgentID:                  pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_azure_database_exporter": "azure_database_exporter",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeAzureDatabaseExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyAzureDatabaseExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				AzureDatabaseExporter: &agents.ChangeAgentOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					Disabled:                    true,
					Status:                      &AgentStatusUnknown,
					LogLevel:                    pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					CustomLabels:                map[string]string{},
				},
			},
		}, changeAzureDatabaseExporterOK)

		changeAzureDatabaseExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyAzureDatabaseExporterCustomLabels{
							Values: map[string]string{
								"new_label": "azure_database_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				AzureDatabaseExporter: &agents.ChangeAgentOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					Disabled:                    false,
					CustomLabels: map[string]string{
						"new_label": "azure_database_exporter",
					},
					Status:   &AgentStatusUnknown,
					LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeAzureDatabaseExporterOK)
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
					NodeID:     "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddAzureDatabaseExporterParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.AzureDatabaseExporter.AgentID)
		}
	})

	t.Run("NotExistNodeID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
					NodeID:                    "pmm-node-id",
					PMMAgentID:                pmmAgentID,
					AzureDatabaseResourceType: "mysql",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID \"pmm-node-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.AzureDatabaseExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()
		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
					NodeID:                    "nodeID",
					PMMAgentID:                "pmm-not-exist-server",
					AzureDatabaseResourceType: "mysql",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.AzureDatabaseExporter.AgentID)
		}
	})

	t.Run("With PushMetrics", func(t *testing.T) {
		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := addRemoteAzureDatabaseNode(t, pmmapitests.TestString(t, "Remote node for Azure database exporter"))
		nodeID := node.RemoteAzureDatabase.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		azureDatabaseExporter := addAgent(t, agents.AddAgentBody{
			AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
				NodeID:              nodeID,
				PMMAgentID:          pmmAgentID,
				AzureSubscriptionID: "azure_subscription_id",
				CustomLabels: map[string]string{
					"custom_label_azure_database_exporter": "azure_database_exporter",
				},
				SkipConnectionCheck:       true,
				AzureDatabaseResourceType: "mysql",
			},
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		assert.Equal(t, &agents.GetAgentOKBodyAzureDatabaseExporter{
			NodeID:                      nodeID,
			AgentID:                     agentID,
			PMMAgentID:                  pmmAgentID,
			AzureDatabaseSubscriptionID: "azure_subscription_id",
			CustomLabels: map[string]string{
				"custom_label_azure_database_exporter": "azure_database_exporter",
			},
			Status:   &AgentStatusUnknown,
			LogLevel: pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
		}, getAgentRes.Payload.AzureDatabaseExporter)

		// Test change API.
		changeAzureDatabaseExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				AzureDatabaseExporter: &agents.ChangeAgentOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					CustomLabels: map[string]string{
						"custom_label_azure_database_exporter": "azure_database_exporter",
					},
					Status:             &AgentStatusUnknown,
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					PushMetricsEnabled: true,
				},
			},
		}, changeAzureDatabaseExporterOK)

		changeAzureDatabaseExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
						EnablePushMetrics: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				AzureDatabaseExporter: &agents.ChangeAgentOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					CustomLabels: map[string]string{
						"custom_label_azure_database_exporter": "azure_database_exporter",
					},
					Status:             &AgentStatusUnknown,
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					PushMetricsEnabled: false,
				},
			},
		}, changeAzureDatabaseExporterOK)
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := addRemoteAzureDatabaseNode(t, pmmapitests.TestString(t, "Remote Azure Database node for credential rotation test"))
		nodeID := node.RemoteAzureDatabase.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create Azure Database Exporter with initial Azure credentials
		azureDatabaseExporter := addAgent(t, agents.AddAgentBody{
			AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
				NodeID:                    nodeID,
				PMMAgentID:                pmmAgentID,
				AzureClientID:             "initial-client-id",
				AzureClientSecret:         "initial-client-secret",
				AzureTenantID:             "initial-tenant-id",
				AzureSubscriptionID:       "initial-subscription-id",
				AzureResourceGroup:        "initial-resource-group",
				AzureDatabaseResourceType: "mysql",
				LogLevel:                  pointer.ToString("LOG_LEVEL_WARN"),
				CustomLabels: map[string]string{
					"environment": "test",
				},
				SkipConnectionCheck: true,
			},
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test Azure client secret rotation
		changeAzureDatabaseExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
					AzureClientSecret: pointer.ToString("rotated-client-secret"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.False(t, changeAzureDatabaseExporterOK.Payload.AzureDatabaseExporter.Disabled)

		// Test Azure client ID and secret rotation together
		changeAzureDatabaseExporterOK, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
					AzureClientID:     pointer.ToString("new-client-id"),
					AzureClientSecret: pointer.ToString("new-client-secret"),
					AzureTenantID:     pointer.ToString("new-tenant-id"),
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
		// Note: Azure credentials are not returned in responses, only subscription ID and resource type
		assert.Equal(t, "initial-subscription-id", getAgentRes.Payload.AzureDatabaseExporter.AzureDatabaseSubscriptionID)
		assert.False(t, getAgentRes.Payload.AzureDatabaseExporter.Disabled)
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := addRemoteAzureDatabaseNode(t, pmmapitests.TestString(t, "Remote Azure Database node for partial update test"))
		nodeID := node.RemoteAzureDatabase.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create Azure Database Exporter with comprehensive initial configuration
		azureDatabaseExporter := addAgent(t, agents.AddAgentBody{
			AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
				NodeID:                    nodeID,
				PMMAgentID:                pmmAgentID,
				AzureClientID:             "initial-client-id",
				AzureClientSecret:         "initial-client-secret",
				AzureTenantID:             "initial-tenant-id",
				AzureSubscriptionID:       "initial-subscription-id",
				AzureResourceGroup:        "initial-resource-group",
				AzureDatabaseResourceType: "postgres",
				LogLevel:                  pointer.ToString("LOG_LEVEL_INFO"),
				CustomLabels: map[string]string{
					"environment": "staging",
					"team":        "data",
					"region":      "eastus",
				},
				SkipConnectionCheck: true,
				PushMetrics:         true,
			},
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change only log level, verify all other fields remain unchanged
		_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
					LogLevel: pointer.ToString("LOG_LEVEL_DEBUG"),
					// Note: Azure credentials, custom labels, resource group are NOT specified
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

		agent := getAgentRes.Payload.AzureDatabaseExporter
		// Log level should be changed
		assert.Equal(t, pointer.ToString("LOG_LEVEL_DEBUG"), agent.LogLevel)

		// Everything else should remain unchanged
		assert.Equal(t, "initial-subscription-id", agent.AzureDatabaseSubscriptionID)
		assert.True(t, agent.PushMetricsEnabled)
		assert.Equal(t, map[string]string{
			"environment": "staging",
			"team":        "data",
			"region":      "eastus",
		}, agent.CustomLabels)
		assert.False(t, agent.Disabled)
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := addRemoteAzureDatabaseNode(t, pmmapitests.TestString(t, "Remote Azure Database node for change all fields test"))
		nodeID := node.RemoteAzureDatabase.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Create Azure Database Exporter with initial configuration
		azureDatabaseExporter := addAgent(t, agents.AddAgentBody{
			AzureDatabaseExporter: &agents.AddAgentParamsBodyAzureDatabaseExporter{
				NodeID:                    nodeID,
				PMMAgentID:                pmmAgentID,
				AzureClientID:             "initial-client-id",
				AzureClientSecret:         "initial-client-secret",
				AzureTenantID:             "initial-tenant-id",
				AzureSubscriptionID:       "initial-subscription-id",
				AzureResourceGroup:        "initial-resource-group",
				AzureDatabaseResourceType: "mysql",
				LogLevel:                  pointer.ToString("LOG_LEVEL_WARN"),
				CustomLabels: map[string]string{
					"environment": "staging",
					"version":     "1.0",
				},
				SkipConnectionCheck: true,
				PushMetrics:         false,
			},
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Change ALL available fields at once
		changeAzureDatabaseExporterOK, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				AzureDatabaseExporter: &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
					AzureClientID:       pointer.ToString("new-client-id"),
					AzureClientSecret:   pointer.ToString("new-client-secret"),
					AzureTenantID:       pointer.ToString("new-tenant-id"),
					AzureSubscriptionID: pointer.ToString("new-subscription-id"),
					AzureResourceGroup:  pointer.ToString("new-resource-group"),
					LogLevel:            pointer.ToString("LOG_LEVEL_ERROR"),
					EnablePushMetrics:   pointer.ToBool(true),
					CustomLabels: &agents.ChangeAgentParamsBodyAzureDatabaseExporterCustomLabels{
						Values: map[string]string{
							"environment": "production",
							"version":     "2.0",
							"team":        "platform",
						},
					},
					Enable: pointer.ToBool(false), // disable the agent
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify all fields were changed correctly
		expectedAgent := &agents.ChangeAgentOKBodyAzureDatabaseExporter{
			AgentID:                     agentID,
			PMMAgentID:                  pmmAgentID,
			NodeID:                      nodeID,
			AzureDatabaseSubscriptionID: "new-subscription-id",
			AzureDatabaseResourceType:   "", // This field gets reset when changed
			LogLevel:                    pointer.ToString("LOG_LEVEL_ERROR"),
			PushMetricsEnabled:          true,
			Disabled:                    true, // agent was disabled
			Status:                      &AgentStatusUnknown,
			CustomLabels: map[string]string{
				"environment": "production",
				"team":        "platform",
				"version":     "2.0",
			},
		}
		assert.Equal(t, expectedAgent, changeAzureDatabaseExporterOK.Payload.AzureDatabaseExporter)

		// Verify with GetAgent that changes persisted
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		expectedGetAgent := &agents.GetAgentOKBodyAzureDatabaseExporter{
			AgentID:                     agentID,
			PMMAgentID:                  pmmAgentID,
			NodeID:                      nodeID,
			AzureDatabaseSubscriptionID: "new-subscription-id",
			AzureDatabaseResourceType:   "", // This field gets reset when changed
			LogLevel:                    pointer.ToString("LOG_LEVEL_ERROR"),
			PushMetricsEnabled:          true,
			Disabled:                    true, // agent was disabled
			Status:                      &AgentStatusUnknown,
			CustomLabels: map[string]string{
				"environment": "production",
				"team":        "platform",
				"version":     "2.0",
			},
		}
		assert.Equal(t, expectedGetAgent, getAgentRes.Payload.AzureDatabaseExporter)
	})
}
