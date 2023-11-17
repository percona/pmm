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

	managementpb "github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/services/management"
)

// TODO merge into ../proxysql.go.
type proxySQLServer struct {
	svc *management.ProxySQLService

	managementpb.UnimplementedProxySQLServiceServer
}

// NewManagementProxySQLServer creates Management ProxySQL Server.
func NewManagementProxySQLServer(s *management.ProxySQLService) managementpb.ProxySQLServiceServer { //nolint:ireturn
	return &proxySQLServer{svc: s}
}

// AddProxySQL adds "ProxySQL Service", "Postgres Exporter Agent".
func (s *proxySQLServer) AddProxySQL(ctx context.Context, req *managementpb.AddProxySQLRequest) (*managementpb.AddProxySQLResponse, error) {
	return s.svc.Add(ctx, req)
}
