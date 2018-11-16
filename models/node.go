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

// Node types.
const (
	PMMServerNodeType NodeType = "pmm-server" // FIXME remove

	BareMetalNodeType      NodeType = "bare-metal"
	VirtualMachineNodeType NodeType = "virtual-machine"
	ContainerNodeType      NodeType = "container"
	RemoteNodeType         NodeType = "remote"
	AWSRDSNodeType         NodeType = "aws-rds"
)

const RemoteNodeRegion string = "remote"

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

	Hostname *string `reform:"hostname"`
	Region   *string `reform:"region"`
}

// TODO remove types below

//reform:nodes
type AWSRDSNode struct {
	ID   uint32   `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier

	// Hostname *string `reform:"hostname"`
	Region *string `reform:"region"`
}

//reform:nodes
type RemoteNode struct {
	ID   uint32   `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier

	// Hostname *string `reform:"hostname"`
	Region *string `reform:"region"`
}
