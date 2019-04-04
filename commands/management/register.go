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
	"text/template"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
)

var (
	nodeTypes = map[string]string{
		"generic":   node.RegisterBodyNodeTypeGENERICNODE,
		"container": node.RegisterBodyNodeTypeCONTAINERNODE,
	}
)

var registerResultT = template.Must(template.New("").Parse(strings.TrimSpace(`
pmm-agent registered.
pmm-agent ID: {{ .PMMAgent.AgentID }}
Node ID     : {{ .PMMAgent.RunsOnNodeID }}
`)))

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
	ContainerID   string
	ContainerName string
}

func (cmd *registerCommand) Run() (commands.Result, error) {
	params := &node.RegisterParams{
		Body: node.RegisterBody{
			NodeName:      cmd.NodeName,
			NodeType:      pointer.ToString(nodeTypes[cmd.NodeType]),
			MachineID:     cmd.MachineID,
			ContainerID:   cmd.ContainerID,
			ContainerName: cmd.ContainerName,
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

// register commands
var (
	Register  = new(registerCommand)
	RegisterC = kingpin.Command("register", "Register current Node at PMM Server.")
)

func init() {
	nodeTypeKeys := make([]string, 0, len(nodeTypes))
	for k := range nodeTypes {
		nodeTypeKeys = append(nodeTypeKeys, k)
	}
	nodeTypeDefault := nodeTypeKeys[0]
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s. Default: %s.", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	RegisterC.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&Register.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name. Default: %s.", hostname)
	RegisterC.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&Register.NodeName)

	RegisterC.Flag("machine-id", "Node machine-id.").StringVar(&Register.MachineID)
	RegisterC.Flag("container-id", "Container ID.").StringVar(&Register.ContainerID)
	RegisterC.Flag("container-name", "Container name.").StringVar(&Register.ContainerName)
}
