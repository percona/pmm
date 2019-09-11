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

var addNodeContainerResultT = commands.ParseTemplate(`
Container Node added.
Node ID  : {{ .Node.NodeID }}
Node name: {{ .Node.NodeName }}

Machine ID    : {{ .Node.MachineID }}
Container ID  : {{ .Node.ContainerID }}
Container name: {{ .Node.ContainerName }}
Custom labels : {{ .Node.CustomLabels }}

Region    : {{ .Node.Region }}
Az        : {{ .Node.Az }}
Node model: {{ .Node.NodeModel }}
`)

type addNodeContainerResult struct {
	Node *nodes.AddContainerNodeOKBodyContainer `json:"container"`
}

func (res *addNodeContainerResult) Result() {}

func (res *addNodeContainerResult) String() string {
	return commands.RenderTemplate(addNodeContainerResultT, res)
}

type addNodeContainerCommand struct {
	NodeName      string
	MachineID     string
	ContainerID   string
	ContainerName string
	Address       string
	CustomLabels  string
	Region        string
	Az            string
	NodeModel     string
}

func (cmd *addNodeContainerCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &nodes.AddContainerNodeParams{
		Body: nodes.AddContainerNodeBody{
			NodeName:      cmd.NodeName,
			MachineID:     cmd.MachineID,
			ContainerID:   cmd.ContainerID,
			ContainerName: cmd.ContainerName,
			Address:       cmd.Address,
			CustomLabels:  customLabels,

			Region:    cmd.Region,
			Az:        cmd.Az,
			NodeModel: cmd.NodeModel,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Nodes.AddContainerNode(params)
	if err != nil {
		return nil, err
	}
	return &addNodeContainerResult{
		Node: resp.Payload.Container,
	}, nil
}

// register command
var (
	AddNodeContainer  = new(addNodeContainerCommand)
	AddNodeContainerC = addNodeC.Command("container", "Add container node to inventory").Hide(hide)
)

func init() {
	AddNodeContainerC.Arg("name", "Node name").StringVar(&AddNodeContainer.NodeName)

	AddNodeContainerC.Flag("machine-id", "Linux machine-id").StringVar(&AddNodeContainer.MachineID)
	AddNodeContainerC.Flag("container-id", "Container identifier; if specified, must be a unique Docker container identifier").
		StringVar(&AddNodeContainer.ContainerID)
	AddNodeContainerC.Flag("container-name", "Container name").StringVar(&AddNodeContainer.ContainerName)
	AddNodeContainerC.Flag("address", "Address").StringVar(&AddNodeContainer.Address)
	AddNodeContainerC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddNodeContainer.CustomLabels)

	AddNodeContainerC.Flag("region", "Node region").StringVar(&AddNodeContainer.Region)
	AddNodeContainerC.Flag("az", "Node availability zone").StringVar(&AddNodeContainer.Az)
	AddNodeContainerC.Flag("node-model", "Node model").StringVar(&AddNodeContainer.NodeModel)
}
