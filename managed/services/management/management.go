// Copyright (C) 2024 Percona LLC
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

// Package management contains management business logic and APIs.
package management

import (
	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
)

func nodeID(tx *reform.TX, nodeID, nodeName string, addNodeParams *managementpb.AddNodeParams, address string) (string, error) {
	if err := validateNodeParamsOneOf(nodeID, nodeName, addNodeParams); err != nil {
		return "", err
	}
	switch {
	case nodeID != "":
		node, err := models.FindNodeByID(tx.Querier, nodeID)
		if err != nil {
			return "", err
		}
		if err = validateExistingNodeType(node); err != nil {
			return "", err
		}
		return node.NodeID, err
	case nodeName != "":
		node, err := models.FindNodeByName(tx.Querier, nodeName)
		if err != nil {
			return "", err
		}
		if err = validateExistingNodeType(node); err != nil {
			return "", err
		}
		return node.NodeID, err
	case addNodeParams != nil:
		if addNodeParams.NodeType != inventorypb.NodeType_REMOTE_NODE {
			return "", status.Errorf(codes.InvalidArgument, "add_node structure can be used only for remote nodes")
		}
		node, err := addNode(tx, addNodeParams, address)
		if err != nil {
			return "", err
		}
		return node.NodeID, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "node_id, node_name or add_node is required")
	}
}

func validateExistingNodeType(node *models.Node) error {
	switch node.NodeType {
	case models.GenericNodeType, models.ContainerNodeType:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "node_id or node_name can be used only for generic nodes or container nodes")
	}
}

func addNode(tx *reform.TX, addNodeParams *managementpb.AddNodeParams, address string) (*models.Node, error) {
	nodeType, err := nodeType(addNodeParams.NodeType)
	if err != nil {
		return nil, err
	}
	node, err := models.CreateNode(tx.Querier, nodeType, &models.CreateNodeParams{
		NodeName:      addNodeParams.NodeName,
		MachineID:     pointer.ToStringOrNil(addNodeParams.MachineId),
		Distro:        addNodeParams.Distro,
		NodeModel:     addNodeParams.NodeModel,
		AZ:            addNodeParams.Az,
		ContainerID:   pointer.ToStringOrNil(addNodeParams.ContainerId),
		ContainerName: pointer.ToStringOrNil(addNodeParams.ContainerName),
		CustomLabels:  addNodeParams.CustomLabels,
		Address:       address,
		Region:        pointer.ToStringOrNil(addNodeParams.Region),
	})
	if err != nil {
		return nil, err
	}
	return node, nil
}

func nodeType(inputNodeType inventorypb.NodeType) (models.NodeType, error) {
	switch inputNodeType {
	case inventorypb.NodeType_GENERIC_NODE:
		return models.GenericNodeType, nil
	case inventorypb.NodeType_CONTAINER_NODE:
		return models.ContainerNodeType, nil
	case inventorypb.NodeType_REMOTE_NODE:
		return models.RemoteNodeType, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "Unsupported Node type %q.", inputNodeType)
	}
}

func validateNodeParamsOneOf(nodeID, nodeName string, addNodeParams *managementpb.AddNodeParams) error {
	got := 0
	if nodeID != "" {
		got++
	}
	if nodeName != "" {
		got++
	}
	if addNodeParams != nil {
		got++
	}
	if got != 1 {
		return status.Errorf(codes.InvalidArgument, "expected only one param; node id, node name or register node params")
	}
	return nil
}

// PUSH or AUTO variant enables pushMode for the agent.
func isPushMode(variant managementpb.MetricsMode) bool {
	return variant == managementpb.MetricsMode_PUSH || variant == managementpb.MetricsMode_AUTO
}

// Automatically pick metrics mode.
func supportedMetricsMode(q *reform.Querier, metricsMode managementpb.MetricsMode, pmmAgentID string) (managementpb.MetricsMode, error) {
	if pmmAgentID == models.PMMServerAgentID && metricsMode == managementpb.MetricsMode_PUSH {
		return metricsMode, status.Errorf(codes.FailedPrecondition, "push metrics mode is not allowed for exporters running on pmm-server")
	}

	if metricsMode != managementpb.MetricsMode_AUTO {
		return metricsMode, nil
	}

	if pmmAgentID == models.PMMServerAgentID {
		return managementpb.MetricsMode_PULL, nil
	}

	pmmAgent, err := models.FindAgentByID(q, pmmAgentID)
	if err != nil {
		return metricsMode, err
	}

	if !models.IsPushMetricsSupported(pmmAgent.Version) {
		return managementpb.MetricsMode_PULL, nil
	}

	return metricsMode, nil
}
