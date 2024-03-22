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

func TestPostgresExporter(t *testing.T) {
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
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		PostgresExporter := addAgent(t, agents.AddAgentBody{
			PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				SkipConnectionCheck:    true,
				MaxExporterConnections: 10,
			},
		})
		agentID := PostgresExporter.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				PostgresExporter: &agents.GetAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					DisabledCollectors:     make([]string, 0),
					LogLevel:               pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					Status:                 &AgentStatusUnknown,
					MaxExporterConnections: 10,
				},
			},
		}, getAgentRes)

		// Test change API.
		changePostgresExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyPostgresExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:                agentID,
					ServiceID:              serviceID,
					Username:               "username",
					PMMAgentID:             pmmAgentID,
					Disabled:               true,
					Status:                 &AgentStatusUnknown,
					CustomLabels:           map[string]string{},
					DisabledCollectors:     make([]string, 0),
					LogLevel:               pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxExporterConnections: 10,
				},
			},
		}, changePostgresExporterOK)

		changePostgresExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyPostgresExporterCustomLabels{
							Values: map[string]string{
								"new_label": "postgres_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "postgres_exporter",
					},
					Status:                 &AgentStatusUnknown,
					DisabledCollectors:     make([]string, 0),
					LogLevel:               pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					MaxExporterConnections: 10,
				},
			},
		}, changePostgresExporterOK)
	})

	t.Run("AddServiceIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		pmmAgent := pmmapitests.AddPMMAgent(t, genericNodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:  "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgresExporterParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.PostgresExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:  serviceID,
					PMMAgentID: "",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgresExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.PostgresExporter.AgentID)
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

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:  "pmm-service-id",
					PMMAgentID: pmmAgentID,
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-service-id\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.PostgresExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
					ServiceID:  serviceID,
					PMMAgentID: "pmm-not-exist-server",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID \"pmm-not-exist-server\" not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.PostgresExporter.AgentID)
		}
	})

	t.Run("With PushMetrics", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for Node exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
			},
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		PostgresExporter := addAgent(t, agents.AddAgentBody{
			PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"custom_label_postgres_exporter": "postgres_exporter",
				},
				SkipConnectionCheck: true,
				PushMetrics:         true,
			},
		})
		agentID := PostgresExporter.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				PostgresExporter: &agents.GetAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changePostgresExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						EnablePushMetrics: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changePostgresExporterOK)

		changePostgresExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					PostgresExporter: &agents.ChangeAgentParamsBodyPostgresExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				PostgresExporter: &agents.ChangeAgentOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changePostgresExporterOK)
	})
}
