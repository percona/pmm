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

	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	"github.com/percona/pmm/api/inventorypb/types"
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
					{ServiceType: types.ServiceTypeMySQLService, ServiceID: "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a", ServiceName: "mysql-service"},
				},
				Agents: []listResultAgent{
					{AgentType: types.AgentTypeMySQLdExporter, AgentID: "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140", ServiceID: "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a", Status: "RUNNING", MetricsMode: "pull", Port: 3306},
				},
			},
			expected: strings.TrimSpace(`
Service type        Service name         Address and port        Service ID
MySQL               mysql-service                                /service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a

Agent type             Status         Metrics Mode        Agent ID                                              Service ID                                              Port
mysqld_exporter        Running        pull                /agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140        /service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a        3306
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
					{ServiceType: types.ServiceTypeExternalService, ServiceID: "/service_id/8ff49c41-80a1-4030-bc02-cd76e3b0b84a", ServiceName: "myhost-redis", Group: "redis"},
				},
				Agents: []listResultAgent{
					{AgentType: types.AgentTypeExternalExporter, AgentID: "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1149", ServiceID: "/service_id/8ff49c41-80a1-4030-bc02-cd76e3b0b84a", Status: "RUNNING", Port: 8080},
				},
			},
			expected: strings.TrimSpace(`
Service type          Service name        Address and port        Service ID
External:redis        myhost-redis                                /service_id/8ff49c41-80a1-4030-bc02-cd76e3b0b84a

Agent type               Status         Metrics Mode        Agent ID                                              Service ID                                              Port
external-exporter        Running                            /agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1149        /service_id/8ff49c41-80a1-4030-bc02-cd76e3b0b84a        8080
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
		services := &services.ListServicesOK{
			Payload: &services.ListServicesOKBody{
				Mysql: []*services.ListServicesOKBodyMysqlItems0{
					{
						ServiceID:   "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
						ServiceName: "mysql-service",
						Address:     "127.0.0.1",
						Port:        3306,
					},
				},
			},
		}
		agents := &agents.ListAgentsOK{
			Payload: &agents.ListAgentsOKBody{
				PMMAgent: []*agents.ListAgentsOKBodyPMMAgentItems0{
					{
						AgentID:      "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
						RunsOnNodeID: "/node_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
						Connected:    true,
					},
				},
				MysqldExporter: []*agents.ListAgentsOKBodyMysqldExporterItems0{
					{
						AgentID:            "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1198",
						PMMAgentID:         "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
						ServiceID:          "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
						Status:             pointer.ToString("RUNNING"),
						PushMetricsEnabled: false,
						ListenPort:         3306,
					},
				},
			},
		}
		result := listResult{
			Services: servicesList(services),
			Agents:   agentsList(agents, "/node_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140"),
		}

		res, err := json.Marshal(result)
		assert.NoError(t, err)
		expected := `
		{
			"service": [
				{
					"service_type": "MYSQL_SERVICE",
					"service_id": "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
					"service_name": "mysql-service",
					"address_port": "127.0.0.1:3306",
					"external_group": ""
				}
			],
			"agent": [
				{
					"agent_type": "PMM_AGENT",
					"agent_id": "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140",
					"service_id": "",
					"status": "CONNECTED",
					"disabled": false,
					"push_metrics_enabled": ""
				},
				{
					"agent_type": "MYSQLD_EXPORTER",
					"agent_id": "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1198",
					"service_id": "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a",
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
			Services: servicesList(&services.ListServicesOK{
				Payload: &services.ListServicesOKBody{},
			}),
			Agents: agentsList(&agents.ListAgentsOK{
				Payload: &agents.ListAgentsOKBody{},
			}, ""),
		}

		res, err := json.Marshal(result)
		assert.NoError(t, err)
		expected := `{"service":[],"agent":[]}`
		assert.Equal(t, expected, string(res))
	})
}
