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
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addPMMAgentResultT = commands.ParseTemplate(`
PMM-agent added.
Agent ID       : {{ .Agent.AgentID }}
Runs on node ID: {{ .Agent.RunsOnNodeID }}
Connected      : {{ .Agent.Connected }}
Custom labels  : {{ .Agent.CustomLabels }}
`)

type addPMMAgentResult struct {
	Agent *agents.AddAgentOKBodyPMMAgent `json:"pmm_agent"`
}

func (res *addPMMAgentResult) Result() {}

func (res *addPMMAgentResult) String() string {
	return commands.RenderTemplate(addPMMAgentResultT, res)
}

// AddPMMAgentCommand is used by Kong for CLI flags and commands.
type AddPMMAgentCommand struct {
	RunsOnNodeID string            `arg:"" help:"Node identifier where this instance runs"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the AddPMMAgentCommand and returns the result.
func (cmd *AddPMMAgentCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			PMMAgent: &agents.AddAgentParamsBodyPMMAgent{
				RunsOnNodeID: cmd.RunsOnNodeID,
				CustomLabels: *customLabels,
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addPMMAgentResult{
		Agent: resp.Payload.PMMAgent,
	}, nil
}
