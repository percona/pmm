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
	"fmt"

	"github.com/AlekSi/pointer"
	api "github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/inventory"
)

type servicesServer struct {
	s *inventory.ServicesService
}

// NewServicesServer returns Inventory API handler for managing Services.
func NewServicesServer(s *inventory.ServicesService) api.ServicesServer {
	return &servicesServer{
		s: s,
	}
}

// ListServices returns a list of all Services.
func (s *servicesServer) ListServices(ctx context.Context, req *api.ListServicesRequest) (*api.ListServicesResponse, error) {
	services, err := s.s.List(ctx)
	if err != nil {
		return nil, err
	}

	res := new(api.ListServicesResponse)
	for _, service := range services {
		switch service := service.(type) {
		case *api.MySQLService:
			res.Mysql = append(res.Mysql, service)
		case *api.AmazonRDSMySQLService:
			res.AmazonRdsMysql = append(res.AmazonRdsMysql, service)
		default:
			panic(fmt.Errorf("unhandled inventory Service type %T", service))
		}
	}
	return res, nil
}

// GetService returns a single Service by ID.
func (s *servicesServer) GetService(ctx context.Context, req *api.GetServiceRequest) (*api.GetServiceResponse, error) {
	service, err := s.s.Get(ctx, req.ServiceId)
	if err != nil {
		return nil, err
	}

	res := new(api.GetServiceResponse)
	switch service := service.(type) {
	case *api.MySQLService:
		res.Service = &api.GetServiceResponse_Mysql{Mysql: service}
	case *api.AmazonRDSMySQLService:
		res.Service = &api.GetServiceResponse_AmazonRdsMysql{AmazonRdsMysql: service}
	default:
		panic(fmt.Errorf("unhandled inventory Service type %T", service))
	}
	return res, nil
}

// AddMySQLService adds MySQL Service.
func (s *servicesServer) AddMySQLService(ctx context.Context, req *api.AddMySQLServiceRequest) (*api.AddMySQLServiceResponse, error) {
	address := pointer.ToStringOrNil(req.Address)
	port := pointer.ToUint16OrNil(uint16(req.Port))
	service, err := s.s.AddMySQL(ctx, req.ServiceName, req.NodeId, address, port)
	if err != nil {
		return nil, err
	}

	res := &api.AddMySQLServiceResponse{
		Mysql: service,
	}
	return res, nil
}

// AddMongoDBService adds MongoDB Service.
func (s *servicesServer) AddMongoDBService(ctx context.Context, req *api.AddMongoDBServiceRequest) (*api.AddMongoDBServiceResponse, error) {
	panic("not implemented yet")
}

// AddAmazonRDSMySQLService adds AmazonRDSMySQL Service.
func (s *servicesServer) AddAmazonRDSMySQLService(ctx context.Context, req *api.AddAmazonRDSMySQLServiceRequest) (*api.AddAmazonRDSMySQLServiceResponse, error) {
	panic("not implemented yet")
}

// ChangeMySQLService changes MySQL Service.
func (s *servicesServer) ChangeMySQLService(ctx context.Context, req *api.ChangeMySQLServiceRequest) (*api.ChangeMySQLServiceResponse, error) {
	service, err := s.s.Change(ctx, req.ServiceId, req.ServiceName)
	if err != nil {
		return nil, err
	}

	res := &api.ChangeMySQLServiceResponse{
		Mysql: service.(*api.MySQLService),
	}
	return res, nil
}

// ChangeAmazonRDSMySQLService changes AmazonRDSMySQL Service.
func (s *servicesServer) ChangeAmazonRDSMySQLService(ctx context.Context, req *api.ChangeAmazonRDSMySQLServiceRequest) (*api.ChangeAmazonRDSMySQLServiceResponse, error) {
	panic("not implemented yet")
}

// RemoveService removes Service without any Agents.
func (s *servicesServer) RemoveService(ctx context.Context, req *api.RemoveServiceRequest) (*api.RemoveServiceResponse, error) {
	if err := s.s.Remove(ctx, req.ServiceId); err != nil {
		return nil, err
	}

	return new(api.RemoveServiceResponse), nil
}
