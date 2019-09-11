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

var addNodeGenericResultT = commands.ParseTemplate(`
Generic Node added.
Node ID  : {{ .Node.NodeID }}
Node name: {{ .Node.NodeName }}

Machine ID    : {{ .Node.MachineID }}
Distro        : {{ .Node.Distro }}
Address       : {{ .Node.Address }}
Custom labels : {{ .Node.CustomLabels }}

Region    : {{ .Node.Region }}
Az        : {{ .Node.Az }}
Node model: {{ .Node.NodeModel }}
`)

type addNodeGenericResult struct {
	Node *nodes.AddGenericNodeOKBodyGeneric `json:"generic"`
}

func (res *addNodeGenericResult) Result() {}

func (res *addNodeGenericResult) String() string {
	return commands.RenderTemplate(addNodeGenericResultT, res)
}

type addNodeGenericCommand struct {
	NodeName     string
	MachineID    string
	Distro       string
	Address      string
	CustomLabels string
	Region       string
	Az           string
	NodeModel    string
}

func (cmd *addNodeGenericCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &nodes.AddGenericNodeParams{
		Body: nodes.AddGenericNodeBody{
			NodeName:     cmd.NodeName,
			MachineID:    cmd.MachineID,
			Distro:       cmd.Distro,
			Address:      cmd.Address,
			CustomLabels: customLabels,

			Region:    cmd.Region,
			Az:        cmd.Az,
			NodeModel: cmd.NodeModel,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Nodes.AddGenericNode(params)
	if err != nil {
		return nil, err
	}
	return &addNodeGenericResult{
		Node: resp.Payload.Generic,
	}, nil
}

// register command
var (
	AddNodeGeneric  = new(addNodeGenericCommand)
	AddNodeGenericC = addNodeC.Command("generic", "Add generic node to inventory").Hide(hide)
)

func init() {
	AddNodeGenericC.Arg("name", "Node name").StringVar(&AddNodeGeneric.NodeName)

	AddNodeGenericC.Flag("machine-id", "Linux machine-id").StringVar(&AddNodeGeneric.MachineID)
	AddNodeGenericC.Flag("distro", "Linux distribution (if any)").StringVar(&AddNodeGeneric.Distro)
	AddNodeGenericC.Flag("address", "Address").StringVar(&AddNodeGeneric.Address)
	AddNodeGenericC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddNodeGeneric.CustomLabels)

	AddNodeGenericC.Flag("region", "Node region").StringVar(&AddNodeGeneric.Region)
	AddNodeGenericC.Flag("az", "Node availability zone").StringVar(&AddNodeGeneric.Az)
	AddNodeGenericC.Flag("node-model", "Node model").StringVar(&AddNodeGeneric.NodeModel)
}
