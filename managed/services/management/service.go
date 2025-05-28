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
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// ManagementService allows to interact with services.
type ManagementService struct { //nolint:revive
	db            *reform.DB
	r             agentsRegistry
	state         agentsStateUpdater
	cc            connectionChecker
	sib           serviceInfoBroker
	vmdb          prometheusService
	vc            versionCache
	grafanaClient grafanaClient
	vmClient      victoriaMetricsClient
	l             *logrus.Entry

	managementv1.UnimplementedManagementServiceServer
}

type statusMetrics struct {
	status      int
	serviceType string
}

// NewManagementService creates a ManagementService instance.
func NewManagementService(
	db *reform.DB,
	r agentsRegistry,
	state agentsStateUpdater,
	cc connectionChecker,
	sib serviceInfoBroker,
	vmdb prometheusService,
	vc versionCache,
	grafanaClient grafanaClient,
	vmClient victoriaMetricsClient,
) *ManagementService {
	return &ManagementService{
		db:            db,
		r:             r,
		state:         state,
		cc:            cc,
		sib:           sib,
		vmdb:          vmdb,
		vc:            vc,
		grafanaClient: grafanaClient,
		vmClient:      vmClient,
		l:             logrus.WithField("service", "management"),
	}
}

// A map to check if the service is supported.
// NOTE: known external services appear to match the vendor names,
// (e.g. "mysql", "mongodb", "postgresql", "proxysql", "haproxy"),
// which is why ServiceType_EXTERNAL_SERVICE is not part of this map.
var supportedServices = map[string]inventoryv1.ServiceType{
	string(models.MySQLServiceType):      inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE,
	string(models.MongoDBServiceType):    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
	string(models.PostgreSQLServiceType): inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
	string(models.ProxySQLServiceType):   inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE,
	string(models.HAProxyServiceType):    inventoryv1.ServiceType_SERVICE_TYPE_HAPROXY_SERVICE,
}

// AddService add a Service and its Agents.
func (s *ManagementService) AddService(ctx context.Context, req *managementv1.AddServiceRequest) (*managementv1.AddServiceResponse, error) {
	switch req.Service.(type) {
	case *managementv1.AddServiceRequest_Mysql:
		return s.addMySQL(ctx, req.GetMysql())
	case *managementv1.AddServiceRequest_Mongodb:
		return s.addMongoDB(ctx, req.GetMongodb())
	case *managementv1.AddServiceRequest_Postgresql:
		return s.addPostgreSQL(ctx, req.GetPostgresql())
	case *managementv1.AddServiceRequest_Proxysql:
		return s.addProxySQL(ctx, req.GetProxysql())
	case *managementv1.AddServiceRequest_Haproxy:
		return s.addHAProxy(ctx, req.GetHaproxy())
	case *managementv1.AddServiceRequest_External:
		return s.addExternal(ctx, req.GetExternal())
	case *managementv1.AddServiceRequest_Rds:
		return s.addRDS(ctx, req.GetRds())
	case *managementv1.AddServiceRequest_Valkey:
		return s.addValkey(ctx, req.GetValkey())
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid service type")
	}
}

// ListServices returns a filtered list of Services with some attributes from Agents and Nodes.
func (s *ManagementService) ListServices(ctx context.Context, req *managementv1.ListServicesRequest) (*managementv1.ListServicesResponse, error) {
	filters := models.ServiceFilters{
		NodeID:        req.NodeId,
		ServiceType:   services.ProtoToModelServiceType(req.ServiceType),
		ExternalGroup: req.ExternalGroup,
	}

	agentToAPI := func(agent *models.Agent) *managementv1.UniversalAgent {
		return &managementv1.UniversalAgent{
			AgentId:     agent.AgentID,
			AgentType:   string(agent.AgentType),
			Status:      agent.Status,
			IsConnected: s.r.IsConnected(agent.AgentID),
		}
	}

	query := `pg_up{collector="exporter",job=~".*_hr$"}
		or mysql_up{job=~".*_hr$"}
		or mongodb_up{job=~".*_hr$"}
		or proxysql_up{job=~".*_hr$"}
		or haproxy_backend_status{state="UP"}
		or redis_up{job=~".*_hr$"}
	`
	result, _, err := s.vmClient.Query(ctx, query, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute an instant VM query")
	}

	metrics := make(map[string]statusMetrics, len(result.(model.Vector))) //nolint:forcetypeassert
	for _, v := range result.(model.Vector) {                             //nolint:forcetypeassert
		serviceID := string(v.Metric[model.LabelName("service_id")])
		serviceType := string(v.Metric[model.LabelName("service_type")])
		metrics[serviceID] = statusMetrics{status: int(v.Value), serviceType: serviceType}
	}

	var (
		services []*models.Service
		agents   []*models.Agent
		nodes    []*models.Node
	)

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		services, err = models.FindServices(tx.Querier, filters)
		if err != nil {
			return err
		}

		agentFilters := models.AgentFilters{}

		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		agentFilters.IgnoreNomad = !settings.IsNomadEnabled()

		agents, err = models.FindAgents(tx.Querier, agentFilters)
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

	resultSvc := make([]*managementv1.UniversalService, len(services))
	for i, service := range services {
		labels, err := service.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		svc := &managementv1.UniversalService{
			Address:        pointer.GetString(service.Address),
			Agents:         []*managementv1.UniversalAgent{},
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
			Version:        pointer.GetString(service.Version),
		}

		if metric, ok := metrics[service.ServiceID]; ok {
			switch metric.status {
			// We assume there can only be values of either 1(UP) or 0(DOWN).
			case 0:
				svc.Status = managementv1.UniversalService_STATUS_DOWN
			case 1:
				svc.Status = managementv1.UniversalService_STATUS_UP
			}
		} else {
			// In case there is no metric, we need to assign different values for supported and unsupported service types.
			if _, ok := supportedServices[metric.serviceType]; ok {
				svc.Status = managementv1.UniversalService_STATUS_UNKNOWN
			} else {
				svc.Status = managementv1.UniversalService_STATUS_UNSPECIFIED
			}
		}

		nodeName, ok := nodeMap[service.NodeID]
		if ok {
			svc.NodeName = nodeName
		}

		var uAgents []*managementv1.UniversalAgent

		for _, agent := range agents {
			if IsNodeAgent(agent, service) || IsVMAgent(agent, service) || IsServiceAgent(agent, service) {
				uAgents = append(uAgents, agentToAPI(agent))
			}
		}

		svc.Agents = uAgents
		resultSvc[i] = svc
	}

	return &managementv1.ListServicesResponse{Services: resultSvc}, nil
}

// RemoveService removes a Service along with its Agents.
func (s *ManagementService) RemoveService(ctx context.Context, req *managementv1.RemoveServiceRequest) (*managementv1.RemoveServiceResponse, error) {
	err := s.validateRequest(req)
	if err != nil {
		return nil, err
	}
	pmmAgentIDs := make(map[string]struct{})
	var reloadPrometheusConfig bool

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var service *models.Service
		var err error

		if LooksLikeID(req.ServiceId) {
			service, err = models.FindServiceByID(tx.Querier, req.ServiceId)
		} else {
			// if it's not a service ID, it is a service name then
			service, err = models.FindServiceByName(tx.Querier, req.ServiceId)
		}
		if err != nil {
			return err
		}

		if req.ServiceType != inventoryv1.ServiceType_SERVICE_TYPE_UNSPECIFIED {
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

		// For RDS and Azure we also want to remove the node.
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
	return &managementv1.RemoveServiceResponse{}, nil
}

func (s *ManagementService) checkServiceType(service *models.Service, serviceType inventoryv1.ServiceType) error {
	if expected, ok := services.ServiceTypes[serviceType]; ok && expected == service.ServiceType {
		return nil
	}
	return status.Error(codes.InvalidArgument, "wrong service type")
}

func (s *ManagementService) validateRequest(request *managementv1.RemoveServiceRequest) error {
	if request.ServiceId == "" {
		return status.Error(codes.InvalidArgument, "service_id or service_name expected")
	}
	return nil
}
