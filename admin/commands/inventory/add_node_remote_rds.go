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
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
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
	Node *nodes.AddRemoteRDSNodeOKBodyRemoteRDS `json:"remote_rds"`
}

func (res *addNodeRemoteRDSResult) Result() {}

func (res *addNodeRemoteRDSResult) String() string {
	return commands.RenderTemplate(addNodeRemoteRDSResultT, res)
}

func (cmd *AddNodeRemoteRDSCommand) RunCmd() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &nodes.AddRemoteRDSNodeParams{
		Body: nodes.AddRemoteRDSNodeBody{
			NodeName:     cmd.NodeName,
			Address:      cmd.Address,
			NodeModel:    cmd.NodeModel,
			Region:       cmd.Region,
			Az:           cmd.Az,
			CustomLabels: customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Nodes.AddRemoteRDSNode(params)
	if err != nil {
		return nil, err
	}
	return &addNodeRemoteRDSResult{
		Node: resp.Payload.RemoteRDS,
	}, nil
}
