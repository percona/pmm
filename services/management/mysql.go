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
	"github.com/percona/pmm-managed/services"
)

// MySQLService MySQL Management Service.
type MySQLService struct {
	db       *reform.DB
	registry agentsRegistry
}

// NewMySQLService creates new MySQL Management Service.
func NewMySQLService(db *reform.DB, registry agentsRegistry) *MySQLService {
	return &MySQLService{db, registry}
}

// Add adds "MySQL Service", "MySQL Exporter Agent" and "QAN MySQL PerfSchema Agent".
func (s *MySQLService) Add(ctx context.Context, req *managementpb.AddMySQLRequest) (*managementpb.AddMySQLResponse, error) {
	res := new(managementpb.AddMySQLResponse)

	if err := validateNodeParamsOneOf(req.NodeId, req.NodeName, req.AddNode); err != nil {
		return nil, err
	}

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		var nodeID string
		switch {
		case req.NodeId != "":
			nodeID = req.NodeId
		case req.NodeName != "":
			node, err := models.FindNodeByName(tx.Querier, req.NodeName)
			if err != nil {
				return err
			}
			nodeID = node.NodeID
		case req.AddNode != nil:
			var nodeType models.NodeType
			switch req.AddNode.NodeType {
			case inventorypb.NodeType_GENERIC_NODE:
				nodeType = models.GenericNodeType
			case inventorypb.NodeType_CONTAINER_NODE:
				nodeType = models.ContainerNodeType
			case inventorypb.NodeType_REMOTE_NODE:
				nodeType = models.ContainerNodeType
			default:
				return status.Errorf(codes.InvalidArgument, "Unsupported Node type %q.", req.AddNode.NodeType)
			}
			node, err := models.CreateNode(tx.Querier, nodeType, &models.CreateNodeParams{
				NodeName:      req.AddNode.NodeName,
				MachineID:     pointer.ToStringOrNil(req.AddNode.MachineId),
				Distro:        req.AddNode.Distro,
				NodeModel:     req.AddNode.NodeModel,
				AZ:            req.AddNode.Az,
				ContainerID:   pointer.ToStringOrNil(req.AddNode.ContainerId),
				ContainerName: pointer.ToStringOrNil(req.AddNode.ContainerName),
				CustomLabels:  req.AddNode.CustomLabels,
				Address:       req.Address,
				Region:        pointer.ToStringOrNil(req.AddNode.Region),
			})
			if err != nil {
				return err
			}
			nodeID = node.NodeID
		}
		service, err := models.AddNewService(tx.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         nodeID,
			Environment:    req.Environment,
			Cluster:        req.Cluster,
			ReplicationSet: req.ReplicationSet,
			Address:        pointer.ToStringOrNil(req.Address),
			Port:           pointer.ToUint16OrNil(uint16(req.Port)),
			CustomLabels:   req.CustomLabels,
		})
		if err != nil {
			return err
		}

		invService, err := services.ToAPIService(service)
		if err != nil {
			return err
		}
		res.Service = invService.(*inventorypb.MySQLService)

		row, err := models.CreateAgent(tx.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
			PMMAgentID: req.PmmAgentId,
			ServiceID:  service.ServiceID,
			Username:   req.Username,
			Password:   req.Password,
		})
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			if err = s.registry.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res.MysqldExporter = agent.(*inventorypb.MySQLdExporter)

		if req.QanMysqlPerfschema {
			row, err = models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, &models.CreateAgentParams{
				PMMAgentID: req.PmmAgentId,
				ServiceID:  service.ServiceID,
				Username:   req.Username,
				Password:   req.Password,
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanMysqlPerfschema = agent.(*inventorypb.QANMySQLPerfSchemaAgent)
		}

		if req.QanMysqlSlowlog {
			row, err = models.CreateAgent(tx.Querier, models.QANMySQLSlowlogAgentType, &models.CreateAgentParams{
				PMMAgentID: req.PmmAgentId,
				ServiceID:  service.ServiceID,
				Username:   req.Username,
				Password:   req.Password,
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanMysqlSlowlog = agent.(*inventorypb.QANMySQLSlowlogAgent)
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.registry.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, nil
}

func validateNodeParamsOneOf(nodeID string, nodeName string, addNodeParams *managementpb.AddNodeParams) error {
	got := 0
	if nodeID != "" {
		got++
	}
	if nodeName != "" {
		got++
	}
	if addNodeParams != nil {
		got++
	}
	if got != 1 {
		return status.Errorf(codes.InvalidArgument, "expected only one param; node id, node name or register node params")
	}
	return nil
}
