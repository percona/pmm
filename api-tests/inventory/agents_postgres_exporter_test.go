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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		PostgresExporter := addPostgresExporter(t, agents.AddPostgresExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"custom_label_postgres_exporter": "postgres_exporter",
			},

			SkipConnectionCheck:    true,
			MaxExporterConnections: 10,
		})
		agentID := PostgresExporter.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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
					Status:                 &AgentStatusUnknown,
					MaxExporterConnections: 10,
				},
			},
		}, getAgentRes)

		// Test change API.
		changePostgresExporterOK, err := client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
			Body: agents.ChangePostgresExporterBody{
				AgentID: agentID,
				Common: &agents.ChangePostgresExporterParamsBodyCommon{
					Disable:            true,
					RemoveCustomLabels: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangePostgresExporterOK{
			Payload: &agents.ChangePostgresExporterOKBody{
				PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
					AgentID:                agentID,
					ServiceID:              serviceID,
					Username:               "username",
					PMMAgentID:             pmmAgentID,
					Disabled:               true,
					Status:                 &AgentStatusUnknown,
					MaxExporterConnections: 10,
				},
			},
		}, changePostgresExporterOK)

		changePostgresExporterOK, err = client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
			Body: agents.ChangePostgresExporterBody{
				AgentID: agentID,
				Common: &agents.ChangePostgresExporterParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "postgres_exporter",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangePostgresExporterOK{
			Payload: &agents.ChangePostgresExporterOKBody{
				PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "postgres_exporter",
					},
					Status:                 &AgentStatusUnknown,
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

		res, err := client.Default.Agents.AddPostgresExporter(&agents.AddPostgresExporterParams{
			Body: agents.AddPostgresExporterBody{
				ServiceID:  "",
				PMMAgentID: pmmAgentID,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgresExporterRequest.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveNodes(t, res.Payload.PostgresExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for agent"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddPostgresExporter(&agents.AddPostgresExporterParams{
			Body: agents.AddPostgresExporterBody{
				ServiceID:  serviceID,
				PMMAgentID: "",
				Username:   "username",
				Password:   "password",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgresExporterRequest.PmmAgentId: value length must be at least 1 runes")
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

		res, err := client.Default.Agents.AddPostgresExporter(&agents.AddPostgresExporterParams{
			Body: agents.AddPostgresExporterBody{
				ServiceID:  "pmm-service-id",
				PMMAgentID: pmmAgentID,
				Username:   "username",
				Password:   "password",
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for not exists node ID"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.Agents.AddPostgresExporter(&agents.AddPostgresExporterParams{
			Body: agents.AddPostgresExporterBody{
				ServiceID:  serviceID,
				PMMAgentID: "pmm-not-exist-server",
				Username:   "username",
				Password:   "password",
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

		service := addPostgreSQLService(t, services.AddPostgreSQLServiceBody{
			NodeID:      genericNodeID,
			Address:     "localhost",
			Port:        5432,
			ServiceName: pmmapitests.TestString(t, "PostgreSQL Service for PostgresExporter test"),
		})
		serviceID := service.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		PostgresExporter := addPostgresExporter(t, agents.AddPostgresExporterBody{
			ServiceID:  serviceID,
			Username:   "username",
			Password:   "password",
			PMMAgentID: pmmAgentID,
			CustomLabels: map[string]string{
				"custom_label_postgres_exporter": "postgres_exporter",
			},

			SkipConnectionCheck: true,
			PushMetrics:         true,
		})
		agentID := PostgresExporter.PostgresExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.Agents.GetAgent(&agents.GetAgentParams{
			Body:    agents.GetAgentBody{AgentID: agentID},
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
				},
			},
		}, getAgentRes)

		// Test change API.
		changePostgresExporterOK, err := client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
			Body: agents.ChangePostgresExporterBody{
				AgentID: agentID,
				Common: &agents.ChangePostgresExporterParamsBodyCommon{
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangePostgresExporterOK{
			Payload: &agents.ChangePostgresExporterOKBody{
				PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					Status: &AgentStatusUnknown,
				},
			},
		}, changePostgresExporterOK)

		changePostgresExporterOK, err = client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
			Body: agents.ChangePostgresExporterBody{
				AgentID: agentID,
				Common: &agents.ChangePostgresExporterParamsBodyCommon{
					EnablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.Equal(t, &agents.ChangePostgresExporterOK{
			Payload: &agents.ChangePostgresExporterOKBody{
				PostgresExporter: &agents.ChangePostgresExporterOKBodyPostgresExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"custom_label_postgres_exporter": "postgres_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
				},
			},
		}, changePostgresExporterOK)

		_, err = client.Default.Agents.ChangePostgresExporter(&agents.ChangePostgresExporterParams{
			Body: agents.ChangePostgresExporterBody{
				AgentID: agentID,
				Common: &agents.ChangePostgresExporterParamsBodyCommon{
					Enable: true,
					CustomLabels: map[string]string{
						"new_label": "postgres_exporter",
					},
					EnablePushMetrics:  true,
					DisablePushMetrics: true,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "expected one of  param: enable_push_metrics or disable_push_metrics")
	})
}
