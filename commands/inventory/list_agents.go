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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/types"

	"github.com/percona/pmm-admin/commands"
)

var listAgentsResultT = commands.ParseTemplate(`
Agents list.

{{ printf "%-27s" "Agent type" }} {{ printf "%-15s" "Status" }} {{ printf "%-47s" "Agent ID" }} {{ printf "%-47s" "PMM-Agent ID" }} {{ printf "%-47s" "Service ID" }}
{{ range .Agents }}
{{- printf "%-27s" .HumanReadableAgentType }} {{ printf "%-15s" .Status }} {{ .AgentID }}  {{ .PMMAgentID }}  {{ .ServiceID }}
{{ end }}
`)

type listResultAgent struct {
	AgentType  string `json:"agent_type"`
	AgentID    string `json:"agent_id"`
	PMMAgentID string `json:"pmm_agent_id"`
	ServiceID  string `json:"service_id"`
	Status     string `json:"status"`
}

func (a listResultAgent) HumanReadableAgentType() string {
	return types.AgentTypeName(a.AgentType)
}

type listAgentsResult struct {
	Agents []listResultAgent `json:"agents"`
}

func (res *listAgentsResult) Result() {}

func (res *listAgentsResult) String() string {
	return commands.RenderTemplate(listAgentsResultT, res)
}

type listAgentsCommand struct {
}

func getAgentStatus(s *string, disabled bool) string {
	res := strings.ToLower(pointer.GetString(s))
	if res == "" {
		res = "unknown"
	}
	if disabled {
		res += " (disabled)"
	}
	return res
}

func (cmd *listAgentsCommand) Run() (commands.Result, error) {
	params := &agents.ListAgentsParams{
		Context: commands.Ctx,
	}
	agentsRes, err := client.Default.Agents.ListAgents(params)
	if err != nil {
		return nil, err
	}

	var agentsList []listResultAgent
	// Contanst values set to AgentType should be from api/inventorypb/types/agent_types.go.
	// We use hardcoded constants to avoid big dependencies.
	for _, a := range agentsRes.Payload.PMMAgent {
		status := "disconnected"
		if a.Connected {
			status = "connected"
		}
		agentsList = append(agentsList, listResultAgent{
			AgentType: types.AgentTypePMMAgent,
			AgentID:   a.AgentID,
			Status:    status,
		})
	}
	for _, a := range agentsRes.Payload.NodeExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeNodeExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.MysqldExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeMySQLdExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.MongodbExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeMongoDBExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.PostgresExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypePostgresExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.ProxysqlExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeProxySQLExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.RDSExporter {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeRDSExporter,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANMysqlPerfschemaAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANMySQLPerfSchemaAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANMysqlSlowlogAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANMySQLSlowlogAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANMongodbProfilerAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANMongoDBProfilerAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANPostgresqlPgstatementsAgent {
		agentsList = append(agentsList, listResultAgent{
			AgentType:  types.AgentTypeQANPostgreSQLPgStatementsAgent,
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}

	return &listAgentsResult{
		Agents: agentsList,
	}, nil
}

// register command
var (
	ListAgents  = new(listAgentsCommand)
	ListAgentsC = inventoryListC.Command("agents", "Show agents in inventory").Hide(hide)
)
