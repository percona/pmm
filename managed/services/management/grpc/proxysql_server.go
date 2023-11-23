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

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/services/management"
)

// TODO merge into ../proxysql.go.
type proxySQLServer struct {
	svc *management.ProxySQLService

	managementv1.UnimplementedProxySQLServiceServer
}

// NewManagementProxySQLServer creates Management ProxySQL Server.
func NewManagementProxySQLServer(s *management.ProxySQLService) managementv1.ProxySQLServiceServer { //nolint:ireturn
	return &proxySQLServer{svc: s}
}

// AddProxySQL adds "ProxySQL Service", "Postgres Exporter Agent".
func (s *proxySQLServer) AddProxySQL(ctx context.Context, req *managementv1.AddProxySQLRequest) (*managementv1.AddProxySQLResponse, error) {
	return s.svc.Add(ctx, req)
}
