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

package types

import "fmt"

// this list should be in sync with inventorypb/nodes.pb.go.
const (
	NodeTypeGenericNode             = "GENERIC_NODE"
	NodeTypeContainerNode           = "CONTAINER_NODE"
	NodeTypeRemoteNode              = "REMOTE_NODE"
	NodeTypeRemoteRDSNode           = "REMOTE_RDS_NODE"
	NodeTypeRemoteAzureDatabaseNode = "REMOTE_AZURE_DATABASE_NODE"
)

var nodeTypeNames = map[string]string{
	// no invalid
	NodeTypeGenericNode:             "Generic",
	NodeTypeContainerNode:           "Container",
	NodeTypeRemoteNode:              "Remote",
	NodeTypeRemoteRDSNode:           "Remote RDS",
	NodeTypeRemoteAzureDatabaseNode: "Remote Azure database",
}

// NodeTypeName returns human friendly node type to be used in reports.
func NodeTypeName(t string) string {
	res := nodeTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Node Type %s", t))
	}

	return res
}
