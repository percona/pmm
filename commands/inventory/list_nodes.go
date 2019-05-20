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

	return &listNodesResult{
		Nodes: nodes,
	}, nil
}

// register command
var (
	ListNodes  = new(listNodeCommand)
	ListNodesC = inventoryListC.Command("nodes", "Show nodes in inventory.")
)
