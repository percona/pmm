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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"

	"github.com/percona/pmm-admin/commands"
)

var addAgentPostgresExporterResultT = commands.ParseTemplate(`
Postgres Exporter added.
Agent ID     : {{ .Agent.AgentID }}
PMM-Agent ID : {{ .Agent.PMMAgentID }}
Service ID   : {{ .Agent.ServiceID }}
Username     : {{ .Agent.Username }}
Password     : {{ .Agent.Password }}
Listen port  : {{ .Agent.ListenPort }}

Status       : {{ .Agent.Status }}
Disabled     : {{ .Agent.Disabled }}
Custom labels: {{ .Agent.CustomLabels }}
`)

type addAgentPostgresExporterResult struct {
	Agent *agents.AddPostgresExporterOKBodyPostgresExporter `json:"postgres_exporter"`
}

func (res *addAgentPostgresExporterResult) Result() {}

func (res *addAgentPostgresExporterResult) String() string {
	return commands.RenderTemplate(addAgentPostgresExporterResultT, res)
}

type addAgentPostgresExporterCommand struct {
	PMMAgentID   string
	ServiceID    string
	Username     string
	Password     string
	CustomLabels string
}

func (cmd *addAgentPostgresExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddPostgresExporterParams{
		Body: agents.AddPostgresExporterBody{
			PMMAgentID:   cmd.PMMAgentID,
			ServiceID:    cmd.ServiceID,
			Username:     cmd.Username,
			Password:     cmd.Password,
			CustomLabels: customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddPostgresExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentPostgresExporterResult{
		Agent: resp.Payload.PostgresExporter,
	}, nil
}

// register command
var (
	AddAgentPostgresExporter  = new(addAgentPostgresExporterCommand)
	AddAgentPostgresExporterC = addAgentC.Command("postgres-exporter", "Add postgres_exporter to inventory.")
)

func init() {
	AddAgentPostgresExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance.").StringVar(&AddAgentPostgresExporter.PMMAgentID)
	AddAgentPostgresExporterC.Arg("service-id", "Service identifier.").StringVar(&AddAgentPostgresExporter.ServiceID)
	AddAgentPostgresExporterC.Arg("username", "PostgreSQL username for scraping metrics.").Default("postgres").StringVar(&AddAgentPostgresExporter.Username)
	AddAgentPostgresExporterC.Flag("password", "PostgreSQL password for scraping metrics.").StringVar(&AddAgentPostgresExporter.Password)
	AddAgentPostgresExporterC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddAgentPostgresExporter.CustomLabels)
}
