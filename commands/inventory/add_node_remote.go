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

type addNodeRemoteCommand struct {
	NodeName     string
	Address      string
	CustomLabels string
	Region       string
	Az           string
}

func (cmd *addNodeRemoteCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
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

// register command
var (
	AddNodeRemote  = new(addNodeRemoteCommand)
	AddNodeRemoteC = addNodeC.Command("remote", "Add Remote node to inventory")
)

func init() {
	AddNodeRemoteC.Arg("name", "Node name").StringVar(&AddNodeRemote.NodeName)

	AddNodeRemoteC.Flag("address", "Address").StringVar(&AddNodeRemote.Address)
	AddNodeRemoteC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddNodeRemote.CustomLabels)

	AddNodeRemoteC.Flag("region", "Node region").StringVar(&AddNodeRemote.Region)
	AddNodeRemoteC.Flag("az", "Node availability zone").StringVar(&AddNodeRemote.Az)
}
