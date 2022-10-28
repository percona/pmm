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
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
)

var serviceTypes = map[inventorypb.ServiceType]models.ServiceType{
	inventorypb.ServiceType_MYSQL_SERVICE:      models.MySQLServiceType,
	inventorypb.ServiceType_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventorypb.ServiceType_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventorypb.ServiceType_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventorypb.ServiceType_HAPROXY_SERVICE:    models.HAProxyServiceType,
	inventorypb.ServiceType_EXTERNAL_SERVICE:   models.ExternalServiceType,
}

// ServiceService represents service for working with services.
type ServiceService struct {
	db    *reform.DB
	state agentsStateUpdater
	vmdb  prometheusService

	managementpb.UnimplementedServiceServer
}

// NewServiceService creates ServiceService instance.
func NewServiceService(db *reform.DB, state agentsStateUpdater, vmdb prometheusService) *ServiceService {
	return &ServiceService{
		db:    db,
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

	if e := s.db.InTransaction(func(tx *reform.TX) error {
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
		if req.ServiceType != inventorypb.ServiceType_SERVICE_TYPE_INVALID {
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
	}); e != nil {
		return nil, e
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

// AddCustomLabels adds custom labels to a service.
func (s *ServiceService) AddCustomLabels(ctx context.Context, req *managementpb.AddCustomLabelsRequest) (*managementpb.AddCustomLabelsResponse, error) {
	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service_id is required")
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		labels, err := service.GetCustomLabels()
		if err != nil {
			return err
		}
		if labels == nil {
			labels = make(map[string]string)
		}

		for k, v := range req.CustomLabels {
			labels[k] = v
		}

		err = service.SetCustomLabels(labels)
		if err != nil {
			return err
		}

		err = tx.UpdateColumns(service, "custom_labels")
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &managementpb.AddCustomLabelsResponse{}, nil
}

// RemoveCustomLabels adds custom labels to a service.
func (s *ServiceService) RemoveCustomLabels(ctx context.Context, req *managementpb.RemoveCustomLabelsRequest) (*managementpb.RemoveCustomLabelsResponse, error) {
	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service_id is required")
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		labels, err := service.GetCustomLabels()
		if err != nil {
			return err
		}
		if labels == nil {
			return nil
		}

		for _, k := range req.CustomLabelKeys {
			delete(labels, k)
		}

		err = service.SetCustomLabels(labels)
		if err != nil {
			return err
		}

		err = tx.UpdateColumns(service, "custom_labels")
		if err != nil {
			return err
		}

		logrus.Info(string(service.CustomLabels))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &managementpb.RemoveCustomLabelsResponse{}, nil
}

func (s *ServiceService) checkServiceType(service *models.Service, serviceType inventorypb.ServiceType) error {
	if expected, ok := serviceTypes[serviceType]; ok && expected == service.ServiceType {
		return nil
	}
	return status.Error(codes.InvalidArgument, "wrong service type")
}

func (s *ServiceService) validateRequest(request *managementpb.RemoveServiceRequest) error {
	if request.ServiceName == "" && request.ServiceId == "" {
		return status.Error(codes.InvalidArgument, "service_id or service_name expected")
	}
	if request.ServiceName != "" && request.ServiceId != "" {
		return status.Error(codes.InvalidArgument, "service_id or service_name expected; not both")
	}
	return nil
}
