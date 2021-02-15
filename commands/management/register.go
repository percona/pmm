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

package management

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/percona/pmm/utils/nodeinfo"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
)

var registerResultT = commands.ParseTemplate(`
pmm-agent registered.
pmm-agent ID: {{ .PMMAgent.AgentID }}
Node ID     : {{ .PMMAgent.RunsOnNodeID }}
`)

type registerResult struct {
	GenericNode   *node.RegisterNodeOKBodyGenericNode   `json:"generic_node"`
	ContainerNode *node.RegisterNodeOKBodyContainerNode `json:"container_node"`
	PMMAgent      *node.RegisterNodeOKBodyPMMAgent      `json:"pmm_agent"`
}

func (res *registerResult) Result() {}

func (res *registerResult) String() string {
	return commands.RenderTemplate(registerResultT, res)
}

type registerCommand struct {
	NodeType          string
	NodeName          string
	MachineID         string
	Distro            string
	ContainerID       string
	ContainerName     string
	NodeModel         string
	Region            string
	Az                string
	CustomLabels      string
	Address           string
	MetricsMode       string
	DisableCollectors string

	Force bool
}

func (cmd *registerCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	params := &node.RegisterNodeParams{
		Body: node.RegisterNodeBody{
			NodeType:      pointer.ToString(allNodeTypes[cmd.NodeType]),
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

			Reregister:        cmd.Force,
			MetricsMode:       pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors: commands.ParseDisableCollectors(cmd.DisableCollectors),
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.Node.RegisterNode(params)
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

	registerNodeTypeKeys := []string{"generic", "container"} // "remote" Node can't be registered with that API
	nodeTypeDefault := "generic"
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(registerNodeTypeKeys, ", "), nodeTypeDefault)
	RegisterC.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&Register.NodeType, registerNodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s)", hostname)
	RegisterC.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&Register.NodeName)

	var defaultMachineID string
	if nodeinfo.MachineID != "" {
		defaultMachineID = "/machine_id/" + nodeinfo.MachineID
	}
	RegisterC.Flag("machine-id", "Node machine-id (default is autodetected)").Default(defaultMachineID).StringVar(&Register.MachineID)
	RegisterC.Flag("distro", "Node OS distribution (default is autodetected)").Default(nodeinfo.Distro).StringVar(&Register.Distro)
	RegisterC.Flag("container-id", "Container ID").StringVar(&Register.ContainerID)
	RegisterC.Flag("container-name", "Container name").StringVar(&Register.ContainerName)
	RegisterC.Flag("node-model", "Node model").StringVar(&Register.NodeModel)
	RegisterC.Flag("region", "Node region").StringVar(&Register.Region)
	RegisterC.Flag("az", "Node availability zone").StringVar(&Register.Az)
	RegisterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&Register.CustomLabels)

	RegisterC.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist").BoolVar(&Register.Force)
	RegisterC.Flag("metrics-mode", "Metrics flow mode, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server.").Default("auto").EnumVar(&Register.MetricsMode, "auto", "pull", "push")
	RegisterC.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter").StringVar(&Register.DisableCollectors)
}
