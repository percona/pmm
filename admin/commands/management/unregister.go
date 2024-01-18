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
	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
)

type unregisterResult struct {
	NodeID   string
	NodeName string
}

var unregisterNodeResultT = commands.ParseTemplate(`
Node with ID {{ .NodeID }} and name {{ .NodeName }} unregistered.
`)

func (res *unregisterResult) Result() {}

func (res *unregisterResult) String() string {
	return commands.RenderTemplate(unregisterNodeResultT, res)
}

// UnregisterCommand is used by Kong for CLI flags and commands.
type UnregisterCommand struct {
	Force    bool   `help:"Remove this node with all dependencies"`
	NodeName string `help:"Node name (autodetected default: ${hostname})"`
}

// RunCmd runs the command for UnregisterCommand.
func (cmd *UnregisterCommand) RunCmd() (commands.Result, error) {
	var nodeName string
	var nodeID string
	var err error
	if cmd.NodeName != "" {
		nodeName = cmd.NodeName
		nodeID, err = nodeIDFromNodeName(cmd.NodeName)
		if err != nil {
			return nil, err
		}
	} else {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}

		nodeID = status.NodeID
		node, err := client.Default.Nodes.GetNode(&nodes.GetNodeParams{
			Context: commands.Ctx,
			Body: nodes.GetNodeBody{
				NodeID: nodeID,
			},
		})
		if err != nil {
			return nil, err
		}
		nodeName, err = helpers.GetNodeName(node.Payload)
		if err != nil {
			return nil, err
		}
	}

	params := &nodes.RemoveNodeParams{
		Body: nodes.RemoveNodeBody{
			NodeID: nodeID,
			Force:  cmd.Force,
		},
		Context: commands.Ctx,
	}

	_, err = client.Default.Nodes.RemoveNode(params)
	if err != nil {
		return nil, err
	}

	return &unregisterResult{
		NodeID:   nodeID,
		NodeName: nodeName,
	}, nil
}

func nodeIDFromNodeName(nodeName string) (string, error) {
	listNodes, err := client.Default.Nodes.ListNodes(nil)
	if err != nil {
		return "", err
	}
	for _, node := range listNodes.Payload.Generic {
		if node.NodeName == nodeName {
			return node.NodeID, nil
		}
	}
	for _, node := range listNodes.Payload.Remote {
		if node.NodeName == nodeName {
			return node.NodeID, nil
		}
	}
	for _, node := range listNodes.Payload.Container {
		if node.NodeName == nodeName {
			return node.NodeID, nil
		}
	}
	for _, node := range listNodes.Payload.RemoteAzureDatabase {
		if node.NodeName == nodeName {
			return node.NodeID, nil
		}
	}
	for _, node := range listNodes.Payload.RemoteRDS {
		if node.NodeName == nodeName {
			return node.NodeID, nil
		}
	}
	return "", errors.Errorf("node %s is not found", nodeName)
}
