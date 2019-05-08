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

var addAgentMongodbExporterResultT = commands.ParseTemplate(`
MongoDB Exporter added.
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

type addAgentMongodbExporterResult struct {
	Agent *agents.AddMongoDBExporterOKBodyMongodbExporter `json:"mongodb_exporter"`
}

func (res *addAgentMongodbExporterResult) Result() {}

func (res *addAgentMongodbExporterResult) String() string {
	return commands.RenderTemplate(addAgentMongodbExporterResultT, res)
}

type addAgentMongodbExporterCommand struct {
	PMMAgentID   string
	ServiceID    string
	Username     string
	Password     string
	CustomLabels string
}

func (cmd *addAgentMongodbExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddMongoDBExporterParams{
		Body: agents.AddMongoDBExporterBody{
			PMMAgentID:   cmd.PMMAgentID,
			ServiceID:    cmd.ServiceID,
			Username:     cmd.Username,
			Password:     cmd.Password,
			CustomLabels: customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddMongoDBExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentMongodbExporterResult{
		Agent: resp.Payload.MongodbExporter,
	}, nil
}

// register command
var (
	AddAgentMongodbExporter  = new(addAgentMongodbExporterCommand)
	AddAgentMongodbExporterC = addAgentC.Command("mongodb-exporter", "Add mongodb_exporter to inventory.")
)

func init() {
	AddAgentMongodbExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance.").StringVar(&AddAgentMongodbExporter.PMMAgentID)
	AddAgentMongodbExporterC.Arg("service-id", "Service identifier.").StringVar(&AddAgentMongodbExporter.ServiceID)
	AddAgentMongodbExporterC.Arg("username", "MongoDB username for scraping metrics.").StringVar(&AddAgentMongodbExporter.Username)
	AddAgentMongodbExporterC.Flag("password", "MongoDB password for scraping metrics.").StringVar(&AddAgentMongodbExporter.Password)
	AddAgentMongodbExporterC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddAgentMongodbExporter.CustomLabels)
}
