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
	"strings"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/node"
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

func (cmd *RegisterCmd) RunCmd() (commands.Result, error) {
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
			AgentPassword: cmd.AgentPassword,

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
