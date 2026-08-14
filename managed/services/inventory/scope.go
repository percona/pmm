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

package inventory

import (
	"context"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/auth"
)

// scopeNodes drops every Node outside the caller's scope. Unconfined callers keep the
// full list.
func scopeNodes(ctx context.Context, nodes []*models.Node) []*models.Node {
	scoped, ok := auth.NodeScope(ctx)
	if !ok {
		return nodes
	}

	return slices.DeleteFunc(nodes, func(n *models.Node) bool { return n.NodeID != scoped })
}

// scopeServices drops every Service that does not belong to the caller's node.
func scopeServices(ctx context.Context, list []*models.Service) []*models.Service {
	scoped, ok := auth.NodeScope(ctx)
	if !ok {
		return list
	}

	return slices.DeleteFunc(list, func(s *models.Service) bool { return s.NodeID != scoped })
}

// scopeAgents drops every Agent that does not belong to the caller's node.
func scopeAgents(ctx context.Context, q *reform.Querier, list []*models.Agent) ([]*models.Agent, error) {
	scoped, ok := auth.NodeScope(ctx)
	if !ok {
		return list, nil
	}

	kept := make([]*models.Agent, 0, len(list))
	for _, a := range list {
		nodeID, err := models.FindNodeIDForAgent(q, a.AgentID)
		if err != nil {
			return nil, err
		}
		if nodeID == scoped {
			kept = append(kept, a)
		}
	}

	return kept, nil
}

// CheckNodeScope returns an error when the caller's token is bound to a different Node.
func (s *NodesService) CheckNodeScope(ctx context.Context, nodeID string) error {
	return auth.CheckNodeScope(ctx, nodeID)
}

// CheckAddNodeScope refuses Node creation for a caller confined to a node: adding nodes is
// what registration does, and a node's own token may not enroll further ones.
func (s *NodesService) CheckAddNodeScope(ctx context.Context) error {
	if _, scoped := auth.NodeScope(ctx); scoped {
		return status.Error(codes.PermissionDenied, "This token may not add nodes.")
	}

	return nil
}

// CheckServiceScope returns an error when the caller's token is bound to a Node other than
// the one the Service belongs to.
func (ss *ServicesService) CheckServiceScope(ctx context.Context, serviceID string) error {
	return auth.CheckServiceScope(ctx, ss.db.Querier, serviceID)
}

// CheckNodeScope returns an error when the caller's token is bound to a different Node.
func (ss *ServicesService) CheckNodeScope(ctx context.Context, nodeID string) error {
	return auth.CheckNodeScope(ctx, nodeID)
}

// CheckAgentScope returns an error when the caller's token is bound to a Node other than the
// one the Agent belongs to.
func (as *AgentsService) CheckAgentScope(ctx context.Context, agentID string) error {
	return auth.CheckAgentScope(ctx, as.db.Querier, agentID)
}

// CheckAddAgentScope confines Agent creation to the caller's own Node. The new agent is
// placed by its pmm-agent, its service or its node, whichever the request carries.
func (as *AgentsService) CheckAddAgentScope(ctx context.Context, pmmAgentID, serviceID, nodeID string) error {
	if _, scoped := auth.NodeScope(ctx); !scoped {
		return nil
	}

	switch {
	case pmmAgentID != "":
		return auth.CheckAgentScope(ctx, as.db.Querier, pmmAgentID)
	case serviceID != "":
		return auth.CheckServiceScope(ctx, as.db.Querier, serviceID)
	case nodeID != "":
		return auth.CheckNodeScope(ctx, nodeID)
	default:
		return status.Error(codes.PermissionDenied, "This token may only add agents to its own node.")
	}
}
