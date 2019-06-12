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

var addAgentProxysqlExporterResultT = commands.ParseTemplate(`
Proxysql Exporter added.
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

type addAgentProxysqlExporterResult struct {
	Agent *agents.AddProxySQLExporterOKBodyProxysqlExporter `json:"proxysql_exporter"`
}

func (res *addAgentProxysqlExporterResult) Result() {}

func (res *addAgentProxysqlExporterResult) String() string {
	return commands.RenderTemplate(addAgentProxysqlExporterResultT, res)
}

type addAgentProxysqlExporterCommand struct {
	PMMAgentID          string
	ServiceID           string
	Username            string
	Password            string
	CustomLabels        string
	SkipConnectionCheck bool
}

func (cmd *addAgentProxysqlExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddProxySQLExporterParams{
		Body: agents.AddProxySQLExporterBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddProxySQLExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentProxysqlExporterResult{
		Agent: resp.Payload.ProxysqlExporter,
	}, nil
}

// register command
var (
	AddAgentProxysqlExporter  = new(addAgentProxysqlExporterCommand)
	AddAgentProxysqlExporterC = addAgentC.Command("proxysql-exporter", "Add proxysql_exporter to inventory")
)

func init() {
	AddAgentProxysqlExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").StringVar(&AddAgentProxysqlExporter.PMMAgentID)
	AddAgentProxysqlExporterC.Arg("service-id", "Service identifier").StringVar(&AddAgentProxysqlExporter.ServiceID)
	AddAgentProxysqlExporterC.Arg("username", "ProxySQL username for scraping metrics").Default("admin").StringVar(&AddAgentProxysqlExporter.Username)
	AddAgentProxysqlExporterC.Flag("password", "ProxySQL password for scraping metrics").Default("admin").StringVar(&AddAgentProxysqlExporter.Password)
	AddAgentProxysqlExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentProxysqlExporter.CustomLabels)
	AddAgentProxysqlExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentProxysqlExporter.SkipConnectionCheck)
}
