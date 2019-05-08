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

var addAgentQANMySQLPerfSchemaAgentResultT = commands.ParseTemplate(`
QAN MySQL perf schema agent added.
Agent ID     : {{ .Agent.AgentID }}
PMM-Agent ID : {{ .Agent.PMMAgentID }}
Service ID   : {{ .Agent.ServiceID }}
Username     : {{ .Agent.Username }}
Password     : {{ .Agent.Password }}

Status       : {{ .Agent.Status }}
Disabled     : {{ .Agent.Disabled }}
Custom labels: {{ .Agent.CustomLabels }}
`)

type addAgentQANMySQLPerfSchemaAgentResult struct {
	Agent *agents.AddQANMySQLPerfSchemaAgentOKBodyQANMysqlPerfschemaAgent `json:"qan_mysql_perfschema_agent"`
}

func (res *addAgentQANMySQLPerfSchemaAgentResult) Result() {}

func (res *addAgentQANMySQLPerfSchemaAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANMySQLPerfSchemaAgentResultT, res)
}

type addAgentQANMySQLPerfSchemaAgentCommand struct {
	PMMAgentID   string
	ServiceID    string
	Username     string
	Password     string
	CustomLabels string
}

func (cmd *addAgentQANMySQLPerfSchemaAgentCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddQANMySQLPerfSchemaAgentParams{
		Body: agents.AddQANMySQLPerfSchemaAgentBody{
			PMMAgentID:   cmd.PMMAgentID,
			ServiceID:    cmd.ServiceID,
			Username:     cmd.Username,
			Password:     cmd.Password,
			CustomLabels: customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddQANMySQLPerfSchemaAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentQANMySQLPerfSchemaAgentResult{
		Agent: resp.Payload.QANMysqlPerfschemaAgent,
	}, nil
}

// register command
var (
	AddAgentQANMySQLPerfSchemaAgent  = new(addAgentQANMySQLPerfSchemaAgentCommand)
	AddAgentQANMySQLPerfSchemaAgentC = addAgentC.Command("qan-mysql-perfschema-agent", "add QAN MySQL perf schema agent to inventory.")
)

func init() {
	AddAgentQANMySQLPerfSchemaAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance.").StringVar(&AddAgentQANMySQLPerfSchemaAgent.PMMAgentID)
	AddAgentQANMySQLPerfSchemaAgentC.Arg("service-id", "Service identifier.").StringVar(&AddAgentQANMySQLPerfSchemaAgent.ServiceID)
	AddAgentQANMySQLPerfSchemaAgentC.Arg("username", "MySQL username for scraping metrics.").Default("root").StringVar(&AddAgentQANMySQLPerfSchemaAgent.Username)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("password", "MySQL password for scraping metrics.").StringVar(&AddAgentQANMySQLPerfSchemaAgent.Password)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddAgentQANMySQLPerfSchemaAgent.CustomLabels)
}
