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

// TODO merge into ../mongodb.go.
type mongoDBServer struct {
	svc *management.MongoDBService

	managementv1.UnimplementedMongoDBServiceServer
}

// NewManagementMongoDBServer creates Management MongoDB Server.
func NewManagementMongoDBServer(s *management.MongoDBService) managementv1.MongoDBServiceServer { //nolint:ireturn
	return &mongoDBServer{svc: s}
}

// AddMongoDB adds "MongoDB Service", "MongoDB Exporter Agent" and "QAN MongoDB Profiler".
func (s *mongoDBServer) AddMongoDB(ctx context.Context, req *managementv1.AddMongoDBRequest) (*managementv1.AddMongoDBResponse, error) {
	return s.svc.Add(ctx, req)
}
