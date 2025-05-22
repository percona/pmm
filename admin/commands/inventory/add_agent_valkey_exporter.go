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

var addAgentValkeyExporterResultT = commands.ParseTemplate(`
Valkey Exporter added.
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

type addAgentValkeyExporterResult struct {
	Agent      *agents.AddAgentOKBodyValkeyExporter `json:"Valkey_exporter"`
	TableCount int32                                `json:"table_count,omitempty"`
}

func (res *addAgentValkeyExporterResult) Result() {}

func (res *addAgentValkeyExporterResult) String() string {
	return commands.RenderTemplate(addAgentValkeyExporterResultT, res)
}

// AddAgentValkeyExporterCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddAgentValkeyExporterCommand struct {
	PMMAgentID          string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string            `arg:"" help:"Service identifier"`
	Username            string            `arg:"" optional:"" help:"Valkey username for scraping metrics"`
	Password            string            `help:"Valkey password for scraping metrics"`
	AgentPassword       string            `help:"Custom password for /metrics endpoint"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
	TLSCAFile           string            `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile         string            `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile          string            `name:"tls-key" help:"Path to client key file"`
	PushMetrics         bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	ExposeExporter      bool              `help:"Expose the address of the exporter publicly on 0.0.0.0"`
	DisableCollectors   []string          `help:"Comma-separated list of collector names to exclude from exporter"`

	flags.LogLevelNoFatalFlags
}

// RunCmd executes the AddAgentValkeyExporterCommand and returns the result.
func (cmd *AddAgentValkeyExporterCommand) RunCmd() (commands.Result, error) {
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

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			ValkeyExporter: &agents.AddAgentParamsBodyValkeyExporter{
				PMMAgentID:          cmd.PMMAgentID,
				ServiceID:           cmd.ServiceID,
				Username:            cmd.Username,
				Password:            cmd.Password,
				AgentPassword:       cmd.AgentPassword,
				CustomLabels:        customLabels,
				SkipConnectionCheck: cmd.SkipConnectionCheck,
				TLS:                 cmd.TLS,
				TLSSkipVerify:       cmd.TLSSkipVerify,
				TLSCa:               tlsCa,
				TLSCert:             tlsCert,
				TLSKey:              tlsKey,
				PushMetrics:         cmd.PushMetrics,
				ExposeExporter:      cmd.ExposeExporter,
				DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
				LogLevel:            cmd.LogLevelNoFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentValkeyExporterResult{
		Agent: resp.Payload.ValkeyExporter,
	}, nil
}
