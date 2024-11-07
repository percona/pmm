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

var addAgentPostgresExporterResultT = commands.ParseTemplate(`
Postgres Exporter added.
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

type addAgentPostgresExporterResult struct {
	Agent *agents.AddAgentOKBodyPostgresExporter `json:"postgres_exporter"`
}

func (res *addAgentPostgresExporterResult) Result() {}

func (res *addAgentPostgresExporterResult) String() string {
	return commands.RenderTemplate(addAgentPostgresExporterResultT, res)
}

// AddAgentPostgresExporterCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddAgentPostgresExporterCommand struct {
	PMMAgentID          string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string            `arg:"" help:"Service identifier"`
	Username            string            `arg:"" optional:"" help:"PostgreSQL username for scraping metrics"`
	Password            string            `help:"PostgreSQL password for scraping metrics"`
	AgentPassword       string            `help:"Custom password for /metrics endpoint"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	PushMetrics         bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	ExposeExporter      bool              `help:"Expose the address of the exporter publicly on 0.0.0.0"`
	DisableCollectors   []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
	TLSCAFile           string            `help:"TLS CA certificate file"`
	TLSCertFile         string            `help:"TLS certificate file"`
	TLSKeyFile          string            `help:"TLS certificate key file"`
	AutoDiscoveryLimit  int32             `default:"0" placeholder:"NUMBER" help:"Auto-discovery will be disabled if there are more than that number of databases (default: server-defined, -1: always disabled)"`

	flags.LogLevelNoFatalFlags
}

// RunCmd executes the AddAgentPostgresExporterCommand and returns the result.
func (cmd *AddAgentPostgresExporterCommand) RunCmd() (commands.Result, error) {
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
			PostgresExporter: &agents.AddAgentParamsBodyPostgresExporter{
				PMMAgentID:          cmd.PMMAgentID,
				ServiceID:           cmd.ServiceID,
				Username:            cmd.Username,
				Password:            cmd.Password,
				AgentPassword:       cmd.AgentPassword,
				CustomLabels:        customLabels,
				SkipConnectionCheck: cmd.SkipConnectionCheck,
				PushMetrics:         cmd.PushMetrics,
				ExposeExporter:      cmd.ExposeExporter,
				DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
				AutoDiscoveryLimit:  cmd.AutoDiscoveryLimit,

				TLS:           cmd.TLS,
				TLSSkipVerify: cmd.TLSSkipVerify,
				TLSCa:         tlsCa,
				TLSCert:       tlsCert,
				TLSKey:        tlsKey,
				LogLevel:      cmd.LogLevelNoFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentPostgresExporterResult{
		Agent: resp.Payload.PostgresExporter,
	}, nil
}
