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

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// Service provides API for managing Real-Time Analytics.
type Service struct {
	db           *reform.DB
	registry     agentsRegistry
	stateUpdater agentsStateUpdater
	store        *Store

	rtav1.UnimplementedRealtimeAnalyticsServiceServer
	rtav1.UnimplementedCollectorServiceServer
}

// NewService creates a new Real-Time Analytics service.
func NewService(db *reform.DB, registry agentsRegistry, stateUpdater agentsStateUpdater, store *Store) *Service {
	return &Service{
		db:           db,
		registry:     registry,
		stateUpdater: stateUpdater,
		store:        store,
	}
}

// ListSessions returns the list of currently running Real-Time Analytics Sessions (gRPC handler).
func (s *Service) ListSessions(_ context.Context, req *rtav1.ListSessionsRequest) (*rtav1.ListSessionsResponse, error) {
	response := &rtav1.ListSessionsResponse{
		Sessions: []*rtav1.Session{},
	}

	for _, at := range models.GetRTAAgentTypes() {
		// fetch all RTA agents of this type
		agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{
			AgentType: &at,
			Disabled:  pointer.To(false), // fetch enabled only
		})
		if err != nil {
			return nil, err
		}

		for _, agent := range agents {
			// Skip agents not linked to a service
			if agent.ServiceID == nil {
				continue
			}

			service, err := models.FindServiceByID(s.db.Querier, *agent.ServiceID)
			if err != nil {
				return nil, err
			}

			// Apply cluster filter if specified
			if req.ClusterName != "" && service.Cluster != req.ClusterName {
				continue
			}

			response.Sessions = append(response.Sessions, s.convertAgentToSession(agent, service))
		}
	}

	return response, nil
}

// StartSession starts Real-Time Analytics Session for a specified service (gRPC handler).
func (s *Service) StartSession(ctx context.Context, req *rtav1.StartSessionRequest) (*rtav1.StartSessionResponse, error) {
	var err error
	var session *rtav1.Session
	// Contains pmm-agent ID to be updated after the change with RTA agent.
	var pmmAgentIDToUpdate string
	err = s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		// Validate that the service exists and is of a supported type for RTA
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		var rtaAgentType models.AgentType
		// Check that service type supports RTA
		if rtaAgentType, err = getRTAAgentTypeForServiceType(service.ServiceType); err != nil {
			return status.Errorf(codes.InvalidArgument,
				"Service %s of type %s does not support Real-Time Analytics",
				req.ServiceId, service.ServiceType)
		}

		// Find existing RTA agents for this service
		existingRTAAgents, err := models.FindAgents(tx.Querier, models.AgentFilters{
			ServiceID: req.ServiceId,
			AgentType: &rtaAgentType,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to find Real-Time Analytics agents for service %s: %v", req.ServiceId, err)
		}

		if len(existingRTAAgents) != 0 {
			// RTA Agent exists - update its state if required
			rtaAgent := existingRTAAgents[0]
			if !rtaAgent.Disabled {
				session = s.convertAgentToSession(rtaAgent, service)
				return nil // Already enabled, nothing to do
			}

			rtaAgent.Disabled = false
			rtaAgent.CreatedAt = time.Now()
			if err := tx.Update(rtaAgent); err != nil {
				return status.Errorf(codes.Internal, "Failed to update Real-Time Analytics agent %s: %v", rtaAgent.AgentID, err)
			}

			// Request state update to pmm-agent
			pmmAgentIDToUpdate = *rtaAgent.PMMAgentID
			session = s.convertAgentToSession(rtaAgent, service)
			return nil
		}

		// Create new RTA agent for the requested service

		// In this context we do not have any credentials for connecting to the service.
		// So we need to copy them from existing agents for this service.
		// Retrieve credentials and pmm-agent ID from existing MongoDB agents for this service
		// Try to find from QAN or exporter agents
		var agentTypes []models.AgentType
		switch service.ServiceType {
		case models.MongoDBServiceType:
			agentTypes = []models.AgentType{
				models.QANMongoDBProfilerAgentType,
				models.QANMongoDBMongologAgentType,
				models.MongoDBExporterType,
			}
			// Add other service types once RTA is supported for them
		default:
			return status.Errorf(codes.InvalidArgument,
				"Service %s of type %s does not support Real-Time Analytics",
				req.ServiceId, service.ServiceType)
		}

		var existingAgent *models.Agent
		for _, agentType := range agentTypes {
			agents, err := models.FindAgents(tx.Querier, models.AgentFilters{
				ServiceID: service.ServiceID,
				AgentType: &agentType,
			})
			if err != nil {
				return err
			}
			if len(agents) != 0 {
				existingAgent = agents[0]
				break
			}
		}

		if existingAgent == nil {
			return status.Errorf(codes.FailedPrecondition,
				"No existing %s agent found for service %s to retrieve credentials and pmm-agent ID",
				service.ServiceType, service.ServiceID)
		}

		if existingAgent.PMMAgentID == nil {
			return status.Errorf(codes.FailedPrecondition,
				"Existing %s agent for service %s has no pmm-agent ID",
				service.ServiceType, service.ServiceID)
		}

		// Create the RTA agent with credentials and pmm-agent ID from existing agent for the requested service.
		rtaAgent, err := models.CreateAgent(tx.Querier, rtaAgentType, &models.CreateAgentParams{
			PMMAgentID:        *existingAgent.PMMAgentID,
			ServiceID:         service.ServiceID,
			Username:          pointer.GetString(existingAgent.Username),
			Password:          pointer.GetString(existingAgent.Password),
			TLS:               existingAgent.TLS,
			TLSSkipVerify:     existingAgent.TLSSkipVerify,
			MongoDBOptions:    existingAgent.MongoDBOptions,
			MySQLOptions:      existingAgent.MySQLOptions,
			PostgreSQLOptions: existingAgent.PostgreSQLOptions,
			ValkeyOptions:     existingAgent.ValkeyOptions,
			LogLevel:          services.SpecifyLogLevel(inventoryv1.LogLevel_LOG_LEVEL_UNSPECIFIED, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			Disabled:          false,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to create Real-Time Analytics agent for service %s: %v", req.ServiceId, err)
		}
		pmmAgentIDToUpdate = *rtaAgent.PMMAgentID
		session = s.convertAgentToSession(rtaAgent, service)

		return nil
	})
	if err != nil {
		return nil, err
	}

	if pmmAgentIDToUpdate != "" {
		// Request state update to pmm-agent
		s.stateUpdater.RequestStateUpdate(ctx, pmmAgentIDToUpdate)
	}
	return &rtav1.StartSessionResponse{Session: session}, nil
}

// StopSession stops Real-Time Analytics Session for a specified service (gRPC handler).
func (s *Service) StopSession(ctx context.Context, req *rtav1.StopSessionRequest) (*rtav1.StopSessionResponse, error) {
	// Contains pmm-agent ID to be updated after the change with RTA agent.
	var pmmAgentIDToUpdate string
	err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		// Validate that the service exists and is of a supported type for RTA
		service, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		var agentType models.AgentType
		// Check that service type supports RTA
		if agentType, err = getRTAAgentTypeForServiceType(service.ServiceType); err != nil {
			return status.Errorf(codes.InvalidArgument,
				"Service %s of type %s does not support Real-Time Analytics",
				req.ServiceId, service.ServiceType)
		}

		// Find existing RTA agents for this service
		existingRTAAgents, err := models.FindAgents(tx.Querier, models.AgentFilters{
			ServiceID: req.ServiceId,
			AgentType: &agentType,
			Disabled:  pointer.To(false), // fetch enabled only
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to find Real-Time Analytics agents for service %s: %v", req.ServiceId, err)
		}

		if len(existingRTAAgents) == 0 {
			// No RTA Agent exists for this service or already disabled - nothing to do
			return nil
		}

		// RTA Agent exists - update its state
		rtaAgent := existingRTAAgents[0]
		rtaAgent.Disabled = true
		if err = tx.Update(rtaAgent); err != nil {
			return status.Errorf(codes.Internal, "Failed to update Real-Time Analytics agent %s: %v", rtaAgent.AgentID, err)
		}
		pmmAgentIDToUpdate = *rtaAgent.PMMAgentID
		return nil
	})
	if err != nil {
		return nil, err
	}

	if pmmAgentIDToUpdate != "" {
		// Request state update to pmm-agent
		s.stateUpdater.RequestStateUpdate(ctx, pmmAgentIDToUpdate)
		// Clear query data from store
		s.store.Clear(req.ServiceId)
	}

	return &rtav1.StopSessionResponse{}, nil
}

// SearchQueries returns the list of currently running Database Queries for specified services. (gRPC handler).
func (s *Service) SearchQueries(_ context.Context, req *rtav1.SearchQueriesRequest) (*rtav1.SearchQueriesResponse, error) {
	// Validate that all the requested services exist
	for _, serviceID := range req.ServiceIds {
		if _, err := models.FindServiceByID(s.db.Querier, serviceID); err != nil {
			return nil, err
		}
	}

	resp := &rtav1.SearchQueriesResponse{
		Queries: []*rtav1.QueryData{},
	}

	for _, serviceID := range req.ServiceIds {
		// Queries for a particular service requested
		resp.Queries = append(resp.Queries, s.store.Get(serviceID)...)

		// Apply limit if specified
		if req.Limit > 0 && int64(len(resp.Queries)) > req.Limit {
			resp.Queries = resp.Queries[:req.Limit]
			break
		}
	}

	return resp, nil
}

// Collect handles incoming streaming RTA query data from agents (gRPC handler).
func (s *Service) Collect(_ grpc.ClientStreamingServer[rtav1.CollectRequest, rtav1.CollectResponse]) error {
	return status.Errorf(codes.Unimplemented, "ListQueries is not implemented yet")
}

// Helpers

func convertAgentStatusToSessionStatus(status inventoryv1.AgentStatus) rtav1.SessionStatus {
	switch status {
	case inventoryv1.AgentStatus_AGENT_STATUS_STARTING,
		inventoryv1.AgentStatus_AGENT_STATUS_RUNNING:
		return rtav1.SessionStatus_SESSION_STATUS_RUNNING
	case inventoryv1.AgentStatus_AGENT_STATUS_DONE,
		inventoryv1.AgentStatus_AGENT_STATUS_STOPPING:
		return rtav1.SessionStatus_SESSION_STATUS_DOWN
	case inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR,
		inventoryv1.AgentStatus_AGENT_STATUS_WAITING:
		return rtav1.SessionStatus_SESSION_STATUS_ERROR
	default:
		return rtav1.SessionStatus_SESSION_STATUS_UNSPECIFIED
	}
}

func (s *Service) convertAgentToSession(agent *models.Agent, service *models.Service) *rtav1.Session {
	var status rtav1.SessionStatus
	if agent.PMMAgentID == nil || !s.registry.IsConnected(*agent.PMMAgentID) {
		status = rtav1.SessionStatus_SESSION_STATUS_UNSPECIFIED
	} else {
		status = convertAgentStatusToSessionStatus(inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]))
	}

	return &rtav1.Session{
		ServiceId:   service.ServiceID,
		ServiceName: service.ServiceName,
		ClusterName: service.Cluster,
		StartTime:   timestamppb.New(agent.CreatedAt),
		Status:      status,
	}
}

func getRTAAgentTypeForServiceType(serviceType models.ServiceType) (models.AgentType, error) {
	switch serviceType {
	case models.MongoDBServiceType:
		return models.RTAMongoDBAgentType, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "Service type %s does not support Real-Time Analytics", serviceType)
	}
}
