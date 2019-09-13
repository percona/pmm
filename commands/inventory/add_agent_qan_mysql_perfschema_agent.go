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

var addAgentQANMySQLPerfSchemaAgentResultT = commands.ParseTemplate(`
QAN MySQL perf schema agent added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Password              : {{ .Agent.Password }}
Query examples        : {{ .QueryExamples }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentQANMySQLPerfSchemaAgentResult struct {
	Agent *agents.AddQANMySQLPerfSchemaAgentOKBodyQANMysqlPerfschemaAgent `json:"qan_mysql_perfschema_agent"`
}

func (res *addAgentQANMySQLPerfSchemaAgentResult) Result() {}

func (res *addAgentQANMySQLPerfSchemaAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANMySQLPerfSchemaAgentResultT, res)
}

func (res *addAgentQANMySQLPerfSchemaAgentResult) QueryExamples() string {
	if res.Agent.QueryExamplesDisabled {
		return "disabled"
	}
	return "enabled"
}

type addAgentQANMySQLPerfSchemaAgentCommand struct {
	PMMAgentID           string
	ServiceID            string
	Username             string
	Password             string
	CustomLabels         string
	SkipConnectionCheck  bool
	DisableQueryExamples bool
	TLS                  bool
	TLSSkipVerify        bool
}

func (cmd *addAgentQANMySQLPerfSchemaAgentCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddQANMySQLPerfSchemaAgentParams{
		Body: agents.AddQANMySQLPerfSchemaAgentBody{
			PMMAgentID:           cmd.PMMAgentID,
			ServiceID:            cmd.ServiceID,
			Username:             cmd.Username,
			Password:             cmd.Password,
			CustomLabels:         customLabels,
			SkipConnectionCheck:  cmd.SkipConnectionCheck,
			DisableQueryExamples: cmd.DisableQueryExamples,
			TLS:                  cmd.TLS,
			TLSSkipVerify:        cmd.TLSSkipVerify,
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
	AddAgentQANMySQLPerfSchemaAgentC = addAgentC.Command("qan-mysql-perfschema-agent", "add QAN MySQL perf schema agent to inventory").Hide(hide)
)

func init() {
	AddAgentQANMySQLPerfSchemaAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").StringVar(&AddAgentQANMySQLPerfSchemaAgent.PMMAgentID)
	AddAgentQANMySQLPerfSchemaAgentC.Arg("service-id", "Service identifier").StringVar(&AddAgentQANMySQLPerfSchemaAgent.ServiceID)
	AddAgentQANMySQLPerfSchemaAgentC.Arg("username", "MySQL username for scraping metrics").Default("root").StringVar(&AddAgentQANMySQLPerfSchemaAgent.Username)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("password", "MySQL password for scraping metrics").StringVar(&AddAgentQANMySQLPerfSchemaAgent.Password)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentQANMySQLPerfSchemaAgent.CustomLabels)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentQANMySQLPerfSchemaAgent.SkipConnectionCheck)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("disable-queryexamples", "Disable collection of query examples").BoolVar(&AddAgentQANMySQLPerfSchemaAgent.DisableQueryExamples)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddAgentQANMySQLPerfSchemaAgent.TLS)
	AddAgentQANMySQLPerfSchemaAgentC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddAgentQANMySQLPerfSchemaAgent.TLSSkipVerify)
}
