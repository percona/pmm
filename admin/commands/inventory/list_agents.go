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

package inventory

import (
	"strings"

	"github.com/AlekSi/pointer"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

//nolint:lll
var listAgentsResultT = commands.ParseTemplate(`
Agents list.

{{ printf "%-29s" "Agent type" }} {{ printf "%-15s" "Status" }} {{ printf "%-39s" "Agent ID" }} {{ printf "%-39s" "PMM-Agent ID" }} {{ printf "%-38s" "Service ID" }} {{ printf "%-20s" "Port" }}
{{ range .Agents }}
{{- printf "%-29s" .HumanReadableAgentType }} {{ printf "%-15s" .NiceAgentStatus }} {{ printf "%-38s" .AgentID }}  {{ printf "%-38s" .PMMAgentID }}  {{ printf "%-38s" .ServiceID }} {{ .Port }}
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
	res := cases.Title(language.English).String(strings.ToLower(a.Status))
	if a.Disabled {
		res += " (disabled)"
	}
	return res
}

func (res *listAgentsResult) Result() {}

func (res *listAgentsResult) String() string {
	return commands.RenderTemplate(listAgentsResultT, res)
}

// This is used in the json output. By convention, statuses must be in uppercase.
func getAgentStatus(status *string) string {
	res := pointer.GetString(status)
	if res == "" {
		res = "UNKNOWN"
	}
	res, _ = strings.CutPrefix(res, "AGENT_STATUS_")
	return res
}

// ListAgentsCommand is used by Kong for CLI flags and commands.
type ListAgentsCommand struct {
	PMMAgentID string `help:"Filter by pmm-agent identifier"`
	ServiceID  string `help:"Filter by Service identifier"`
	NodeID     string `help:"Filter by Node identifier"`
	AgentType  string `help:"Filter by Agent type"`
}

// RunCmd executes the ListAgentsCommand and returns the result.
func (cmd *ListAgentsCommand) RunCmd() (commands.Result, error) {
	agentType, err := formatTypeValue(acceptableAgentTypes, cmd.AgentType)
	if err != nil {
		return nil, err
	}

	params := &agents.ListAgentsParams{
		PMMAgentID: pointer.ToString(cmd.PMMAgentID),
		ServiceID:  pointer.ToString(cmd.ServiceID),
		NodeID:     pointer.ToString(cmd.NodeID),
		AgentType:  agentType,
		Context:    commands.Ctx,
	}
	agentsRes, err := client.Default.AgentsService.ListAgents(params)
	if err != nil {
		return nil, err
	}

	var agentsList []listResultAgent //nolint:prealloc
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
