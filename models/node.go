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
	PMMServerNodeType  NodeType = "pmm-server"
	RDSNodeType        NodeType = "rds"
	PostgreSQLNodeType NodeType = "postgresql"
)

//reform:nodes
type Node struct {
	ID   int32    `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"`
}

//reform:nodes
type RDSNode struct {
	ID   int32    `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier

	Region string `reform:"region"` // not a pointer, see database structure
}

//reform:nodes
type PostgreSQLNode struct {
	ID   int32    `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier
}
