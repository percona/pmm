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
)

type RDSServer struct {
	RDS *rds.Service
}

func (s *RDSServer) Discover(ctx context.Context, req *api.RDSDiscoverRequest) (*api.RDSDiscoverResponse, error) {
	res, err := s.RDS.Discover(ctx, req.AwsAccessKeyId, req.AwsSecretAccessKey)
	if err != nil {
		return nil, err
	}

	var resp api.RDSDiscoverResponse
	for _, db := range res {
		resp.Instances = append(resp.Instances, &api.RDSInstance{
			Id:                 db.DBInstanceIdentifier,
			RegionId:           db.Region,
			EndpointAddress:    db.EndpointAddress,
			EndpointPort:       uint32(db.EndpointPort),
			MasterUsername:     db.MasterUsername,
			Engine:             db.Engine,
			EngineVersion:      db.EngineVersion,
			MonitoringInterval: uint32(db.MonitoringInterval.Seconds()),
		})
	}
	return &resp, nil
}

// check interface
var _ api.RDSServer = (*RDSServer)(nil)
