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

var addPMMAgentResultT = commands.ParseTemplate(`
PMM-agent added.
Agent ID       : {{ .Agent.AgentID }}
Runs on node ID: {{ .Agent.RunsOnNodeID }}
Connected      : {{ .Agent.Connected }}
Custom labels  : {{ .Agent.CustomLabels }}
`)

type addPMMAgentResult struct {
	Agent *agents.AddPMMAgentOKBodyPMMAgent `json:"pmm_agent"`
}

func (res *addPMMAgentResult) Result() {}

func (res *addPMMAgentResult) String() string {
	return commands.RenderTemplate(addPMMAgentResultT, res)
}

type addPMMAgentCommand struct {
	RunsOnNodeID string
	CustomLabels string
}

func (cmd *addPMMAgentCommand) Run() (commands.Result, error) {
	customLabels, err := parseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddPMMAgentParams{
		Body: agents.AddPMMAgentBody{
			RunsOnNodeID: cmd.RunsOnNodeID,
			CustomLabels: customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddPMMAgent(params)
	if err != nil {
		return nil, err
	}
	return &addPMMAgentResult{
		Agent: resp.Payload.PMMAgent,
	}, nil
}

// register command
var (
	AddAgentPMMAgent  = new(addPMMAgentCommand)
	AddAgentPMMAgentC = addAgentC.Command("pmm-agent", "add PMM agent to inventory.")
)

func init() {
	AddAgentPMMAgentC.Arg("runs-on-node-id", "Node identifier where this instance runs.").StringVar(&AddAgentPMMAgent.RunsOnNodeID)
	AddAgentPMMAgentC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddAgentPMMAgent.CustomLabels)
}
