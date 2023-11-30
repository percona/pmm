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

func TestProxySQLExporter(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for ProxySQLExporter test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		ProxySQLExporter := addProxySQLExporter(t, agents.AddProxySQLExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"custom_label_proxysql_exporter": "proxysql_exporter",
			},

			SkipConnectionCheck: true,
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				ProxysqlExporter: &agents.GetAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeProxySQLExporterOK, err := client.Default.AgentsService.ChangeProxySQLExporter(&agents.ChangeProxySQLExporterParams{
			Body: agents.ChangeProxySQLExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeProxySQLExporterParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeProxySQLExporterOK{
			Payload: &agents.ChangeProxySQLExporterOKBody{
				ProxysqlExporter: &agents.ChangeProxySQLExporterOKBodyProxysqlExporter{
					AgentID:            agentID,
					ServiceID:          serviceID,
					Username:           "username",
					PMMAgentID:         pmmAgentID,
					Disabled:           true,
					Status:             &AgentStatusUnknown,
					CustomLabels:       map[string]string{},
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)

		changeProxySQLExporterOK, err = client.Default.AgentsService.ChangeProxySQLExporter(&agents.ChangeProxySQLExporterParams{
			Body: agents.ChangeProxySQLExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeProxySQLExporterParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "proxysql_exporter",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeProxySQLExporterOK{
			Payload: &agents.ChangeProxySQLExporterOKBody{
				ProxysqlExporter: &agents.ChangeProxySQLExporterOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddProxySQLExporter(&agents.AddProxySQLExporterParams{
			Body: agents.AddProxySQLExporterBody{
				ServiceID:  "",
				PMMAgentID: pmmAgentID,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddProxySQLExporterRequest.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for agent"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddProxySQLExporter(&agents.AddProxySQLExporterParams{
			Body: agents.AddProxySQLExporterBody{
				ServiceID:  serviceID,
				PMMAgentID: "",
				Username:   "username",
				Password:   "password",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddProxySQLExporterRequest.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})

	t.Run("NotExistServiceID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddProxySQLExporter(&agents.AddProxySQLExporterParams{
			Body: agents.AddProxySQLExporterBody{
				ServiceID:  "pmm-service-id",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for not exists node ID"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddProxySQLExporter(&agents.AddProxySQLExporterParams{
			Body: agents.AddProxySQLExporterBody{
				ServiceID:  serviceID,
				PMMAgentID: "pmm-not-exist-server",
				Username:   "username",
				Password:   "password",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.ProxysqlExporter.AgentID)
		}
	})
	t.Run("With PushMetrics", func(t *testing.T) {
		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Proxysql: &services.AddServiceParamsBodyProxysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "ProxySQL Service for ProxySQLExporter test"),
			},
		})
		serviceID := service.Proxysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		ProxySQLExporter := addProxySQLExporter(t, agents.AddProxySQLExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"custom_label_proxysql_exporter": "proxysql_exporter",
			},

			SkipConnectionCheck: true,
		})
		agentID := ProxySQLExporter.ProxysqlExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				ProxysqlExporter: &agents.GetAgentOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeProxySQLExporterOK, err := client.Default.AgentsService.ChangeProxySQLExporter(&agents.ChangeProxySQLExporterParams{
			Body: agents.ChangeProxySQLExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeProxySQLExporterParamsBodyCommon{
					EnablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeProxySQLExporterOK{
			Payload: &agents.ChangeProxySQLExporterOKBody{
				ProxysqlExporter: &agents.ChangeProxySQLExporterOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)

		changeProxySQLExporterOK, err = client.Default.AgentsService.ChangeProxySQLExporter(&agents.ChangeProxySQLExporterParams{
			Body: agents.ChangeProxySQLExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeProxySQLExporterParamsBodyCommon{
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeProxySQLExporterOK{
			Payload: &agents.ChangeProxySQLExporterOKBody{
				ProxysqlExporter: &agents.ChangeProxySQLExporterOKBodyProxysqlExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_proxysql_exporter": "proxysql_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeProxySQLExporterOK)

		_, err = client.Default.AgentsService.ChangeProxySQLExporter(&agents.ChangeProxySQLExporterParams{
			Body: agents.ChangeProxySQLExporterBody{
				AgentID: agentID,
				Common: &agents.ChangeProxySQLExporterParamsBodyCommon{
					EnablePushMetrics:  true,
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected one of  param: enable_push_metrics or disable_push_metrics")
	})
}
