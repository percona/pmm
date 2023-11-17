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
	"github.com/percona/pmm/api/inventorypb/v1/json/client"
	agents "github.com/percona/pmm/api/inventorypb/v1/json/client/agents_service"
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

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				NodeExporter: &agents.GetAgentOKBodyNodeExporter{
					AgentID:      agentID,
					PMMAgentID:   pmmAgentID,
					Disabled:     false,
					CustomLabels: customLabels,
					Status:       &AgentStatusUnknown,
				},
			},
		}, getAgentRes)

		// Test change API.
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeNodeExporter(&agents.ChangeNodeExporterParams{
			Body: agents.ChangeNodeExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeNodeExporterParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeNodeExporterOK{
			Payload: &agents.ChangeNodeExporterOKBody{
				NodeExporter: &agents.ChangeNodeExporterOKBodyNodeExporter{
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					Disabled:   true,
					Status:     &AgentStatusUnknown,
				},
			},
		}, changeNodeExporterOK)

		changeNodeExporterOK, err = client.Default.AgentsService.ChangeNodeExporter(&agents.ChangeNodeExporterParams{
			Body: agents.ChangeNodeExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeNodeExporterParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "node_exporter",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeNodeExporterOK{
			Payload: &agents.ChangeNodeExporterOKBody{
				NodeExporter: &agents.ChangeNodeExporterOKBodyNodeExporter{
					AgentID:    agentID,
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "node_exporter",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, changeNodeExporterOK)
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.AgentsService.AddNodeExporter(&agents.AddNodeExporterParams{
			Body:    agents.AddNodeExporterBody{PMMAgentID: ""},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddNodeExporterRequest.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.NodeExporter.AgentID)
		}
	})

	t.Run("NotExistPmmAgentID", func(t *testing.T) {
		t.Parallel()

		res, err := client.Default.AgentsService.AddNodeExporter(&agents.AddNodeExporterParams{
			Body:    agents.AddNodeExporterBody{PMMAgentID: "pmm-node-exporter-node"},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-node-exporter-node\" not found.")
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
		res, err := client.Default.AgentsService.AddNodeExporter(&agents.AddNodeExporterParams{
			Body: agents.AddNodeExporterBody{
				PMMAgentID:   pmmAgentID,
				CustomLabels: customLabels,
				PushMetrics:  true,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Payload.NodeExporter)
		require.Equal(t, pmmAgentID, res.Payload.NodeExporter.PMMAgentID)
		agentID := res.Payload.NodeExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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
				},
			},
		}, getAgentRes)

		// Test change API.
		changeNodeExporterOK, err := client.Default.AgentsService.ChangeNodeExporter(&agents.ChangeNodeExporterParams{
			Body: agents.ChangeNodeExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeNodeExporterParamsBodyCommon{
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeNodeExporterOK{
			Payload: &agents.ChangeNodeExporterOKBody{
				NodeExporter: &agents.ChangeNodeExporterOKBodyNodeExporter{
					AgentID:      agentID,
					PMMAgentID:   pmmAgentID,
					Disabled:     false,
					CustomLabels: customLabels,
					Status:       &AgentStatusUnknown,
				},
			},
		}, changeNodeExporterOK)

		changeNodeExporterOK, err = client.Default.AgentsService.ChangeNodeExporter(&agents.ChangeNodeExporterParams{
			Body: agents.ChangeNodeExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeNodeExporterParamsBodyCommon{
					EnablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeNodeExporterOK{
			Payload: &agents.ChangeNodeExporterOKBody{
				NodeExporter: &agents.ChangeNodeExporterOKBodyNodeExporter{
					AgentID:            agentID,
					PMMAgentID:         pmmAgentID,
					Disabled:           false,
					CustomLabels:       customLabels,
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
				},
			},
		}, changeNodeExporterOK)
		_, err = client.Default.AgentsService.ChangeNodeExporter(&agents.ChangeNodeExporterParams{
			Body: agents.ChangeNodeExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeNodeExporterParamsBodyCommon{
					EnablePushMetrics:  true,
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected one of  param: enable_push_metrics or disable_push_metrics")
	})
}
