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

var addNodeRemoteRDSResultT = commands.ParseTemplate(`
Remote RDS Node added.
Node ID  : {{ .Node.NodeID }}
Node name: {{ .Node.NodeName }}

Address       : {{ .Node.Address }}
Model         : {{ .Node.NodeModel }}
Custom labels : {{ .Node.CustomLabels }}

Region    : {{ .Node.Region }}
Az        : {{ .Node.Az }}
`)

type addNodeRemoteRDSResult struct {
	Node *nodes.AddNodeOKBodyRemoteRDS `json:"remote_rds"`
}

func (res *addNodeRemoteRDSResult) Result() {}

func (res *addNodeRemoteRDSResult) String() string {
	return commands.RenderTemplate(addNodeRemoteRDSResultT, res)
}

// AddNodeRemoteRDSCommand is used by Kong for CLI flags and commands.
type AddNodeRemoteRDSCommand struct {
	NodeName     string            `arg:"" optional:"" name:"name" help:"Node name"`
	Address      string            `help:"Address"`
	NodeModel    string            `help:"Node mddel"`
	Region       string            `help:"Node region"`
	Az           string            `help:"Node availability zone"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the AddNodeRemoteRDSCommand and returns the result.
func (cmd *AddNodeRemoteRDSCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)
	params := &nodes.AddNodeParams{
		Body: nodes.AddNodeBody{
			RemoteRDS: &nodes.AddNodeParamsBodyRemoteRDS{
				NodeName:     cmd.NodeName,
				Address:      cmd.Address,
				NodeModel:    cmd.NodeModel,
				Region:       cmd.Region,
				Az:           cmd.Az,
				CustomLabels: *customLabels,
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.NodesService.AddNode(params)
	if err != nil {
		return nil, err
	}
	return &addNodeRemoteRDSResult{
		Node: resp.Payload.RemoteRDS,
	}, nil
}
