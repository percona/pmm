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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/auth"
)

// RegisterNode performs the registration of a new node.
func (s *ManagementService) RegisterNode(ctx context.Context, req *managementv1.RegisterNodeRequest) (*managementv1.RegisterNodeResponse, error) {
	res := &managementv1.RegisterNodeResponse{}

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		node, err := models.FindNodeByName(tx.Querier, req.NodeName)
		switch status.Code(err) { //nolint:exhaustive
		case codes.OK:
			if !req.Reregister {
				return status.Errorf(codes.AlreadyExists, "Node with name %q already exists.", req.NodeName)
			}
			err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade)
		case codes.NotFound:
			err = nil
		}
		if err != nil {
			return err
		}

		node, err = models.CheckUniqueNodeInstanceRegion(tx.Querier, req.Address, &req.Region)
		switch status.Code(err) { //nolint:exhaustive
		case codes.OK:
			// nothing
		case codes.AlreadyExists:
			if !req.Reregister {
				return err
			}
			err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade)
		}
		if err != nil {
			return err
		}

		nodeType, err := nodeType(req.NodeType)
		if err != nil {
			return err
		}
		node, err = models.CreateNode(tx.Querier, nodeType, &models.CreateNodeParams{
			NodeName:      req.NodeName,
			MachineID:     pointer.ToStringOrNil(req.MachineId),
			Distro:        req.Distro,
			NodeModel:     req.NodeModel,
			AZ:            req.Az,
			ContainerID:   pointer.ToStringOrNil(req.ContainerId),
			ContainerName: pointer.ToStringOrNil(req.ContainerName),
			CustomLabels:  req.CustomLabels,
			Address:       req.Address,
			Region:        pointer.ToStringOrNil(req.Region),
		})
		if err != nil {
			return err
		}

		n, err := services.ToAPINode(node)
		if err != nil {
			return err
		}
		switch n := n.(type) {
		case *inventoryv1.GenericNode:
			res.GenericNode = n
		case *inventoryv1.ContainerNode:
			res.ContainerNode = n
		default:
			return status.Errorf(codes.InvalidArgument, "Unsupported Node type %q.", req.NodeType)
		}

		pmmAgent, err := models.CreatePMMAgent(tx.Querier, node.NodeID, nil)
		if err != nil {
			return err
		}

		a, err := services.ToAPIAgent(tx.Querier, pmmAgent)
		if err != nil {
			return err
		}
		res.PmmAgent = a.(*inventoryv1.PMMAgent) //nolint:forcetypeassert

		_, err = models.CreateNodeExporter(tx.Querier, pmmAgent.AgentID, nil, isPushMode(req.MetricsMode), req.ExposeExporter,
			req.DisableCollectors, pointer.ToStringOrNil(req.AgentPassword), "")
		if err != nil {
			return err
		}
		return err
	})
	if e != nil {
		return nil, e
	}

	authHeaders, _ := auth.GetHeadersFromContext(ctx)
	token := auth.GetTokenFromHeaders(authHeaders)
	if token != "" {
		res.Token = token
	} else {
		_, res.Token, e = s.grafanaClient.CreateServiceAccount(ctx, req.NodeName, req.Reregister)
		if e != nil {
			return nil, e
		}
	}

	return res, nil
}

// UnregisterNode unregisters the node.
func (s *ManagementService) UnregisterNode(ctx context.Context, req *managementv1.UnregisterNodeRequest) (*managementv1.UnregisterNodeResponse, error) {
	idsToKick := make(map[string]struct{})
	idsToSetState := make(map[string]struct{})

	node, err := models.FindNodeByID(s.db.Querier, req.NodeId)
	if err != nil {
		return nil, err
	}

	if e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if req.Force {
			mode = models.RemoveCascade

			agents, err := models.FindPMMAgentsRunningOnNode(tx.Querier, node.NodeID)
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				idsToKick[a.AgentID] = struct{}{}
			}

			agents, err = models.FindAgents(tx.Querier, models.AgentFilters{NodeID: node.NodeID})
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				if a.PMMAgentID != nil {
					idsToSetState[pointer.GetString(a.PMMAgentID)] = struct{}{}
				}
			}

			agents, err = models.FindPMMAgentsForServicesOnNode(tx.Querier, node.NodeID)
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				idsToSetState[a.AgentID] = struct{}{}
			}
		}
		return models.RemoveNode(tx.Querier, node.NodeID, mode)
	}); e != nil {
		return nil, e
	}

	for id := range idsToSetState {
		s.state.RequestStateUpdate(ctx, id)
	}
	for id := range idsToKick {
		s.r.Kick(ctx, id)
	}

	if req.Force {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		s.vmdb.RequestConfigurationUpdate()
	}

	warning, err := s.grafanaClient.DeleteServiceAccount(ctx, node.NodeName, req.Force)
	if err != nil {
		// TODO: need to pass the logger to the service
		// s.l.WithError(err).Error("deleting service account")
		return &managementv1.UnregisterNodeResponse{ //nolint:nilerr
			Warning: err.Error(),
		}, nil
	}

	return &managementv1.UnregisterNodeResponse{
		Warning: warning,
	}, nil
}

const upQuery = `up{job=~".*_hr$"}`

// ListNodes returns a filtered list of Nodes.
func (s *ManagementService) ListNodes(ctx context.Context, req *managementv1.ListNodesRequest) (*managementv1.ListNodesResponse, error) {
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

		nodes, err = models.FindNodes(tx.Querier, filters)
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

		services, err = models.FindServices(tx.Querier, models.ServiceFilters{})
		if err != nil {
			return err
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	convertAgentToProto := func(agent *models.Agent) *managementv1.UniversalNode_Agent {
		return &managementv1.UniversalNode_Agent{
			AgentId:     agent.AgentID,
			AgentType:   string(agent.AgentType),
			Status:      agent.Status,
			IsConnected: s.r.IsConnected(agent.AgentID),
		}
	}

	aMap := make(map[string][]*managementv1.UniversalNode_Agent, len(nodes))
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

	sMap := make(map[string][]*managementv1.UniversalNode_Service, len(services))
	for _, s := range services {
		sMap[s.NodeID] = append(sMap[s.NodeID], &managementv1.UniversalNode_Service{
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

	res := make([]*managementv1.UniversalNode, len(nodes))
	for i, node := range nodes {
		labels, err := node.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		uNode := &managementv1.UniversalNode{
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
				uNode.Status = managementv1.UniversalNode_STATUS_DOWN
			case 1:
				uNode.Status = managementv1.UniversalNode_STATUS_UP
			}
		} else {
			uNode.Status = managementv1.UniversalNode_STATUS_UNKNOWN
		}

		if uAgents, ok := aMap[node.NodeID]; ok {
			uNode.Agents = uAgents
		}

		if uServices, ok := sMap[node.NodeID]; ok {
			uNode.Services = uServices
		}

		res[i] = uNode
	}

	return &managementv1.ListNodesResponse{
		Nodes: res,
	}, nil
}

const nodeUpQuery = `up{job=~".*_hr$",node_id=%q}`

// GetNode returns a single Node by ID.
func (s *ManagementService) GetNode(ctx context.Context, req *managementv1.GetNodeRequest) (*managementv1.GetNodeResponse, error) {
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

	uNode := &managementv1.UniversalNode{
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
			uNode.Status = managementv1.UniversalNode_STATUS_DOWN
		case 1:
			uNode.Status = managementv1.UniversalNode_STATUS_UP
		}
	} else {
		uNode.Status = managementv1.UniversalNode_STATUS_UNKNOWN
	}

	return &managementv1.GetNodeResponse{
		Node: uNode,
	}, nil
}
