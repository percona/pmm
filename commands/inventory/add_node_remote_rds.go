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

type addNodeRemoteRDSCommand struct {
	NodeName     string
	Address      string
	NodeModel    string
	Region       string
	Az           string
	CustomLabels string
}

func (cmd *addNodeRemoteRDSCommand) Run() (commands.Result, error) {
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

// register command
var (
	AddNodeRemoteRDS  = new(addNodeRemoteRDSCommand)
	AddNodeRemoteRDSC = addNodeC.Command("remote-rds", "Add Remote RDS node to inventory").Hide(hide)
)

func init() {
	AddNodeRemoteRDSC.Arg("name", "Node name").StringVar(&AddNodeRemoteRDS.NodeName)

	AddNodeRemoteRDSC.Flag("address", "Address").StringVar(&AddNodeRemoteRDS.Address)
	AddNodeRemoteRDSC.Flag("node-model", "Node model").StringVar(&AddNodeRemoteRDS.NodeModel)
	AddNodeRemoteRDSC.Flag("region", "Node region").StringVar(&AddNodeRemoteRDS.Region)
	AddNodeRemoteRDSC.Flag("az", "Node availability zone").StringVar(&AddNodeRemoteRDS.Az)
	AddNodeRemoteRDSC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddNodeRemoteRDS.CustomLabels)
}
