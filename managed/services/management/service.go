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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func convertServiceType(serviceType inventorypb.ServiceType) *models.ServiceType {
	if serviceType == inventorypb.ServiceType_SERVICE_TYPE_INVALID {
		return nil
	}
	result := serviceTypes[serviceType]
	return &result
}

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

// ListServices returns a filtered list of Services with some attributes from Agents and Nodes.
//
//nolint:unparam
func (s *ServiceService) ListServices(ctx context.Context, req *managementpb.ListServiceRequest) (*managementpb.ListServiceResponse, error) {
	filters := models.ServiceFilters{
		NodeID:        req.NodeId,
		ServiceType:   convertServiceType(req.ServiceType),
		ExternalGroup: req.ExternalGroup,
	}

	var services []*models.Service
	var agents []*models.Agent
	var nodes []*models.Node

	agentToAPI := func(agent *models.Agent) *managementpb.UniversalAgent {
		return &managementpb.UniversalAgent{
			AgentId:     agent.AgentID,
			AgentType:   string(agent.AgentType),
			Status:      agent.Status,
			IsConnected: s.r.IsConnected(agent.AgentID),
		}
	}

	// TODO: provide a higher level of data consistency guarantee by using a locking mechanism.
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		services, err = models.FindServices(tx.Querier, filters)
		if err != nil {
			return err
		}

		agents, err = models.FindAgents(tx.Querier, models.AgentFilters{})
		if err != nil {
			return err
		}

		nodes, err = models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	nodeMap := make(map[string]string, len(nodes))
	for _, node := range nodes {
		nodeMap[node.NodeID] = node.NodeName
	}

	resultSvc := make([]*managementpb.UniversalService, len(services))
	for i, service := range services {
		labels, err := service.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		svc := &managementpb.UniversalService{
			Address:        pointer.GetString(service.Address),
			Agents:         []*managementpb.UniversalAgent{},
			Cluster:        service.Cluster,
			CreatedAt:      timestamppb.New(service.CreatedAt),
			CustomLabels:   labels,
			DatabaseName:   service.DatabaseName,
			Environment:    service.Environment,
			ExternalGroup:  service.ExternalGroup,
			NodeId:         service.NodeID,
			Port:           uint32(pointer.GetUint16(service.Port)),
			ReplicationSet: service.ReplicationSet,
			ServiceId:      service.ServiceID,
			ServiceType:    string(service.ServiceType),
			ServiceName:    service.ServiceName,
			Socket:         pointer.GetString(service.Socket),
			UpdatedAt:      timestamppb.New(service.UpdatedAt),
		}

		nodeName, ok := nodeMap[service.NodeID]
		if ok {
			svc.NodeName = nodeName
		}

		var svcAgents []*managementpb.UniversalAgent

		for _, agent := range agents {
			if IsNodeAgent(agent, service) || IsVMAgent(agent, service) || IsServiceAgent(agent, service) {
				svcAgents = append(svcAgents, agentToAPI(agent))
			}

			svc.Agents = svcAgents
		}

		resultSvc[i] = svc
	}

	return &managementpb.ListServiceResponse{Services: resultSvc}, nil
}
