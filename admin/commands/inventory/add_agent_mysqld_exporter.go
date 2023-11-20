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

	s := "enabled" //nolint:goconst
	if res.Agent.TablestatsGroupDisabled {
		s = "disabled" //nolint:goconst
	}

	switch {
	case res.Agent.TablestatsGroupTableLimit == 0: // server defined
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

// AddAgentMysqldExporterCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddAgentMysqldExporterCommand struct {
	PMMAgentID                string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                 string            `arg:"" help:"Service identifier"`
	Username                  string            `arg:"" optional:"" help:"MySQL username for scraping metrics"`
	Password                  string            `help:"MySQL password for scraping metrics"`
	AgentPassword             string            `help:"Custom password for /metrics endpoint"`
	CustomLabels              map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck       bool              `help:"Skip connection check"`
	TLS                       bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify             bool              `help:"Skip TLS certificates validation"`
	TLSCAFile                 string            `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile               string            `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile                string            `name:"tls-key" help:"Path to client key file"`
	TablestatsGroupTableLimit int32             `placeholder:"number" help:"Tablestats group collectors will be disabled if there are more than that number of tables (default: server-defined, -1: always disabled)"`
	PushMetrics               bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	ExposeExporter            bool              `help:"Expose the address of the exporter publicly on 0.0.0.0"`
	DisableCollectors         []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	LogLevel                  string            `enum:"debug,info,warn,error" default:"warn" help:"Service logging level. One of: [debug, info, warn, error]"`
}

func (cmd *AddAgentMysqldExporterCommand) RunCmd() (commands.Result, error) {
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
			ExposeExporter:            cmd.ExposeExporter,
			DisableCollectors:         commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:                  &cmd.LogLevel,
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
