// pmm-admin
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package inventory

import (
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"

	"github.com/percona/pmm-admin/commands"
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
	PMMAgentID   string
	CustomLabels string
}

func (cmd *addAgentNodeExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddNodeExporterParams{
		Body: agents.AddNodeExporterBody{
			PMMAgentID:   cmd.PMMAgentID,
			CustomLabels: customLabels,
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
	AddAgentNodeExporter  = new(addAgentNodeExporterCommand)
	AddAgentNodeExporterC = addAgentC.Command("node-exporter", "add Node exporter to inventory")
)

func init() {
	AddAgentNodeExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").StringVar(&AddAgentNodeExporter.PMMAgentID)
	AddAgentNodeExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentNodeExporter.CustomLabels)
}
