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

import (
	"time"

	"gopkg.in/reform.v1"
)

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
	ID        uint32    `reform:"id,pk"`
	Type      NodeType  `reform:"type"`
	Name      string    `reform:"name"`
	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`

	Hostname *string `reform:"hostname"`
	Region   *string `reform:"region"`
}

func (nr *NodeRow) BeforeInsert() error {
	now := time.Now().Truncate(time.Microsecond).UTC()
	nr.CreatedAt = now
	nr.UpdatedAt = now
	return nil
}

func (nr *NodeRow) BeforeUpdate() error {
	now := time.Now().Truncate(time.Microsecond).UTC()
	nr.UpdatedAt = now
	return nil
}

func (nr *NodeRow) AfterFind() error {
	nr.CreatedAt = nr.CreatedAt.UTC()
	nr.UpdatedAt = nr.UpdatedAt.UTC()
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*NodeRow)(nil)
	_ reform.BeforeUpdater  = (*NodeRow)(nil)
	_ reform.AfterFinder    = (*NodeRow)(nil)
)

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
