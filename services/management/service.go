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

package management

import (
	"context"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

var (
	errNoParamsNotFound    = status.Error(codes.InvalidArgument, "params not found")
	errOneOfParamsExpected = status.Error(codes.InvalidArgument, "service_id or service_name expected; not both")
	serviceTypes           = map[inventorypb.ServiceType]models.ServiceType{
		inventorypb.ServiceType_MYSQL_SERVICE:      models.MySQLServiceType,
		inventorypb.ServiceType_MONGODB_SERVICE:    models.MongoDBServiceType,
		inventorypb.ServiceType_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	}
)

// ServiceService represents service for working with services.
type ServiceService struct {
	db       *reform.DB
	registry registry
}

// NewServiceService creates ServiceService instance.
func NewServiceService(db *reform.DB, registry registry) *ServiceService {
	return &ServiceService{
		db:       db,
		registry: registry,
	}
}

// RemoveService removes Service with Agents.
func (ss *ServiceService) RemoveService(ctx context.Context, req *managementpb.RemoveServiceRequest) (*managementpb.RemoveServiceResponse, error) {
	err := ss.validateRequest(req)
	if err != nil {
		return nil, err
	}
	pmmAgentIDs := make(map[string]bool)

	if e := ss.db.InTransaction(func(tx *reform.TX) error {
		var service *models.Service
		var err error
		switch {
		case req.ServiceName != "":
			service, err = models.FindServiceByName(ss.db.Querier, req.ServiceName)
		case req.ServiceId != "":
			service, err = models.FindServiceByID(ss.db.Querier, req.ServiceId)
		}
		if err != nil {
			return err
		}
		if req.ServiceType != inventorypb.ServiceType_SERVICE_TYPE_INVALID {
			err := ss.checkServiceType(service, req.ServiceType)
			if err != nil {
				return err
			}
		}

		agents, err := models.AgentsForService(ss.db.Querier, service.ServiceID)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			_, err := models.RemoveAgent(ss.db.Querier, agent.AgentID)
			if err != nil {
				return err
			}
			if agent.PMMAgentID != nil {
				pmmAgentIDs[pointer.GetString(agent.PMMAgentID)] = true
			}
		}
		err = models.RemoveService(ss.db.Querier, service.ServiceID)
		if err != nil {
			return err
		}
		return nil
	}); e != nil {
		return nil, e
	}
	for agentID := range pmmAgentIDs {
		ss.registry.SendSetStateRequest(ctx, agentID)
	}
	return &managementpb.RemoveServiceResponse{}, nil
}

func (ss *ServiceService) checkServiceType(service *models.Service, serviceType inventorypb.ServiceType) error {
	if expected, ok := serviceTypes[serviceType]; ok && expected == service.ServiceType {
		return nil
	}
	return status.Error(codes.InvalidArgument, "wrong service type")
}

func (ss *ServiceService) validateRequest(request *managementpb.RemoveServiceRequest) error {
	if request.ServiceName == "" && request.ServiceId == "" {
		return errNoParamsNotFound
	}
	if request.ServiceName != "" && request.ServiceId != "" {
		return errOneOfParamsExpected
	}
	return nil
}
