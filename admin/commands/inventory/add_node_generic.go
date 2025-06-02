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
	"github.com/percona/pmm/api/inventory/v1/json/client"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
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
	Node *nodes.AddNodeOKBodyGeneric `json:"generic"`
}

func (res *addNodeGenericResult) Result() {}

func (res *addNodeGenericResult) String() string {
	return commands.RenderTemplate(addNodeGenericResultT, res)
}

// AddNodeGenericCommand is used by Kong for CLI flags and commands.
type AddNodeGenericCommand struct {
	NodeName     string            `arg:"" optional:"" name:"name" help:"Node name"`
	MachineID    string            `help:"Linux machine-id"`
	Distro       string            `help:"Linux distribution (if any)"`
	Address      string            `help:"Address"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	Region       string            `help:"Node region"`
	Az           string            `help:"Node availability zone"`
	NodeModel    string            `help:"Node mddel"`
}

// RunCmd executes the AddNodeGenericCommand and returns the result.
func (cmd *AddNodeGenericCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)
	params := &nodes.AddNodeParams{
		Body: nodes.AddNodeBody{
			Generic: &nodes.AddNodeParamsBodyGeneric{
				NodeName:     cmd.NodeName,
				MachineID:    cmd.MachineID,
				Distro:       cmd.Distro,
				Address:      cmd.Address,
				CustomLabels: *customLabels,

				Region:    cmd.Region,
				Az:        cmd.Az,
				NodeModel: cmd.NodeModel,
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.NodesService.AddNode(params)
	if err != nil {
		return nil, err
	}
	return &addNodeGenericResult{
		Node: resp.Payload.Generic,
	}, nil
}
