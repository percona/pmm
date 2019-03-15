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
	inventorypb "github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/inventory"
)

type servicesServer struct {
	s *inventory.ServicesService
}

// NewServicesServer returns Inventory API handler for managing Services.
func NewServicesServer(s *inventory.ServicesService) inventorypb.ServicesServer {
	return &servicesServer{
		s: s,
	}
}

// ListServices returns a list of all Services.
func (s *servicesServer) ListServices(ctx context.Context, req *inventorypb.ListServicesRequest) (*inventorypb.ListServicesResponse, error) {
	services, err := s.s.List(ctx)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.ListServicesResponse)
	for _, service := range services {
		switch service := service.(type) {
		case *inventorypb.MySQLService:
			res.Mysql = append(res.Mysql, service)
		case *inventorypb.AmazonRDSMySQLService:
			res.AmazonRdsMysql = append(res.AmazonRdsMysql, service)
		case *inventorypb.MongoDBService:
			res.Mongodb = append(res.Mongodb, service)
		default:
			panic(fmt.Errorf("unhandled inventory Service type %T", service))
		}
	}
	return res, nil
}

// GetService returns a single Service by ID.
func (s *servicesServer) GetService(ctx context.Context, req *inventorypb.GetServiceRequest) (*inventorypb.GetServiceResponse, error) {
	service, err := s.s.Get(ctx, req.ServiceId)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.GetServiceResponse)
	switch service := service.(type) {
	case *inventorypb.MySQLService:
		res.Service = &inventorypb.GetServiceResponse_Mysql{Mysql: service}
	case *inventorypb.AmazonRDSMySQLService:
		res.Service = &inventorypb.GetServiceResponse_AmazonRdsMysql{AmazonRdsMysql: service}
	case *inventorypb.MongoDBService:
		res.Service = &inventorypb.GetServiceResponse_Mongodb{Mongodb: service}
	default:
		panic(fmt.Errorf("unhandled inventory Service type %T", service))
	}
	return res, nil
}

// AddMySQLService adds MySQL Service.
func (s *servicesServer) AddMySQLService(ctx context.Context, req *inventorypb.AddMySQLServiceRequest) (*inventorypb.AddMySQLServiceResponse, error) {
	address := pointer.ToStringOrNil(req.Address)
	port := pointer.ToUint16OrNil(uint16(req.Port))
	service, err := s.s.AddMySQL(ctx, req.ServiceName, req.NodeId, address, port)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMySQLServiceResponse{
		Mysql: service,
	}
	return res, nil
}

// AddAmazonRDSMySQLService adds AmazonRDSMySQL Service.
func (s *servicesServer) AddAmazonRDSMySQLService(ctx context.Context, req *inventorypb.AddAmazonRDSMySQLServiceRequest) (*inventorypb.AddAmazonRDSMySQLServiceResponse, error) {
	panic("not implemented yet")
}

// ChangeMySQLService changes MySQL Service.
func (s *servicesServer) ChangeMySQLService(ctx context.Context, req *inventorypb.ChangeMySQLServiceRequest) (*inventorypb.ChangeMySQLServiceResponse, error) {
	service, err := s.s.Change(ctx, req.ServiceId, req.ServiceName)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeMySQLServiceResponse{
		Mysql: service.(*inventorypb.MySQLService),
	}
	return res, nil
}

// ChangeAmazonRDSMySQLService changes AmazonRDSMySQL Service.
func (s *servicesServer) ChangeAmazonRDSMySQLService(ctx context.Context, req *inventorypb.ChangeAmazonRDSMySQLServiceRequest) (*inventorypb.ChangeAmazonRDSMySQLServiceResponse, error) {
	panic("not implemented yet")
}

func (s *servicesServer) AddMongoDBService(ctx context.Context, req *inventorypb.AddMongoDBServiceRequest) (*inventorypb.AddMongoDBServiceResponse, error) {
	address := pointer.ToStringOrNil(req.Address)
	port := pointer.ToUint16OrNil(uint16(req.Port))
	service, err := s.s.AddMongoDB(ctx, req.ServiceName, req.NodeId, address, port)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMongoDBServiceResponse{
		Mongodb: service,
	}
	return res, nil
}

// RemoveService removes Service without any Agents.
func (s *servicesServer) RemoveService(ctx context.Context, req *inventorypb.RemoveServiceRequest) (*inventorypb.RemoveServiceResponse, error) {
	if err := s.s.Remove(ctx, req.ServiceId); err != nil {
		return nil, err
	}

	return new(inventorypb.RemoveServiceResponse), nil
}
