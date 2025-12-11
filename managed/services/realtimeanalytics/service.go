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

// Package realtimeanalytics provides service for managing Real-Time Analytics.
package realtimeanalytics

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
)

// agentsRegistry provides information about running agents.
type agentsRegistry interface {
	IsConnected(pmmAgentID string) bool
}

// Service provides API for managing Real-Time Analytics.
type Service struct {
	db       *reform.DB
	registry agentsRegistry

	rtav1.UnimplementedRealtimeAnalyticsServiceServer
}

// NewService creates a new Real-Time Analytics service.
func NewService(db *reform.DB, registry agentsRegistry) *Service {
	return &Service{
		db:       db,
		registry: registry,
	}
}

// ListRunningRealtimeAgents returns the list of currently running RTA agents (gRPC handler).
func (s *Service) ListRunningRealtimeAgents(_ context.Context, req *rtav1.ListRunningRealtimeAgentsRequest) (*rtav1.ListRunningRealtimeAgentsResponse, error) {
	realtimeAgentType := models.MongoDBRealtimeAgentType
	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{
		AgentType: &realtimeAgentType,
	})
	if err != nil {
		return nil, err
	}

	response := &rtav1.ListRunningRealtimeAgentsResponse{
		Agents: []*rtav1.RunningRealtimeAgent{},
	}

	for _, agent := range agents {
		// Skip disabled agents
		if agent.Disabled {
			continue
		}

		// Get service details
		if agent.ServiceID == nil {
			continue
		}
		service, err := models.FindServiceByID(s.db.Querier, *agent.ServiceID)
		if err != nil {
			return nil, err
		}

		// Apply cluster filter if specified
		if req.Cluster != "" && service.Cluster != req.Cluster {
			continue
		}

		// Determine started_at from RTAOptions.EnabledAt or fall back to CreatedAt
		startedAt := agent.CreatedAt
		if agent.RTAOptions.EnabledAt != nil {
			startedAt = *agent.RTAOptions.EnabledAt
		}

		// Determine status: if pmm-agent is disconnected, show UNKNOWN
		status := inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status])
		if agent.PMMAgentID == nil || !s.registry.IsConnected(*agent.PMMAgentID) {
			status = inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN
		}

		response.Agents = append(response.Agents, &rtav1.RunningRealtimeAgent{
			AgentId:     agent.AgentID,
			ServiceId:   service.ServiceID,
			ServiceName: service.ServiceName,
			Cluster:     service.Cluster,
			StartedAt:   timestamppb.New(startedAt),
			Status:      status,
		})
	}

	return response, nil
}
