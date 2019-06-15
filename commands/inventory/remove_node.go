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
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"

	"github.com/percona/pmm-admin/commands"
)

var removeNodeGenericResultT = commands.ParseTemplate(`
Node removed.
`)

type removeNodeResult struct{}

func (res *removeNodeResult) Result() {}

func (res *removeNodeResult) String() string {
	return commands.RenderTemplate(removeNodeGenericResultT, res)
}

type removeNodeCommand struct {
	NodeID string
	Force  bool
}

func (cmd *removeNodeCommand) Run() (commands.Result, error) {
	params := &nodes.RemoveNodeParams{
		Body: nodes.RemoveNodeBody{
			NodeID: cmd.NodeID,
			Force:  cmd.Force,
		},
		Context: commands.Ctx,
	}
	_, err := client.Default.Nodes.RemoveNode(params)
	if err != nil {
		return nil, err
	}
	return new(removeNodeResult), nil
}

// register command
var (
	RemoveNode  = new(removeNodeCommand)
	RemoveNodeC = inventoryRemoveC.Command("node", "Remove node from inventory")
)

func init() {
	RemoveNodeC.Arg("node-id", "Node ID").StringVar(&RemoveNode.NodeID)
	RemoveNodeC.Flag("force", "Remove node with all dependencies").BoolVar(&RemoveNode.Force)
}
