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

var removeAgentResultT = commands.ParseTemplate(`
Agent removed.
`)

type removeAgentResult struct{}

func (res *removeAgentResult) Result() {}

func (res *removeAgentResult) String() string {
	return commands.RenderTemplate(removeAgentResultT, res)
}

type removeAgentCommand struct {
	AgentID string
	Force   bool
}

func (cmd *removeAgentCommand) Run() (commands.Result, error) {
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

// register command
var (
	RemoveAgent  = new(removeAgentCommand)
	RemoveAgentC = inventoryRemoveC.Command("agent", "Remove agent from inventory.")
)

func init() {
	RemoveAgentC.Arg("agent-id", "Agent ID").StringVar(&RemoveAgent.AgentID)
	RemoveAgentC.Flag("force", "Remove agent with all dependencies").BoolVar(&RemoveAgent.Force)
}
