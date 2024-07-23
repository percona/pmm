// Copyright (C) 2023 Percona LLC
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

package inventoryv1

//go-sumtype:decl Node

// Node is a common interface for all types of Nodes.
type Node interface {
	sealedNode()
}

// Ordered the same as NodeType enum.

func (*GenericNode) sealedNode()             {}
func (*ContainerNode) sealedNode()           {}
func (*RemoteNode) sealedNode()              {}
func (*RemoteRDSNode) sealedNode()           {}
func (*RemoteAzureDatabaseNode) sealedNode() {}
