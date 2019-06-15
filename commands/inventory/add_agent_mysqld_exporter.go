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

var addAgentMysqldExporterResultT = commands.ParseTemplate(`
Mysqld Exporter added.
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

type addAgentMysqldExporterResult struct {
	Agent *agents.AddMySqldExporterOKBodyMysqldExporter `json:"mysqld_exporter"`
}

func (res *addAgentMysqldExporterResult) Result() {}

func (res *addAgentMysqldExporterResult) String() string {
	return commands.RenderTemplate(addAgentMysqldExporterResultT, res)
}

type addAgentMysqldExporterCommand struct {
	PMMAgentID          string
	ServiceID           string
	Username            string
	Password            string
	CustomLabels        string
	SkipConnectionCheck bool
}

func (cmd *addAgentMysqldExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddMySqldExporterParams{
		Body: agents.AddMySqldExporterBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddMySqldExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentMysqldExporterResult{
		Agent: resp.Payload.MysqldExporter,
	}, nil
}

// register command
var (
	AddAgentMysqldExporter  = new(addAgentMysqldExporterCommand)
	AddAgentMysqldExporterC = addAgentC.Command("mysqld-exporter", "Add mysqld_exporter to inventory")
)

func init() {
	AddAgentMysqldExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").StringVar(&AddAgentMysqldExporter.PMMAgentID)
	AddAgentMysqldExporterC.Arg("service-id", "Service identifier").StringVar(&AddAgentMysqldExporter.ServiceID)
	AddAgentMysqldExporterC.Arg("username", "MySQL username for scraping metrics").Default("root").StringVar(&AddAgentMysqldExporter.Username)
	AddAgentMysqldExporterC.Flag("password", "MySQL password for scraping metrics").StringVar(&AddAgentMysqldExporter.Password)
	AddAgentMysqldExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentMysqldExporter.CustomLabels)
	AddAgentMysqldExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentMysqldExporter.SkipConnectionCheck)
}
