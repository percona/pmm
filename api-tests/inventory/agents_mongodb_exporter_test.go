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

func TestMongoDBExporter(t *testing.T) {
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
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for MongoDBExporter test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		mongoDBExporter := addAgent(t, agents.AddAgentBody{
			MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"new_label": "mongodb_exporter",
				},

				SkipConnectionCheck: true,
			},
		})
		agentID := mongoDBExporter.MongodbExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				MongodbExporter: &agents.GetAgentOKBodyMongodbExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "mongodb_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					StatsCollections:   make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeMongoDBExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
						Enable:       pointer.ToBool(false),
						CustomLabels: &agents.ChangeAgentParamsBodyMongodbExporterCustomLabels{},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				MongodbExporter: &agents.ChangeAgentOKBodyMongodbExporter{
					AgentID:            agentID,
					ServiceID:          serviceID,
					Username:           "username",
					PMMAgentID:         pmmAgentID,
					Disabled:           true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					StatsCollections:   make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					CustomLabels:       map[string]string{},
				},
			},
		}, changeMongoDBExporterOK)

		changeMongoDBExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
						Enable: pointer.ToBool(true),
						CustomLabels: &agents.ChangeAgentParamsBodyMongodbExporterCustomLabels{
							Values: map[string]string{
								"new_label": "mongodb_exporter",
							},
						},
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				MongodbExporter: &agents.ChangeAgentOKBodyMongodbExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					Disabled:   false,
					CustomLabels: map[string]string{
						"new_label": "mongodb_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					StatsCollections:   make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeMongoDBExporterOK)
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
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					ServiceID:  "",
					PMMAgentID: pmmAgentID,
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMongoDBExporterParams.ServiceId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.MongodbExporter.AgentID)
		}
	})

	t.Run("AddPMMAgentIDEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for agent"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					ServiceID:  serviceID,
					PMMAgentID: "",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMongoDBExporterParams.PmmAgentId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.MongodbExporter.AgentID)
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
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
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
			pmmapitests.RemoveAgents(t, res.Payload.MongodbExporter.AgentID)
		}
	})

	t.Run("NotExistPMMAgentID", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		service := addService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for not exists node ID"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					ServiceID:  serviceID,
					PMMAgentID: "pmm-not-exist-server",
					Username:   "username",
					Password:   "password",
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Agent with ID pmm-not-exist-server not found.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveAgents(t, res.Payload.MongodbExporter.AgentID)
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
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for MongoDBExporter test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		mongoDBExporter := addAgent(t, agents.AddAgentBody{
			MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,
				CustomLabels: map[string]string{
					"new_label": "mongodb_exporter",
				},

				SkipConnectionCheck: true,
				PushMetrics:         true,
			},
		})
		agentID := mongoDBExporter.MongodbExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		getAgentRes, err := client.Default.AgentsService.GetAgent(
			&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.GetAgentOK{
			Payload: &agents.GetAgentOKBody{
				MongodbExporter: &agents.GetAgentOKBodyMongodbExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "mongodb_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
					StatsCollections:   make([]string, 0),
				},
			},
		}, getAgentRes)

		// Test change API.
		changeMongoDBExporterOK, err := client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
						EnablePushMetrics: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				MongodbExporter: &agents.ChangeAgentOKBodyMongodbExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "mongodb_exporter",
					},
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					StatsCollections:   make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeMongoDBExporterOK)

		changeMongoDBExporterOK, err = client.Default.AgentsService.ChangeAgent(
			&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
						EnablePushMetrics: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
		require.NoError(t, err)
		assert.Equal(t, &agents.ChangeAgentOK{
			Payload: &agents.ChangeAgentOKBody{
				MongodbExporter: &agents.ChangeAgentOKBodyMongodbExporter{
					AgentID:    agentID,
					ServiceID:  serviceID,
					Username:   "username",
					PMMAgentID: pmmAgentID,
					CustomLabels: map[string]string{
						"new_label": "mongodb_exporter",
					},
					PushMetricsEnabled: true,
					Status:             &AgentStatusUnknown,
					DisabledCollectors: make([]string, 0),
					StatsCollections:   make([]string, 0),
					LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
				},
			},
		}, changeMongoDBExporterOK)
	})

	t.Run("ChangeAllAvailableFields", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mongodb exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for MongoDBExporter test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Add agent with skip connection check
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					ServiceID:           serviceID,
					Username:            "username",
					Password:            "password",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MongodbExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing ALL available MongoDB exporter fields
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					// Core agent settings
					Enable:            pointer.ToBool(true),
					EnablePushMetrics: pointer.ToBool(true),
					Username:          pointer.ToString("new-mongodb-user"),

					// TLS configuration
					TLS:                           pointer.ToBool(true),
					TLSSkipVerify:                 pointer.ToBool(false),
					TLSCertificateKey:             pointer.ToString("test-cert-key"),
					TLSCertificateKeyFilePassword: pointer.ToString("test-password"),
					TLSCa:                         pointer.ToString("test-ca-cert"),

					// Authentication
					AuthenticationMechanism: pointer.ToString("MONGODB-X509"),
					AuthenticationDatabase:  pointer.ToString("$external"),

					// Collection and monitoring settings
					SkipConnectionCheck: pointer.ToBool(true),
					StatsCollections:    []string{"db1.coll1", "db2.coll2"},
					CollectionsLimit:    pointer.ToInt32(500),
					EnableAllCollectors: pointer.ToBool(true),
					DisableCollectors:   []string{"collstats", "indexstats"},

					// Agent configuration
					AgentPassword:  pointer.ToString("new-agent-password"),
					LogLevel:       pointer.ToString(agents.ChangeAgentParamsBodyMongodbExporterLogLevelLOGLEVELDEBUG),
					ExposeExporter: pointer.ToBool(true),

					// Metrics configuration
					MetricsResolutions: &agents.ChangeAgentParamsBodyMongodbExporterMetricsResolutions{
						Hr: "5s",
						Mr: "10s",
						Lr: "60s",
					},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify ALL the fields were applied correctly
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MongodbExporter)

		mongodbExporter := getAgentRes.Payload.MongodbExporter

		// Core agent settings
		assert.False(t, mongodbExporter.Disabled) // Enable: true means Disabled: false
		assert.Equal(t, "new-mongodb-user", mongodbExporter.Username)

		// TLS configuration
		assert.True(t, mongodbExporter.TLS)
		assert.False(t, mongodbExporter.TLSSkipVerify)

		// Collection and monitoring settings
		assert.ElementsMatch(t, []string{"collstats", "indexstats"}, mongodbExporter.DisabledCollectors)
		assert.ElementsMatch(t, []string{"db1.coll1", "db2.coll2"}, mongodbExporter.StatsCollections)
		assert.Equal(t, int32(500), mongodbExporter.CollectionsLimit)
		assert.True(t, mongodbExporter.EnableAllCollectors)

		// Agent configuration
		assert.Equal(t, pointer.ToString("LOG_LEVEL_DEBUG"), mongodbExporter.LogLevel)
		assert.True(t, mongodbExporter.ExposeExporter)
		assert.True(t, mongodbExporter.PushMetricsEnabled)

		// Metrics configuration
		assert.Equal(t, "5s", mongodbExporter.MetricsResolutions.Hr)
		assert.Equal(t, "10s", mongodbExporter.MetricsResolutions.Mr)
		assert.Equal(t, "60s", mongodbExporter.MetricsResolutions.Lr)

		// Note: TLS cert/key, agent_password, and authentication fields are not returned in GetAgent for security reasons
	})

	t.Run("ChangeEnableAllCollectors", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mongodb exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for EnableAllCollectors test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Add MongoDB exporter without EnableAllCollectors
		addAgentRes, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					PMMAgentID:          pmmAgentID,
					ServiceID:           serviceID,
					Username:            "test-user",
					Password:            "test-password",
					SkipConnectionCheck: true,
					EnableAllCollectors: false, // Start with disabled
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := addAgentRes.Payload.MongodbExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test enabling all collectors
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					EnableAllCollectors: pointer.ToBool(true),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify EnableAllCollectors is enabled
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MongodbExporter)

		mongodbExporter := getAgentRes.Payload.MongodbExporter
		assert.True(t, mongodbExporter.EnableAllCollectors, "EnableAllCollectors should be true")

		// Test disabling all collectors
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					EnableAllCollectors: pointer.ToBool(false),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify EnableAllCollectors is disabled
		getAgentRes2, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes2.Payload.MongodbExporter)

		mongodbExporter2 := getAgentRes2.Payload.MongodbExporter
		assert.False(t, mongodbExporter2.EnableAllCollectors, "EnableAllCollectors should be false")
	})

	t.Run("ChangePassword_PasswordRotation", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mongodb exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for password rotation test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Add agent with initial password
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					ServiceID:           serviceID,
					Username:            "mongodb-user",
					Password:            "initial-mongodb-password-123",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MongodbExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Test changing password (simulating password rotation)
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					Password: pointer.ToString("rotated-mongodb-password-456"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify agent still works after password change
		getAgentRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MongodbExporter)

		mongodbExporter := getAgentRes.Payload.MongodbExporter
		assert.Equal(t, "mongodb-user", mongodbExporter.Username) // Username unchanged
		assert.False(t, mongodbExporter.Disabled)                 // Agent still enabled

		// Note: Password is not returned in GetAgent response for security reasons
		// This test verifies that the password change operation completes successfully
		// without returning the actual password value

		// Test changing both username, password, and authentication settings together
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					Username:                pointer.ToString("new-mongodb-user"),
					Password:                pointer.ToString("final-mongodb-password-789"),
					AuthenticationMechanism: pointer.ToString("SCRAM-SHA-256"),
					AuthenticationDatabase:  pointer.ToString("admin"),
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify username and authentication changes completed
		getAgentRes, err = client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, getAgentRes.Payload.MongodbExporter)

		assert.Equal(t, "new-mongodb-user", getAgentRes.Payload.MongodbExporter.Username)
		assert.False(t, getAgentRes.Payload.MongodbExporter.Disabled)
		// Note: AuthenticationMechanism and AuthenticationDatabase are not returned for security reasons
	})

	t.Run("ChangeOnlySpecifiedFields_KeepOthersUnchanged", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for mongodb exporter"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mongodb: &services.AddServiceParamsBodyMongodb{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        27017,
				ServiceName: pmmapitests.TestString(t, "MongoDB Service for partial change test"),
			},
		})
		serviceID := service.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		// Add agent with specific initial values for multiple fields
		res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
			Body: agents.AddAgentBody{
				MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
					ServiceID:           serviceID,
					Username:            "initial-mongo-user",
					Password:            "initial-mongo-password",
					PMMAgentID:          pmmAgentID,
					SkipConnectionCheck: true,
					CollectionsLimit:    1000,
					EnableAllCollectors: true,
					StatsCollections:    []string{"db1.coll1", "db2.coll2"},
					CustomLabels: map[string]string{
						"env":        "staging",
						"database":   "mongodb",
						"monitoring": "enabled",
					},
					TLS:                           true,
					TLSSkipVerify:                 false,
					AuthenticationMechanism:       "SCRAM-SHA-1",
					AuthenticationDatabase:        "admin",
					PushMetrics:                   true,
					LogLevel:                      pointer.ToString("LOG_LEVEL_WARN"),
					ExposeExporter:                true,
					DisableCollectors:             []string{"collstats", "indexstats"},
					TLSCertificateKey:             "initial-cert-key",
					TLSCertificateKeyFilePassword: "initial-cert-password",
					TLSCa:                         "initial-ca-cert",
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		agentID := res.Payload.MongodbExporter.AgentID
		defer pmmapitests.RemoveAgents(t, agentID)

		// Get initial state to capture all original values
		initialAgent, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, initialAgent.Payload.MongodbExporter)

		initialExporter := initialAgent.Payload.MongodbExporter

		// Change ONLY the password - all other fields should remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					Password: pointer.ToString("new-password-only"),
					// All other fields are intentionally NOT set (nil)
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify that ONLY the password-related behavior changed, all other fields preserved
		updatedAgent, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedAgent.Payload.MongodbExporter)

		updatedExporter := updatedAgent.Payload.MongodbExporter

		// Verify all original fields are preserved (password can't be checked as it's not returned)
		assert.Equal(t, initialExporter.Username, updatedExporter.Username, "Username should remain unchanged")
		assert.Equal(t, initialExporter.CollectionsLimit, updatedExporter.CollectionsLimit, "CollectionsLimit should remain unchanged")
		assert.Equal(t, initialExporter.EnableAllCollectors, updatedExporter.EnableAllCollectors, "EnableAllCollectors should remain unchanged")
		assert.Equal(t, initialExporter.StatsCollections, updatedExporter.StatsCollections, "StatsCollections should remain unchanged")
		assert.Equal(t, initialExporter.CustomLabels, updatedExporter.CustomLabels, "CustomLabels should remain unchanged")
		assert.Equal(t, initialExporter.TLS, updatedExporter.TLS, "TLS should remain unchanged")
		assert.Equal(t, initialExporter.TLSSkipVerify, updatedExporter.TLSSkipVerify, "TLSSkipVerify should remain unchanged")
		assert.Equal(t, initialExporter.PushMetricsEnabled, updatedExporter.PushMetricsEnabled, "PushMetricsEnabled should remain unchanged")
		assert.Equal(t, initialExporter.LogLevel, updatedExporter.LogLevel, "LogLevel should remain unchanged")
		assert.Equal(t, initialExporter.ExposeExporter, updatedExporter.ExposeExporter, "ExposeExporter should remain unchanged")
		assert.Equal(t, initialExporter.DisabledCollectors, updatedExporter.DisabledCollectors, "DisabledCollectors should remain unchanged")
		assert.Equal(t, initialExporter.Disabled, updatedExporter.Disabled, "Disabled status should remain unchanged")

		// Now change ONLY the collections limit - all other fields should remain unchanged
		_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				MongodbExporter: &agents.ChangeAgentParamsBodyMongodbExporter{
					CollectionsLimit: pointer.ToInt32(2000),
					// All other fields are intentionally NOT set (nil)
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		// Verify that ONLY the collections limit changed
		finalAgent, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, finalAgent.Payload.MongodbExporter)

		finalExporter := finalAgent.Payload.MongodbExporter

		// Collections limit should be changed
		assert.Equal(t, int32(2000), finalExporter.CollectionsLimit, "CollectionsLimit should be changed")

		// All other fields should still match the initial values
		assert.Equal(t, initialExporter.Username, finalExporter.Username, "Username should remain unchanged")
		assert.Equal(t, initialExporter.EnableAllCollectors, finalExporter.EnableAllCollectors, "EnableAllCollectors should remain unchanged")
		assert.Equal(t, initialExporter.StatsCollections, finalExporter.StatsCollections, "StatsCollections should remain unchanged")
		assert.Equal(t, initialExporter.CustomLabels, finalExporter.CustomLabels, "CustomLabels should remain unchanged")
		assert.Equal(t, initialExporter.TLS, finalExporter.TLS, "TLS should remain unchanged")
		assert.Equal(t, initialExporter.TLSSkipVerify, finalExporter.TLSSkipVerify, "TLSSkipVerify should remain unchanged")
		assert.Equal(t, initialExporter.PushMetricsEnabled, finalExporter.PushMetricsEnabled, "PushMetricsEnabled should remain unchanged")
		assert.Equal(t, initialExporter.LogLevel, finalExporter.LogLevel, "LogLevel should remain unchanged")
		assert.Equal(t, initialExporter.ExposeExporter, finalExporter.ExposeExporter, "ExposeExporter should remain unchanged")
		assert.Equal(t, initialExporter.DisabledCollectors, finalExporter.DisabledCollectors, "DisabledCollectors should remain unchanged")
		assert.Equal(t, initialExporter.Disabled, finalExporter.Disabled, "Disabled status should remain unchanged")
	})
}
