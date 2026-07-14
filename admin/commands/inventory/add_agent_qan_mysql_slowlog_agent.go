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
	"strconv"

	"github.com/alecthomas/units"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var addAgentQANMySQLSlowlogAgentResultT = commands.ParseTemplate(`
QAN MySQL slowlog agent added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Query examples        : {{ .QueryExamples }}
Slowlog rotation      : {{ .SlowlogRotation }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentQANMySQLSlowlogAgentResult struct {
	Agent *agents.AddQANMySQLSlowlogAgentOKBodyQANMysqlSlowlogAgent `json:"qan_mysql_slowlog_agent"`
}

func (res *addAgentQANMySQLSlowlogAgentResult) Result() {}

func (res *addAgentQANMySQLSlowlogAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANMySQLSlowlogAgentResultT, res)
}

func (res *addAgentQANMySQLSlowlogAgentResult) QueryExamples() string {
	if res.Agent.QueryExamplesDisabled {
		return "disabled"
	}
	return "enabled"
}

func (res *addAgentQANMySQLSlowlogAgentResult) SlowlogRotation() string {
	// TODO units.ParseBase2Bytes, etc
	return res.Agent.MaxSlowlogFileSize
}

// AddAgentQANMySQLSlowlogAgentCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddAgentQANMySQLSlowlogAgentCommand struct {
	PMMAgentID           string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID            string            `arg:"" help:"Service identifier"`
	Username             string            `arg:"" optional:"" help:"MySQL username for scraping metrics"`
	Password             string            `help:"MySQL password for scraping metrics"`
	CustomLabels         map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool              `help:"Skip connection check"`
	CommentsParsing      string            `enum:"on,off" default:"off" help:"Enable/disable parsing comments from queries. One of: [on, off]"`
	MaxQueryLength       int32             `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`
	DisableQueryExamples bool              `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MaxSlowlogFileSize   units.Base2Bytes  `name:"size-slow-logs" placeholder:"size" help:"Rotate slow log file at this size (default: 0; 0 or negative value disables rotation). Ex.: 1GiB"`
	TLS                  bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify        bool              `help:"Skip TLS certificates validation"`
	TLSCAFile            string            `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile          string            `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile           string            `name:"tls-key" help:"Path to client key file"`
	LogLevel             string            `enum:"debug,info,warn,error,fatal" default:"warn" help:"Service logging level. One of: [debug, info, warn, error, fatal]"`
}

// RunCmd executes the AddAgentQANMySQLSlowlogAgentCommand and returns the result.
func (cmd *AddAgentQANMySQLSlowlogAgentCommand) RunCmd() (commands.Result, error) {
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

	params := &agents.AddQANMySQLSlowlogAgentParams{
		Body: agents.AddQANMySQLSlowlogAgentBody{
			PMMAgentID:             cmd.PMMAgentID,
			ServiceID:              cmd.ServiceID,
			Username:               cmd.Username,
			Password:               cmd.Password,
			CustomLabels:           customLabels,
			SkipConnectionCheck:    cmd.SkipConnectionCheck,
			DisableCommentsParsing: disableCommentsParsing,
			MaxQueryLength:         cmd.MaxQueryLength,
			DisableQueryExamples:   cmd.DisableQueryExamples,
			MaxSlowlogFileSize:     strconv.FormatInt(int64(cmd.MaxSlowlogFileSize), 10),
			TLS:                    cmd.TLS,
			TLSSkipVerify:          cmd.TLSSkipVerify,
			TLSCa:                  tlsCa,
			TLSCert:                tlsCert,
			TLSKey:                 tlsKey,
			LogLevel:               &cmd.LogLevel,
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
