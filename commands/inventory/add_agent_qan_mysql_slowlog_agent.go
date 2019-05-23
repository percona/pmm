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

var addAgentQANMySQLSlowlogAgentResultT = commands.ParseTemplate(`
QAN MySQL slowlog agent added.
Agent ID     : {{ .Agent.AgentID }}
PMM-Agent ID : {{ .Agent.PMMAgentID }}
Service ID   : {{ .Agent.ServiceID }}
Username     : {{ .Agent.Username }}
Password     : {{ .Agent.Password }}

Status       : {{ .Agent.Status }}
Disabled     : {{ .Agent.Disabled }}
Custom labels: {{ .Agent.CustomLabels }}
`)

type addAgentQANMySQLSlowlogAgentResult struct {
	Agent *agents.AddQANMySQLSlowlogAgentOKBodyQANMysqlSlowlogAgent `json:"qan_mysql_slowlog_agent"`
}

func (res *addAgentQANMySQLSlowlogAgentResult) Result() {}

func (res *addAgentQANMySQLSlowlogAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANMySQLSlowlogAgentResultT, res)
}

type addAgentQANMySQLSlowlogAgentCommand struct {
	PMMAgentID          string
	ServiceID           string
	Username            string
	Password            string
	CustomLabels        string
	SkipConnectionCheck bool
}

func (cmd *addAgentQANMySQLSlowlogAgentCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddQANMySQLSlowlogAgentParams{
		Body: agents.AddQANMySQLSlowlogAgentBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddQANMySQLSlowlogAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentQANMySQLSlowlogAgentResult{
		Agent: resp.Payload.QANMysqlSlowlogAgent,
	}, nil
}

// register command
var (
	AddAgentQANMySQLSlowlogAgent  = new(addAgentQANMySQLSlowlogAgentCommand)
	AddAgentQANMySQLSlowlogAgentC = addAgentC.Command("qan-mysql-slowlog-agent", "add QAN MySQL slowlog agent to inventory.")
)

func init() {
	AddAgentQANMySQLSlowlogAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance.").StringVar(&AddAgentQANMySQLSlowlogAgent.PMMAgentID)
	AddAgentQANMySQLSlowlogAgentC.Arg("service-id", "Service identifier.").StringVar(&AddAgentQANMySQLSlowlogAgent.ServiceID)
	AddAgentQANMySQLSlowlogAgentC.Arg("username", "MySQL username for scraping metrics.").Default("root").StringVar(&AddAgentQANMySQLSlowlogAgent.Username)
	AddAgentQANMySQLSlowlogAgentC.Flag("password", "MySQL password for scraping metrics.").StringVar(&AddAgentQANMySQLSlowlogAgent.Password)
	AddAgentQANMySQLSlowlogAgentC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddAgentQANMySQLSlowlogAgent.CustomLabels)
	AddAgentQANMySQLSlowlogAgentC.Flag("skip-connection-check", "Skip connection check.").BoolVar(&AddAgentQANMySQLSlowlogAgent.SkipConnectionCheck)
}
