// pmm-admin
// Copyright 2019 Percona LLC
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

package inventory

import (
	"strings"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/types"
)

var listAgentsResultT = commands.ParseTemplate(`
Agents list.

{{ printf "%-27s" "Agent type" }} {{ printf "%-15s" "Status" }} {{ printf "%-47s" "Agent ID" }} {{ printf "%-47s" "PMM-Agent ID" }} {{ printf "%-47s" "Service ID" }} {{ printf "%-47s" "Port" }}
{{ range .Agents }}
{{- printf "%-27s" .HumanReadableAgentType }} {{ printf "%-15s" .NiceAgentStatus }} {{ .AgentID }}  {{ .PMMAgentID }}  {{ .ServiceID }} {{ .Port }}
{{ end }}
`)

var acceptableAgentTypes = map[string][]string{
	types.AgentTypePMMAgent:                        {types.AgentTypeName(types.AgentTypePMMAgent), "pmm-agent"},
	types.AgentTypeNodeExporter:                    {types.AgentTypeName(types.AgentTypeNodeExporter), "node-exporter"},
	types.AgentTypeMySQLdExporter:                  {types.AgentTypeName(types.AgentTypeMySQLdExporter), "mysqld-exporter"},
	types.AgentTypeMongoDBExporter:                 {types.AgentTypeName(types.AgentTypeMongoDBExporter), "mongodb-exporter"},
	types.AgentTypePostgresExporter:                {types.AgentTypeName(types.AgentTypePostgresExporter), "postgres-exporter"},
	types.AgentTypeProxySQLExporter:                {types.AgentTypeName(types.AgentTypeProxySQLExporter), "proxysql-exporter"},
	types.AgentTypeQANMySQLPerfSchemaAgent:         {types.AgentTypeName(types.AgentTypeQANMySQLPerfSchemaAgent), "qan-mysql-perfschema-agent"},
	types.AgentTypeQANMySQLSlowlogAgent:            {types.AgentTypeName(types.AgentTypeQANMySQLSlowlogAgent), "qan-mysql-slowlog-agent"},
	types.AgentTypeQANMongoDBProfilerAgent:         {types.AgentTypeName(types.AgentTypeQANMongoDBProfilerAgent), "qan-mongodb-profiler-agent"},
	types.AgentTypeQANPostgreSQLPgStatementsAgent:  {types.AgentTypeName(types.AgentTypeQANPostgreSQLPgStatementsAgent), "qan-postgresql-pgstatements-agent"},
	types.AgentTypeQANPostgreSQLPgStatMonitorAgent: {types.AgentTypeName(types.AgentTypeQANPostgreSQLPgStatMonitorAgent), "qan-postgresql-pgstatmonitor-agent"},
	types.AgentTypeRDSExporter:                     {types.AgentTypeName(types.AgentTypeRDSExporter), "rds-exporter"},
}

type listResultAgent struct {
	AgentType  string `json:"agent_type"`
	AgentID    string `json:"agent_id"`
	PMMAgentID string `json:"pmm_agent_id"`
	ServiceID  string `json:"service_id"`
	Status     string `json:"status"`
	Disabled   bool   `json:"disabled"`
	Port       int64  `json:"port,omitempty"`
}

func (a listResultAgent) HumanReadableAgentType() string {
	return types.AgentTypeName(a.AgentType)
}

type listAgentsResult struct {
	Agents []listResultAgent `json:"agents"`
}

func (a listResultAgent) NiceAgentStatus() string {
	res := a.Status
	res = strings.Title(strings.ToLower(res))
	if a.Disabled {
		res += " (disabled)"
	}
	return res
}

func (res *listAgentsResult) Result() {}

func (res *listAgentsResult) String() string {
	return commands.RenderTemplate(listAgentsResultT, res)
}

type listAgentsCommand struct {
	filters   agents.ListAgentsBody
	agentType string
}

// This is used in the json output. By convention, statuses must be in uppercase
func getAgentStatus(status *string) string {
	res := pointer.GetString(status)
	if res == "" {
		res = "UNKNOWN"
	}
	return res
}

func (cmd *listAgentsCommand) Run() (commands.Result, error) {
	agentType, err := formatTypeValue(acceptableAgentTypes, cmd.agentType)
	if err != nil {
		return nil, err
	}

	cmd.filters.AgentType = agentType

	params := &agents.ListAgentsParams{
		Body:    cmd.filters,
		Context: commands.Ctx,
	}
	agentsRes, err := client.Default.Agents.ListAgents(params)
	if err != nil {
		return nil, err
	}

	var agentsList []listResultAgent
	for _, a := range agentsRes.Payload.PMMAgent {
		status := "disconnected"
		if a.Connected {
			status = "connected"
		}
		agentsList = append(agentsList, listResultAgent{
			AgentType: types.AgentTypePMMAgent,
			AgentID:   a.AgentID,
			Status:    strings.ToUpper(status),
		})
	}
	for _, a := range agentsRes.Payload.NodeExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeNodeExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
			Port:       a.ListenPort,
		})
	}
	for _, a := range agentsRes.Payload.MysqldExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeMySQLdExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
			Port:       a.ListenPort,
		})
	}
	for _, a := range agentsRes.Payload.MongodbExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeMongoDBExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
			Port:       a.ListenPort,
		})
	}
	for _, a := range agentsRes.Payload.PostgresExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypePostgresExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
			Port:       a.ListenPort,
		})
	}
	for _, a := range agentsRes.Payload.ProxysqlExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeProxySQLExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
			Port:       a.ListenPort,
		})
	}
	for _, a := range agentsRes.Payload.RDSExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeRDSExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
			Port:       a.ListenPort,
		})
	}
	for _, a := range agentsRes.Payload.QANMysqlPerfschemaAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANMySQLPerfSchemaAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
		})
	}
	for _, a := range agentsRes.Payload.QANMysqlSlowlogAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANMySQLSlowlogAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
		})
	}
	for _, a := range agentsRes.Payload.QANMongodbProfilerAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANMongoDBProfilerAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
		})
	}
	for _, a := range agentsRes.Payload.QANPostgresqlPgstatementsAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANPostgreSQLPgStatementsAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
		})
	}
	for _, a := range agentsRes.Payload.QANPostgresqlPgstatmonitorAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANPostgreSQLPgStatMonitorAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status),
			Disabled:   a.Disabled,
		})
	}
	for _, a := range agentsRes.Payload.ExternalExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType: types.AgentTypeExternalExporter,
			AgentID:   a.AgentID,
			ServiceID: a.ServiceID,
			Status:    getAgentStatus(nil),
			Disabled:  a.Disabled,
			Port:      a.ListenPort,
		})
	}

	return &listAgentsResult{
		Agents: agentsList,
	}, nil
}

// register command
var (
	ListAgents  listAgentsCommand
	ListAgentsC = inventoryListC.Command("agents", "Show agents in inventory").Hide(hide)
)

func init() {
	ListAgentsC.Flag("pmm-agent-id", "Filter by pmm-agent identifier").StringVar(&ListAgents.filters.PMMAgentID)
	ListAgentsC.Flag("service-id", "Filter by Service identifier").StringVar(&ListAgents.filters.ServiceID)
	ListAgentsC.Flag("node-id", "Filter by Node identifier").StringVar(&ListAgents.filters.NodeID)
	ListAgentsC.Flag("agent-type", "Filter by Agent type").StringVar(&ListAgents.agentType)
}
