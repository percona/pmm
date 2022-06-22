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

package grpc

import (
	"context"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/services/management"
)

// TODO merge into ../postgresql.go
type postgreSQLServer struct {
	svc *management.PostgreSQLService

	managementpb.UnimplementedPostgreSQLServer
}

// NewManagementPostgreSQLServer creates Management PostgreSQL Server.
func NewManagementPostgreSQLServer(s *management.PostgreSQLService) managementpb.PostgreSQLServer {
	return &postgreSQLServer{svc: s}
}

// AddPostgreSQL adds "PostgreSQL Service", "Postgres Exporter Agent".
func (s *postgreSQLServer) AddPostgreSQL(ctx context.Context, req *managementpb.AddPostgreSQLRequest) (*managementpb.AddPostgreSQLResponse, error) {
	return s.svc.Add(ctx, req)
}
