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

package management

import (
	"context"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// ServiceService represents service for working with services.
type ServiceService struct {
	db    *reform.DB
	r     agentsRegistry
	state agentsStateUpdater
	vmdb  prometheusService

	managementpb.UnimplementedServiceServer
}

// NewServiceService creates ServiceService instance.
func NewServiceService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService) *ServiceService {
	return &ServiceService{
		db:    db,
		r:     r,
		state: state,
		vmdb:  vmdb,
	}
}

// RemoveService removes Service with Agents.
func (s *ServiceService) RemoveService(ctx context.Context, req *managementpb.RemoveServiceRequest) (*managementpb.RemoveServiceResponse, error) {
	err := s.validateRequest(req)
	if err != nil {
		return nil, err
	}
	pmmAgentIDs := make(map[string]struct{})
	var reloadPrometheusConfig bool

	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var service *models.Service
		var err error
		switch {
		case req.ServiceName != "":
			service, err = models.FindServiceByName(tx.Querier, req.ServiceName)
		case req.ServiceId != "":
			service, err = models.FindServiceByID(tx.Querier, req.ServiceId)
		}
		if err != nil {
			return err
		}
		if req.ServiceType != inventorypb.ServiceType_SERVICE_TYPE_UNSPECIFIED {
			err := s.checkServiceType(service, req.ServiceType)
			if err != nil {
				return err
			}
		}

		agents, err := models.FindAgents(tx.Querier, models.AgentFilters{ServiceID: service.ServiceID})
		if err != nil {
			return err
		}
		for _, agent := range agents {
			_, err := models.RemoveAgent(tx.Querier, agent.AgentID, models.RemoveRestrict)
			if err != nil {
				return err
			}
			if agent.PMMAgentID != nil {
				pmmAgentIDs[pointer.GetString(agent.PMMAgentID)] = struct{}{}
			} else {
				reloadPrometheusConfig = true
			}
		}

		err = models.RemoveService(tx.Querier, service.ServiceID, models.RemoveCascade)
		if err != nil {
			return err
		}

		node, err := models.FindNodeByID(s.db.Querier, service.NodeID)
		if err != nil {
			return err
		}

		// For RDS and Azure remove also node.
		if node.NodeType == models.RemoteRDSNodeType || node.NodeType == models.RemoteAzureDatabaseNodeType {
			agents, err = models.FindAgents(tx.Querier, models.AgentFilters{NodeID: node.NodeID})
			if err != nil {
				return err
			}
			for _, a := range agents {
				_, err := models.RemoveAgent(s.db.Querier, a.AgentID, models.RemoveRestrict)
				if err != nil {
					return err
				}
				if a.PMMAgentID != nil {
					pmmAgentIDs[pointer.GetString(a.PMMAgentID)] = struct{}{}
				}
			}

			if len(pmmAgentIDs) <= 1 {
				if err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	for agentID := range pmmAgentIDs {
		s.state.RequestStateUpdate(ctx, agentID)
	}
	if reloadPrometheusConfig {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		s.vmdb.RequestConfigurationUpdate()
	}
	return &managementpb.RemoveServiceResponse{}, nil
}

func (s *ServiceService) checkServiceType(service *models.Service, serviceType inventorypb.ServiceType) error {
	if expected, ok := services.ServiceTypes[serviceType]; ok && expected == service.ServiceType {
		return nil
	}
	return status.Error(codes.InvalidArgument, "wrong service type")
}
