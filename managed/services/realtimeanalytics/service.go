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

// ChangeRealtimeAnalytics enables or disables RTA for a service or cluster (gRPC handler).
func (s *Service) ChangeRealtimeAnalytics(_ context.Context, req *rtav1.ChangeRealtimeAnalyticsRequest) (*rtav1.ChangeRealtimeAnalyticsResponse, error) {
	// Validate request: must have either service_id or cluster
	if req.GetServiceId() == "" && req.GetCluster() == "" {
		return nil, status.Error(codes.InvalidArgument, "Either service_id or cluster must be specified")
	}

	var serviceIDs []string

	// Get list of services based on target
	switch target := req.Target.(type) {
	case *rtav1.ChangeRealtimeAnalyticsRequest_ServiceId:
		// Single service
		serviceIDs = []string{target.ServiceId}

	case *rtav1.ChangeRealtimeAnalyticsRequest_Cluster:
		// All MongoDB services in cluster (RTA only supports MongoDB)
		serviceType := models.MongoDBServiceType
		services, err := models.FindServices(s.db.Querier, models.ServiceFilters{
			Cluster:     target.Cluster,
			ServiceType: &serviceType,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to find services in cluster: %v", err)
		}

		if len(services) == 0 {
			return nil, status.Errorf(codes.NotFound, "No MongoDB services found in cluster: %s", target.Cluster)
		}

		for _, service := range services {
			serviceIDs = append(serviceIDs, service.ServiceID)
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "Either service_id or cluster must be specified")
	}

	// Apply enable/disable to all target services
	err := s.db.InTransaction(func(tx *reform.TX) error {
		for _, serviceID := range serviceIDs {
			// Find existing RTA agents for this service
			agentType := models.MongoDBRealtimeAgentType
			existingAgents, err := models.FindAgents(tx.Querier, models.AgentFilters{
				ServiceID: serviceID,
				AgentType: &agentType,
			})
			if err != nil {
				return status.Errorf(codes.Internal, "Failed to find RTA agents for service %s: %v", serviceID, err)
			}

			if len(existingAgents) > 0 {
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
				}

				if err := tx.Update(agent); err != nil {
					return status.Errorf(codes.Internal, "Failed to update RTA agent %s: %v", agent.AgentID, err)
				}
			} else if req.Enable {
				// Agent doesn't exist - create it with appropriate state
				// CreateMongoDBRealtimeAgent will validate service type and find pmm-agent
				_, err = models.CreateMongoDBRealtimeAgent(tx.Querier, serviceID, nil, !req.Enable)
				if err != nil {
					return status.Errorf(codes.Internal, "Failed to create RTA agent for service %s: %v", serviceID, err)
				}
			}
			// TODO: send set state request to pmm-agent
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &rtav1.ChangeRealtimeAnalyticsResponse{}, nil
}
