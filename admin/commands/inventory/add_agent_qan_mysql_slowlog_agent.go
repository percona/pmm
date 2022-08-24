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

type addAgentQANMySQLSlowlogAgentCommand struct {
	PMMAgentID           string
	ServiceID            string
	Username             string
	Password             string
	CustomLabels         string
	SkipConnectionCheck  bool
	DisableQueryExamples bool
	MaxSlowlogFileSize   units.Base2Bytes
	TLS                  bool
	TLSSkipVerify        bool
	TLSCaFile            string
	TLSCertFile          string
	TLSKeyFile           string
}

func (cmd *addAgentQANMySQLSlowlogAgentCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	var tlsCa, tlsCert, tlsKey string
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCaFile)
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

	params := &agents.AddQANMySQLSlowlogAgentParams{
		Body: agents.AddQANMySQLSlowlogAgentBody{
			PMMAgentID:           cmd.PMMAgentID,
			ServiceID:            cmd.ServiceID,
			Username:             cmd.Username,
			Password:             cmd.Password,
			CustomLabels:         customLabels,
			SkipConnectionCheck:  cmd.SkipConnectionCheck,
			DisableQueryExamples: cmd.DisableQueryExamples,
			MaxSlowlogFileSize:   strconv.FormatInt(int64(cmd.MaxSlowlogFileSize), 10),
			TLS:                  cmd.TLS,
			TLSSkipVerify:        cmd.TLSSkipVerify,
			TLSCa:                tlsCa,
			TLSCert:              tlsCert,
			TLSKey:               tlsKey,
			LogLevel:             &addExporterLogLevel,
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
	AddAgentQANMySQLSlowlogAgent  addAgentQANMySQLSlowlogAgentCommand
	AddAgentQANMySQLSlowlogAgentC = addAgentC.Command("qan-mysql-slowlog-agent", "add QAN MySQL slowlog agent to inventory").Hide(hide)
)

func init() {
	AddAgentQANMySQLSlowlogAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentQANMySQLSlowlogAgent.PMMAgentID)
	AddAgentQANMySQLSlowlogAgentC.Arg("service-id", "Service identifier").Required().StringVar(&AddAgentQANMySQLSlowlogAgent.ServiceID)
	AddAgentQANMySQLSlowlogAgentC.Arg("username", "MySQL username for scraping metrics").Default("root").StringVar(&AddAgentQANMySQLSlowlogAgent.Username)
	AddAgentQANMySQLSlowlogAgentC.Flag("password", "MySQL password for scraping metrics").StringVar(&AddAgentQANMySQLSlowlogAgent.Password)
	AddAgentQANMySQLSlowlogAgentC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentQANMySQLSlowlogAgent.CustomLabels)
	AddAgentQANMySQLSlowlogAgentC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentQANMySQLSlowlogAgent.SkipConnectionCheck)
	AddAgentQANMySQLSlowlogAgentC.Flag("disable-queryexamples", "Disable collection of query examples").BoolVar(&AddAgentQANMySQLSlowlogAgent.DisableQueryExamples)
	AddAgentQANMySQLSlowlogAgentC.Flag("size-slow-logs", "Rotate slow log file at this size (default: 0; 0 or negative value disables rotation). Ex.: 1GiB").
		BytesVar(&AddAgentQANMySQLSlowlogAgent.MaxSlowlogFileSize)
	AddAgentQANMySQLSlowlogAgentC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddAgentQANMySQLSlowlogAgent.TLS)
	AddAgentQANMySQLSlowlogAgentC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddAgentQANMySQLSlowlogAgent.TLSSkipVerify)
	AddAgentQANMySQLSlowlogAgentC.Flag("tls-ca", "Path to certificate authority certificate file").StringVar(&AddAgentQANMySQLSlowlogAgent.TLSCaFile)
	AddAgentQANMySQLSlowlogAgentC.Flag("tls-cert", "Path to client certificate file").StringVar(&AddAgentQANMySQLSlowlogAgent.TLSCertFile)
	AddAgentQANMySQLSlowlogAgentC.Flag("tls-key", "Path to client key file").StringVar(&AddAgentQANMySQLSlowlogAgent.TLSKeyFile)
	addExporterGlobalFlags(AddAgentQANMySQLSlowlogAgentC, false)
}
