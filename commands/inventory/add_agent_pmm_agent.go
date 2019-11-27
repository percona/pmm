// pmm-admin
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
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
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
	AddAgentPMMAgentC = addAgentC.Command("pmm-agent", "add PMM agent to inventory").Hide(hide)
)

func init() {
	AddAgentPMMAgentC.Arg("runs-on-node-id", "Node identifier where this instance runs").Required().StringVar(&AddAgentPMMAgent.RunsOnNodeID)
	AddAgentPMMAgentC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentPMMAgent.CustomLabels)
}
