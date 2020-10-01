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

var addAgentQANPostgreSQLPgStatMonitorAgentResultT = commands.ParseTemplate(`
PostgreSQL QAN Pg Stat Monitor Agent added.
Agent ID             : {{ .Agent.AgentID }}
PMM-Agent ID         : {{ .Agent.PMMAgentID }}
Service ID           : {{ .Agent.ServiceID }}
Username             : {{ .Agent.Username }}
TLS enabled          : {{ .Agent.TLS }}
Skip TLS verification: {{ .Agent.TLSSkipVerify }}

Status               : {{ .Agent.Status }}
Disabled             : {{ .Agent.Disabled }}
Custom labels        : {{ .Agent.CustomLabels }}
Query examples       : {{ .Agent.QueryExamplesDisabled }}
`)

type addAgentQANPostgreSQLPgStatMonitorAgentResult struct {
	Agent *agents.AddQANPostgreSQLPgStatMonitorAgentOKBodyQANPostgresqlPgstatmonitorAgent `json:"qan_postgresql_pgstatmonitor_agent"`
}

func (res *addAgentQANPostgreSQLPgStatMonitorAgentResult) Result() {}

func (res *addAgentQANPostgreSQLPgStatMonitorAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANPostgreSQLPgStatMonitorAgentResultT, res)
}

type addAgentQANPostgreSQLPgStatMonitorAgentCommand struct {
	PMMAgentID            string
	ServiceID             string
	Username              string
	Password              string
	CustomLabels          string
	SkipConnectionCheck   bool
	TLS                   bool
	TLSSkipVerify         bool
	QueryExamplesDisabled bool
}

func (cmd *addAgentQANPostgreSQLPgStatMonitorAgentCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddQANPostgreSQLPgStatMonitorAgentParams{
		Body: agents.AddQANPostgreSQLPgStatMonitorAgentBody{
			PMMAgentID:           cmd.PMMAgentID,
			ServiceID:            cmd.ServiceID,
			Username:             cmd.Username,
			Password:             cmd.Password,
			CustomLabels:         customLabels,
			SkipConnectionCheck:  cmd.SkipConnectionCheck,
			TLS:                  cmd.TLS,
			TLSSkipVerify:        cmd.TLSSkipVerify,
			DisableQueryExamples: cmd.QueryExamplesDisabled,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddQANPostgreSQLPgStatMonitorAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentQANPostgreSQLPgStatMonitorAgentResult{
		Agent: resp.Payload.QANPostgresqlPgstatmonitorAgent,
	}, nil
}

// register command
var (
	AddAgentQANPostgreSQLPgStatMonitorAgent  = new(addAgentQANPostgreSQLPgStatMonitorAgentCommand)
	AddAgentQANPostgreSQLPgStatMonitorAgentC = addAgentC.Command("qan-postgresql-pgstatmonitor-agent", "Add QAN PostgreSQL Stat Monitor Agent to inventory").Hide(hide)
)

func init() {
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.PMMAgentID)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Arg("service-id", "Service identifier").Required().StringVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.ServiceID)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Arg("username", "PostgreSQL username for QAN agent").Default("postgres").StringVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.Username)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Flag("password", "PostgreSQL password for QAN agent").StringVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.Password)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.CustomLabels)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.SkipConnectionCheck)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.TLS)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.TLSSkipVerify)
	AddAgentQANPostgreSQLPgStatMonitorAgentC.Flag("disable-queryexamples", "Disable collection of query examples").BoolVar(&AddAgentQANPostgreSQLPgStatMonitorAgent.QueryExamplesDisabled)
}
