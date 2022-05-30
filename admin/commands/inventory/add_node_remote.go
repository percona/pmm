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
	AddNodeRemoteC = addNodeC.Command("remote", "Add Remote node to inventory").Hide(hide)
)

func init() {
	AddNodeRemoteC.Arg("name", "Node name").StringVar(&AddNodeRemote.NodeName)

	AddNodeRemoteC.Flag("address", "Address").StringVar(&AddNodeRemote.Address)
	AddNodeRemoteC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddNodeRemote.CustomLabels)

	AddNodeRemoteC.Flag("region", "Node region").StringVar(&AddNodeRemote.Region)
	AddNodeRemoteC.Flag("az", "Node availability zone").StringVar(&AddNodeRemote.Az)
}
