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
	"github.com/percona/pmm-managed/services/mysql"
	"github.com/percona/pmm-managed/utils/logger"
)

// MySQLServer handles requests to manage MySQL nodes and services.
type MySQLServer struct {
	MySQL *mysql.Service
}

// List returns a list of MySQL instances.
func (s *MySQLServer) List(ctx context.Context, req *api.MySQLListRequest) (*api.MySQLListResponse, error) {
	res, err := s.MySQL.List(ctx)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.MySQLListResponse
	for _, db := range res {
		resp.Instances = append(resp.Instances, &api.MySQLInstance{
			Node: &api.MySQLNode{
				Name: db.Node.Name,
			},
			Service: &api.MySQLService{
				Address:       *db.Service.Address,
				Port:          uint32(*db.Service.Port),
				Engine:        *db.Service.Engine,
				EngineVersion: *db.Service.EngineVersion,
			},
		})
	}
	return &resp, nil
}

// Add adds new MySQL instance.
func (s *MySQLServer) Add(ctx context.Context, req *api.MySQLAddRequest) (*api.MySQLAddResponse, error) {

	id, err := s.MySQL.Add(ctx, req.Name, req.Address, req.Port, req.Username, req.Password)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	resp := api.MySQLAddResponse{
		Id: id,
	}
	return &resp, nil
}

// Remove removes MySQL instance.
func (s *MySQLServer) Remove(ctx context.Context, req *api.MySQLRemoveRequest) (*api.MySQLRemoveResponse, error) {
	if err := s.MySQL.Remove(ctx, req.Id); err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.MySQLRemoveResponse
	return &resp, nil
}

// check interfaces
var (
	_ api.MySQLServer = (*MySQLServer)(nil)
)
