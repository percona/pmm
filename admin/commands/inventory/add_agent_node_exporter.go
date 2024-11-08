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

var addAgentNodeExporterResultT = commands.ParseTemplate(`
Node Exporter added.
Agent ID     : {{ .Agent.AgentID }}
PMM-Agent ID : {{ .Agent.PMMAgentID }}
Listen port  : {{ .Agent.ListenPort }}

Status       : {{ .Agent.Status }}
Disabled     : {{ .Agent.Disabled }}
Custom labels: {{ .Agent.CustomLabels }}
`)

type addAgentNodeExporterResult struct {
	Agent *agents.AddAgentOKBodyNodeExporter `json:"node_exporter"`
}

func (res *addAgentNodeExporterResult) Result() {}

func (res *addAgentNodeExporterResult) String() string {
	return commands.RenderTemplate(addAgentNodeExporterResultT, res)
}

// AddAgentNodeExporterCommand is used by Kong for CLI flags and commands.
type AddAgentNodeExporterCommand struct {
	PMMAgentID        string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	CustomLabels      map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	PushMetrics       bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	ExposeExporter    bool              `help:"Expose the address of the exporter publicly on 0.0.0.0"`
	DisableCollectors []string          `help:"Comma-separated list of collector names to exclude from exporter"`

	flags.LogLevelNoFatalFlags
}

// RunCmd runs the command for AddAgentNodeExporterCommand.
func (cmd *AddAgentNodeExporterCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)
	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
				PMMAgentID:        cmd.PMMAgentID,
				CustomLabels:      customLabels,
				PushMetrics:       cmd.PushMetrics,
				ExposeExporter:    cmd.ExposeExporter,
				DisableCollectors: commands.ParseDisableCollectors(cmd.DisableCollectors),
				LogLevel:          cmd.LogLevelNoFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentNodeExporterResult{
		Agent: resp.Payload.NodeExporter,
	}, nil
}
