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

var listNodesResultT = commands.ParseTemplate(`
Nodes list.

{{ printf "%-13s" "Node type" }} {{ printf "%-20s" "Node name" }} {{ printf "%-17s" "Address" }} {{ "Node ID" }}
{{ range .Nodes }}
{{- printf "%-13s" .NodeType }} {{ printf "%-20s" .NodeName }} {{ printf "%-17s" .Address }} {{ .NodeID }}
{{ end }}
`)

type listResultNode struct {
	NodeType string `json:"node_type"`
	NodeName string `json:"node_name"`
	Address  string `json:"address"`
	NodeID   string `json:"node_id"`
}

type listNodesResult struct {
	Nodes []listResultNode `json:"nodes"`
}

func (res *listNodesResult) Result() {}

func (res *listNodesResult) String() string {
	return commands.RenderTemplate(listNodesResultT, res)
}

type listNodeCommand struct {
}

func (cmd *listNodeCommand) Run() (commands.Result, error) {
	params := &nodes.ListNodesParams{
		Context: commands.Ctx,
	}
	result, err := client.Default.Nodes.ListNodes(params)
	if err != nil {
		return nil, err
	}

	var nodes []listResultNode
	for _, n := range result.Payload.Generic {
		nodes = append(nodes, listResultNode{
			NodeType: "Generic",
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}
	for _, n := range result.Payload.Container {
		nodes = append(nodes, listResultNode{
			NodeType: "Container",
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}
	for _, n := range result.Payload.Remote {
		nodes = append(nodes, listResultNode{
			NodeType: "Remote",
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}
	for _, n := range result.Payload.RemoteRDS {
		nodes = append(nodes, listResultNode{
			NodeType: "RemoteRDS",
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}

	return &listNodesResult{
		Nodes: nodes,
	}, nil
}

// register command
var (
	ListNodes  = new(listNodeCommand)
	ListNodesC = inventoryListC.Command("nodes", "Show nodes in inventory").Hide(hide)
)
