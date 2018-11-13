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

	"github.com/percona/pmm-managed/services/inventory"
)

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
	panic("not implemented")
}

func (s *NodesServer) AddVirtualMachineNode(ctx context.Context, req *api.AddVirtualMachineNodeRequest) (*api.AddVirtualMachineNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddContainerNode(ctx context.Context, req *api.AddContainerNodeRequest) (*api.AddContainerNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddRemoteNode(ctx context.Context, req *api.AddRemoteNodeRequest) (*api.AddRemoteNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddRDSNode(ctx context.Context, req *api.AddRDSNodeRequest) (*api.AddRDSNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) ChangeBareMetalNode(ctx context.Context, req *api.ChangeBareMetalNodeRequest) (*api.ChangeBareMetalNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) ChangeVirtualMachineNode(ctx context.Context, req *api.ChangeVirtualMachineNodeRequest) (*api.ChangeVirtualMachineNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) ChangeContainerNode(ctx context.Context, req *api.ChangeContainerNodeRequest) (*api.ChangeContainerNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) ChangeRemoteNode(ctx context.Context, req *api.ChangeRemoteNodeRequest) (*api.ChangeRemoteNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) ChangeRDSNode(ctx context.Context, req *api.ChangeRDSNodeRequest) (*api.ChangeRDSNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) RemoveNode(ctx context.Context, req *api.RemoveNodeRequest) (*api.RemoveNodeResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ api.NodesServer = (*NodesServer)(nil)
)
