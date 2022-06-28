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
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var addAgentQANPostgreSQLPgStatementsAgentResultT = commands.ParseTemplate(`
PostgreSQL QAN Pg Stat Statements Agent added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentQANPostgreSQLPgStatementsAgentResult struct {
	Agent *agents.AddQANPostgreSQLPgStatementsAgentOKBodyQANPostgresqlPgstatementsAgent `json:"qan_postgresql_pgstatements_agent"`
}

func (res *addAgentQANPostgreSQLPgStatementsAgentResult) Result() {}

func (res *addAgentQANPostgreSQLPgStatementsAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANPostgreSQLPgStatementsAgentResultT, res)
}

func (cmd *AddQANPostgreSQLPgStatementsAgentCommand) RunCmd() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	var tlsCa, tlsCert, tlsKey string
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCAFile)
		if err != nil {
			return nil, err
		}

		tlsCert, err = commands.ReadFile(cmd.TLSCertFile)
		if err != nil {
			return nil, err
		}

		tlsKey, err = commands.ReadFile(cmd.TLSKeyFile)
		if err != nil {
			return nil, err
		}
	}

	params := &agents.AddQANPostgreSQLPgStatementsAgentParams{
		Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,

			TLS:           cmd.TLS,
			TLSSkipVerify: cmd.TLSSkipVerify,
			TLSCa:         tlsCa,
			TLSCert:       tlsCert,
			TLSKey:        tlsKey,
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
