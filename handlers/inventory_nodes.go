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

	"github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/agents"
)

type NodesServer struct {
	Store  *agents.Store
	Agents map[uint32]*agents.Conn
}

func (s *NodesServer) ListNodes(ctx context.Context, req *inventory.ListNodesRequest) (*inventory.ListNodesResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) GetNode(ctx context.Context, req *inventory.GetNodeRequest) (*inventory.GetNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddNode(ctx context.Context, req *inventory.AddNodeRequest) (*inventory.AddNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddRemoveNode(ctx context.Context, req *inventory.AddRemoveNodeRequest) (*inventory.AddRemoveNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) AddRDSNode(ctx context.Context, req *inventory.AddRDSNodeRequest) (*inventory.AddRDSNodeResponse, error) {
	panic("not implemented")
}

func (s *NodesServer) RemoveNode(ctx context.Context, req *inventory.RemoveNodeRequest) (*inventory.RemoveNodeResponse, error) {
	panic("not implemented")
}

// func (s *NodesServer) List(ctx context.Context, req *inventory.NodesListRequest) (*inventory.NodesListResponse, error) {
// 	logger.Get(ctx).Infof("%#v", req)
// 	return &inventory.NodesListResponse{
// 		Node: []*inventory.Node{
// 			{
// 				Id: 1,
// 			},
// 		},
// 		BareMetal: []*inventory.BareMetalNode{
// 			{
// 				Id: 2,
// 			},
// 		},
// 		Container: []*inventory.ContainerNode{
// 			{
// 				Id: 3,
// 			},
// 		},
// 	}, nil
// }

// func (s *NodesServer) Get(ctx context.Context, req *inventory.NodesGetRequest) (*inventory.NodesGetResponse, error) {
// 	logger.Get(ctx).Infof("%#v", req)
// 	return &inventory.NodesGetResponse{
// 		OnlyOne: &inventory.NodesGetResponse_BareMetal{
// 			BareMetal: &inventory.BareMetalNode{
// 				Id: 1,
// 			},
// 		},
// 	}, nil
// }

// func (s *NodesServer) AddBareMetal(ctx context.Context, req *inventory.AddBareMetalRequest) (*inventory.AddBareMetalResponse, error) {
// 	node := s.Store.AddBareMetalNode(req)
// 	return &inventory.AddBareMetalResponse{
// 		Node: node,
// 	}, nil
// }

// func (s *NodesServer) AddMySQLdExporter(ctx context.Context, req *inventory.AddMySQLdExporterRequest) (*inventory.AddMySQLdExporterResponse, error) {
// 	exporter := s.Store.AddMySQLdExporter(req)
// 	return &inventory.AddMySQLdExporterResponse{
// 		Agent: exporter,
// 	}, nil
// }

// check interfaces
var (
	_ inventory.NodesServer = (*NodesServer)(nil)
)
