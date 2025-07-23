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
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addAgentProxysqlExporterResultT = commands.ParseTemplate(`
Proxysql Exporter added.
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
`)

type addAgentProxysqlExporterResult struct {
	Agent *agents.AddAgentOKBodyProxysqlExporter `json:"proxysql_exporter"`
}

func (res *addAgentProxysqlExporterResult) Result() {}

func (res *addAgentProxysqlExporterResult) String() string {
	return commands.RenderTemplate(addAgentProxysqlExporterResultT, res)
}

// AddAgentProxysqlExporterCommand is used by Kong for CLI flags and commands.
type AddAgentProxysqlExporterCommand struct {
	PMMAgentID          string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string            `arg:"" help:"Service identifier"`
	Username            string            `arg:"" optional:"" help:"ProxySQL username for scraping metrics"`
	Password            string            `help:"ProxySQL password for scraping metrics"`
	AgentPassword       string            `help:"Custom password for /metrics endpoint"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificate verification"`
	PushMetrics         bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	ExposeExporter      bool              `help:"Expose the address of the exporter publicly on 0.0.0.0"`
	DisableCollectors   []string          `help:"Comma-separated list of collector names to exclude from exporter"`

	flags.LogLevelFatalFlags
}

// RunCmd executes the AddAgentProxysqlExporterCommand and returns the result.
func (cmd *AddAgentProxysqlExporterCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)
	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			ProxysqlExporter: &agents.AddAgentParamsBodyProxysqlExporter{
				PMMAgentID:          cmd.PMMAgentID,
				ServiceID:           cmd.ServiceID,
				Username:            cmd.Username,
				Password:            cmd.Password,
				AgentPassword:       cmd.AgentPassword,
				CustomLabels:        *customLabels,
				SkipConnectionCheck: cmd.SkipConnectionCheck,
				TLS:                 cmd.TLS,
				TLSSkipVerify:       cmd.TLSSkipVerify,
				PushMetrics:         cmd.PushMetrics,
				ExposeExporter:      cmd.ExposeExporter,
				DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
				LogLevel:            cmd.LogLevelFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentProxysqlExporterResult{
		Agent: resp.Payload.ProxysqlExporter,
	}, nil
}
