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

	"github.com/percona/pmm-managed/services/agents"
	"github.com/percona/pmm-managed/services/inventory"
)

type NodesServer struct {
	Nodes  *inventory.NodesService
	Store  *agents.Store
	Agents map[uint32]*agents.Conn
}

func (s *NodesServer) ListNodes(ctx context.Context, req *api.ListNodesRequest) (*api.ListNodesResponse, error) {
	return s.Nodes.List(ctx)
}

func (s *NodesServer) GetNode(ctx context.Context, req *api.GetNodeRequest) (*api.GetNodeResponse, error) {
	node, err := s.Nodes.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	res := new(api.GetNodeResponse)
	switch node := node.(type) {
	case *api.BareMetalNode:
		res.Node = &api.GetNodeResponse_BareMetal{
			BareMetal: node,
		}
	case *api.VirtualMachineNode:
		res.Node = &api.GetNodeResponse_VirtualMachine{
			VirtualMachine: node,
		}
	case *api.ContainerNode:
		res.Node = &api.GetNodeResponse_Container{
			Container: node,
		}
	case *api.RemoteNode:
		res.Node = &api.GetNodeResponse_Remote{
			Remote: node,
		}
	case *api.RDSNode:
		res.Node = &api.GetNodeResponse_Rds{
			Rds: node,
		}
	default:
		panic(fmt.Errorf("unhandled inventory Node type %T", node))
	}
	return res, nil
}

func (s *NodesServer) AddNode(ctx context.Context, req *api.AddNodeRequest) (*api.AddNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddRemoveNode(ctx context.Context, req *api.AddRemoveNodeRequest) (*api.AddRemoveNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddRDSNode(ctx context.Context, req *api.AddRDSNodeRequest) (*api.AddRDSNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) RemoveNode(ctx context.Context, req *api.RemoveNodeRequest) (*api.RemoveNodeResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ api.NodesServer = (*NodesServer)(nil)
)
