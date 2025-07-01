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
	s *inventory.ServicesService

	inventoryv1.UnimplementedServicesServiceServer
}

// NewServicesServer returns Inventory API handler for managing Services.
func NewServicesServer(s *inventory.ServicesService) inventoryv1.ServicesServiceServer { //nolint:ireturn
	return &servicesServer{
		s: s,
	}
}

var serviceTypes = map[inventoryv1.ServiceType]models.ServiceType{
	inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE:      models.MySQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_HAPROXY_SERVICE:    models.HAProxyServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_EXTERNAL_SERVICE:   models.ExternalServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_VALKEY_SERVICE:     models.ValkeyServiceType,
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
	req *inventoryv1.ListActiveServiceTypesRequest, //nolint:revive
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
	service, err := s.s.Get(ctx, req.GetServiceId())
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

// AddService adds any type of Service.
func (s *servicesServer) AddService(ctx context.Context, req *inventoryv1.AddServiceRequest) (*inventoryv1.AddServiceResponse, error) {
	switch req.Service.(type) {
	case *inventoryv1.AddServiceRequest_Mysql:
		return s.addMySQLService(ctx, req.GetMysql())
	case *inventoryv1.AddServiceRequest_Mongodb:
		return s.addMongoDBService(ctx, req.GetMongodb())
	case *inventoryv1.AddServiceRequest_Postgresql:
		return s.addPostgreSQLService(ctx, req.GetPostgresql())
	case *inventoryv1.AddServiceRequest_Valkey:
		return s.addValkeyService(ctx, req.GetValkey())
	case *inventoryv1.AddServiceRequest_Proxysql:
		return s.addProxySQLService(ctx, req.GetProxysql())
	case *inventoryv1.AddServiceRequest_Haproxy:
		return s.addHAProxyService(ctx, req.GetHaproxy())
	case *inventoryv1.AddServiceRequest_External:
		return s.addExternalService(ctx, req.GetExternal())
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported service type %T", req.Service)
	}
}

// addMySQLService adds MySQL Service.
func (s *servicesServer) addMySQLService(ctx context.Context, params *inventoryv1.AddMySQLServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddMySQL(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        pointer.ToStringOrNil(params.Address),
		Port:           pointer.ToUint16OrNil(uint16(params.Port)), //nolint:gosec // port is not expected to overflow uint16
		Socket:         pointer.ToStringOrNil(params.Socket),
		CustomLabels:   params.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_Mysql{
			Mysql: service,
		},
	}
	return res, nil
}

// addValkeyService adds Valkey Service.
func (s *servicesServer) addValkeyService(ctx context.Context, params *inventoryv1.AddValkeyServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddValkey(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        pointer.ToStringOrNil(params.Address),
		Port:           pointer.ToUint16OrNil(uint16(params.Port)), //nolint:gosec // port is not expected to overflow uint16
		Socket:         pointer.ToStringOrNil(params.Socket),
		CustomLabels:   params.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_Valkey{
			Valkey: service,
		},
	}
	return res, nil
}

func (s *servicesServer) addMongoDBService(ctx context.Context, params *inventoryv1.AddMongoDBServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddMongoDB(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        pointer.ToStringOrNil(params.Address),
		Port:           pointer.ToUint16OrNil(uint16(params.Port)), //nolint:gosec // port is not expected to overflow uint16
		Socket:         pointer.ToStringOrNil(params.Socket),
		CustomLabels:   params.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_Mongodb{
			Mongodb: service,
		},
	}
	return res, nil
}

func (s *servicesServer) addPostgreSQLService(ctx context.Context, params *inventoryv1.AddPostgreSQLServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        pointer.ToStringOrNil(params.Address),
		Port:           pointer.ToUint16OrNil(uint16(params.Port)), //nolint:gosec // port is not expected to overflow uint16
		Socket:         pointer.ToStringOrNil(params.Socket),
		CustomLabels:   params.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_Postgresql{
			Postgresql: service,
		},
	}
	return res, nil
}

func (s *servicesServer) addProxySQLService(ctx context.Context, params *inventoryv1.AddProxySQLServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddProxySQL(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        pointer.ToStringOrNil(params.Address),
		Port:           pointer.ToUint16OrNil(uint16(params.Port)), //nolint:gosec // port is not expected to overflow uint16
		Socket:         pointer.ToStringOrNil(params.Socket),
		CustomLabels:   params.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_Proxysql{
			Proxysql: service,
		},
	}
	return res, nil
}

func (s *servicesServer) addHAProxyService(ctx context.Context, params *inventoryv1.AddHAProxyServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddHAProxyService(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		CustomLabels:   params.CustomLabels,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_Haproxy{
			Haproxy: service,
		},
	}
	return res, nil
}

func (s *servicesServer) addExternalService(ctx context.Context, params *inventoryv1.AddExternalServiceParams) (*inventoryv1.AddServiceResponse, error) {
	service, err := s.s.AddExternalService(ctx, &models.AddDBMSServiceParams{
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeId,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		CustomLabels:   params.CustomLabels,
		ExternalGroup:  params.Group,
	})
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddServiceResponse{
		Service: &inventoryv1.AddServiceResponse_External{
			External: service,
		},
	}
	return res, nil
}

// RemoveService removes Service.
func (s *servicesServer) RemoveService(ctx context.Context, req *inventoryv1.RemoveServiceRequest) (*inventoryv1.RemoveServiceResponse, error) {
	if err := s.s.Remove(ctx, req.GetServiceId(), req.GetForce()); err != nil {
		return nil, err
	}

	return &inventoryv1.RemoveServiceResponse{}, nil
}

// ChangeService changes service configuration.
func (s *servicesServer) ChangeService(ctx context.Context, req *inventoryv1.ChangeServiceRequest) (*inventoryv1.ChangeServiceResponse, error) {
	sl := &models.ChangeStandardLabelsParams{
		ServiceID:      req.ServiceId,
		Cluster:        req.Cluster,
		Environment:    req.Environment,
		ReplicationSet: req.ReplicationSet,
		ExternalGroup:  req.ExternalGroup,
	}

	service, err := s.s.ChangeService(ctx, sl, req.GetCustomLabels())
	if err != nil {
		return nil, toAPIError(err)
	}

	res := &inventoryv1.ChangeServiceResponse{}
	switch service := service.(type) {
	case *inventoryv1.MySQLService:
		res.Service = &inventoryv1.ChangeServiceResponse_Mysql{Mysql: service}
	case *inventoryv1.MongoDBService:
		res.Service = &inventoryv1.ChangeServiceResponse_Mongodb{Mongodb: service}
	case *inventoryv1.PostgreSQLService:
		res.Service = &inventoryv1.ChangeServiceResponse_Postgresql{Postgresql: service}
	case *inventoryv1.ProxySQLService:
		res.Service = &inventoryv1.ChangeServiceResponse_Proxysql{Proxysql: service}
	case *inventoryv1.HAProxyService:
		res.Service = &inventoryv1.ChangeServiceResponse_Haproxy{Haproxy: service}
	case *inventoryv1.ExternalService:
		res.Service = &inventoryv1.ChangeServiceResponse_External{External: service}
	default:
		panic(fmt.Errorf("unhandled inventory Service type %T", service))
	}
	return res, nil
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
