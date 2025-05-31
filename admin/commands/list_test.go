// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

func TestListResultString(t *testing.T) {
	tests := []struct {
		name       string
		listResult listResult
		expected   string
	}{
		{
			name: "filled",
			listResult: listResult{
				Services: []listResultService{
					{ServiceType: types.ServiceTypeMySQLService, ServiceID: "4ff49c41-80a1-4030-bc02-cd76e3b0b84a", ServiceName: "mysql-service"},
				},
				Agents: []listResultAgent{
					{AgentType: types.AgentTypeMySQLdExporter, AgentID: "8b732ac3-8256-40b0-a98b-0fd5fa9a1140", ServiceID: "4ff49c41-80a1-4030-bc02-cd76e3b0b84a", Status: "RUNNING", MetricsMode: "pull", Port: 3306},
				},
			},
			expected: strings.TrimSpace(`
Service type        Service name         Address and port        Service ID
MySQL               mysql-service                                4ff49c41-80a1-4030-bc02-cd76e3b0b84a

Agent type             Status         Metrics Mode        Agent ID                                    Service ID                                  Port
mysqld_exporter        Running        pull                8b732ac3-8256-40b0-a98b-0fd5fa9a1140        4ff49c41-80a1-4030-bc02-cd76e3b0b84a        3306
`),
		},
		{
			name:       "empty",
			listResult: listResult{},
			expected: strings.TrimSpace(`
Service type        Service name        Address and port        Service ID

Agent type        Status        Metrics Mode        Agent ID        Service ID        Port
`),
		},
		{
			name: "external",
			listResult: listResult{
				Services: []listResultService{
					{ServiceType: types.ServiceTypeExternalService, ServiceID: "8ff49c41-80a1-4030-bc02-cd76e3b0b84a", ServiceName: "myhost-redis", Group: "redis"},
				},
				Agents: []listResultAgent{
					{AgentType: types.AgentTypeExternalExporter, AgentID: "8b732ac3-8256-40b0-a98b-0fd5fa9a1149", ServiceID: "8ff49c41-80a1-4030-bc02-cd76e3b0b84a", Status: "RUNNING", Port: 8080},
				},
			},
			expected: strings.TrimSpace(`
Service type          Service name        Address and port        Service ID
External:redis        myhost-redis                                8ff49c41-80a1-4030-bc02-cd76e3b0b84a

Agent type               Status         Metrics Mode        Agent ID                                    Service ID                                  Port
external-exporter        Running                            8b732ac3-8256-40b0-a98b-0fd5fa9a1149        8ff49c41-80a1-4030-bc02-cd76e3b0b84a        8080
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := strings.TrimSpace(tt.listResult.String())
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNiceAgentStatus(t *testing.T) {
	type fields struct {
		Status   string
		Disabled bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "emptyStatus",
			fields: fields{
				Status: "",
			},
			want: "Unknown",
		},
		{
			name: "disabled",
			fields: fields{
				Disabled: true,
			},
			want: "Unknown (disabled)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := listResultAgent{
				Status:   tt.fields.Status,
				Disabled: tt.fields.Disabled,
			}
			assert.Equal(t, tt.want, a.NiceAgentStatus())
		})
	}
}

func TestListJSONOutput(t *testing.T) {
	t.Parallel()
	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		services := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mysql: []*services_service.ListServicesOKBodyMysqlItems0{
					{
						ServiceID:   "4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
						ServiceName: "mysql-service",
						Address:     "127.0.0.1",
						Port:        3306,
					},
				},
			},
		}
		agents := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				PMMAgent: []*agents_service.ListAgentsOKBodyPMMAgentItems0{
					{
						AgentID:      "8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
						RunsOnNodeID: "8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
						Connected:    true,
					},
				},
				MysqldExporter: []*agents_service.ListAgentsOKBodyMysqldExporterItems0{
					{
						AgentID:            "8b732ac3-8256-40b0-a98b-0fd5fa9a1198",
						PMMAgentID:         "8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
						ServiceID:          "4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
						Status:             pointer.ToString("RUNNING"),
						PushMetricsEnabled: false,
						ListenPort:         3306,
					},
				},
			},
		}
		result := listResult{
			Services: servicesList(services),
			Agents:   agentsList(agents, "8b732ac3-8256-40b0-a98b-0fd5fa9a1140"),
		}

		res, err := json.Marshal(result)
		assert.NoError(t, err)
		expected := `
		{
			"service": [
				{
					"service_type": "SERVICE_TYPE_MYSQL_SERVICE",
					"service_id": "4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
					"service_name": "mysql-service",
					"address_port": "127.0.0.1:3306",
					"external_group": ""
				}
			],
			"agent": [
				{
					"agent_type": "AGENT_TYPE_PMM_AGENT",
					"agent_id": "8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
					"service_id": "",
					"status": "CONNECTED",
					"disabled": false,
					"push_metrics_enabled": ""
				},
				{
					"agent_type": "AGENT_TYPE_MYSQLD_EXPORTER",
					"agent_id": "8b732ac3-8256-40b0-a98b-0fd5fa9a1198",
					"service_id": "4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
					"status": "RUNNING",
					"disabled": false,
					"push_metrics_enabled": "pull",
					"port": 3306
				}
			]
		}
		`
		expected = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(expected, "\t", ""), "\n", ""), " ", "")
		assert.Equal(t, expected, string(res))
	})
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		result := listResult{
			Services: servicesList(&services_service.ListServicesOK{
				Payload: &services_service.ListServicesOKBody{},
			}),
			Agents: agentsList(&agents_service.ListAgentsOK{
				Payload: &agents_service.ListAgentsOKBody{},
			}, ""),
		}

		res, err := json.Marshal(result)
		assert.NoError(t, err)
		expected := `{"service":[],"agent":[]}`
		assert.Equal(t, expected, string(res))
	})
}

func TestAgentsList(t *testing.T) {
	t.Parallel()

	nodeID := "test-node-id"
	pmmAgentID := "pmm-agent-id"

	t.Run("empty payload", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{},
		}

		result := agentsList(agentsRes, nodeID)
		assert.Empty(t, result)
	})

	t.Run("all agent types", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				PMMAgent: []*agents_service.ListAgentsOKBodyPMMAgentItems0{
					{
						AgentID:      pmmAgentID,
						RunsOnNodeID: nodeID,
						Connected:    true,
					},
				},
				NodeExporter: []*agents_service.ListAgentsOKBodyNodeExporterItems0{
					{
						AgentID:            "node-exporter-id",
						PMMAgentID:         pmmAgentID,
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						PushMetricsEnabled: false,
						ListenPort:         9100,
					},
				},
				MysqldExporter: []*agents_service.ListAgentsOKBodyMysqldExporterItems0{
					{
						AgentID:            "mysqld-exporter-id",
						PMMAgentID:         pmmAgentID,
						ServiceID:          "mysql-service-id",
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						PushMetricsEnabled: true,
						ListenPort:         9104,
					},
				},
				MongodbExporter: []*agents_service.ListAgentsOKBodyMongodbExporterItems0{
					{
						AgentID:            "mongodb-exporter-id",
						PMMAgentID:         pmmAgentID,
						ServiceID:          "mongodb-service-id",
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						PushMetricsEnabled: false,
						ListenPort:         9216,
					},
				},
				PostgresExporter: []*agents_service.ListAgentsOKBodyPostgresExporterItems0{
					{
						AgentID:            "postgres-exporter-id",
						PMMAgentID:         pmmAgentID,
						ServiceID:          "postgres-service-id",
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						PushMetricsEnabled: true,
						ListenPort:         9187,
					},
				},
				ProxysqlExporter: []*agents_service.ListAgentsOKBodyProxysqlExporterItems0{
					{
						AgentID:            "proxysql-exporter-id",
						PMMAgentID:         pmmAgentID,
						ServiceID:          "proxysql-service-id",
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						PushMetricsEnabled: false,
						ListenPort:         6032,
					},
				},
				RDSExporter: []*agents_service.ListAgentsOKBodyRDSExporterItems0{
					{
						AgentID:            "rds-exporter-id",
						PMMAgentID:         pmmAgentID,
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						PushMetricsEnabled: true,
						ListenPort:         9042,
					},
				},
				QANMysqlPerfschemaAgent: []*agents_service.ListAgentsOKBodyQANMysqlPerfschemaAgentItems0{
					{
						AgentID:    "qan-mysql-perfschema-id",
						PMMAgentID: pmmAgentID,
						ServiceID:  "mysql-service-id",
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					},
				},
				QANMysqlSlowlogAgent: []*agents_service.ListAgentsOKBodyQANMysqlSlowlogAgentItems0{
					{
						AgentID:    "qan-mysql-slowlog-id",
						PMMAgentID: pmmAgentID,
						ServiceID:  "mysql-service-id",
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					},
				},
				QANMongodbProfilerAgent: []*agents_service.ListAgentsOKBodyQANMongodbProfilerAgentItems0{
					{
						AgentID:    "qan-mongodb-profiler-id",
						PMMAgentID: pmmAgentID,
						ServiceID:  "mongodb-service-id",
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					},
				},
				QANPostgresqlPgstatementsAgent: []*agents_service.ListAgentsOKBodyQANPostgresqlPgstatementsAgentItems0{
					{
						AgentID:    "qan-postgres-pgstatements-id",
						PMMAgentID: pmmAgentID,
						ServiceID:  "postgres-service-id",
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					},
				},
				QANPostgresqlPgstatmonitorAgent: []*agents_service.ListAgentsOKBodyQANPostgresqlPgstatmonitorAgentItems0{
					{
						AgentID:    "qan-postgres-pgstatmonitor-id",
						PMMAgentID: pmmAgentID,
						ServiceID:  "postgres-service-id",
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					},
				},
				ExternalExporter: []*agents_service.ListAgentsOKBodyExternalExporterItems0{
					{
						AgentID:            "external-exporter-id",
						RunsOnNodeID:       nodeID,
						ServiceID:          "external-service-id",
						PushMetricsEnabled: false,
						ListenPort:         8080,
					},
				},
				VMAgent: []*agents_service.ListAgentsOKBodyVMAgentItems0{
					{
						AgentID:    "vm-agent-id",
						PMMAgentID: pmmAgentID,
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
						ListenPort: 8429,
					},
				},
				NomadAgent: []*agents_service.ListAgentsOKBodyNomadAgentItems0{
					{
						AgentID:    "nomad-agent-id",
						PMMAgentID: pmmAgentID,
						Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
						ListenPort: 4646,
					},
				},
			},
		}

		result := agentsList(agentsRes, nodeID)

		// Should have 15 agents total
		assert.Len(t, result, 15)

		// Verify each agent type is present
		agentTypes := make(map[string]bool)
		for _, agent := range result {
			agentTypes[agent.AgentType] = true
		}

		expectedTypes := []string{
			types.AgentTypePMMAgent,
			types.AgentTypeNodeExporter,
			types.AgentTypeMySQLdExporter,
			types.AgentTypeMongoDBExporter,
			types.AgentTypePostgresExporter,
			types.AgentTypeProxySQLExporter,
			types.AgentTypeRDSExporter,
			types.AgentTypeQANMySQLPerfSchemaAgent,
			types.AgentTypeQANMySQLSlowlogAgent,
			types.AgentTypeQANMongoDBProfilerAgent,
			types.AgentTypeQANPostgreSQLPgStatementsAgent,
			types.AgentTypeQANPostgreSQLPgStatMonitorAgent,
			types.AgentTypeExternalExporter,
			types.AgentTypeVMAgent,
			types.AgentTypeNomadAgent,
		}

		for _, expectedType := range expectedTypes {
			assert.True(t, agentTypes[expectedType], "Expected agent type %s not found", expectedType)
		}
	})
}

func TestPmmAgents(t *testing.T) {
	t.Parallel()

	nodeID := "test-node-id"
	pmmAgentIDs := make(map[string]struct{})

	t.Run("connected agent", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				PMMAgent: []*agents_service.ListAgentsOKBodyPMMAgentItems0{
					{
						AgentID:      "pmm-agent-1",
						RunsOnNodeID: nodeID,
						Connected:    true,
					},
				},
			},
		}

		result := pmmAgents(agentsRes, nodeID, pmmAgentIDs)

		assert.Len(t, result, 1)
		assert.Equal(t, types.AgentTypePMMAgent, result[0].AgentType)
		assert.Equal(t, "pmm-agent-1", result[0].AgentID)
		assert.Equal(t, "CONNECTED", result[0].Status)
		assert.Contains(t, pmmAgentIDs, "pmm-agent-1")
	})

	t.Run("disconnected agent", func(t *testing.T) {
		t.Parallel()
		pmmAgentIDs := make(map[string]struct{})
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				PMMAgent: []*agents_service.ListAgentsOKBodyPMMAgentItems0{
					{
						AgentID:      "pmm-agent-2",
						RunsOnNodeID: nodeID,
						Connected:    false,
					},
				},
			},
		}

		result := pmmAgents(agentsRes, nodeID, pmmAgentIDs)

		assert.Len(t, result, 1)
		assert.Equal(t, "DISCONNECTED", result[0].Status)
	})

	t.Run("different node", func(t *testing.T) {
		t.Parallel()
		pmmAgentIDs := make(map[string]struct{})
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				PMMAgent: []*agents_service.ListAgentsOKBodyPMMAgentItems0{
					{
						AgentID:      "pmm-agent-3",
						RunsOnNodeID: "different-node",
						Connected:    true,
					},
				},
			},
		}

		result := pmmAgents(agentsRes, nodeID, pmmAgentIDs)

		assert.Empty(t, result)
		assert.NotContains(t, pmmAgentIDs, "pmm-agent-3")
	})
}

func TestMongodbExporters(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	t.Run("valid exporter", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				MongodbExporter: []*agents_service.ListAgentsOKBodyMongodbExporterItems0{
					{
						AgentID:            "mongodb-exporter-1",
						PMMAgentID:         "pmm-agent-1",
						ServiceID:          "mongodb-service-1",
						Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
						Disabled:           false,
						PushMetricsEnabled: true,
						ListenPort:         9216,
					},
				},
			},
		}

		result := mongodbExporters(agentsRes, pmmAgentIDs)

		assert.Len(t, result, 1)
		assert.Equal(t, types.AgentTypeMongoDBExporter, result[0].AgentType)
		assert.Equal(t, "mongodb-exporter-1", result[0].AgentID)
		assert.Equal(t, "mongodb-service-1", result[0].ServiceID)
		assert.Equal(t, "RUNNING", result[0].Status)
		assert.False(t, result[0].Disabled)
		assert.Equal(t, "push", result[0].MetricsMode)
		assert.Equal(t, int64(9216), result[0].Port)
	})

	t.Run("invalid pmm agent", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				MongodbExporter: []*agents_service.ListAgentsOKBodyMongodbExporterItems0{
					{
						AgentID:    "mongodb-exporter-2",
						PMMAgentID: "invalid-pmm-agent",
						ServiceID:  "mongodb-service-2",
					},
				},
			},
		}

		result := mongodbExporters(agentsRes, pmmAgentIDs)

		assert.Empty(t, result)
	})
}

func TestPostgresExporters(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	t.Run("valid exporter", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				PostgresExporter: []*agents_service.ListAgentsOKBodyPostgresExporterItems0{
					{
						AgentID:            "postgres-exporter-1",
						PMMAgentID:         "pmm-agent-1",
						ServiceID:          "postgres-service-1",
						Status:             pointer.ToString("AGENT_STATUS_STOPPED"),
						Disabled:           true,
						PushMetricsEnabled: false,
						ListenPort:         9187,
					},
				},
			},
		}

		result := postgresExporters(agentsRes, pmmAgentIDs)

		assert.Len(t, result, 1)
		assert.Equal(t, types.AgentTypePostgresExporter, result[0].AgentType)
		assert.Equal(t, "postgres-exporter-1", result[0].AgentID)
		assert.Equal(t, "postgres-service-1", result[0].ServiceID)
		assert.Equal(t, "STOPPED", result[0].Status)
		assert.True(t, result[0].Disabled)
		assert.Equal(t, "pull", result[0].MetricsMode)
		assert.Equal(t, int64(9187), result[0].Port)
	})
}

func TestProxysqlExporters(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			ProxysqlExporter: []*agents_service.ListAgentsOKBodyProxysqlExporterItems0{
				{
					AgentID:            "proxysql-exporter-1",
					PMMAgentID:         "pmm-agent-1",
					ServiceID:          "proxysql-service-1",
					Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
					PushMetricsEnabled: true,
					ListenPort:         6032,
				},
			},
		},
	}

	result := proxysqlExporters(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeProxySQLExporter, result[0].AgentType)
	assert.Equal(t, "push", result[0].MetricsMode)
}

func TestRdsExporters(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			RDSExporter: []*agents_service.ListAgentsOKBodyRDSExporterItems0{
				{
					AgentID:            "rds-exporter-1",
					PMMAgentID:         "pmm-agent-1",
					Status:             pointer.ToString("AGENT_STATUS_RUNNING"),
					PushMetricsEnabled: false,
					ListenPort:         9042,
				},
			},
		},
	}

	result := rdsExporters(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeRDSExporter, result[0].AgentType)
	assert.Empty(t, result[0].ServiceID) // RDS exporters don't have service ID
}

func TestQanMysqlPerfschemaAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			QANMysqlPerfschemaAgent: []*agents_service.ListAgentsOKBodyQANMysqlPerfschemaAgentItems0{
				{
					AgentID:    "qan-mysql-perfschema-1",
					PMMAgentID: "pmm-agent-1",
					ServiceID:  "mysql-service-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					Disabled:   false,
				},
			},
		},
	}

	result := qanMysqlPerfschemaAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeQANMySQLPerfSchemaAgent, result[0].AgentType)
	assert.Empty(t, result[0].MetricsMode)    // QAN agents don't have metrics mode
	assert.Equal(t, int64(0), result[0].Port) // QAN agents don't have ports
}

func TestQanMysqlSlowlogAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			QANMysqlSlowlogAgent: []*agents_service.ListAgentsOKBodyQANMysqlSlowlogAgentItems0{
				{
					AgentID:    "qan-mysql-slowlog-1",
					PMMAgentID: "pmm-agent-1",
					ServiceID:  "mysql-service-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
				},
			},
		},
	}

	result := qanMysqlSlowlogAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeQANMySQLSlowlogAgent, result[0].AgentType)
}

func TestQanMongodbProfilerAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			QANMongodbProfilerAgent: []*agents_service.ListAgentsOKBodyQANMongodbProfilerAgentItems0{
				{
					AgentID:    "qan-mongodb-profiler-1",
					PMMAgentID: "pmm-agent-1",
					ServiceID:  "mongodb-service-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
				},
			},
		},
	}

	result := qanMongodbProfilerAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeQANMongoDBProfilerAgent, result[0].AgentType)
}

func TestQanPostgresqlPgstatementsAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			QANPostgresqlPgstatementsAgent: []*agents_service.ListAgentsOKBodyQANPostgresqlPgstatementsAgentItems0{
				{
					AgentID:    "qan-postgres-pgstatements-1",
					PMMAgentID: "pmm-agent-1",
					ServiceID:  "postgres-service-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
				},
			},
		},
	}

	result := qanPostgresqlPgstatementsAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeQANPostgreSQLPgStatementsAgent, result[0].AgentType)
}

func TestQanPostgresqlPgstatmonitorAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			QANPostgresqlPgstatmonitorAgent: []*agents_service.ListAgentsOKBodyQANPostgresqlPgstatmonitorAgentItems0{
				{
					AgentID:    "qan-postgres-pgstatmonitor-1",
					PMMAgentID: "pmm-agent-1",
					ServiceID:  "postgres-service-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
				},
			},
		},
	}

	result := qanPostgresqlPgstatmonitorAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeQANPostgreSQLPgStatMonitorAgent, result[0].AgentType)
}

func TestExternalExporters(t *testing.T) {
	t.Parallel()

	nodeID := "test-node-id"

	t.Run("valid exporter", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				ExternalExporter: []*agents_service.ListAgentsOKBodyExternalExporterItems0{
					{
						AgentID:            "external-exporter-1",
						RunsOnNodeID:       nodeID,
						ServiceID:          "external-service-1",
						Disabled:           false,
						PushMetricsEnabled: true,
						ListenPort:         8080,
					},
				},
			},
		}

		result := externalExporters(agentsRes, nodeID)

		assert.Len(t, result, 1)
		assert.Equal(t, types.AgentTypeExternalExporter, result[0].AgentType)
		assert.Equal(t, "external-exporter-1", result[0].AgentID)
		assert.Equal(t, "external-service-1", result[0].ServiceID)
		assert.Equal(t, "UNKNOWN", result[0].Status) // External exporters get nil status
		assert.Equal(t, "push", result[0].MetricsMode)
		assert.Equal(t, int64(8080), result[0].Port)
	})

	t.Run("different node", func(t *testing.T) {
		t.Parallel()
		agentsRes := &agents_service.ListAgentsOK{
			Payload: &agents_service.ListAgentsOKBody{
				ExternalExporter: []*agents_service.ListAgentsOKBodyExternalExporterItems0{
					{
						AgentID:      "external-exporter-2",
						RunsOnNodeID: "different-node",
						ServiceID:    "external-service-2",
					},
				},
			},
		}

		result := externalExporters(agentsRes, nodeID)

		assert.Empty(t, result)
	})
}

func TestVmAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			VMAgent: []*agents_service.ListAgentsOKBodyVMAgentItems0{
				{
					AgentID:    "vm-agent-1",
					PMMAgentID: "pmm-agent-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					ListenPort: 8429,
				},
			},
		},
	}

	result := vmAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeVMAgent, result[0].AgentType)
	assert.Equal(t, "push", result[0].MetricsMode) // VM agents always push
	assert.Empty(t, result[0].ServiceID)           // VM agents don't have service ID
}

func TestNomadAgents(t *testing.T) {
	t.Parallel()

	pmmAgentIDs := map[string]struct{}{
		"pmm-agent-1": {},
	}

	agentsRes := &agents_service.ListAgentsOK{
		Payload: &agents_service.ListAgentsOKBody{
			NomadAgent: []*agents_service.ListAgentsOKBodyNomadAgentItems0{
				{
					AgentID:    "nomad-agent-1",
					PMMAgentID: "pmm-agent-1",
					Status:     pointer.ToString("AGENT_STATUS_RUNNING"),
					Disabled:   false,
					ListenPort: 4646,
				},
			},
		},
	}

	result := nomadAgents(agentsRes, pmmAgentIDs)

	assert.Len(t, result, 1)
	assert.Equal(t, types.AgentTypeNomadAgent, result[0].AgentType)
	assert.Empty(t, result[0].MetricsMode) // Nomad agents don't have metrics mode
	assert.Empty(t, result[0].ServiceID)   // Nomad agents don't have service ID
}

func TestGetStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   *string
		expected string
	}{
		{
			name:     "nil status",
			status:   nil,
			expected: "UNKNOWN",
		},
		{
			name:     "empty status",
			status:   pointer.ToString(""),
			expected: "UNKNOWN",
		},
		{
			name:     "status with prefix",
			status:   pointer.ToString("AGENT_STATUS_RUNNING"),
			expected: "RUNNING",
		},
		{
			name:     "status without prefix",
			status:   pointer.ToString("RUNNING"),
			expected: "RUNNING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMetricsMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected string
	}{
		{
			name:     "push enabled",
			enabled:  true,
			expected: "push",
		},
		{
			name:     "push disabled",
			enabled:  false,
			expected: "pull",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getMetricsMode(tt.enabled)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServicesList(t *testing.T) {
	t.Parallel()

	t.Run("empty payload", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{},
		}

		result := servicesList(servicesRes)
		assert.Empty(t, result)
	})

	t.Run("all service types", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mysql: []*services_service.ListServicesOKBodyMysqlItems0{
					{
						ServiceID:   "mysql-service-id",
						ServiceName: "mysql-service",
						Address:     "127.0.0.1",
						Port:        3306,
						Socket:      "",
					},
				},
				Mongodb: []*services_service.ListServicesOKBodyMongodbItems0{
					{
						ServiceID:   "mongodb-service-id",
						ServiceName: "mongodb-service",
						Address:     "127.0.0.1",
						Port:        27017,
						Socket:      "",
					},
				},
				Postgresql: []*services_service.ListServicesOKBodyPostgresqlItems0{
					{
						ServiceID:   "postgres-service-id",
						ServiceName: "postgres-service",
						Address:     "127.0.0.1",
						Port:        5432,
						Socket:      "",
					},
				},
				Proxysql: []*services_service.ListServicesOKBodyProxysqlItems0{
					{
						ServiceID:   "proxysql-service-id",
						ServiceName: "proxysql-service",
						Address:     "127.0.0.1",
						Port:        6032,
						Socket:      "",
					},
				},
				Haproxy: []*services_service.ListServicesOKBodyHaproxyItems0{
					{
						ServiceID:   "haproxy-service-id",
						ServiceName: "haproxy-service",
					},
				},
				External: []*services_service.ListServicesOKBodyExternalItems0{
					{
						ServiceID:   "external-service-id",
						ServiceName: "external-service",
						Group:       "redis",
					},
				},
			},
		}

		result := servicesList(servicesRes)

		// Should have 6 services total
		assert.Len(t, result, 6)

		// Verify each service type is present
		serviceTypes := make(map[string]bool)
		for _, service := range result {
			serviceTypes[service.ServiceType] = true
		}

		expectedTypes := []string{
			types.ServiceTypeMySQLService,
			types.ServiceTypeMongoDBService,
			types.ServiceTypePostgreSQLService,
			types.ServiceTypeProxySQLService,
			types.ServiceTypeHAProxyService,
			types.ServiceTypeExternalService,
		}

		for _, expectedType := range expectedTypes {
			assert.True(t, serviceTypes[expectedType], "Expected service type %s not found", expectedType)
		}

		// Verify specific service details
		for _, service := range result {
			switch service.ServiceType {
			case types.ServiceTypeMySQLService:
				assert.Equal(t, "mysql-service-id", service.ServiceID)
				assert.Equal(t, "mysql-service", service.ServiceName)
				assert.Equal(t, "127.0.0.1:3306", service.AddressPort)
			case types.ServiceTypeMongoDBService:
				assert.Equal(t, "mongodb-service-id", service.ServiceID)
				assert.Equal(t, "mongodb-service", service.ServiceName)
				assert.Equal(t, "127.0.0.1:27017", service.AddressPort)
			case types.ServiceTypePostgreSQLService:
				assert.Equal(t, "postgres-service-id", service.ServiceID)
				assert.Equal(t, "postgres-service", service.ServiceName)
				assert.Equal(t, "127.0.0.1:5432", service.AddressPort)
			case types.ServiceTypeProxySQLService:
				assert.Equal(t, "proxysql-service-id", service.ServiceID)
				assert.Equal(t, "proxysql-service", service.ServiceName)
				assert.Equal(t, "127.0.0.1:6032", service.AddressPort)
			case types.ServiceTypeHAProxyService:
				assert.Equal(t, "haproxy-service-id", service.ServiceID)
				assert.Equal(t, "haproxy-service", service.ServiceName)
				assert.Empty(t, service.AddressPort) // HAProxy services don't have address/port
			case types.ServiceTypeExternalService:
				assert.Equal(t, "external-service-id", service.ServiceID)
				assert.Equal(t, "external-service", service.ServiceName)
				assert.Equal(t, "redis", service.Group)
				assert.Empty(t, service.AddressPort) // External services don't have address/port
			}
		}
	})
}

func TestMysqlServices(t *testing.T) {
	t.Parallel()

	t.Run("valid mysql service", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mysql: []*services_service.ListServicesOKBodyMysqlItems0{
					{
						ServiceID:   "mysql-service-1",
						ServiceName: "mysql-db",
						Address:     "localhost",
						Port:        3306,
						Socket:      "",
					},
				},
			},
		}

		result := mysqlServices(servicesRes)

		assert.Len(t, result, 1)
		assert.Equal(t, types.ServiceTypeMySQLService, result[0].ServiceType)
		assert.Equal(t, "mysql-service-1", result[0].ServiceID)
		assert.Equal(t, "mysql-db", result[0].ServiceName)
		assert.Equal(t, "localhost:3306", result[0].AddressPort)
		assert.Empty(t, result[0].Group)
	})

	t.Run("mysql service with socket", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mysql: []*services_service.ListServicesOKBodyMysqlItems0{
					{
						ServiceID:   "mysql-service-2",
						ServiceName: "mysql-socket",
						Address:     "localhost",
						Port:        3306,
						Socket:      "/var/run/mysqld/mysqld.sock",
					},
				},
			},
		}

		result := mysqlServices(servicesRes)

		assert.Len(t, result, 1)
		assert.Equal(t, "/var/run/mysqld/mysqld.sock", result[0].AddressPort)
	})

	t.Run("empty mysql services", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mysql: []*services_service.ListServicesOKBodyMysqlItems0{},
			},
		}

		result := mysqlServices(servicesRes)
		assert.Empty(t, result)
	})
}

func TestMongodbServices(t *testing.T) {
	t.Parallel()

	t.Run("valid mongodb service", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mongodb: []*services_service.ListServicesOKBodyMongodbItems0{
					{
						ServiceID:   "mongodb-service-1",
						ServiceName: "mongodb-db",
						Address:     "mongodb.example.com",
						Port:        27017,
						Socket:      "",
					},
				},
			},
		}

		result := mongodbServices(servicesRes)

		assert.Len(t, result, 1)
		assert.Equal(t, types.ServiceTypeMongoDBService, result[0].ServiceType)
		assert.Equal(t, "mongodb-service-1", result[0].ServiceID)
		assert.Equal(t, "mongodb-db", result[0].ServiceName)
		assert.Equal(t, "mongodb.example.com:27017", result[0].AddressPort)
	})

	t.Run("mongodb service with socket", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mongodb: []*services_service.ListServicesOKBodyMongodbItems0{
					{
						ServiceID:   "mongodb-service-2",
						ServiceName: "mongodb-socket",
						Address:     "localhost",
						Port:        27017,
						Socket:      "/tmp/mongodb-27017.sock",
					},
				},
			},
		}

		result := mongodbServices(servicesRes)

		assert.Len(t, result, 1)
		assert.Equal(t, "/tmp/mongodb-27017.sock", result[0].AddressPort)
	})
}

func TestPostgresqlServices(t *testing.T) {
	t.Parallel()

	servicesRes := &services_service.ListServicesOK{
		Payload: &services_service.ListServicesOKBody{
			Postgresql: []*services_service.ListServicesOKBodyPostgresqlItems0{
				{
					ServiceID:   "postgres-service-1",
					ServiceName: "postgres-db",
					Address:     "postgres.example.com",
					Port:        5432,
					Socket:      "",
				},
			},
		},
	}

	result := postgresqlServices(servicesRes)

	assert.Len(t, result, 1)
	assert.Equal(t, types.ServiceTypePostgreSQLService, result[0].ServiceType)
	assert.Equal(t, "postgres-service-1", result[0].ServiceID)
	assert.Equal(t, "postgres-db", result[0].ServiceName)
	assert.Equal(t, "postgres.example.com:5432", result[0].AddressPort)
}

func TestProxysqlServices(t *testing.T) {
	t.Parallel()

	servicesRes := &services_service.ListServicesOK{
		Payload: &services_service.ListServicesOKBody{
			Proxysql: []*services_service.ListServicesOKBodyProxysqlItems0{
				{
					ServiceID:   "proxysql-service-1",
					ServiceName: "proxysql-db",
					Address:     "proxysql.example.com",
					Port:        6032,
					Socket:      "",
				},
			},
		},
	}

	result := proxysqlServices(servicesRes)

	assert.Len(t, result, 1)
	assert.Equal(t, types.ServiceTypeProxySQLService, result[0].ServiceType)
	assert.Equal(t, "proxysql-service-1", result[0].ServiceID)
	assert.Equal(t, "proxysql-db", result[0].ServiceName)
	assert.Equal(t, "proxysql.example.com:6032", result[0].AddressPort)
}

func TestHaproxyServices(t *testing.T) {
	t.Parallel()

	servicesRes := &services_service.ListServicesOK{
		Payload: &services_service.ListServicesOKBody{
			Haproxy: []*services_service.ListServicesOKBodyHaproxyItems0{
				{
					ServiceID:   "haproxy-service-1",
					ServiceName: "haproxy-lb",
				},
			},
		},
	}

	result := haproxyServices(servicesRes)

	assert.Len(t, result, 1)
	assert.Equal(t, types.ServiceTypeHAProxyService, result[0].ServiceType)
	assert.Equal(t, "haproxy-service-1", result[0].ServiceID)
	assert.Equal(t, "haproxy-lb", result[0].ServiceName)
	assert.Empty(t, result[0].AddressPort) // HAProxy services don't have address/port
	assert.Empty(t, result[0].Group)
}

func TestExternalServices(t *testing.T) {
	t.Parallel()

	t.Run("valid external service", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				External: []*services_service.ListServicesOKBodyExternalItems0{
					{
						ServiceID:   "external-service-1",
						ServiceName: "redis-cache",
						Group:       "redis",
					},
				},
			},
		}

		result := externalServices(servicesRes)

		assert.Len(t, result, 1)
		assert.Equal(t, types.ServiceTypeExternalService, result[0].ServiceType)
		assert.Equal(t, "external-service-1", result[0].ServiceID)
		assert.Equal(t, "redis-cache", result[0].ServiceName)
		assert.Equal(t, "redis", result[0].Group)
		assert.Empty(t, result[0].AddressPort) // External services don't have address/port
	})

	t.Run("external service with different group", func(t *testing.T) {
		t.Parallel()
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				External: []*services_service.ListServicesOKBodyExternalItems0{
					{
						ServiceID:   "external-service-2",
						ServiceName: "elastic-search",
						Group:       "elasticsearch",
					},
				},
			},
		}

		result := externalServices(servicesRes)

		assert.Len(t, result, 1)
		assert.Equal(t, "elasticsearch", result[0].Group)
	})
}

func TestGetSocketOrHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		socket   string
		address  string
		port     int64
		expected string
	}{
		{
			name:     "socket provided",
			socket:   "/var/run/mysqld/mysqld.sock",
			address:  "localhost",
			port:     3306,
			expected: "/var/run/mysqld/mysqld.sock",
		},
		{
			name:     "no socket, use address and port",
			socket:   "",
			address:  "127.0.0.1",
			port:     3306,
			expected: "127.0.0.1:3306",
		},
		{
			name:     "IPv6 address",
			socket:   "",
			address:  "::1",
			port:     5432,
			expected: "[::1]:5432",
		},
		{
			name:     "hostname with port",
			socket:   "",
			address:  "database.example.com",
			port:     27017,
			expected: "database.example.com:27017",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getSocketOrHost(tt.socket, tt.address, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServicesListPreallocation(t *testing.T) {
	t.Parallel()

	t.Run("pre-allocation efficiency", func(t *testing.T) {
		t.Parallel()
		// Create a payload with known number of services
		servicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{
				Mysql: []*services_service.ListServicesOKBodyMysqlItems0{
					{ServiceID: "mysql-1", ServiceName: "mysql-service-1", Address: "127.0.0.1", Port: 3306},
					{ServiceID: "mysql-2", ServiceName: "mysql-service-2", Address: "127.0.0.1", Port: 3307},
				},
				Mongodb: []*services_service.ListServicesOKBodyMongodbItems0{
					{ServiceID: "mongodb-1", ServiceName: "mongodb-service-1", Address: "127.0.0.1", Port: 27017},
				},
				Postgresql: []*services_service.ListServicesOKBodyPostgresqlItems0{
					{ServiceID: "postgres-1", ServiceName: "postgres-service-1", Address: "127.0.0.1", Port: 5432},
					{ServiceID: "postgres-2", ServiceName: "postgres-service-2", Address: "127.0.0.1", Port: 5433},
					{ServiceID: "postgres-3", ServiceName: "postgres-service-3", Address: "127.0.0.1", Port: 5434},
				},
				Proxysql: []*services_service.ListServicesOKBodyProxysqlItems0{
					{ServiceID: "proxysql-1", ServiceName: "proxysql-service-1", Address: "127.0.0.1", Port: 6032},
				},
				Haproxy: []*services_service.ListServicesOKBodyHaproxyItems0{
					{ServiceID: "haproxy-1", ServiceName: "haproxy-service-1"},
				},
				External: []*services_service.ListServicesOKBodyExternalItems0{
					{ServiceID: "external-1", ServiceName: "external-service-1", Group: "redis"},
					{ServiceID: "external-2", ServiceName: "external-service-2", Group: "elasticsearch"},
				},
			},
		}

		result := servicesList(servicesRes)

		// Verify we get exactly the expected number of services (2+1+3+1+1+2 = 10)
		expectedCount := 10
		assert.Len(t, result, expectedCount)

		// Verify all service types are present
		serviceTypeCount := make(map[string]int)
		for _, service := range result {
			serviceTypeCount[service.ServiceType]++
		}

		assert.Equal(t, 2, serviceTypeCount[types.ServiceTypeMySQLService])
		assert.Equal(t, 1, serviceTypeCount[types.ServiceTypeMongoDBService])
		assert.Equal(t, 3, serviceTypeCount[types.ServiceTypePostgreSQLService])
		assert.Equal(t, 1, serviceTypeCount[types.ServiceTypeProxySQLService])
		assert.Equal(t, 1, serviceTypeCount[types.ServiceTypeHAProxyService])
		assert.Equal(t, 2, serviceTypeCount[types.ServiceTypeExternalService])
	})

	t.Run("individual service functions pre-allocation", func(t *testing.T) {
		t.Parallel()
		// Test that individual service functions handle empty slices correctly
		emptyServicesRes := &services_service.ListServicesOK{
			Payload: &services_service.ListServicesOKBody{},
		}

		// All individual functions should return empty slices for empty input
		assert.Empty(t, mysqlServices(emptyServicesRes))
		assert.Empty(t, mongodbServices(emptyServicesRes))
		assert.Empty(t, postgresqlServices(emptyServicesRes))
		assert.Empty(t, proxysqlServices(emptyServicesRes))
		assert.Empty(t, haproxyServices(emptyServicesRes))
		assert.Empty(t, externalServices(emptyServicesRes))
	})
}
