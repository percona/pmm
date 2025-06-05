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

var addAgentRDSExporterResultT = commands.ParseTemplate(`
RDS Exporter added.
Agent ID                  : {{ .Agent.AgentID }}
PMM-Agent ID              : {{ .Agent.PMMAgentID }}
Node ID                   : {{ .Agent.NodeID }}
Listen port               : {{ .Agent.ListenPort }}

Status                    : {{ .Agent.Status }}
Disabled                  : {{ .Agent.Disabled }}
Basic metrics disabled    : {{ .Agent.BasicMetricsDisabled }}
Enhanced metrics disabled : {{ .Agent.EnhancedMetricsDisabled }}
Custom labels             : {{ .Agent.CustomLabels }}
`)

type addAgentRDSExporterResult struct {
	Agent *agents.AddAgentOKBodyRDSExporter `json:"rds_exporter"`
}

func (res *addAgentRDSExporterResult) Result() {}

func (res *addAgentRDSExporterResult) String() string {
	return commands.RenderTemplate(addAgentRDSExporterResultT, res)
}

// AddAgentRDSExporterCommand is used by Kong for CLI flags and commands.
type AddAgentRDSExporterCommand struct {
	PMMAgentID             string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	NodeID                 string            `arg:"" help:"Node identifier"`
	AWSAccessKey           string            `help:"AWS Access Key ID"`
	AWSSecretKey           string            `help:"AWS Secret Access Key"`
	CustomLabels           map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck    bool              `help:"Skip connection check"`
	DisableBasicMetrics    bool              `help:"Disable basic metrics"`
	DisableEnhancedMetrics bool              `help:"Disable enhanced metrics"`
	PushMetrics            bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`

	flags.LogLevelFatalFlags
}

// RunCmd executes the AddAgentRDSExporterCommand and returns the result.
func (cmd *AddAgentRDSExporterCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)
	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			RDSExporter: &agents.AddAgentParamsBodyRDSExporter{
				PMMAgentID:             cmd.PMMAgentID,
				NodeID:                 cmd.NodeID,
				AWSAccessKey:           cmd.AWSAccessKey,
				AWSSecretKey:           cmd.AWSSecretKey,
				CustomLabels:           *customLabels,
				SkipConnectionCheck:    cmd.SkipConnectionCheck,
				DisableBasicMetrics:    cmd.DisableBasicMetrics,
				DisableEnhancedMetrics: cmd.DisableEnhancedMetrics,
				PushMetrics:            cmd.PushMetrics,
				LogLevel:               cmd.LogLevelFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentRDSExporterResult{
		Agent: resp.Payload.RDSExporter,
	}, nil
}
