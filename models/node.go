// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package models

//go:generate reform

type NodeType string

// Node types
const (
	PMMServerNodeType NodeType = "pmm-server" // FIXME remove

	BareMetalNodeType      NodeType = "bare-metal"
	VirtualMachineNodeType NodeType = "virtual-machine"
	ContainerNodeType      NodeType = "container"
	RemoteNodeType         NodeType = "remote"
	RDSNodeType            NodeType = "rds"
)

const RemoteNodeRegion string = "remote"

/*
func InventoryNodeType(m NodeType) inventory.NodeType {
	switch m {
	default:
		panic(fmt.Errorf("unhandled models node type %s", m))
	}
}

func ModelsNodeType(i inventory.NodeType) NodeType {
	switch i {
	case inventory.NodeType_BARE_METAL:
		return BareMetalNodeType
	case inventory.NodeType_VIRTUAL_MACHINE:
		return VirtualMachineNodeType
	case inventory.NodeType_CONTAINER:
		return ContainerNodeType
	case inventory.NodeType_REMOTE:
		return RemoteNodeType
	case inventory.NodeType_RDS:
		return RDSNodeType
	default:
		panic(fmt.Errorf("unhandled inventory node type %s (%d)", i, i))
	}
}
*/

//reform:nodes
type Node struct {
	ID   uint32   `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"`
}

//reform:nodes
type NodeRow struct {
	ID   uint32   `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"`

	Region   string  `reform:"region"` // not a pointer, see database structure
	Hostname *string `reform:"hostname"`
}

// TODO remove types below

//reform:nodes
type RDSNode struct {
	ID   uint32   `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier

	Region string `reform:"region"` // not a pointer, see database structure
}

//reform:nodes
type RemoteNode struct {
	ID   uint32   `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier

	Region string `reform:"region"` // not a pointer, see database structure
}
