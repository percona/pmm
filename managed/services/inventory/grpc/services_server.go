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
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
	"github.com/percona/pmm/managed/services/management/common"
)

type servicesServer struct {
	s            *inventory.ServicesService
	mgmtServices common.MgmtServices

	inventoryv1.UnimplementedServicesServiceServer
}

// NewServicesServer returns Inventory API handler for managing Services.
func NewServicesServer(s *inventory.ServicesService, mgmtServices common.MgmtServices) inventoryv1.ServicesServiceServer { //nolint:ireturn
	return &servicesServer{
		s:            s,
		mgmtServices: mgmtServices,
	}
}

var serviceTypes = map[inventoryv1.ServiceType]models.ServiceType{
	inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE:      models.MySQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_HAPROXY_SERVICE:    models.HAProxyServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_EXTERNAL_SERVICE:   models.ExternalServiceType,
}

func serviceType(serviceType inventoryv1.ServiceType) *models.ServiceType {
	if serviceType == inventoryv1.ServiceType_SERVICE_TYPE_UNSPECIFIED {
		return nil
	}
	result := serviceTypes[serviceType]
	return &result
}

// ListServices returns a list of Services for a given filters.
func (s *servicesServer) ListServices(ctx context.Context, req *inventoryv1.ListServicesRequest) (*inventoryv1.ListServicesResponse, error) {
	filters := models.ServiceFilters{
		NodeID:        req.GetNodeId(),
		ServiceType:   serviceType(req.GetServiceType()),
		ExternalGroup: req.GetExternalGroup(),
	}
	services, err := s.s.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ListServicesResponse{}
	for _, service := range services {
		switch service := service.(type) {
		case *inventoryv1.MySQLService:
			res.Mysql = append(res.Mysql, service)
		case *inventoryv1.MongoDBService:
			res.Mongodb = append(res.Mongodb, service)
		case *inventoryv1.PostgreSQLService:
			res.Postgresql = append(res.Postgresql, service)
		case *inventoryv1.ProxySQLService:
			res.Proxysql = append(res.Proxysql, service)
		case *inventoryv1.HAProxyService:
			res.Haproxy = append(res.Haproxy, service)
		case *inventoryv1.ExternalService:
			res.External = append(res.External, service)
		default:
			panic(fmt.Errorf("unhandled inventory Service type %T", service))
		}
	}
	return res, nil
}

// ListActiveServiceTypes returns list of active Services.
func (s *servicesServer) ListActiveServiceTypes(
	ctx context.Context,
	req *inventoryv1.ListActiveServiceTypesRequest,
) (*inventoryv1.ListActiveServiceTypesResponse, error) {
	types, err := s.s.ListActiveServiceTypes(ctx)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ListActiveServiceTypesResponse{
		ServiceTypes: types,
	}

	return res, nil
}

// GetService returns a single Service by ID.
func (s *servicesServer) GetService(ctx context.Context, req *inventoryv1.GetServiceRequest) (*inventoryv1.GetServiceResponse, error) {
	service, err := s.s.Get(ctx, req.ServiceId)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.GetServiceResponse{}
	switch service := service.(type) {
	case *inventoryv1.MySQLService:
		res.Service = &inventoryv1.GetServiceResponse_Mysql{Mysql: service}
	case *inventoryv1.MongoDBService:
		res.Service = &inventoryv1.GetServiceResponse_Mongodb{Mongodb: service}
	case *inventoryv1.PostgreSQLService:
		res.Service = &inventoryv1.GetServiceResponse_Postgresql{Postgresql: service}
	case *inventoryv1.ProxySQLService:
		res.Service = &inventoryv1.GetServiceResponse_Proxysql{Proxysql: service}
	case *inventoryv1.HAProxyService:
		res.Service = &inventoryv1.GetServiceResponse_Haproxy{Haproxy: service}
	case *inventoryv1.ExternalService:
		res.Service = &inventoryv1.GetServiceResponse_External{External: service}
	default:
		panic(fmt.Errorf("unhandled inventory Service type %T", service))
	}
	return res, nil
}

// AddMySQLService adds MySQL Service.
func (s *servicesServer) AddMySQLService(ctx context.Context, req *inventoryv1.AddMySQLServiceRequest) (*inventoryv1.AddMySQLServiceResponse, error) {
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

	res := &inventoryv1.AddMySQLServiceResponse{
		Mysql: service,
	}
	return res, nil
}

func (s *servicesServer) AddMongoDBService(ctx context.Context, req *inventoryv1.AddMongoDBServiceRequest) (*inventoryv1.AddMongoDBServiceResponse, error) {
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

	res := &inventoryv1.AddMongoDBServiceResponse{
		Mongodb: service,
	}
	return res, nil
}

func (s *servicesServer) AddPostgreSQLService(ctx context.Context, req *inventoryv1.AddPostgreSQLServiceRequest) (*inventoryv1.AddPostgreSQLServiceResponse, error) {
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

	res := &inventoryv1.AddPostgreSQLServiceResponse{
		Postgresql: service,
	}
	return res, nil
}

func (s *servicesServer) AddProxySQLService(ctx context.Context, req *inventoryv1.AddProxySQLServiceRequest) (*inventoryv1.AddProxySQLServiceResponse, error) {
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

	res := &inventoryv1.AddProxySQLServiceResponse{
		Proxysql: service,
	}
	return res, nil
}

func (s *servicesServer) AddHAProxyService(ctx context.Context, req *inventoryv1.AddHAProxyServiceRequest) (*inventoryv1.AddHAProxyServiceResponse, error) {
	service, err := s.s.AddHAProxyService(ctx, &models.AddDBMSServiceParams{
		ServiceName:    req.ServiceName,
		NodeID:         req.NodeId,
		Environment:    req.Environment,
		Cluster:        req.Cluster,
		ReplicationSet: req.ReplicationSet,
		CustomLabels:   req.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddHAProxyServiceResponse{
		Haproxy: service,
	}
	return res, nil
}

func (s *servicesServer) AddExternalService(ctx context.Context, req *inventoryv1.AddExternalServiceRequest) (*inventoryv1.AddExternalServiceResponse, error) {
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

	res := &inventoryv1.AddExternalServiceResponse{
		External: service,
	}
	return res, nil
}

// RemoveService removes Service.
func (s *servicesServer) RemoveService(ctx context.Context, req *inventoryv1.RemoveServiceRequest) (*inventoryv1.RemoveServiceResponse, error) {
	if err := s.s.Remove(ctx, req.ServiceId, req.Force); err != nil {
		return nil, err
	}

	return &inventoryv1.RemoveServiceResponse{}, nil
}

// AddCustomLabels adds or replaces (if key exists) custom labels for a service.
func (s *servicesServer) AddCustomLabels(ctx context.Context, req *inventoryv1.AddCustomLabelsRequest) (*inventoryv1.AddCustomLabelsResponse, error) {
	return s.s.AddCustomLabels(ctx, req)
}

// RemoveCustomLabels removes custom labels from a service.
func (s *servicesServer) RemoveCustomLabels(ctx context.Context, req *inventoryv1.RemoveCustomLabelsRequest) (*inventoryv1.RemoveCustomLabelsResponse, error) {
	return s.s.RemoveCustomLabels(ctx, req)
}

// ChangeService changes service configuration.
func (s *servicesServer) ChangeService(ctx context.Context, req *inventoryv1.ChangeServiceRequest) (*inventoryv1.ChangeServiceResponse, error) {
	err := s.s.ChangeService(ctx, s.mgmtServices, &models.ChangeStandardLabelsParams{
		ServiceID:      req.ServiceId,
		Cluster:        req.Cluster,
		Environment:    req.Environment,
		ReplicationSet: req.ReplicationSet,
		ExternalGroup:  req.ExternalGroup,
	})
	if err != nil {
		return nil, toAPIError(err)
	}

	return &inventoryv1.ChangeServiceResponse{}, nil
}

// toAPIError converts GO errors into API-level errors.
func toAPIError(err error) error {
	switch {
	case errors.Is(err, common.ErrClusterLocked):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return err
	}
}
