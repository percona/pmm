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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var removeAgentResultT = commands.ParseTemplate(`
Agent removed.
`)

type removeAgentResult struct{}

func (res *removeAgentResult) Result() {}

func (res *removeAgentResult) String() string {
	return commands.RenderTemplate(removeAgentResultT, res)
}

// RemoveAgentCommand is used by Kong for CLI flags and commands.
type RemoveAgentCommand struct {
	AgentID string `arg:"" optional:"" help:"Agent ID"`
	Force   bool   `help:"Remove agent with all dependencies"`
}

// RunCmd executes the RemoveAgentCommand and returns the result.
func (cmd *RemoveAgentCommand) RunCmd() (commands.Result, error) {
	params := &agents.RemoveAgentParams{
		Body: agents.RemoveAgentBody{
			AgentID: cmd.AgentID,
			Force:   cmd.Force,
		},
		Context: commands.Ctx,
	}
	_, err := client.Default.Agents.RemoveAgent(params)
	if err != nil {
		return nil, err
	}
	return &removeAgentResult{}, nil
}
