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
	PMMAgentID          string
	ServiceID           string
	Username            string
	Password            string
	CustomLabels        string
	SkipConnectionCheck bool
}

func (cmd *addAgentPostgresExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddPostgresExporterParams{
		Body: agents.AddPostgresExporterBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
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
	AddAgentPostgresExporterC = addAgentC.Command("postgres-exporter", "Add postgres_exporter to inventory")
)

func init() {
	AddAgentPostgresExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").StringVar(&AddAgentPostgresExporter.PMMAgentID)
	AddAgentPostgresExporterC.Arg("service-id", "Service identifier").StringVar(&AddAgentPostgresExporter.ServiceID)
	AddAgentPostgresExporterC.Arg("username", "PostgreSQL username for scraping metrics").Default("postgres").StringVar(&AddAgentPostgresExporter.Username)
	AddAgentPostgresExporterC.Flag("password", "PostgreSQL password for scraping metrics").StringVar(&AddAgentPostgresExporter.Password)
	AddAgentPostgresExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentPostgresExporter.CustomLabels)
	AddAgentPostgresExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentPostgresExporter.SkipConnectionCheck)
}
