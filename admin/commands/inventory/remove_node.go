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
	RemoveNodeC = inventoryRemoveC.Command("node", "Remove node from inventory").Hide(hide)
)

func init() {
	RemoveNodeC.Arg("node-id", "Node ID").StringVar(&RemoveNode.NodeID)
	RemoveNodeC.Flag("force", "Remove node with all dependencies").BoolVar(&RemoveNode.Force)
}
