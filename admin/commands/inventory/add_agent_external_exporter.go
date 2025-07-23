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
	"strings"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addAgentExternalExporterResultT = commands.ParseTemplate(`
External Exporter added.
Agent ID              : {{ .Agent.AgentID }}
Runs on node ID       : {{ .Agent.RunsOnNodeID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Scheme                : {{ .Agent.Scheme }}
Metrics path          : {{ .Agent.MetricsPath }}
Listen port           : {{ .Agent.ListenPort }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentExternalExporterResult struct {
	Agent *agents.AddAgentOKBodyExternalExporter `json:"external_exporter"`
}

func (res *addAgentExternalExporterResult) Result() {}

func (res *addAgentExternalExporterResult) String() string {
	return commands.RenderTemplate(addAgentExternalExporterResultT, res)
}

// AddAgentExternalExporterCommand is used by Kong for CLI flags and commands.
type AddAgentExternalExporterCommand struct {
	RunsOnNodeID  string            `required:"" help:"Node identifier where this instance runs"`
	ServiceID     string            `required:"" help:"Service identifier"`
	Username      string            `help:"HTTP Basic auth username for scraping metrics"`
	Password      string            `help:"HTTP Basic auth password for scraping metrics"`
	Scheme        string            `help:"Scheme to generate URI to exporter metrics endpoints (http, https)"`
	MetricsPath   string            `help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort    int64             `required:"" placeholder:"port" help:"Listen port for scraping metrics"`
	CustomLabels  map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	PushMetrics   bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	TLSSkipVerify bool              `help:"Skip TLS certificate verification"`
}

// RunCmd executes the AddAgentExternalExporterCommand and returns the result.
func (cmd *AddAgentExternalExporterCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			ExternalExporter: &agents.AddAgentParamsBodyExternalExporter{
				RunsOnNodeID:  cmd.RunsOnNodeID,
				ServiceID:     cmd.ServiceID,
				Username:      cmd.Username,
				Password:      cmd.Password,
				Scheme:        cmd.Scheme,
				MetricsPath:   cmd.MetricsPath,
				ListenPort:    cmd.ListenPort,
				CustomLabels:  *customLabels,
				PushMetrics:   cmd.PushMetrics,
				TLSSkipVerify: cmd.TLSSkipVerify,
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentExternalExporterResult{
		Agent: resp.Payload.ExternalExporter,
	}, nil
}
