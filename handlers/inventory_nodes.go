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

	"github.com/AlekSi/pointer"
	inventorypb "github.com/percona/pmm/api/inventory"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/inventory"
)

type nodesServer struct {
	db *reform.DB
	s  *inventory.NodesService
}

// NewNodesServer returns Inventory API handler for managing Nodes.
func NewNodesServer(db *reform.DB, s *inventory.NodesService) inventorypb.NodesServer {
	return &nodesServer{
		db: db,
		s:  s,
	}
}

// ListNodes returns a list of all Nodes.
func (s *nodesServer) ListNodes(ctx context.Context, req *inventorypb.ListNodesRequest) (*inventorypb.ListNodesResponse, error) {
	nodes, err := s.s.List(ctx, s.db.Querier)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.ListNodesResponse)
	for _, node := range nodes {
		switch node := node.(type) {
		case *inventorypb.GenericNode:
			res.Generic = append(res.Generic, node)
		case *inventorypb.ContainerNode:
			res.Container = append(res.Container, node)
		case *inventorypb.RemoteNode:
			res.Remote = append(res.Remote, node)
		case *inventorypb.RemoteAmazonRDSNode:
			res.RemoteAmazonRds = append(res.RemoteAmazonRds, node)
		default:
			panic(fmt.Errorf("unhandled inventory Node type %T", node))
		}
	}
	return res, nil
}

// GetNode returns a single Node by ID.
func (s *nodesServer) GetNode(ctx context.Context, req *inventorypb.GetNodeRequest) (*inventorypb.GetNodeResponse, error) {
	node, err := s.s.Get(ctx, s.db.Querier, req.NodeId)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.GetNodeResponse)
	switch node := node.(type) {
	case *inventorypb.GenericNode:
		res.Node = &inventorypb.GetNodeResponse_Generic{Generic: node}
	case *inventorypb.ContainerNode:
		res.Node = &inventorypb.GetNodeResponse_Container{Container: node}
	case *inventorypb.RemoteNode:
		res.Node = &inventorypb.GetNodeResponse_Remote{Remote: node}
	case *inventorypb.RemoteAmazonRDSNode:
		res.Node = &inventorypb.GetNodeResponse_RemoteAmazonRds{RemoteAmazonRds: node}
	default:
		panic(fmt.Errorf("unhandled inventory Node type %T", node))
	}
	return res, nil
}

// AddGenericNode adds Generic Node.
func (s *nodesServer) AddGenericNode(ctx context.Context, req *inventorypb.AddGenericNodeRequest) (*inventorypb.AddGenericNodeResponse, error) {
	params := &inventory.AddNodeParams{
		NodeType: models.GenericNodeType,
		Name:     req.NodeName,
		Address:  pointer.ToStringOrNil(req.Address),
	}
	node, err := s.s.Add(ctx, s.db.Querier, params)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddGenericNodeResponse{
		Generic: node.(*inventorypb.GenericNode),
	}
	return res, nil
}

// AddContainerNode adds Container Node.
func (s *nodesServer) AddContainerNode(ctx context.Context, req *inventorypb.AddContainerNodeRequest) (*inventorypb.AddContainerNodeResponse, error) {
	params := &inventory.AddNodeParams{
		NodeType: models.ContainerNodeType,
		Name:     req.NodeName,
	}
	node, err := s.s.Add(ctx, s.db.Querier, params)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddContainerNodeResponse{
		Container: node.(*inventorypb.ContainerNode),
	}
	return res, nil
}

// AddRemoteNode adds Remote Node.
func (s *nodesServer) AddRemoteNode(ctx context.Context, req *inventorypb.AddRemoteNodeRequest) (*inventorypb.AddRemoteNodeResponse, error) {
	params := &inventory.AddNodeParams{
		NodeType: models.RemoteNodeType,
		Name:     req.NodeName,
	}
	node, err := s.s.Add(ctx, s.db.Querier, params)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddRemoteNodeResponse{
		Remote: node.(*inventorypb.RemoteNode),
	}
	return res, nil
}

// AddRemoteAmazonRDSNode adds Amazon (AWS) RDS remote Node.
func (s *nodesServer) AddRemoteAmazonRDSNode(ctx context.Context, req *inventorypb.AddRemoteAmazonRDSNodeRequest) (*inventorypb.AddRemoteAmazonRDSNodeResponse, error) {
	params := &inventory.AddNodeParams{
		NodeType: models.RemoteAmazonRDSNodeType,
		Name:     req.NodeName,
		Address:  &req.Instance,
		Region:   &req.Region,
	}
	node, err := s.s.Add(ctx, s.db.Querier, params)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddRemoteAmazonRDSNodeResponse{
		RemoteAmazonRds: node.(*inventorypb.RemoteAmazonRDSNode),
	}
	return res, nil
}

// RemoveNode removes Node without any Agents and Services.
func (s *nodesServer) RemoveNode(ctx context.Context, req *inventorypb.RemoveNodeRequest) (*inventorypb.RemoveNodeResponse, error) {
	if err := s.s.Remove(ctx, s.db.Querier, req.NodeId); err != nil {
		return nil, err
	}

	return new(inventorypb.RemoveNodeResponse), nil
}
