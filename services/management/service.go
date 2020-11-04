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
	serviceTypes = map[inventorypb.ServiceType]models.ServiceType{
		inventorypb.ServiceType_MYSQL_SERVICE:      models.MySQLServiceType,
		inventorypb.ServiceType_MONGODB_SERVICE:    models.MongoDBServiceType,
		inventorypb.ServiceType_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
		inventorypb.ServiceType_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
		inventorypb.ServiceType_EXTERNAL_SERVICE:   models.ExternalServiceType,
	}
)

// ServiceService represents service for working with services.
type ServiceService struct {
	db       *reform.DB
	registry agentsRegistry
	vmdb     prometheusService
}

// NewServiceService creates ServiceService instance.
func NewServiceService(db *reform.DB, registry agentsRegistry, vmdb prometheusService) *ServiceService {
	return &ServiceService{
		db:       db,
		registry: registry,
		vmdb:     vmdb,
	}
}

// RemoveService removes Service with Agents.
func (s *ServiceService) RemoveService(ctx context.Context, req *managementpb.RemoveServiceRequest) (*managementpb.RemoveServiceResponse, error) {
	err := s.validateRequest(req)
	if err != nil {
		return nil, err
	}
	pmmAgentIDs := make(map[string]bool)
	var reloadPrometheusConfig bool

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		var service *models.Service
		var err error
		switch {
		case req.ServiceName != "":
			service, err = models.FindServiceByName(s.db.Querier, req.ServiceName)
		case req.ServiceId != "":
			service, err = models.FindServiceByID(s.db.Querier, req.ServiceId)
		}
		if err != nil {
			return err
		}
		if req.ServiceType != inventorypb.ServiceType_SERVICE_TYPE_INVALID {
			err := s.checkServiceType(service, req.ServiceType)
			if err != nil {
				return err
			}
		}

		agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: service.ServiceID})
		if err != nil {
			return err
		}
		for _, agent := range agents {
			_, err := models.RemoveAgent(s.db.Querier, agent.AgentID, models.RemoveRestrict)
			if err != nil {
				return err
			}
			if agent.PMMAgentID != nil {
				pmmAgentIDs[pointer.GetString(agent.PMMAgentID)] = true
			} else {
				reloadPrometheusConfig = true
			}
		}
		err = models.RemoveService(s.db.Querier, service.ServiceID, models.RemoveCascade)
		if err != nil {
			return err
		}
		return nil
	}); e != nil {
		return nil, e
	}
	for agentID := range pmmAgentIDs {
		s.registry.SendSetStateRequest(ctx, agentID)
	}
	if reloadPrometheusConfig {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		s.vmdb.RequestConfigurationUpdate()
	}
	return &managementpb.RemoveServiceResponse{}, nil
}

func (s *ServiceService) checkServiceType(service *models.Service, serviceType inventorypb.ServiceType) error {
	if expected, ok := serviceTypes[serviceType]; ok && expected == service.ServiceType {
		return nil
	}
	return status.Error(codes.InvalidArgument, "wrong service type")
}

func (s *ServiceService) validateRequest(request *managementpb.RemoveServiceRequest) error {
	if request.ServiceName == "" && request.ServiceId == "" {
		return status.Error(codes.InvalidArgument, "params not found")
	}
	if request.ServiceName != "" && request.ServiceId != "" {
		return status.Error(codes.InvalidArgument, "service_id or service_name expected; not both")
	}
	return nil
}
