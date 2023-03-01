// Copyright (C) 2017 Percona LLC
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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

func TestAzureDatabaseExporter(t *testing.T) {
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

		azureDatabaseExporter := addAzureDatabaseExporter(t, agents.AddAzureDatabaseExporterBody{
			NodeID:                    nodeID,
			PMMAgentID:                pmmAgentID,
			AzureDatabaseResourceType: "mysql",
			AzureSubscriptionID:       "azure_subscription_id",
			CustomLabels: map[string]string{
				"custom_label_azure_database_exporter": "azure_database_exporter",
			},
			SkipConnectionCheck: true,
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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
					Status: &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeAzureDatabaseExporterOK, err := client.Default.Agents.ChangeAzureDatabaseExporter(&agents.ChangeAzureDatabaseExporterParams{
			Body: agents.ChangeAzureDatabaseExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeAzureDatabaseExporterParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAzureDatabaseExporterOK{
			Payload: &agents.ChangeAzureDatabaseExporterOKBody{
				AzureDatabaseExporter: &agents.ChangeAzureDatabaseExporterOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					Disabled:                    true,
					Status:                      &AgentStatusUnknown,
				},
			},
		}, changeAzureDatabaseExporterOK)

		changeAzureDatabaseExporterOK, err = client.Default.Agents.ChangeAzureDatabaseExporter(&agents.ChangeAzureDatabaseExporterParams{
			Body: agents.ChangeAzureDatabaseExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeAzureDatabaseExporterParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "azure_database_exporter",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAzureDatabaseExporterOK{
			Payload: &agents.ChangeAzureDatabaseExporterOKBody{
				AzureDatabaseExporter: &agents.ChangeAzureDatabaseExporterOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					Disabled:                    false,
					CustomLabels: map[string]string{
						"new_label": "azure_database_exporter",
					},
					Status: &AgentStatusUnknown,
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

		res, err := client.Default.Agents.AddAzureDatabaseExporter(&agents.AddAzureDatabaseExporterParams{
			Body: agents.AddAzureDatabaseExporterBody{
				NodeID:     "",
				PMMAgentID: pmmAgentID,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field NodeId: value '' must not be an empty string")
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

		res, err := client.Default.Agents.AddAzureDatabaseExporter(&agents.AddAzureDatabaseExporterParams{
			Body: agents.AddAzureDatabaseExporterBody{
				NodeID:                    "pmm-node-id",
				PMMAgentID:                pmmAgentID,
				AzureDatabaseResourceType: "mysql",
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

		res, err := client.Default.Agents.AddAzureDatabaseExporter(&agents.AddAzureDatabaseExporterParams{
			Body: agents.AddAzureDatabaseExporterBody{
				NodeID:                    "nodeID",
				PMMAgentID:                "pmm-not-exist-server",
				AzureDatabaseResourceType: "mysql",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
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

		azureDatabaseExporter := addAzureDatabaseExporter(t, agents.AddAzureDatabaseExporterBody{
			NodeID:              nodeID,
			PMMAgentID:          pmmAgentID,
			AzureSubscriptionID: "azure_subscription_id",
			CustomLabels: map[string]string{
				"custom_label_azure_database_exporter": "azure_database_exporter",
			},
			SkipConnectionCheck:       true,
			AzureDatabaseResourceType: "mysql",
		})
		agentID := azureDatabaseExporter.AzureDatabaseExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				AzureDatabaseExporter: &agents.GetAgentOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					CustomLabels: map[string]string{
						"custom_label_azure_database_exporter": "azure_database_exporter",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeAzureDatabaseExporterOK, err := client.Default.Agents.ChangeAzureDatabaseExporter(&agents.ChangeAzureDatabaseExporterParams{
			Body: agents.ChangeAzureDatabaseExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeAzureDatabaseExporterParamsBodyCommon{
					EnablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAzureDatabaseExporterOK{
			Payload: &agents.ChangeAzureDatabaseExporterOKBody{
				AzureDatabaseExporter: &agents.ChangeAzureDatabaseExporterOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					CustomLabels: map[string]string{
						"custom_label_azure_database_exporter": "azure_database_exporter",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, changeAzureDatabaseExporterOK)

		changeAzureDatabaseExporterOK, err = client.Default.Agents.ChangeAzureDatabaseExporter(&agents.ChangeAzureDatabaseExporterParams{
			Body: agents.ChangeAzureDatabaseExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeAzureDatabaseExporterParamsBodyCommon{
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAzureDatabaseExporterOK{
			Payload: &agents.ChangeAzureDatabaseExporterOKBody{
				AzureDatabaseExporter: &agents.ChangeAzureDatabaseExporterOKBodyAzureDatabaseExporter{
					NodeID:                      nodeID,
					AgentID:                     agentID,
					PMMAgentID:                  pmmAgentID,
					AzureDatabaseSubscriptionID: "azure_subscription_id",
					CustomLabels: map[string]string{
						"custom_label_azure_database_exporter": "azure_database_exporter",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, changeAzureDatabaseExporterOK)
		_, err = client.Default.Agents.ChangeAzureDatabaseExporter(&agents.ChangeAzureDatabaseExporterParams{
			Body: agents.ChangeAzureDatabaseExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeAzureDatabaseExporterParamsBodyCommon{
					EnablePushMetrics:  true,
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected one of  param: enable_push_metrics or disable_push_metrics")
	})
}
