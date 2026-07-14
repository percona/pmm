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

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
	"github.com/percona/pmm/managed/services/management/common"
)

type servicesServer struct {
	s            *inventory.ServicesService
	mgmtServices common.MgmtServices

	inventorypb.UnimplementedServicesServer
}

// NewServicesServer returns Inventory API handler for managing Services.
func NewServicesServer(s *inventory.ServicesService, mgmtServices common.MgmtServices) inventorypb.ServicesServer { //nolint:ireturn
	return &servicesServer{
		s:            s,
		mgmtServices: mgmtServices,
	}
}

var serviceTypes = map[inventorypb.ServiceType]models.ServiceType{
	inventorypb.ServiceType_MYSQL_SERVICE:      models.MySQLServiceType,
	inventorypb.ServiceType_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventorypb.ServiceType_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventorypb.ServiceType_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventorypb.ServiceType_HAPROXY_SERVICE:    models.HAProxyServiceType,
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
		NodeID:        req.GetNodeId(),
		ServiceType:   serviceType(req.GetServiceType()),
		ExternalGroup: req.GetExternalGroup(),
	}
	services, err := s.s.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ListServicesResponse{}
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
		case *inventorypb.HAProxyService:
			res.Haproxy = append(res.Haproxy, service)
		case *inventorypb.ExternalService:
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
	req *inventorypb.ListActiveServiceTypesRequest, //nolint:revive
) (*inventorypb.ListActiveServiceTypesResponse, error) {
	types, err := s.s.ListActiveServiceTypes(ctx)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ListActiveServiceTypesResponse{
		ServiceTypes: types,
	}

	return res, nil
}

// GetService returns a single Service by ID.
func (s *servicesServer) GetService(ctx context.Context, req *inventorypb.GetServiceRequest) (*inventorypb.GetServiceResponse, error) {
	service, err := s.s.Get(ctx, req.ServiceId)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.GetServiceResponse{}
	switch service := service.(type) {
	case *inventorypb.MySQLService:
		res.Service = &inventorypb.GetServiceResponse_Mysql{Mysql: service}
	case *inventorypb.MongoDBService:
		res.Service = &inventorypb.GetServiceResponse_Mongodb{Mongodb: service}
	case *inventorypb.PostgreSQLService:
		res.Service = &inventorypb.GetServiceResponse_Postgresql{Postgresql: service}
	case *inventorypb.ProxySQLService:
		res.Service = &inventorypb.GetServiceResponse_Proxysql{Proxysql: service}
	case *inventorypb.HAProxyService:
		res.Service = &inventorypb.GetServiceResponse_Haproxy{Haproxy: service}
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

func (s *servicesServer) AddHAProxyService(ctx context.Context, req *inventorypb.AddHAProxyServiceRequest) (*inventorypb.AddHAProxyServiceResponse, error) {
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

	res := &inventorypb.AddHAProxyServiceResponse{
		Haproxy: service,
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

	return &inventorypb.RemoveServiceResponse{}, nil
}

// AddCustomLabels adds or replaces (if key exists) custom labels for a service.
func (s *servicesServer) AddCustomLabels(ctx context.Context, req *inventorypb.AddCustomLabelsRequest) (*inventorypb.AddCustomLabelsResponse, error) {
	return s.s.AddCustomLabels(ctx, req)
}

// RemoveCustomLabels removes custom labels from a service.
func (s *servicesServer) RemoveCustomLabels(ctx context.Context, req *inventorypb.RemoveCustomLabelsRequest) (*inventorypb.RemoveCustomLabelsResponse, error) {
	return s.s.RemoveCustomLabels(ctx, req)
}

// ChangeService changes service configuration.
func (s *servicesServer) ChangeService(ctx context.Context, req *inventorypb.ChangeServiceRequest) (*inventorypb.ChangeServiceResponse, error) {
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

	return &inventorypb.ChangeServiceResponse{}, nil
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
