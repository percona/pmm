// pmm-admin
// Copyright (C) 2018 Percona LLC
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
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"

	"github.com/percona/pmm-admin/commands"
)

var listAgentsResultT = commands.ParseTemplate(`
Agents list.

{{ printf "%-27s" "Agent type" }} {{ printf "%-15s" "Status" }} {{ printf "%-47s" "Agent ID" }} {{ printf "%-47s" "PMM-Agent ID" }} {{ printf "%-47s" "Service ID" }}
{{ range .Agents }}
{{- printf "%-27s" .AgentType }} {{ printf "%-15s" .Status }} {{ .AgentID }}  {{ .PMMAgentID }}  {{ .ServiceID }}
{{ end }}
`)

type listResultAgent struct {
	AgentType  string `json:"agent_type"`
	AgentID    string `json:"agent_id"`
	PMMAgentID string `json:"pmm_agent_id"`
	ServiceID  string `json:"service_id"`
	Status     string `json:"status"`
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

	var agents []listResultAgent
	for _, a := range agentsRes.Payload.PMMAgent {
		status := "disconnected"
		if a.Connected {
			status = "connected"
		}
		agents = append(agents, listResultAgent{
			AgentType: "pmm-agent",
			AgentID:   a.AgentID,
			Status:    status,
		})
	}
	for _, a := range agentsRes.Payload.NodeExporter {
		agents = append(agents, listResultAgent{
			AgentType:  "node_exporter",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.MysqldExporter {
		agents = append(agents, listResultAgent{
			AgentType:  "mysqld_exporter",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.MongodbExporter {
		agents = append(agents, listResultAgent{
			AgentType:  "mongodb_exporter",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.PostgresExporter {
		agents = append(agents, listResultAgent{
			AgentType:  "postgres_exporter",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANMysqlPerfschemaAgent {
		agents = append(agents, listResultAgent{
			AgentType:  "qan-mysql-perfschema-agent",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANMysqlSlowlogAgent {
		agents = append(agents, listResultAgent{
			AgentType:  "qan-mysql-slowlog-agent",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANMongodbProfilerAgent {
		agents = append(agents, listResultAgent{
			AgentType:  "qan-mongodb-profiler-agent",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.ProxysqlExporter {
		agents = append(agents, listResultAgent{
			AgentType:  "proxysql_exporter",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}
	for _, a := range agentsRes.Payload.QANPostgresqlPgstatementsAgent {
		agents = append(agents, listResultAgent{
			AgentType:  "qan-postgresql-pgstatements-agent",
			AgentID:    a.AgentID,
			PMMAgentID: a.PMMAgentID,
			ServiceID:  a.ServiceID,
			Status:     getAgentStatus(a.Status, a.Disabled),
		})
	}

	return &listAgentsResult{
		Agents: agents,
	}, nil
}

// register command
var (
	ListAgents  = new(listAgentsCommand)
	ListAgentsC = inventoryListC.Command("agents", "Show agents in inventory")
)
