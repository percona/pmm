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

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/services/management"
)

// TODO merge into ../node.go.
type nodeServer struct {
	svc *management.NodeService

	managementpb.UnimplementedNodeServer
}

// NewManagementNodeServer creates Management Node Server.
func NewManagementNodeServer(s *management.NodeService) managementpb.NodeServer { //nolint:ireturn
	return &nodeServer{svc: s}
}

// RegisterNode do registration of new Node.
func (s *nodeServer) RegisterNode(ctx context.Context, req *managementpb.RegisterNodeRequest) (*managementpb.RegisterNodeResponse, error) {
	return s.svc.Register(ctx, req)
}

// UnregisterNode do unregistration of the Node.
func (s *nodeServer) UnregisterNode(ctx context.Context, req *managementpb.UnregisterNodeRequest) (*managementpb.UnregisterNodeResponse, error) {
	return s.svc.Unregister(ctx, req)
}
