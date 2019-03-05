// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package handlers

import (
	"context"
	"fmt"

	api "github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/inventory"
)

type nodesServer struct {
	s *inventory.NodesService
}

// NewNodesServer returns Inventory API handler for managing Nodes.
func NewNodesServer(s *inventory.NodesService) api.NodesServer {
	return &nodesServer{
		s: s,
	}
}

// ListNodes returns a list of all Nodes.
func (s *nodesServer) ListNodes(ctx context.Context, req *api.ListNodesRequest) (*api.ListNodesResponse, error) {
	nodes, err := s.s.List(ctx)
	if err != nil {
		return nil, err
	}

	res := new(api.ListNodesResponse)
	for _, node := range nodes {
		switch node := node.(type) {
		case *api.GenericNode:
			res.Generic = append(res.Generic, node)
		case *api.ContainerNode:
			res.Container = append(res.Container, node)
		case *api.RemoteNode:
			res.Remote = append(res.Remote, node)
		case *api.RemoteAmazonRDSNode:
			res.RemoteAmazonRds = append(res.RemoteAmazonRds, node)
		default:
			panic(fmt.Errorf("unhandled inventory Node type %T", node))
		}
	}
	return res, nil
}

// GetNode returns a single Node by ID.
func (s *nodesServer) GetNode(ctx context.Context, req *api.GetNodeRequest) (*api.GetNodeResponse, error) {
	node, err := s.s.Get(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}

	res := new(api.GetNodeResponse)
	switch node := node.(type) {
	case *api.GenericNode:
		res.Node = &api.GetNodeResponse_Generic{Generic: node}
	case *api.ContainerNode:
		res.Node = &api.GetNodeResponse_Container{Container: node}
	case *api.RemoteNode:
		res.Node = &api.GetNodeResponse_Remote{Remote: node}
	case *api.RemoteAmazonRDSNode:
		res.Node = &api.GetNodeResponse_RemoteAmazonRds{RemoteAmazonRds: node}
	default:
		panic(fmt.Errorf("unhandled inventory Node type %T", node))
	}
	return res, nil
}

// AddGenericNode adds Generic Node.
func (s *nodesServer) AddGenericNode(ctx context.Context, req *api.AddGenericNodeRequest) (*api.AddGenericNodeResponse, error) {
	node, err := s.s.Add(ctx, models.GenericNodeType, req.NodeName, &req.Address, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddGenericNodeResponse{
		Generic: node.(*api.GenericNode),
	}
	return res, nil
}

// AddContainerNode adds Container Node.
func (s *nodesServer) AddContainerNode(ctx context.Context, req *api.AddContainerNodeRequest) (*api.AddContainerNodeResponse, error) {
	node, err := s.s.Add(ctx, models.ContainerNodeType, req.NodeName, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddContainerNodeResponse{
		Container: node.(*api.ContainerNode),
	}
	return res, nil
}

// AddRemoteNode adds Remote Node.
func (s *nodesServer) AddRemoteNode(ctx context.Context, req *api.AddRemoteNodeRequest) (*api.AddRemoteNodeResponse, error) {
	node, err := s.s.Add(ctx, models.RemoteNodeType, req.NodeName, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddRemoteNodeResponse{
		Remote: node.(*api.RemoteNode),
	}
	return res, nil
}

// AddRemoteAmazonRDSNode adds Amazon (AWS) RDS remote Node.
func (s *nodesServer) AddRemoteAmazonRDSNode(ctx context.Context, req *api.AddRemoteAmazonRDSNodeRequest) (*api.AddRemoteAmazonRDSNodeResponse, error) {
	node, err := s.s.Add(ctx, models.RemoteAmazonRDSNodeType, req.NodeName, &req.Instance, &req.Region)
	if err != nil {
		return nil, err
	}

	res := &api.AddRemoteAmazonRDSNodeResponse{
		RemoteAmazonRds: node.(*api.RemoteAmazonRDSNode),
	}
	return res, nil
}

// ChangeGenericNode changes Generic Node.
func (s *nodesServer) ChangeGenericNode(ctx context.Context, req *api.ChangeGenericNodeRequest) (*api.ChangeGenericNodeResponse, error) {
	node, err := s.s.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeGenericNodeResponse{
		Generic: node.(*api.GenericNode),
	}
	return res, nil
}

// ChangeContainerNode changes Container Node.
func (s *nodesServer) ChangeContainerNode(ctx context.Context, req *api.ChangeContainerNodeRequest) (*api.ChangeContainerNodeResponse, error) {
	node, err := s.s.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeContainerNodeResponse{
		Container: node.(*api.ContainerNode),
	}
	return res, nil
}

// ChangeRemoteNode changes Remote Node.
func (s *nodesServer) ChangeRemoteNode(ctx context.Context, req *api.ChangeRemoteNodeRequest) (*api.ChangeRemoteNodeResponse, error) {
	node, err := s.s.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeRemoteNodeResponse{
		Remote: node.(*api.RemoteNode),
	}
	return res, nil
}

// ChangeRemoteAmazonRDSNode changes Amazon (AWS) RDS remote Node.
func (s *nodesServer) ChangeRemoteAmazonRDSNode(ctx context.Context, req *api.ChangeRemoteAmazonRDSNodeRequest) (*api.ChangeRemoteAmazonRDSNodeResponse, error) {
	node, err := s.s.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeRemoteAmazonRDSNodeResponse{
		RemoteAmazonRds: node.(*api.RemoteAmazonRDSNode),
	}
	return res, nil
}

// RemoveNode removes Node without any Agents and Services.
func (s *nodesServer) RemoveNode(ctx context.Context, req *api.RemoveNodeRequest) (*api.RemoveNodeResponse, error) {
	if err := s.s.Remove(ctx, req.NodeId); err != nil {
		return nil, err
	}

	return new(api.RemoveNodeResponse), nil
}
