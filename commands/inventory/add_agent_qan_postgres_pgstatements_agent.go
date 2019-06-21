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

var addAgentQANPostgreSQLPgStatementsAgentResultT = commands.ParseTemplate(`
PostgreSQL QAN Pg Stat Statements Agent added.
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

type addAgentQANPostgreSQLPgStatementsAgentResult struct {
	Agent *agents.AddQANPostgreSQLPgStatementsAgentOKBodyQANPostgresqlPgstatementsAgent `json:"qan_postgresql_pgstatements_agent"`
}

func (res *addAgentQANPostgreSQLPgStatementsAgentResult) Result() {}

func (res *addAgentQANPostgreSQLPgStatementsAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANPostgreSQLPgStatementsAgentResultT, res)
}

type addAgentQANPostgreSQLPgStatementsAgentCommand struct {
	PMMAgentID          string
	ServiceID           string
	Username            string
	Password            string
	CustomLabels        string
	SkipConnectionCheck bool
}

func (cmd *addAgentQANPostgreSQLPgStatementsAgentCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddQANPostgreSQLPgStatementsAgentParams{
		Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddQANPostgreSQLPgStatementsAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentQANPostgreSQLPgStatementsAgentResult{
		Agent: resp.Payload.QANPostgresqlPgstatementsAgent,
	}, nil
}

// register command
var (
	AddAgentQANPostgreSQLPgStatementsAgent  = new(addAgentQANPostgreSQLPgStatementsAgentCommand)
	AddAgentQANPostgreSQLPgStatementsAgentC = addAgentC.Command("qan-postgresql-pgstatements-agent", "Add QAN PostgreSQL Stat Statements Agent to inventory")
)

func init() {
	AddAgentQANPostgreSQLPgStatementsAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").StringVar(&AddAgentQANPostgreSQLPgStatementsAgent.PMMAgentID)
	AddAgentQANPostgreSQLPgStatementsAgentC.Arg("service-id", "Service identifier").StringVar(&AddAgentQANPostgreSQLPgStatementsAgent.ServiceID)
	AddAgentQANPostgreSQLPgStatementsAgentC.Arg("username", "PostgreSQL username for QAN agent").Default("postgres").StringVar(&AddAgentQANPostgreSQLPgStatementsAgent.Username)
	AddAgentQANPostgreSQLPgStatementsAgentC.Flag("password", "PostgreSQL password for QAN agent").StringVar(&AddAgentQANPostgreSQLPgStatementsAgent.Password)
	AddAgentQANPostgreSQLPgStatementsAgentC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentQANPostgreSQLPgStatementsAgent.CustomLabels)
	AddAgentQANPostgreSQLPgStatementsAgentC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentQANPostgreSQLPgStatementsAgent.SkipConnectionCheck)
}
