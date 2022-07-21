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
	"fmt"
	"strconv"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var addAgentMysqldExporterResultT = commands.ParseTemplate(`
Mysqld Exporter added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Listen port           : {{ .Agent.ListenPort }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}

Tablestat collectors  : {{ .TablestatStatus }}
`)

type addAgentMysqldExporterResult struct {
	Agent      *agents.AddMySQLdExporterOKBodyMysqldExporter `json:"mysqld_exporter"`
	TableCount int32                                         `json:"table_count,omitempty"`
}

func (res *addAgentMysqldExporterResult) Result() {}

func (res *addAgentMysqldExporterResult) String() string {
	return commands.RenderTemplate(addAgentMysqldExporterResultT, res)
}

func (res *addAgentMysqldExporterResult) TablestatStatus() string {
	if res.Agent == nil {
		return ""
	}

	s := "enabled"
	if res.Agent.TablestatsGroupDisabled {
		s = "disabled"
	}

	switch {
	case res.Agent.TablestatsGroupTableLimit == 0: // no limit
		s += " (the table count limit is not set)."
	case res.Agent.TablestatsGroupTableLimit < 0: // always disabled
		s += " (always)."
	default:
		count := "unknown"
		if res.TableCount > 0 {
			count = strconv.Itoa(int(res.TableCount))
		}

		s += fmt.Sprintf(" (the limit is %d, the actual table count is %s).", res.Agent.TablestatsGroupTableLimit, count)
	}

	return s
}

type addAgentMysqldExporterCommand struct {
	PMMAgentID                string
	ServiceID                 string
	Username                  string
	Password                  string
	AgentPassword             string
	CustomLabels              string
	SkipConnectionCheck       bool
	TLS                       bool
	TLSSkipVerify             bool
	TLSCaFile                 string
	TLSCertFile               string
	TLSKeyFile                string
	TablestatsGroupTableLimit int32
	PushMetrics               bool
	DisableCollectors         string
}

func (cmd *addAgentMysqldExporterCommand) Run() (commands.Result, error) {
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

	params := &agents.AddMySQLdExporterParams{
		Body: agents.AddMySQLdExporterBody{
			PMMAgentID:                cmd.PMMAgentID,
			ServiceID:                 cmd.ServiceID,
			Username:                  cmd.Username,
			Password:                  cmd.Password,
			AgentPassword:             cmd.AgentPassword,
			CustomLabels:              customLabels,
			SkipConnectionCheck:       cmd.SkipConnectionCheck,
			TLS:                       cmd.TLS,
			TLSSkipVerify:             cmd.TLSSkipVerify,
			TLSCa:                     tlsCa,
			TLSCert:                   tlsCert,
			TLSKey:                    tlsKey,
			TablestatsGroupTableLimit: cmd.TablestatsGroupTableLimit,
			PushMetrics:               cmd.PushMetrics,
			DisableCollectors:         commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:                  &addExporterLogLevel,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddMySQLdExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentMysqldExporterResult{
		Agent:      resp.Payload.MysqldExporter,
		TableCount: resp.Payload.TableCount,
	}, nil
}

// register command
var (
	AddAgentMysqldExporter  addAgentMysqldExporterCommand
	AddAgentMysqldExporterC = addAgentC.Command("mysqld-exporter", "Add mysqld_exporter to inventory").Hide(hide)
)

func init() {
	AddAgentMysqldExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentMysqldExporter.PMMAgentID)
	AddAgentMysqldExporterC.Arg("service-id", "Service identifier").Required().StringVar(&AddAgentMysqldExporter.ServiceID)
	AddAgentMysqldExporterC.Arg("username", "MySQL username for scraping metrics").Default("root").StringVar(&AddAgentMysqldExporter.Username)
	AddAgentMysqldExporterC.Flag("password", "MySQL password for scraping metrics").StringVar(&AddAgentMysqldExporter.Password)
	AddAgentMysqldExporterC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&AddAgentMysqldExporter.AgentPassword)
	AddAgentMysqldExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentMysqldExporter.CustomLabels)
	AddAgentMysqldExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentMysqldExporter.SkipConnectionCheck)
	AddAgentMysqldExporterC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddAgentMysqldExporter.TLS)
	AddAgentMysqldExporterC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddAgentMysqldExporter.TLSSkipVerify)
	AddAgentMysqldExporterC.Flag("tls-ca", "Path to certificate authority certificate file").StringVar(&AddAgentMysqldExporter.TLSCaFile)
	AddAgentMysqldExporterC.Flag("tls-cert", "Path to client certificate file").StringVar(&AddAgentMysqldExporter.TLSCertFile)
	AddAgentMysqldExporterC.Flag("tls-key", "Path to client key file").StringVar(&AddAgentMysqldExporter.TLSKeyFile)
	AddAgentMysqldExporterC.Flag("tablestats-group-table-limit",
		"Tablestats group collectors will be disabled if there are more than that number of tables (default: 0 - always enabled; negative value - always disabled)").
		Int32Var(&AddAgentMysqldExporter.TablestatsGroupTableLimit)
	AddAgentMysqldExporterC.Flag("push-metrics", "Enables push metrics model flow,"+
		" it will be sent to the server by an agent").BoolVar(&AddAgentMysqldExporter.PushMetrics)
	AddAgentMysqldExporterC.Flag("disable-collectors",
		"Comma-separated list of collector names to exclude from exporter").StringVar(&AddAgentMysqldExporter.DisableCollectors)
	addExporterGlobalFlags(AddAgentMysqldExporterC)
}
