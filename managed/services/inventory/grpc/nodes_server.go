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

package grpc

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
)

type nodesServer struct {
	svc *inventory.NodesService

	inventoryv1.UnimplementedNodesServiceServer
}

// NewNodesServer returns Inventory API handler for managing Nodes.
func NewNodesServer(svc *inventory.NodesService) inventoryv1.NodesServiceServer { //nolint:ireturn
	return &nodesServer{svc: svc}
}

var nodeTypes = map[inventoryv1.NodeType]models.NodeType{
	inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE:               models.GenericNodeType,
	inventoryv1.NodeType_NODE_TYPE_CONTAINER_NODE:             models.ContainerNodeType,
	inventoryv1.NodeType_NODE_TYPE_REMOTE_NODE:                models.RemoteNodeType,
	inventoryv1.NodeType_NODE_TYPE_REMOTE_RDS_NODE:            models.RemoteRDSNodeType,
	inventoryv1.NodeType_NODE_TYPE_REMOTE_AZURE_DATABASE_NODE: models.RemoteAzureDatabaseNodeType,
}

func nodeType(nodeType inventoryv1.NodeType) *models.NodeType {
	if nodeType == inventoryv1.NodeType_NODE_TYPE_UNSPECIFIED {
		return nil
	}
	result := nodeTypes[nodeType]
	return &result
}

// ListNodes returns a list of all Nodes.
func (s *nodesServer) ListNodes(ctx context.Context, req *inventoryv1.ListNodesRequest) (*inventoryv1.ListNodesResponse, error) {
	filters := models.NodeFilters{
		NodeType: nodeType(req.NodeType),
	}
	nodes, err := s.svc.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ListNodesResponse{}
	for _, node := range nodes {
		switch node := node.(type) {
		case *inventoryv1.GenericNode:
			res.Generic = append(res.Generic, node)
		case *inventoryv1.ContainerNode:
			res.Container = append(res.Container, node)
		case *inventoryv1.RemoteNode:
			res.Remote = append(res.Remote, node)
		case *inventoryv1.RemoteRDSNode:
			res.RemoteRds = append(res.RemoteRds, node)
		case *inventoryv1.RemoteAzureDatabaseNode:
			res.RemoteAzureDatabase = append(res.RemoteAzureDatabase, node)
		default:
			panic(fmt.Errorf("unhandled inventory Node type %T", node))
		}
	}
	return res, nil
}

// GetNode returns a single Node by ID.
func (s *nodesServer) GetNode(ctx context.Context, req *inventoryv1.GetNodeRequest) (*inventoryv1.GetNodeResponse, error) {
	node, err := s.svc.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.GetNodeResponse{}
	switch node := node.(type) {
	case *inventoryv1.GenericNode:
		res.Node = &inventoryv1.GetNodeResponse_Generic{Generic: node}
	case *inventoryv1.ContainerNode:
		res.Node = &inventoryv1.GetNodeResponse_Container{Container: node}
	case *inventoryv1.RemoteNode:
		res.Node = &inventoryv1.GetNodeResponse_Remote{Remote: node}
	case *inventoryv1.RemoteRDSNode:
		res.Node = &inventoryv1.GetNodeResponse_RemoteRds{RemoteRds: node}
	case *inventoryv1.RemoteAzureDatabaseNode:
		res.Node = &inventoryv1.GetNodeResponse_RemoteAzureDatabase{RemoteAzureDatabase: node}
	default:
		panic(fmt.Errorf("unhandled inventory Node type %T", node))
	}
	return res, nil
}

func (s *nodesServer) AddNode(ctx context.Context, req *inventoryv1.AddNodeRequest) (*inventoryv1.AddNodeResponse, error) {
	return s.svc.AddNode(ctx, req)
}

// RemoveNode removes Node.
func (s *nodesServer) RemoveNode(ctx context.Context, req *inventoryv1.RemoveNodeRequest) (*empty.Empty, error) {
	if err := s.svc.Remove(ctx, req.NodeId, req.Force); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
