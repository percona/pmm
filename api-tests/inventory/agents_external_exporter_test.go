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
	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

func TestExternalExporter(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addExternalService(t, services.AddExternalServiceBody{
			NodeID:      genericNodeID,
			ServiceName: pmmapitests.TestString(t, "External Service for External Exporter test"),
			Group:       "external",
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		ExternalExporter := addExternalExporter(t, agents.AddExternalExporterBody{
			RunsOnNodeID: genericNodeID,
			ServiceID:    serviceID,
			ListenPort:   12345,
			CustomLabels: map[string]string{
				"custom_label_for_external_exporter": "external_exporter",
			},
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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

		service := addExternalService(t, services.AddExternalServiceBody{
			NodeID:      nodeID,
			ServiceName: pmmapitests.TestString(t, "External Service for External Exporter test"),
			Group:       "external",
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		ExternalExporter := addExternalExporter(t, agents.AddExternalExporterBody{
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
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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
			},
		}, getAgentRes.Payload)

		// Test change API.
		changeExternalExporterOK, err := client.Default.Agents.ChangeExternalExporter(&agents.ChangeExternalExporterParams{
			Body: agents.ChangeExternalExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeExternalExporterParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeExternalExporterOKBody{
			ExternalExporter: &agents.ChangeExternalExporterOKBodyExternalExporter{
				AgentID:      agentID,
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
				Username:     "username",
				Scheme:       "https",
				MetricsPath:  "/metrics-hr",
				ListenPort:   12345,
				Disabled:     true,
			},
		}, changeExternalExporterOK.Payload)

		changeExternalExporterOK, err = client.Default.Agents.ChangeExternalExporter(&agents.ChangeExternalExporterParams{
			Body: agents.ChangeExternalExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeExternalExporterParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "external_exporter",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeExternalExporterOKBody{
			ExternalExporter: &agents.ChangeExternalExporterOKBodyExternalExporter{
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
			},
		}, changeExternalExporterOK.Payload)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		res, err := client.Default.Agents.AddExternalExporter(&agents.AddExternalExporterParams{
			Body: agents.AddExternalExporterBody{
				ServiceID:    "",
				RunsOnNodeID: genericNodeID,
				ListenPort:   12345,
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

		service := addExternalService(t, services.AddExternalServiceBody{
			NodeID:      genericNodeID,
			ServiceName: pmmapitests.TestString(t, "External Service for agent"),
			Group:       "external",
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddExternalExporter(&agents.AddExternalExporterParams{
			Body: agents.AddExternalExporterBody{
				ServiceID:    serviceID,
				RunsOnNodeID: genericNodeID,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field ListenPort: value '0' must be greater than '0'")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("AddRunsOnNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addExternalService(t, services.AddExternalServiceBody{
			NodeID:      genericNodeID,
			ServiceName: pmmapitests.TestString(t, "External Service for agent"),
			Group:       "external",
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddExternalExporter(&agents.AddExternalExporterParams{
			Body: agents.AddExternalExporterBody{
				ServiceID:    serviceID,
				RunsOnNodeID: "",
				ListenPort:   12345,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field RunsOnNodeId: value '' must not be an empty string")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ExternalExporter.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		res, err := client.Default.Agents.AddExternalExporter(&agents.AddExternalExporterParams{
			Body: agents.AddExternalExporterBody{
				ServiceID:    "pmm-service-id",
				RunsOnNodeID: genericNodeID,
				ListenPort:   12345,
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

		service := addExternalService(t, services.AddExternalServiceBody{
			NodeID:      genericNodeID,
			ServiceName: pmmapitests.TestString(t, "External Service for not exists node ID"),
			Group:       "external",
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddExternalExporter(&agents.AddExternalExporterParams{
			Body: agents.AddExternalExporterBody{
				ServiceID:    serviceID,
				RunsOnNodeID: "pmm-not-exist-server",
				ListenPort:   12345,
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

		service := addExternalService(t, services.AddExternalServiceBody{
			NodeID:      genericNodeID,
			ServiceName: pmmapitests.TestString(t, "External Service for External Exporter test"),
		})
		serviceID := service.External.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		ExternalExporter := addExternalExporter(t, agents.AddExternalExporterBody{
			RunsOnNodeID: genericNodeID,
			ServiceID:    serviceID,
			ListenPort:   12345,
			CustomLabels: map[string]string{
				"custom_label_for_external_exporter": "external_exporter",
			},
			PushMetrics: true,
		})
		agentID := ExternalExporter.ExternalExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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
		changeExternalExporterOK, err := client.Default.Agents.ChangeExternalExporter(&agents.ChangeExternalExporterParams{
			Body: agents.ChangeExternalExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeExternalExporterParamsBodyCommon{
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeExternalExporterOKBody{
			ExternalExporter: &agents.ChangeExternalExporterOKBodyExternalExporter{
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

		changeExternalExporterOK, err = client.Default.Agents.ChangeExternalExporter(&agents.ChangeExternalExporterParams{
			Body: agents.ChangeExternalExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeExternalExporterParamsBodyCommon{
					EnablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeExternalExporterOKBody{
			ExternalExporter: &agents.ChangeExternalExporterOKBodyExternalExporter{
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

		_, err = client.Default.Agents.ChangeExternalExporter(&agents.ChangeExternalExporterParams{
			Body: agents.ChangeExternalExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeExternalExporterParamsBodyCommon{
					EnablePushMetrics:  true,
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected one of  param: enable_push_metrics or disable_push_metrics")
	})
}
