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
	"github.com/percona/pmm-managed/services/remote"
	"github.com/percona/pmm-managed/utils/logger"
)

// RemoteServer handles requests to return Remote nodes and services list.
type RemoteServer struct {
	Remote *remote.Service
}

// List returns a list of PostgreSQL instances.
func (s *RemoteServer) List(ctx context.Context, req *api.RemoteListRequest) (*api.RemoteListResponse, error) {
	res, err := s.Remote.List(ctx)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.RemoteListResponse
	for _, db := range res {
		resp.Instances = append(resp.Instances, &api.RemoteInstance{
			Node: &api.RemoteNode{
				Id:     db.Node.ID,
				Name:   db.Node.Name,
				Region: db.Node.Region,
			},
			Service: &api.RemoteService{
				Type:          string(db.Service.Type),
				Address:       *db.Service.Address,
				Port:          uint32(*db.Service.Port),
				Engine:        *db.Service.Engine,
				EngineVersion: *db.Service.EngineVersion,
			},
		})
	}
	return &resp, nil
}

// check interfaces
var (
	_ api.RemoteServer = (*RemoteServer)(nil)
)
