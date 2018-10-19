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

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/postgresql"
	"github.com/percona/pmm-managed/utils/logger"
)

// PostgreSQLServer handles requests to manage PostgreSQL nodes and services.
type PostgreSQLServer struct {
	PostgreSQL *postgresql.Service
}

// List returns a list of PostgreSQL instances.
func (s *PostgreSQLServer) List(ctx context.Context, req *api.PostgreSQLListRequest) (*api.PostgreSQLListResponse, error) {
	res, err := s.PostgreSQL.List(ctx)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.PostgreSQLListResponse
	for _, db := range res {
		resp.Instances = append(resp.Instances, &api.PostgreSQLInstance{
			Node: &api.PostgreSQLNode{
				Name: db.Node.Name,
			},
			Service: &api.PostgreSQLService{
				Address:       *db.Service.Address,
				Port:          uint32(*db.Service.Port),
				Engine:        *db.Service.Engine,
				EngineVersion: *db.Service.EngineVersion,
			},
		})
	}
	return &resp, nil
}

// Add adds new PostgreSQL instance.
func (s *PostgreSQLServer) Add(ctx context.Context, req *api.PostgreSQLAddRequest) (*api.PostgreSQLAddResponse, error) {

	id, err := s.PostgreSQL.Add(ctx, req.Name, req.Address, req.Port, req.Username, req.Password)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	resp := api.PostgreSQLAddResponse{
		Id: id,
	}
	return &resp, nil
}

// Remove removes PostgreSQL instance.
func (s *PostgreSQLServer) Remove(ctx context.Context, req *api.PostgreSQLRemoveRequest) (*api.PostgreSQLRemoveResponse, error) {
	if err := s.PostgreSQL.Remove(ctx, req.Id); err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.PostgreSQLRemoveResponse
	return &resp, nil
}

// check interfaces
var (
	_ api.PostgreSQLServer = (*PostgreSQLServer)(nil)
)
