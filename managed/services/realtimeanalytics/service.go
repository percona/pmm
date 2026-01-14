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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	store    *Store

	rtav1.UnimplementedRealtimeAnalyticsServiceServer
	rtav1.UnimplementedCollectorServiceServer
}

// NewService creates a new Real-Time Analytics service.
func NewService(db *reform.DB, registry agentsRegistry, store *Store) *Service {
	return &Service{
		db:       db,
		registry: registry,
		store:    store,
	}
}

// ListRunningRealtimeAgents returns the list of currently running RTA agents (gRPC handler).
func (s *Service) ListRunningRealtimeAgents(_ context.Context, req *rtav1.ListRunningRealtimeAgentsRequest) (*rtav1.ListRunningRealtimeAgentsResponse, error) {
	realtimeAgentType := models.RTAMongoDBAgentType
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

// ChangeRealtimeAnalytics enables or disables RTA for a service (gRPC handler).
func (s *Service) ChangeRealtimeAnalytics(_ context.Context, req *rtav1.ChangeRealtimeAnalyticsRequest) (*rtav1.ChangeRealtimeAnalyticsResponse, error) {
	err := s.db.InTransaction(func(tx *reform.TX) error {
		// Find existing RTA agents for this service
		agentType := models.RTAMongoDBAgentType
		existingAgents, err := models.FindAgents(tx.Querier, models.AgentFilters{
			ServiceID: req.ServiceId,
			AgentType: &agentType,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to find RTA agents for service %s: %v", req.ServiceId, err)
		}

		if len(existingAgents) != 0 {
			// Agent exists - update its state
			agent := existingAgents[0]
			agent.Disabled = !req.Enable

			if req.Enable {
				// Set EnabledAt when enabling
				now := time.Now()
				agent.RTAOptions.EnabledAt = &now
			} else {
				// Clear EnabledAt when disabling
				agent.RTAOptions.EnabledAt = nil
				// Clear query data from store when disabling
				s.store.Clear(req.ServiceId)
			}

			if err := tx.Update(agent); err != nil {
				return status.Errorf(codes.Internal, "Failed to update RTA agent %s: %v", agent.AgentID, err)
			}
		} else if req.Enable {
			// Agent doesn't exist - create it with appropriate state
			// CreateMongoDBRealtimeAgent will validate service type and find pmm-agent
			_, err = models.CreateMongoDBRealtimeAgent(tx.Querier, req.ServiceId, nil, !req.Enable)
			if err != nil {
				return status.Errorf(codes.Internal, "Failed to create RTA agent for service %s: %v", req.ServiceId, err)
			}
		}
		// TODO: send set state request to pmm-agent

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &rtav1.ChangeRealtimeAnalyticsResponse{}, nil
}

// Collect handles incoming streaming RTA query data from agents (gRPC handler).
func (s *Service) Collect(g grpc.ClientStreamingServer[rtav1.RealtimeAnalyticsQueryRequest, rtav1.RealtimeAnalyticsQueryResponse]) error {
	// TODO implement me
	panic("implement me")
}
