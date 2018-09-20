// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"golang.org/x/net/context"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/rds"
	"github.com/percona/pmm-managed/utils/logger"
)

type RDSServer struct {
	RDS *rds.Service
}

func (s *RDSServer) Discover(ctx context.Context, req *api.RDSDiscoverRequest) (*api.RDSDiscoverResponse, error) {
	res, err := s.RDS.Discover(ctx, req.AwsAccessKeyId, req.AwsSecretAccessKey)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.RDSDiscoverResponse
	for _, db := range res {
		resp.Instances = append(resp.Instances, &api.RDSInstance{
			Node: &api.RDSNode{
				Name:   db.Node.Name,
				Region: db.Node.Region,
			},
			Service: &api.RDSService{
				Address:       *db.Service.Address,
				Port:          uint32(*db.Service.Port),
				Engine:        *db.Service.Engine,
				EngineVersion: *db.Service.EngineVersion,
			},
		})
	}
	return &resp, nil
}

func (s *RDSServer) List(ctx context.Context, req *api.RDSListRequest) (*api.RDSListResponse, error) {
	res, err := s.RDS.List(ctx)
	if err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.RDSListResponse
	for _, db := range res {
		resp.Instances = append(resp.Instances, &api.RDSInstance{
			Node: &api.RDSNode{
				Name:   db.Node.Name,
				Region: db.Node.Region,
			},
			Service: &api.RDSService{
				Address:       *db.Service.Address,
				Port:          uint32(*db.Service.Port),
				Engine:        *db.Service.Engine,
				EngineVersion: *db.Service.EngineVersion,
			},
		})
	}
	return &resp, nil
}

func (s *RDSServer) Add(ctx context.Context, req *api.RDSAddRequest) (*api.RDSAddResponse, error) {
	id := &rds.InstanceID{
		Region: req.Id.GetRegion(),
		Name:   req.Id.GetName(),
	}
	if err := s.RDS.Add(ctx, req.AwsAccessKeyId, req.AwsSecretAccessKey, id, req.Username, req.Password); err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.RDSAddResponse
	return &resp, nil
}

func (s *RDSServer) Remove(ctx context.Context, req *api.RDSRemoveRequest) (*api.RDSRemoveResponse, error) {
	id := &rds.InstanceID{
		Region: req.Id.GetRegion(),
		Name:   req.Id.GetName(),
	}
	if err := s.RDS.Remove(ctx, id); err != nil {
		logger.Get(ctx).Errorf("%+v", err)
		return nil, err
	}

	var resp api.RDSRemoveResponse
	return &resp, nil
}

// check interfaces
var (
	_ api.RDSServer = (*RDSServer)(nil)
)
