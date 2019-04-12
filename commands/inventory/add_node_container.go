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

var addNodeContainerResultT = commands.ParseTemplate(`
Container Node added.
Node ID  : {{ .Node.NodeID }}
Node name: {{ .Node.NodeName }}

MachineID          : {{ .Node.MachineID }}
DockerContainerID  : {{ .Node.DockerContainerID }}
DockerContainerName: {{ .Node.DockerContainerName }}
Custom labels      : {{ .Node.CustomLabels }}
`)

type addNodeContainerResult struct {
	Node *nodes.AddContainerNodeOKBodyContainer `json:"container"`
}

func (res *addNodeContainerResult) Result() {}

func (res *addNodeContainerResult) String() string {
	return commands.RenderTemplate(addNodeContainerResultT, res)
}

type addNodeContainerCommand struct {
	NodeName            string
	MachineID           string
	DockerContainerID   string
	DockerContainerName string
	Address             string
	CustomLabels        string
}

func (cmd *addNodeContainerCommand) Run() (commands.Result, error) {
	customLabels, err := parseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &nodes.AddContainerNodeParams{
		Body: nodes.AddContainerNodeBody{
			NodeName:            cmd.NodeName,
			MachineID:           cmd.MachineID,
			DockerContainerID:   cmd.DockerContainerID,
			DockerContainerName: cmd.DockerContainerName,
			Address:             cmd.Address,
			CustomLabels:        customLabels,
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
	AddNodeContainerC = addNodeC.Command("container", "Add container node to inventory.")
)

func init() {
	AddNodeContainerC.Arg("name", "Node name").StringVar(&AddNodeContainer.NodeName)

	AddNodeContainerC.Flag("machine-id", "Linux machine-id.").StringVar(&AddNodeContainer.MachineID)
	AddNodeContainerC.Flag("container-id", "Docker container identifier. If specified, must be a unique Docker container identifier.").
		StringVar(&AddNodeContainer.DockerContainerID)
	AddNodeContainerC.Flag("container-name", "Container name.").StringVar(&AddNodeContainer.DockerContainerName)
	AddNodeContainerC.Flag("address", "Address.").StringVar(&AddNodeContainer.Address)
	AddNodeContainerC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddNodeGeneric.CustomLabels)
}
