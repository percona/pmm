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
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
)

var removeNodeGenericResultT = commands.ParseTemplate(`
Node removed.
`)

type removeNodeResult struct{}

func (res *removeNodeResult) Result() {}

func (res *removeNodeResult) String() string {
	return commands.RenderTemplate(removeNodeGenericResultT, res)
}

// RemoveNodeCommand is used by Kong for CLI flags and commands.
type RemoveNodeCommand struct {
	NodeID string `arg:"" optional:"" help:"Node ID"`
	Force  bool   `help:"Remove node with all dependencies"`
}

// RunCmd runs the command for RemoveNodeCommand.
func (cmd *RemoveNodeCommand) RunCmd() (commands.Result, error) {
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
	return &removeNodeResult{}, nil
}
