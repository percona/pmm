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

// NodesServer handles Inventory API requests to manage Nodes.
type NodesServer struct {
	Nodes *inventory.NodesService
}

// ListNodes returns a list of all Nodes.
func (s *NodesServer) ListNodes(ctx context.Context, req *api.ListNodesRequest) (*api.ListNodesResponse, error) {
	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return nil, err
	}

	res := new(api.ListNodesResponse)
	for _, node := range nodes {
		switch node := node.(type) {
		case *api.GenericNode:
			res.Generic = append(res.Generic, node)
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
func (s *NodesServer) GetNode(ctx context.Context, req *api.GetNodeRequest) (*api.GetNodeResponse, error) {
	node, err := s.Nodes.Get(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}

	res := new(api.GetNodeResponse)
	switch node := node.(type) {
	case *api.GenericNode:
		res.Node = &api.GetNodeResponse_Generic{Generic: node}
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
func (s *NodesServer) AddGenericNode(ctx context.Context, req *api.AddGenericNodeRequest) (*api.AddGenericNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.GenericNodeType, req.NodeName, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddGenericNodeResponse{
		Generic: node.(*api.GenericNode),
	}
	return res, nil
}

// AddContainerNode adds Container Node.
func (s *NodesServer) AddContainerNode(ctx context.Context, req *api.AddContainerNodeRequest) (*api.AddContainerNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.ContainerNodeType, req.NodeName, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddContainerNodeResponse{
		Container: node.(*api.ContainerNode),
	}
	return res, nil
}

// AddRemoteNode adds Remote Node.
func (s *NodesServer) AddRemoteNode(ctx context.Context, req *api.AddRemoteNodeRequest) (*api.AddRemoteNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.RemoteNodeType, req.NodeName, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddRemoteNodeResponse{
		Remote: node.(*api.RemoteNode),
	}
	return res, nil
}

// AddRemoteAmazonRDSNode adds Amazon (AWS) RDS remote Node.
func (s *NodesServer) AddRemoteAmazonRDSNode(ctx context.Context, req *api.AddRemoteAmazonRDSNodeRequest) (*api.AddRemoteAmazonRDSNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.RemoteAmazonRDSNodeType, req.NodeName, &req.Instance, &req.Region)
	if err != nil {
		return nil, err
	}

	res := &api.AddRemoteAmazonRDSNodeResponse{
		RemoteAmazonRds: node.(*api.RemoteAmazonRDSNode),
	}
	return res, nil
}

// ChangeGenericNode changes Generic Node.
func (s *NodesServer) ChangeGenericNode(ctx context.Context, req *api.ChangeGenericNodeRequest) (*api.ChangeGenericNodeResponse, error) {
	node, err := s.Nodes.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeGenericNodeResponse{
		Generic: node.(*api.GenericNode),
	}
	return res, nil
}

// ChangeContainerNode changes Container Node.
func (s *NodesServer) ChangeContainerNode(ctx context.Context, req *api.ChangeContainerNodeRequest) (*api.ChangeContainerNodeResponse, error) {
	node, err := s.Nodes.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeContainerNodeResponse{
		Container: node.(*api.ContainerNode),
	}
	return res, nil
}

// ChangeRemoteNode changes Remote Node.
func (s *NodesServer) ChangeRemoteNode(ctx context.Context, req *api.ChangeRemoteNodeRequest) (*api.ChangeRemoteNodeResponse, error) {
	node, err := s.Nodes.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeRemoteNodeResponse{
		Remote: node.(*api.RemoteNode),
	}
	return res, nil
}

// ChangeRemoteAmazonRDSNode changes Amazon (AWS) RDS remote Node.
func (s *NodesServer) ChangeRemoteAmazonRDSNode(ctx context.Context, req *api.ChangeRemoteAmazonRDSNodeRequest) (*api.ChangeRemoteAmazonRDSNodeResponse, error) {
	node, err := s.Nodes.Change(ctx, req.NodeId, req.NodeName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeRemoteAmazonRDSNodeResponse{
		RemoteAmazonRds: node.(*api.RemoteAmazonRDSNode),
	}
	return res, nil
}

// RemoveNode removes Node without any Agents and Services.
func (s *NodesServer) RemoveNode(ctx context.Context, req *api.RemoveNodeRequest) (*api.RemoveNodeResponse, error) {
	if err := s.Nodes.Remove(ctx, req.NodeId); err != nil {
		return nil, err
	}

	return new(api.RemoveNodeResponse), nil
}

// check interfaces
var (
	_ api.NodesServer = (*NodesServer)(nil)
)
