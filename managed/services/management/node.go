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
func (s *ManagementService) RegisterNode(ctx context.Context, req *managementv1.RegisterNodeRequest) (*managementv1.RegisterNodeResponse, error) { //nolint:gocognit
	res := &managementv1.RegisterNodeResponse{}

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		node, err := models.FindNodeByName(tx.Querier, req.NodeName)
		switch status.Code(err) { //nolint:exhaustive
		case codes.OK:
			if !req.Reregister {
				return status.Errorf(codes.AlreadyExists, "Node with name %s already exists.", req.NodeName)
			}
			err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade)
		case codes.NotFound:
			err = nil
		}
		if err != nil {
			return err
		}

		node, err = models.CheckUniqueNodeAddressRegion(tx.Querier, req.Address, &req.Region)
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
			InstanceID:    req.InstanceId,
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

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if req.Force {
			mode = models.RemoveCascade

			agents, err := models.FindPMMAgentsRunningOnNode(tx.Querier, node.NodeID)
			if err != nil {
				return fmt.Errorf("failed to find pmm-agent on node %s: %w", node.NodeID, err)
			}
			for _, a := range agents {
				idsToKick[a.AgentID] = struct{}{}
			}

			agents, err = models.FindAgents(tx.Querier, models.AgentFilters{NodeID: node.NodeID})
			if err != nil {
				return fmt.Errorf("failed to find agents on node %s: %w", node.NodeID, err)
			}
			for _, a := range agents {
				if a.PMMAgentID != nil {
					idsToSetState[pointer.GetString(a.PMMAgentID)] = struct{}{}
				}
			}

			agents, err = models.FindPMMAgentsForServicesOnNode(tx.Querier, node.NodeID)
			if err != nil {
				return fmt.Errorf("failed to find pmm-agent on node %s: %w", node.NodeID, err)
			}
			for _, a := range agents {
				idsToSetState[a.AgentID] = struct{}{}
			}
		}
		return models.RemoveNode(tx.Querier, node.NodeID, mode)
	})
	if e != nil {
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
func (s *ManagementService) ListNodes(ctx context.Context, req *managementv1.ListNodesRequest) (*managementv1.ListNodesResponse, error) { //nolint:gocognit
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

	metrics, err := s.queryNodeUpMetrics(ctx, upQuery, false)
	if err != nil {
		return nil, err
	}

	// Same fallback as ListServices: while vmagents replay buffered data after an outage,
	// VM has no fresh samples although nodes are fine. Use the last sample within
	// staleStatusWindow for nodes whose pmm-agent is currently connected.
	staleMetrics := map[string]int{}
	for _, node := range nodes {
		if _, ok := metrics[node.NodeID]; !ok {
			staleMetrics, err = s.queryNodeUpMetrics(ctx, upQuery, true)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	connectedCache := make(map[string]bool)

	res := make([]*managementv1.UniversalNode, len(nodes))
	for i, node := range nodes {
		labels, err := node.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		uNode := &managementv1.UniversalNode{
			Address:         node.Address,
			CustomLabels:    labels,
			NodeId:          node.NodeID,
			NodeName:        node.NodeName,
			NodeType:        string(node.NodeType),
			Az:              node.AZ,
			CreatedAt:       timestamppb.New(node.CreatedAt),
			ContainerId:     pointer.GetString(node.ContainerID),
			ContainerName:   pointer.GetString(node.ContainerName),
			Distro:          node.Distro,
			MachineId:       pointer.GetString(node.MachineID),
			NodeModel:       node.NodeModel,
			Region:          pointer.GetString(node.Region),
			UpdatedAt:       timestamppb.New(node.UpdatedAt),
			InstanceId:      node.InstanceID,
			IsPmmServerNode: node.IsPMMServerNode,
		}

		freshUp, hasFresh := metrics[node.NodeID]
		staleUp, hasStale := staleMetrics[node.NodeID]
		switch {
		case hasFresh:
			uNode.Status = nodeStatusFromUp(freshUp)
		case hasStale && s.nodeHasConnectedPMMAgent(agents, node.NodeID, connectedCache):
			uNode.Status = nodeStatusFromUp(staleUp)
		default:
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

	metrics, err := s.queryNodeUpMetrics(ctx, fmt.Sprintf(nodeUpQuery, req.NodeId), false)
	if err != nil {
		return nil, err
	}

	// Same fallback as ListNodes: use the last sample within staleStatusWindow
	// when there is no fresh one and the node's pmm-agent is connected.
	staleMetrics := map[string]int{}
	connectedCache := make(map[string]bool)
	var nodeAgents []*models.Agent
	if _, ok := metrics[node.NodeID]; !ok {
		staleMetrics, err = s.queryNodeUpMetrics(ctx, fmt.Sprintf(nodeUpQuery, req.NodeId), true)
		if err != nil {
			return nil, err
		}
		nodeAgents, err = models.FindAgents(s.db.WithContext(ctx), models.AgentFilters{NodeID: node.NodeID})
		if err != nil {
			return nil, err
		}
	}

	labels, err := node.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	uNode := &managementv1.UniversalNode{
		Address:         node.Address,
		Az:              node.AZ,
		CreatedAt:       timestamppb.New(node.CreatedAt),
		ContainerId:     pointer.GetString(node.ContainerID),
		ContainerName:   pointer.GetString(node.ContainerName),
		CustomLabels:    labels,
		Distro:          node.Distro,
		MachineId:       pointer.GetString(node.MachineID),
		NodeId:          node.NodeID,
		NodeName:        node.NodeName,
		NodeType:        string(node.NodeType),
		NodeModel:       node.NodeModel,
		Region:          pointer.GetString(node.Region),
		UpdatedAt:       timestamppb.New(node.UpdatedAt),
		IsPmmServerNode: node.IsPMMServerNode,
	}

	freshUp, hasFresh := metrics[node.NodeID]
	staleUp, hasStale := staleMetrics[node.NodeID]
	switch {
	case hasFresh:
		uNode.Status = nodeStatusFromUp(freshUp)
	case hasStale && s.nodeHasConnectedPMMAgent(nodeAgents, node.NodeID, connectedCache):
		uNode.Status = nodeStatusFromUp(staleUp)
	default:
		uNode.Status = managementv1.UniversalNode_STATUS_UNKNOWN
	}

	return &managementv1.GetNodeResponse{
		Node: uNode,
	}, nil
}

// queryNodeUpMetrics returns the values of the node "up" metrics keyed by node ID.
// With stale=true it returns the most recent sample within staleStatusWindow instead
// of only fresh (non-stale) samples.
func (s *ManagementService) queryNodeUpMetrics(ctx context.Context, query string, stale bool) (map[string]int, error) {
	if stale {
		query = fmt.Sprintf("last_over_time(%s[%s])", query, staleStatusWindow)
	}

	result, _, err := s.vmClient.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to execute an instant VM query: %w", err)
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected VM query result type %T", result)
	}
	metrics := make(map[string]int, len(vector))
	for _, v := range vector {
		nodeID := string(v.Metric[model.LabelName("node_id")])
		// Sometimes we may see several metrics for the same node, so we just take the first one.
		if _, ok := metrics[nodeID]; !ok {
			metrics[nodeID] = int(v.Value)
		}
	}
	return metrics, nil
}

// nodeStatusFromUp converts an "up" metric value to a node status.
func nodeStatusFromUp(up int) managementv1.UniversalNode_Status {
	// We assume there can only be metric values of either 1(UP) or 0(DOWN).
	switch up {
	case 0:
		return managementv1.UniversalNode_STATUS_DOWN
	case 1:
		return managementv1.UniversalNode_STATUS_UP
	default:
		return managementv1.UniversalNode_STATUS_UNKNOWN
	}
}

// nodeHasConnectedPMMAgent reports whether a pmm-agent providing this node's metrics is
// currently connected. The connectedCache map memoizes registry lookups within one request.
func (s *ManagementService) nodeHasConnectedPMMAgent(agents []*models.Agent, nodeID string, connectedCache map[string]bool) bool {
	for _, agent := range agents {
		if pointer.GetString(agent.NodeID) != nodeID && pointer.GetString(agent.RunsOnNodeID) != nodeID {
			continue
		}
		pmmAgentID := pointer.GetString(agent.PMMAgentID)
		if agent.AgentType == models.PMMAgentType {
			pmmAgentID = agent.AgentID
		}
		if pmmAgentID == "" {
			continue
		}
		connected, ok := connectedCache[pmmAgentID]
		if !ok {
			connected = s.r.IsConnected(pmmAgentID)
			connectedCache[pmmAgentID] = connected
		}
		if connected {
			return true
		}
	}
	return false
}
