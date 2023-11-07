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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	agentv1beta1 "github.com/percona/pmm/api/managementpb/agent"
	servicev1beta1 "github.com/percona/pmm/api/managementpb/service"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// A map to check if the service is supported.
// NOTE: known external services appear to match the vendor names,
// (e.g. "mysql", "mongodb", "postgresql", "proxysql", "haproxy"),
// which is why ServiceType_EXTERNAL_SERVICE is not part of this map.
var supportedServices = map[string]inventorypb.ServiceType{
	string(models.MySQLServiceType):      inventorypb.ServiceType_MYSQL_SERVICE,
	string(models.MongoDBServiceType):    inventorypb.ServiceType_MONGODB_SERVICE,
	string(models.PostgreSQLServiceType): inventorypb.ServiceType_POSTGRESQL_SERVICE,
	string(models.ProxySQLServiceType):   inventorypb.ServiceType_PROXYSQL_SERVICE,
	string(models.HAProxyServiceType):    inventorypb.ServiceType_HAPROXY_SERVICE,
}

// MgmtServiceService is a management service for working with services.
type MgmtServiceService struct {
	db       *reform.DB
	r        agentsRegistry
	state    agentsStateUpdater
	vmdb     prometheusService
	vmClient victoriaMetricsClient

	servicev1beta1.UnimplementedMgmtServiceServer
}

type statusMetrics struct {
	status      int
	serviceType string
}

// NewMgmtServiceService creates MgmtServiceService instance.
func NewMgmtServiceService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService, vmClient victoriaMetricsClient) *MgmtServiceService {
	return &MgmtServiceService{
		db:       db,
		r:        r,
		state:    state,
		vmdb:     vmdb,
		vmClient: vmClient,
	}
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
func (s *MgmtServiceService) ListServices(ctx context.Context, req *servicev1beta1.ListServiceRequest) (*servicev1beta1.ListServiceResponse, error) {
	filters := models.ServiceFilters{
		NodeID:        req.NodeId,
		ServiceType:   services.ProtoToModelServiceType(req.ServiceType),
		ExternalGroup: req.ExternalGroup,
	}

	agentToAPI := func(agent *models.Agent) *agentv1beta1.UniversalAgent {
		return &agentv1beta1.UniversalAgent{
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

	resultSvc := make([]*servicev1beta1.UniversalService, len(services))
	for i, service := range services {
		labels, err := service.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		svc := &servicev1beta1.UniversalService{
			Address:        pointer.GetString(service.Address),
			Agents:         []*agentv1beta1.UniversalAgent{},
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
				svc.Status = servicev1beta1.UniversalService_DOWN
			case 1:
				svc.Status = servicev1beta1.UniversalService_UP
			}
		} else {
			// In case there is no metric, we need to assign different values for supported and unsupported service types.
			if _, ok := supportedServices[metric.serviceType]; ok {
				svc.Status = servicev1beta1.UniversalService_UNKNOWN
			} else {
				svc.Status = servicev1beta1.UniversalService_STATUS_INVALID
			}
		}

		nodeName, ok := nodeMap[service.NodeID]
		if ok {
			svc.NodeName = nodeName
		}

		var uAgents []*agentv1beta1.UniversalAgent

		for _, agent := range agents {
			if IsNodeAgent(agent, service) || IsVMAgent(agent, service) || IsServiceAgent(agent, service) {
				uAgents = append(uAgents, agentToAPI(agent))
			}
		}

		svc.Agents = uAgents
		resultSvc[i] = svc
	}

	return &servicev1beta1.ListServiceResponse{Services: resultSvc}, nil
}
