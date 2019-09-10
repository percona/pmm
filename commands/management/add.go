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
