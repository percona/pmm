// Copyright (C) 2023 Percona LLC
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
{{ if .Warning }}
Warning: {{ .Warning }}
{{- end -}}
`)

type registerResult struct {
	GenericNode   *node.RegisterNodeOKBodyGenericNode   `json:"generic_node"`
	ContainerNode *node.RegisterNodeOKBodyContainerNode `json:"container_node"`
	PMMAgent      *node.RegisterNodeOKBodyPMMAgent      `json:"pmm_agent"`
	Warning       string                                `json:"warning"`
}

func (res *registerResult) Result() {}

func (res *registerResult) String() string {
	return commands.RenderTemplate(registerResultT, res)
}

// RegisterCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type RegisterCommand struct {
	Address           string            `name:"node-address" arg:"" default:"${nodeIp}" help:"Node address (autodetected default: ${nodeIp})"`
	NodeType          string            `arg:"" enum:"generic,container" default:"generic" help:"Node type, one of: generic, container (default: generic)"`
	NodeName          string            `arg:"" default:"${hostname}" help:"Node name (autodetected default: ${hostname})"`
	MachineID         string            `default:"${defaultMachineID}" help:"Node machine-id (default is autodetected)"`
	Distro            string            `default:"${distro}" help:"Node OS distribution (default is autodetected)"`
	ContainerID       string            `help:"Container ID"`
	ContainerName     string            `help:"Container name"`
	NodeModel         string            `help:"Node model"`
	Region            string            `help:"Node region"`
	Az                string            `help:"Node availability zone"`
	CustomLabels      map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	AgentPassword     string            `help:"Custom password for /metrics endpoint"`
	Force             bool              `help:"Re-register Node"`
	MetricsMode       string            `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"`
	DisableCollectors []string          `help:"Comma-separated list of collector names to exclude from exporter"`
}

// RunCmd runs the command for RegisterCommand.
func (cmd *RegisterCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

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
		Warning:       resp.Payload.Warning,
	}, nil
}
