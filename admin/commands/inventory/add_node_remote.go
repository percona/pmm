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

var addNodeRemoteResultT = commands.ParseTemplate(`
Remote Node added.
Node ID  : {{ .Node.NodeID }}
Node name: {{ .Node.NodeName }}

Address       : {{ .Node.Address }}
Custom labels : {{ .Node.CustomLabels }}

Region    : {{ .Node.Region }}
Az        : {{ .Node.Az }}
`)

type addNodeRemoteResult struct {
	Node *nodes.AddRemoteNodeOKBodyRemote `json:"remote"`
}

func (res *addNodeRemoteResult) Result() {}

func (res *addNodeRemoteResult) String() string {
	return commands.RenderTemplate(addNodeRemoteResultT, res)
}

// AddNodeRemoteCommand is used by Kong for CLI flags and commands.
type AddNodeRemoteCommand struct {
	NodeName     string            `arg:"" optional:"" name:"name" help:"Node name"`
	Address      string            `help:"Address"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	Region       string            `help:"Node region"`
	Az           string            `help:"Node availability zone"`
}

// RunCmd executes the AddNodeRemoteCommand and returns the result.
func (cmd *AddNodeRemoteCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)
	params := &nodes.AddRemoteNodeParams{
		Body: nodes.AddRemoteNodeBody{
			NodeName:     cmd.NodeName,
			Address:      cmd.Address,
			CustomLabels: customLabels,

			Region: cmd.Region,
			Az:     cmd.Az,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Nodes.AddRemoteNode(params)
	if err != nil {
		return nil, err
	}
	return &addNodeRemoteResult{
		Node: resp.Payload.Remote,
	}, nil
}
