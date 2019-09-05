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

// Package management provides management commands.
package management

import (
	"fmt"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
)

// register command
var (
	AddC = kingpin.Command("add", "Add Service to monitoring")
)

type addNodeParams struct {
	NodeType      string
	NodeName      string
	MachineID     string
	Distro        string
	ContainerID   string
	ContainerName string
	NodeModel     string
	Region        string
	Az            string
	CustomLabels  string
	Address       string

	Force bool
}

func addNodeFlags(cmd *kingpin.CmdClause, params *addNodeParams) {
	cmd.Arg("node-address", "Node address").StringVar(&params.Address)
	nodeTypeDefault := "remote"
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	cmd.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&params.NodeType, nodeTypeKeys...)
	cmd.Flag("node-machine-id", "Node machine-id (default is autodetected)").StringVar(&params.MachineID)
	cmd.Flag("node-distro", "Node OS distribution (default is autodetected)").StringVar(&params.Distro)
	cmd.Flag("node-container-id", "Container ID").StringVar(&params.ContainerID)
	cmd.Flag("node-container-name", "Container name").StringVar(&params.ContainerName)
	cmd.Flag("node-model", "Node model").StringVar(&params.NodeModel)
	cmd.Flag("node-region", "Node region").StringVar(&params.Region)
	cmd.Flag("node-az", "Node availability zone").StringVar(&params.Az)
	cmd.Flag("node-custom-labels", "Custom user-assigned labels").StringVar(&params.CustomLabels)
}
