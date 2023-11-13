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
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	nodev1beta1 "github.com/percona/pmm/api/managementpb/node"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// MgmtNodeService represents a management API service for working with nodes.
type MgmtNodeService struct {
	db       *reform.DB
	r        agentsRegistry
	vmClient victoriaMetricsClient

	nodev1beta1.UnimplementedMgmtNodeServiceServer
}

// NewMgmtNodeService creates MgmtNodeService instance.
func NewMgmtNodeService(db *reform.DB, r agentsRegistry, vmClient victoriaMetricsClient) *MgmtNodeService {
	return &MgmtNodeService{
		db:       db,
		r:        r,
		vmClient: vmClient,
	}
}

const upQuery = `up{job=~".*_hr$"}`

// ListNodes returns a filtered list of Nodes.
func (s *MgmtNodeService) ListNodes(ctx context.Context, req *nodev1beta1.ListNodeRequest) (*nodev1beta1.ListNodeResponse, error) {
	filters := models.NodeFilters{
		NodeType: services.ProtoToModelNodeType(req.NodeType),
	}

	var (
		nodes    []*models.Node
		agents   []*models.Agent
		services []*models.Service
	)

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error

		nodes, err = models.FindNodes(s.db.Querier, filters)
		if err != nil {
			return err
		}

		agents, err = models.FindAgents(s.db.Querier, models.AgentFilters{})
		if err != nil {
			return err
		}

		services, err = models.FindServices(s.db.Querier, models.ServiceFilters{})
		if err != nil {
			return err
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	convertAgentToProto := func(agent *models.Agent) *nodev1beta1.UniversalNode_Agent {
		return &nodev1beta1.UniversalNode_Agent{
			AgentId:     agent.AgentID,
			AgentType:   string(agent.AgentType),
			Status:      agent.Status,
			IsConnected: s.r.IsConnected(agent.AgentID),
		}
	}

	aMap := make(map[string][]*nodev1beta1.UniversalNode_Agent, len(nodes))
	for _, a := range agents {
		if a.NodeID != nil || a.RunsOnNodeID != nil {
			var nodeID string
			if a.NodeID != nil {
				nodeID = pointer.GetString(a.NodeID)
			} else {
				nodeID = pointer.GetString(a.RunsOnNodeID)
			}
			aMap[nodeID] = append(aMap[nodeID], convertAgentToProto(a))
		}
	}

	sMap := make(map[string][]*nodev1beta1.UniversalNode_Service, len(services))
	for _, s := range services {
		sMap[s.NodeID] = append(sMap[s.NodeID], &nodev1beta1.UniversalNode_Service{
			ServiceId:   s.ServiceID,
			ServiceType: string(s.ServiceType),
			ServiceName: s.ServiceName,
		})
	}

	result, _, err := s.vmClient.Query(ctx, upQuery, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute an instant VM query")
	}

	metrics := make(map[string]int, len(result.(model.Vector))) //nolint:forcetypeassert
	for _, v := range result.(model.Vector) {                   //nolint:forcetypeassert
		nodeID := string(v.Metric[model.LabelName("node_id")])
		// Sometimes we may see several metrics for the same node, so we just take the first one.
		if _, ok := metrics[nodeID]; !ok {
			metrics[nodeID] = int(v.Value)
		}
	}

	res := make([]*nodev1beta1.UniversalNode, len(nodes))
	for i, node := range nodes {
		labels, err := node.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		uNode := &nodev1beta1.UniversalNode{
			Address:       node.Address,
			CustomLabels:  labels,
			NodeId:        node.NodeID,
			NodeName:      node.NodeName,
			NodeType:      string(node.NodeType),
			Az:            node.AZ,
			CreatedAt:     timestamppb.New(node.CreatedAt),
			ContainerId:   pointer.GetString(node.ContainerID),
			ContainerName: pointer.GetString(node.ContainerName),
			Distro:        node.Distro,
			MachineId:     pointer.GetString(node.MachineID),
			NodeModel:     node.NodeModel,
			Region:        pointer.GetString(node.Region),
			UpdatedAt:     timestamppb.New(node.UpdatedAt),
		}

		if metric, ok := metrics[node.NodeID]; ok {
			switch metric {
			// We assume there can only be metric values of either 1(UP) or 0(DOWN).
			case 0:
				uNode.Status = nodev1beta1.UniversalNode_STATUS_DOWN
			case 1:
				uNode.Status = nodev1beta1.UniversalNode_STATUS_UP
			}
		} else {
			uNode.Status = nodev1beta1.UniversalNode_STATUS_UNKNOWN
		}

		if uAgents, ok := aMap[node.NodeID]; ok {
			uNode.Agents = uAgents
		}

		if uServices, ok := sMap[node.NodeID]; ok {
			uNode.Services = uServices
		}

		res[i] = uNode
	}

	return &nodev1beta1.ListNodeResponse{
		Nodes: res,
	}, nil
}

const nodeUpQuery = `up{job=~".*_hr$",node_id=%q}`

// GetNode returns a single Node by ID.
func (s *MgmtNodeService) GetNode(ctx context.Context, req *nodev1beta1.GetNodeRequest) (*nodev1beta1.GetNodeResponse, error) {
	node, err := models.FindNodeByID(s.db.Querier, req.NodeId)
	if err != nil {
		return nil, err
	}

	result, _, err := s.vmClient.Query(ctx, fmt.Sprintf(nodeUpQuery, req.NodeId), time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute an instant VM query")
	}

	metrics := make(map[string]int, len(result.(model.Vector))) //nolint:forcetypeassert
	for _, v := range result.(model.Vector) {                   //nolint:forcetypeassert
		nodeID := string(v.Metric[model.LabelName("node_id")])
		// Sometimes we may see several metrics for the same node, so we just take the first one.
		if _, ok := metrics[nodeID]; !ok {
			metrics[nodeID] = int(v.Value)
		}
	}

	labels, err := node.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	uNode := &nodev1beta1.UniversalNode{
		Address:       node.Address,
		Az:            node.AZ,
		CreatedAt:     timestamppb.New(node.CreatedAt),
		ContainerId:   pointer.GetString(node.ContainerID),
		ContainerName: pointer.GetString(node.ContainerName),
		CustomLabels:  labels,
		Distro:        node.Distro,
		MachineId:     pointer.GetString(node.MachineID),
		NodeId:        node.NodeID,
		NodeName:      node.NodeName,
		NodeType:      string(node.NodeType),
		NodeModel:     node.NodeModel,
		Region:        pointer.GetString(node.Region),
		UpdatedAt:     timestamppb.New(node.UpdatedAt),
	}

	if metric, ok := metrics[node.NodeID]; ok {
		switch metric {
		// We assume there can only be metric values of either 1(UP) or 0(DOWN).
		case 0:
			uNode.Status = nodev1beta1.UniversalNode_STATUS_DOWN
		case 1:
			uNode.Status = nodev1beta1.UniversalNode_STATUS_UP
		}
	} else {
		uNode.Status = nodev1beta1.UniversalNode_STATUS_UNKNOWN
	}

	return &nodev1beta1.GetNodeResponse{
		Node: uNode,
	}, nil
}
