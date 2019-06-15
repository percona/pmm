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

package management

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/percona/pmm/nodeinfo"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
)

var (
	nodeTypes = map[string]string{
		"generic":   node.RegisterBodyNodeTypeGENERICNODE,
		"container": node.RegisterBodyNodeTypeCONTAINERNODE,
	}
	nodeTypeKeys = []string{"generic", "container"}
)

var registerResultT = commands.ParseTemplate(`
pmm-agent registered.
pmm-agent ID: {{ .PMMAgent.AgentID }}
Node ID     : {{ .PMMAgent.RunsOnNodeID }}
`)

type registerResult struct {
	GenericNode   *node.RegisterOKBodyGenericNode   `json:"generic_node"`
	ContainerNode *node.RegisterOKBodyContainerNode `json:"container_node"`
	PMMAgent      *node.RegisterOKBodyPMMAgent      `json:"pmm_agent"`
}

func (res *registerResult) Result() {}

func (res *registerResult) String() string {
	return commands.RenderTemplate(registerResultT, res)
}

type registerCommand struct {
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

func (cmd *registerCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &node.RegisterParams{
		Body: node.RegisterBody{
			NodeType:      pointer.ToString(nodeTypes[cmd.NodeType]),
			NodeName:      cmd.NodeName,
			MachineID:     cmd.MachineID,
			Distro:        cmd.Distro,
			ContainerID:   cmd.ContainerID,
			ContainerName: cmd.ContainerName,
			NodeModel:     cmd.NodeModel,
			Region:        cmd.Region,
			Az:            cmd.Az,
			CustomLabels:  customLabels,
			Address:       cmd.Address,

			Reregister: cmd.Force,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.Node.Register(params)
	if err != nil {
		return nil, err
	}

	return &registerResult{
		GenericNode:   resp.Payload.GenericNode,
		ContainerNode: resp.Payload.ContainerNode,
		PMMAgent:      resp.Payload.PMMAgent,
	}, nil
}

// register command
var (
	Register  = new(registerCommand)
	RegisterC = kingpin.Command("register", "Register current Node at PMM Server")
)

func init() {
	nodeinfo := nodeinfo.Get()

	if nodeinfo.PublicAddress == "" {
		RegisterC.Arg("node-address", "Node address").Required().StringVar(&Register.Address)
	} else {
		help := fmt.Sprintf("Node address (autodetected default: %s)", nodeinfo.PublicAddress)
		RegisterC.Arg("node-address", help).Default(nodeinfo.PublicAddress).StringVar(&Register.Address)
	}

	nodeTypeDefault := nodeTypeKeys[0]
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	RegisterC.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&Register.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s)", hostname)
	RegisterC.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&Register.NodeName)

	RegisterC.Flag("machine-id", "Node machine-id (default is autodetected)").Default(nodeinfo.MachineID).StringVar(&Register.MachineID)
	RegisterC.Flag("distro", "Node OS distribution (default is autodetected)").Default(nodeinfo.Distro).StringVar(&Register.Distro)
	RegisterC.Flag("container-id", "Container ID").StringVar(&Register.ContainerID)
	RegisterC.Flag("container-name", "Container name").StringVar(&Register.ContainerName)
	RegisterC.Flag("node-model", "Node model").StringVar(&Register.NodeModel)
	RegisterC.Flag("region", "Node region").StringVar(&Register.Region)
	RegisterC.Flag("az", "Node availability zone").StringVar(&Register.Az)
	RegisterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&Register.CustomLabels)

	RegisterC.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist").BoolVar(&Register.Force)
}
