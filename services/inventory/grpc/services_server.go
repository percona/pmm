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
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/inventory"
)

type servicesServer struct {
	s *inventory.ServicesService
}

// NewServicesServer returns Inventory API handler for managing Services.
func NewServicesServer(s *inventory.ServicesService) inventorypb.ServicesServer {
	return &servicesServer{s}
}

var serviceTypes = map[inventorypb.ServiceType]models.ServiceType{
	inventorypb.ServiceType_MYSQL_SERVICE:      models.MySQLServiceType,
	inventorypb.ServiceType_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventorypb.ServiceType_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventorypb.ServiceType_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventorypb.ServiceType_EXTERNAL_SERVICE:   models.ExternalServiceType,
}

func serviceType(serviceType inventorypb.ServiceType) *models.ServiceType {
	if serviceType == inventorypb.ServiceType_SERVICE_TYPE_INVALID {
		return nil
	}
	result := serviceTypes[serviceType]
	return &result
}

// ListServices returns a list of Services for a given filters.
func (s *servicesServer) ListServices(ctx context.Context, req *inventorypb.ListServicesRequest) (*inventorypb.ListServicesResponse, error) {
	filters := models.ServiceFilters{
		NodeID:      req.GetNodeId(),
		ServiceType: serviceType(req.GetServiceType()),
	}
	services, err := s.s.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.ListServicesResponse)
	for _, service := range services {
		switch service := service.(type) {
		case *inventorypb.MySQLService:
			res.Mysql = append(res.Mysql, service)
		case *inventorypb.MongoDBService:
			res.Mongodb = append(res.Mongodb, service)
		case *inventorypb.PostgreSQLService:
			res.Postgresql = append(res.Postgresql, service)
		case *inventorypb.ProxySQLService:
			res.Proxysql = append(res.Proxysql, service)
		case *inventorypb.ExternalService:
			res.External = append(res.External, service)
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
	case *inventorypb.MongoDBService:
		res.Service = &inventorypb.GetServiceResponse_Mongodb{Mongodb: service}
	case *inventorypb.PostgreSQLService:
		res.Service = &inventorypb.GetServiceResponse_Postgresql{Postgresql: service}
	case *inventorypb.ProxySQLService:
		res.Service = &inventorypb.GetServiceResponse_Proxysql{Proxysql: service}
	case *inventorypb.ExternalService:
		res.Service = &inventorypb.GetServiceResponse_External{External: service}
	default:
		panic(fmt.Errorf("unhandled inventory Service type %T", service))
	}
	return res, nil
}

// AddMySQLService adds MySQL Service.
func (s *servicesServer) AddMySQLService(ctx context.Context, req *inventorypb.AddMySQLServiceRequest) (*inventorypb.AddMySQLServiceResponse, error) {
	service, err := s.s.AddMySQL(ctx, &models.AddDBMSServiceParams{
		ServiceName:    req.ServiceName,
		NodeID:         req.NodeId,
		Environment:    req.Environment,
		Cluster:        req.Cluster,
		ReplicationSet: req.ReplicationSet,
		Address:        pointer.ToStringOrNil(req.Address),
		Port:           pointer.ToUint16OrNil(uint16(req.Port)),
		Socket:         pointer.ToStringOrNil(req.Socket),
		CustomLabels:   req.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMySQLServiceResponse{
		Mysql: service,
	}
	return res, nil
}

func (s *servicesServer) AddMongoDBService(ctx context.Context, req *inventorypb.AddMongoDBServiceRequest) (*inventorypb.AddMongoDBServiceResponse, error) {
	service, err := s.s.AddMongoDB(ctx, &models.AddDBMSServiceParams{
		ServiceName:    req.ServiceName,
		NodeID:         req.NodeId,
		Environment:    req.Environment,
		Cluster:        req.Cluster,
		ReplicationSet: req.ReplicationSet,
		Address:        pointer.ToStringOrNil(req.Address),
		Port:           pointer.ToUint16OrNil(uint16(req.Port)),
		Socket:         pointer.ToStringOrNil(req.Socket),
		CustomLabels:   req.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMongoDBServiceResponse{
		Mongodb: service,
	}
	return res, nil
}

func (s *servicesServer) AddPostgreSQLService(ctx context.Context, req *inventorypb.AddPostgreSQLServiceRequest) (*inventorypb.AddPostgreSQLServiceResponse, error) {
	service, err := s.s.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
		ServiceName:    req.ServiceName,
		NodeID:         req.NodeId,
		Environment:    req.Environment,
		Cluster:        req.Cluster,
		ReplicationSet: req.ReplicationSet,
		Address:        pointer.ToStringOrNil(req.Address),
		Port:           pointer.ToUint16OrNil(uint16(req.Port)),
		Socket:         pointer.ToStringOrNil(req.Socket),
		CustomLabels:   req.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddPostgreSQLServiceResponse{
		Postgresql: service,
	}
	return res, nil
}

func (s *servicesServer) AddProxySQLService(ctx context.Context, req *inventorypb.AddProxySQLServiceRequest) (*inventorypb.AddProxySQLServiceResponse, error) {
	service, err := s.s.AddProxySQL(ctx, &models.AddDBMSServiceParams{
		ServiceName:    req.ServiceName,
		NodeID:         req.NodeId,
		Environment:    req.Environment,
		Cluster:        req.Cluster,
		ReplicationSet: req.ReplicationSet,
		Address:        pointer.ToStringOrNil(req.Address),
		Port:           pointer.ToUint16OrNil(uint16(req.Port)),
		Socket:         pointer.ToStringOrNil(req.Socket),
		CustomLabels:   req.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddProxySQLServiceResponse{
		Proxysql: service,
	}
	return res, nil
}

func (s *servicesServer) AddExternalService(ctx context.Context, req *inventorypb.AddExternalServiceRequest) (*inventorypb.AddExternalServiceResponse, error) {
	service, err := s.s.AddExternalService(ctx, &models.AddDBMSServiceParams{
		ServiceName:    req.ServiceName,
		NodeID:         req.NodeId,
		Environment:    req.Environment,
		Cluster:        req.Cluster,
		ReplicationSet: req.ReplicationSet,
		CustomLabels:   req.CustomLabels,
		ExternalGroup:  req.Group,
	})
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddExternalServiceResponse{
		External: service,
	}
	return res, nil
}

// RemoveService removes Service.
func (s *servicesServer) RemoveService(ctx context.Context, req *inventorypb.RemoveServiceRequest) (*inventorypb.RemoveServiceResponse, error) {
	if err := s.s.Remove(ctx, req.ServiceId, req.Force); err != nil {
		return nil, err
	}

	return new(inventorypb.RemoveServiceResponse), nil
}
