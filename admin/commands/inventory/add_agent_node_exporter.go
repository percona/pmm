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
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
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
	Agent *agents.AddNodeExporterOKBodyNodeExporter `json:"node_exporter"`
}

func (res *addAgentNodeExporterResult) Result() {}

func (res *addAgentNodeExporterResult) String() string {
	return commands.RenderTemplate(addAgentNodeExporterResultT, res)
}

type addAgentNodeExporterCommand struct {
	PMMAgentID        string
	CustomLabels      string
	PushMetrics       bool
	DisableCollectors string
}

func (cmd *addAgentNodeExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddNodeExporterParams{
		Body: agents.AddNodeExporterBody{
			PMMAgentID:        cmd.PMMAgentID,
			CustomLabels:      customLabels,
			PushMetrics:       cmd.PushMetrics,
			DisableCollectors: commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:          &addExporterLogLevel,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddNodeExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentNodeExporterResult{
		Agent: resp.Payload.NodeExporter,
	}, nil
}

// register command
var (
	AddAgentNodeExporter  addAgentNodeExporterCommand
	AddAgentNodeExporterC = addAgentC.Command("node-exporter", "add Node exporter to inventory").Hide(hide)
)

func init() {
	AddAgentNodeExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentNodeExporter.PMMAgentID)
	AddAgentNodeExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentNodeExporter.CustomLabels)
	AddAgentNodeExporterC.Flag("push-metrics", "Enables push metrics model flow,"+
		" it will be sent to the server by an agent").BoolVar(&AddAgentNodeExporter.PushMetrics)
	AddAgentNodeExporterC.Flag("disable-collectors",
		"Comma-separated list of collector names to exclude from exporter").StringVar(&AddAgentNodeExporter.DisableCollectors)
	addExporterGlobalFlags(AddAgentNodeExporterC)
}
