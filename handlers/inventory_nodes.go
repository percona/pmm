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

func (s *NodesServer) ListNodes(ctx context.Context, req *api.ListNodesRequest) (*api.ListNodesResponse, error) {
	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return nil, err
	}

	res := new(api.ListNodesResponse)
	for _, node := range nodes {
		switch node := node.(type) {
		case *api.BareMetalNode:
			res.BareMetal = append(res.BareMetal, node)
		case *api.VirtualMachineNode:
			res.VirtualMachine = append(res.VirtualMachine, node)
		case *api.ContainerNode:
			res.Container = append(res.Container, node)
		case *api.RemoteNode:
			res.Remote = append(res.Remote, node)
		case *api.RDSNode:
			res.Rds = append(res.Rds, node)
		default:
			panic(fmt.Errorf("unhandled inventory Node type %T", node))
		}
	}
	return res, nil
}

func (s *NodesServer) GetNode(ctx context.Context, req *api.GetNodeRequest) (*api.GetNodeResponse, error) {
	node, err := s.Nodes.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	res := new(api.GetNodeResponse)
	switch node := node.(type) {
	case *api.BareMetalNode:
		res.Node = &api.GetNodeResponse_BareMetal{BareMetal: node}
	case *api.VirtualMachineNode:
		res.Node = &api.GetNodeResponse_VirtualMachine{VirtualMachine: node}
	case *api.ContainerNode:
		res.Node = &api.GetNodeResponse_Container{Container: node}
	case *api.RemoteNode:
		res.Node = &api.GetNodeResponse_Remote{Remote: node}
	case *api.RDSNode:
		res.Node = &api.GetNodeResponse_Rds{Rds: node}
	default:
		panic(fmt.Errorf("unhandled inventory Node type %T", node))
	}
	return res, nil
}

func (s *NodesServer) AddBareMetalNode(ctx context.Context, req *api.AddBareMetalNodeRequest) (*api.AddBareMetalNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.BareMetalNodeType, req.Name, &req.Hostname, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddBareMetalNodeResponse{
		BareMetal: node.(*api.BareMetalNode),
	}
	return res, nil
}

func (s *NodesServer) AddVirtualMachineNode(ctx context.Context, req *api.AddVirtualMachineNodeRequest) (*api.AddVirtualMachineNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.VirtualMachineNodeType, req.Name, &req.Hostname, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddVirtualMachineNodeResponse{
		VirtualMachine: node.(*api.VirtualMachineNode),
	}
	return res, nil
}

func (s *NodesServer) AddContainerNode(ctx context.Context, req *api.AddContainerNodeRequest) (*api.AddContainerNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.ContainerNodeType, req.Name, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddContainerNodeResponse{
		Container: node.(*api.ContainerNode),
	}
	return res, nil
}

func (s *NodesServer) AddRemoteNode(ctx context.Context, req *api.AddRemoteNodeRequest) (*api.AddRemoteNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.RemoteNodeType, req.Name, nil, nil)
	if err != nil {
		return nil, err
	}

	res := &api.AddRemoteNodeResponse{
		Remote: node.(*api.RemoteNode),
	}
	return res, nil
}

func (s *NodesServer) AddRDSNode(ctx context.Context, req *api.AddRDSNodeRequest) (*api.AddRDSNodeResponse, error) {
	node, err := s.Nodes.Add(ctx, models.RemoteNodeType, req.Name, &req.Hostname, &req.Region)
	if err != nil {
		return nil, err
	}

	res := &api.AddRDSNodeResponse{
		Rds: node.(*api.RDSNode),
	}
	return res, nil
}

func (s *NodesServer) ChangeBareMetalNode(ctx context.Context, req *api.ChangeBareMetalNodeRequest) (*api.ChangeBareMetalNodeResponse, error) {
	if err := s.Nodes.Change(ctx, req.Id, req.Name); err != nil {
		return nil, err
	}

	return new(api.ChangeBareMetalNodeResponse), nil
}

func (s *NodesServer) ChangeVirtualMachineNode(ctx context.Context, req *api.ChangeVirtualMachineNodeRequest) (*api.ChangeVirtualMachineNodeResponse, error) {
	if err := s.Nodes.Change(ctx, req.Id, req.Name); err != nil {
		return nil, err
	}

	return new(api.ChangeVirtualMachineNodeResponse), nil
}

func (s *NodesServer) ChangeContainerNode(ctx context.Context, req *api.ChangeContainerNodeRequest) (*api.ChangeContainerNodeResponse, error) {
	if err := s.Nodes.Change(ctx, req.Id, req.Name); err != nil {
		return nil, err
	}

	return new(api.ChangeContainerNodeResponse), nil
}

func (s *NodesServer) ChangeRemoteNode(ctx context.Context, req *api.ChangeRemoteNodeRequest) (*api.ChangeRemoteNodeResponse, error) {
	if err := s.Nodes.Change(ctx, req.Id, req.Name); err != nil {
		return nil, err
	}

	return new(api.ChangeRemoteNodeResponse), nil
}

func (s *NodesServer) ChangeRDSNode(ctx context.Context, req *api.ChangeRDSNodeRequest) (*api.ChangeRDSNodeResponse, error) {
	if err := s.Nodes.Change(ctx, req.Id, req.Name); err != nil {
		return nil, err
	}

	return new(api.ChangeRDSNodeResponse), nil
}

func (s *NodesServer) RemoveNode(ctx context.Context, req *api.RemoveNodeRequest) (*api.RemoveNodeResponse, error) {
	if err := s.Nodes.Remove(ctx, req.Id); err != nil {
		return nil, err
	}

	return new(api.RemoveNodeResponse), nil
}

// check interfaces
var (
	_ api.NodesServer = (*NodesServer)(nil)
)
