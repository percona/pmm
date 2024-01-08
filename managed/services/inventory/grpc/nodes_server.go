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

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
)

type nodesServer struct {
	svc *inventory.NodesService

	inventorypb.UnimplementedNodesServer
}

// NewNodesServer returns Inventory API handler for managing Nodes.
func NewNodesServer(svc *inventory.NodesService) inventorypb.NodesServer { //nolint:ireturn
	return &nodesServer{svc: svc}
}

var nodeTypes = map[inventorypb.NodeType]models.NodeType{
	inventorypb.NodeType_GENERIC_NODE:               models.GenericNodeType,
	inventorypb.NodeType_CONTAINER_NODE:             models.ContainerNodeType,
	inventorypb.NodeType_REMOTE_NODE:                models.RemoteNodeType,
	inventorypb.NodeType_REMOTE_RDS_NODE:            models.RemoteRDSNodeType,
	inventorypb.NodeType_REMOTE_AZURE_DATABASE_NODE: models.RemoteAzureDatabaseNodeType,
}

func nodeType(nodeType inventorypb.NodeType) *models.NodeType {
	if nodeType == inventorypb.NodeType_NODE_TYPE_INVALID {
		return nil
	}
	result := nodeTypes[nodeType]
	return &result
}

// ListNodes returns a list of all Nodes.
func (s *nodesServer) ListNodes(ctx context.Context, req *inventorypb.ListNodesRequest) (*inventorypb.ListNodesResponse, error) {
	filters := models.NodeFilters{
		NodeType: nodeType(req.NodeType),
	}
	nodes, err := s.svc.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ListNodesResponse{}
	for _, node := range nodes {
		switch node := node.(type) {
		case *inventorypb.GenericNode:
			res.Generic = append(res.Generic, node)
		case *inventorypb.ContainerNode:
			res.Container = append(res.Container, node)
		case *inventorypb.RemoteNode:
			res.Remote = append(res.Remote, node)
		case *inventorypb.RemoteRDSNode:
			res.RemoteRds = append(res.RemoteRds, node)
		case *inventorypb.RemoteAzureDatabaseNode:
			res.RemoteAzureDatabase = append(res.RemoteAzureDatabase, node)
		default:
			panic(fmt.Errorf("unhandled inventory Node type %T", node))
		}
	}
	return res, nil
}

// GetNode returns a single Node by ID.
func (s *nodesServer) GetNode(ctx context.Context, req *inventorypb.GetNodeRequest) (*inventorypb.GetNodeResponse, error) {
	node, err := s.svc.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.GetNodeResponse{}
	switch node := node.(type) {
	case *inventorypb.GenericNode:
		res.Node = &inventorypb.GetNodeResponse_Generic{Generic: node}
	case *inventorypb.ContainerNode:
		res.Node = &inventorypb.GetNodeResponse_Container{Container: node}
	case *inventorypb.RemoteNode:
		res.Node = &inventorypb.GetNodeResponse_Remote{Remote: node}
	case *inventorypb.RemoteRDSNode:
		res.Node = &inventorypb.GetNodeResponse_RemoteRds{RemoteRds: node}
	case *inventorypb.RemoteAzureDatabaseNode:
		res.Node = &inventorypb.GetNodeResponse_RemoteAzureDatabase{RemoteAzureDatabase: node}
	default:
		panic(fmt.Errorf("unhandled inventory Node type %T", node))
	}
	return res, nil
}

func (s *nodesServer) AddNode(ctx context.Context, req *inventorypb.AddNodeRequest) (*inventorypb.AddNodeResponse, error) {
	return s.svc.AddNode(ctx, req)
}

// AddGenericNode adds Generic Node.
func (s *nodesServer) AddGenericNode(
	ctx context.Context,
	req *inventorypb.AddGenericNodeRequest,
) (*inventorypb.AddGenericNodeResponse, error) { //nolint:staticcheck
	node, err := s.svc.AddGenericNode(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddGenericNodeResponse{Generic: node} //nolint:staticcheck
	return res, nil
}

// AddContainerNode adds Container Node.
func (s *nodesServer) AddContainerNode(
	ctx context.Context,
	req *inventorypb.AddContainerNodeRequest,
) (*inventorypb.AddContainerNodeResponse, error) { //nolint:staticcheck
	node, err := s.svc.AddContainerNode(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddContainerNodeResponse{Container: node} //nolint:staticcheck
	return res, nil
}

// AddRemoteNode adds Remote Node.
func (s *nodesServer) AddRemoteNode(
	ctx context.Context,
	req *inventorypb.AddRemoteNodeRequest,
) (*inventorypb.AddRemoteNodeResponse, error) { //nolint:staticcheck
	node, err := s.svc.AddRemoteNode(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddRemoteNodeResponse{Remote: node} //nolint:staticcheck
	return res, nil
}

// AddRemoteRDSNode adds Remote RDS Node.
func (s *nodesServer) AddRemoteRDSNode(
	ctx context.Context,
	req *inventorypb.AddRemoteRDSNodeRequest,
) (*inventorypb.AddRemoteRDSNodeResponse, error) { //nolint:staticcheck
	node, err := s.svc.AddRemoteRDSNode(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddRemoteRDSNodeResponse{RemoteRds: node} //nolint:staticcheck
	return res, nil
}

// AddRemoteAzureDatabaseNode adds Remote Azure database Node.
func (s *nodesServer) AddRemoteAzureDatabaseNode(
	ctx context.Context,
	req *inventorypb.AddRemoteAzureDatabaseNodeRequest,
) (*inventorypb.AddRemoteAzureDatabaseNodeResponse, error) { //nolint:staticcheck
	node, err := s.svc.AddRemoteAzureDatabaseNode(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddRemoteAzureDatabaseNodeResponse{RemoteAzureDatabase: node} //nolint:staticcheck
	return res, nil
}

// RemoveNode removes Node.
func (s *nodesServer) RemoveNode(ctx context.Context, req *inventorypb.RemoveNodeRequest) (*inventorypb.RemoveNodeResponse, error) {
	if err := s.svc.Remove(ctx, req.NodeId, req.Force); err != nil {
		return nil, err
	}

	return &inventorypb.RemoveNodeResponse{}, nil
}
