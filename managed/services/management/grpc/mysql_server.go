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

	managementpb "github.com/percona/pmm/api/managementpb/v1"
	"github.com/percona/pmm/managed/services/management"
)

// TODO merge into ../mysql.go.
type mySQLServer struct {
	svc *management.MySQLService

	managementpb.UnimplementedMySQLServiceServer
}

// NewManagementMySQLServer creates Management MySQL Server.
func NewManagementMySQLServer(s *management.MySQLService) managementpb.MySQLServiceServer { //nolint:ireturn
	return &mySQLServer{svc: s}
}

// AddMySQL adds "MySQL Service", "MySQL Exporter Agent" and "QAN MySQL PerfSchema Agent".
func (s *mySQLServer) AddMySQL(ctx context.Context, req *managementpb.AddMySQLRequest) (*managementpb.AddMySQLResponse, error) {
	return s.svc.Add(ctx, req)
}
