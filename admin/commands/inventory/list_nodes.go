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
	"github.com/percona/pmm/api/inventorypb/types"
)

var listNodesResultT = commands.ParseTemplate(`
Nodes list.

{{ printf "%-13s" "Node type" }} {{ printf "%-20s" "Node name" }} {{ printf "%-17s" "Address" }} {{ "Node ID" }}
{{ range .Nodes }}
{{- printf "%-13s" .NodeType }} {{ printf "%-20s" .NodeName }} {{ printf "%-17s" .Address }} {{ .NodeID }}
{{ end }}
`)

var acceptableNodeTypes = map[string][]string{
	types.NodeTypeGenericNode:   {types.NodeTypeName(types.NodeTypeGenericNode)},
	types.NodeTypeContainerNode: {types.NodeTypeName(types.NodeTypeContainerNode)},
	types.NodeTypeRemoteNode:    {types.NodeTypeName(types.NodeTypeRemoteNode)},
	types.NodeTypeRemoteRDSNode: {types.NodeTypeName(types.NodeTypeRemoteRDSNode)},
}

type listResultNode struct {
	NodeType string `json:"node_type"`
	NodeName string `json:"node_name"`
	Address  string `json:"address"`
	NodeID   string `json:"node_id"`
}

func (n listResultNode) HumanReadableNodeType() string {
	return types.NodeTypeName(n.NodeType)
}

type listNodesResult struct {
	Nodes []listResultNode `json:"nodes"`
}

func (res *listNodesResult) Result() {}

func (res *listNodesResult) String() string {
	return commands.RenderTemplate(listNodesResultT, res)
}

type listNodeCommand struct {
	NodeType string
}

func (cmd *listNodeCommand) Run() (commands.Result, error) {
	nodeType, err := formatTypeValue(acceptableNodeTypes, cmd.NodeType)
	if err != nil {
		return nil, err
	}

	params := &nodes.ListNodesParams{
		Body:    nodes.ListNodesBody{NodeType: nodeType},
		Context: commands.Ctx,
	}
	result, err := client.Default.Nodes.ListNodes(params)
	if err != nil {
		return nil, err
	}

	var nodesList []listResultNode
	for _, n := range result.Payload.Generic {
		nodesList = append(nodesList, listResultNode{
			NodeType: types.NodeTypeGenericNode,
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}
	for _, n := range result.Payload.Container {
		nodesList = append(nodesList, listResultNode{
			NodeType: types.NodeTypeContainerNode,
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}
	for _, n := range result.Payload.Remote {
		nodesList = append(nodesList, listResultNode{
			NodeType: types.NodeTypeRemoteNode,
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}
	for _, n := range result.Payload.RemoteRDS {
		nodesList = append(nodesList, listResultNode{
			NodeType: types.NodeTypeRemoteRDSNode,
			NodeName: n.NodeName,
			Address:  n.Address,
			NodeID:   n.NodeID,
		})
	}

	return &listNodesResult{
		Nodes: nodesList,
	}, nil
}

// register command
var (
	ListNodes  listNodeCommand
	ListNodesC = inventoryListC.Command("nodes", "Show nodes in inventory").Hide(hide)
)

func init() {
	ListNodesC.Flag("node-type", "Filter by Node type").StringVar(&ListNodes.NodeType)
}
