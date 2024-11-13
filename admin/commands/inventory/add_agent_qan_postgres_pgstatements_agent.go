// Copyright (C) 2023 Percona LLC
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

// AddAgentQANPostgreSQLPgStatementsAgentCommand is used by Kong for CLI flags and commands.
type AddAgentQANPostgreSQLPgStatementsAgentCommand struct {
	PMMAgentID          string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string            `arg:"" help:"Service identifier"`
	Username            string            `arg:"" optional:"" help:"PostgreSQL username for QAN agent"`
	Password            string            `help:"PostgreSQL password for QAN agent"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	CommentsParsing     string            `enum:"on,off" default:"off" help:"Enable/disable parsing comments from queries. One of: [on, off]"`
	MaxQueryLength      int32             `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
	TLSCAFile           string            `name:"tls-ca-file" help:"TLS CA certificate file"`
	TLSCertFile         string            `help:"TLS certificate file"`
	TLSKeyFile          string            `help:"TLS certificate key file"`
	LogLevel            string            `enum:"debug,info,warn,error,fatal" default:"warn" help:"Service logging level. One of: [debug, info, warn, error, fatal]"`
}

// RunCmd executes the AddAgentQANPostgreSQLPgStatementsAgentCommand and returns the result.
func (cmd *AddAgentQANPostgreSQLPgStatementsAgentCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	var (
		err                    error
		tlsCa, tlsCert, tlsKey string
	)
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

	disableCommentsParsing := true
	if cmd.CommentsParsing == "on" {
		disableCommentsParsing = false
	}

	params := &agents.AddQANPostgreSQLPgStatementsAgentParams{
		Body: agents.AddQANPostgreSQLPgStatementsAgentBody{
			PMMAgentID:             cmd.PMMAgentID,
			ServiceID:              cmd.ServiceID,
			Username:               cmd.Username,
			Password:               cmd.Password,
			CustomLabels:           customLabels,
			SkipConnectionCheck:    cmd.SkipConnectionCheck,
			DisableCommentsParsing: disableCommentsParsing,
			MaxQueryLength:         cmd.MaxQueryLength,

			TLS:           cmd.TLS,
			TLSSkipVerify: cmd.TLSSkipVerify,
			TLSCa:         tlsCa,
			TLSCert:       tlsCert,
			TLSKey:        tlsKey,
			LogLevel:      &cmd.LogLevel,
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
