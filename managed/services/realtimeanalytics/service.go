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
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
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

// ListServices returns a list of Services that support Real-Time Analytics filtered by type (gRPC handler).
func (s *Service) ListServices(ctx context.Context, req *rtav1.ListServicesRequest) (*rtav1.ListServicesResponse, error) {
	var serviceList []*models.Service

	requestedFilterModelServiceType := services.ProtoToModelServiceType(req.GetServiceType())
	if requestedFilterModelServiceType != nil {
		// Request is filtered by service type - validate that the service type
		// is supported for RTA and apply the filter.
		_, err := getRTAAgentTypeForServiceType(*requestedFilterModelServiceType)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				"Service type %s does not support Real-Time Analytics", *requestedFilterModelServiceType)
		}

		// Lookup for services of the requested type.
		serviceList, err = models.FindServices(s.db.Querier, models.ServiceFilters{
			ServiceType: requestedFilterModelServiceType,
		})
		if err != nil {
			return nil, err
		}
	} else {
		// No service type filter specified - return all services that support RTA.
		// For the time being we only support MongoDB, so we can just filter by service type here.
		for _, modelServiceType := range services.ServiceTypes {
			_, err := getRTAAgentTypeForServiceType(modelServiceType)
			if err != nil {
				// Service type is not supported for RTA - skip it.
				continue
			}

			tmpServiceList, err := models.FindServices(s.db.Querier, models.ServiceFilters{
				ServiceType: &modelServiceType,
			})
			if err != nil {
				return nil, err
			}

			serviceList = append(serviceList, tmpServiceList...)
		}
	}

	res := &rtav1.ListServicesResponse{}

	for _, svc := range serviceList {
		select {
		case <-ctx.Done():
			return nil, status.Error(codes.Canceled, "request canceled")
		default:
		}

		// Check that service has pmm-agent with version supporting RTA.
		pmmAgents, err := models.FindPMMAgentsForService(s.db.WithContext(ctx), svc.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to find pmm-agent for service with ID %s: %w", svc.ServiceID, err)
		}

		if len(pmmAgents) == 0 {
			continue // skip services without pmm-agent
		}

		if !isRtaFeatureSupported(*pmmAgents[0].Version) {
			continue // skip services with unsupported pmm-agent version
		}

		// Convert service to API format to be returned in the response.
		apiSvc, svcErr := services.ToAPIService(svc)
		if svcErr != nil {
			return nil, fmt.Errorf("failed to convert service with ID %s to API format: %w", svc.ServiceID, svcErr)
		}

		switch apiSvc := apiSvc.(type) {
		case *inventoryv1.MongoDBService:
			res.Mongodb = append(res.Mongodb, apiSvc)
		// Add other service types once RTA is supported for them
		default:
			return nil, fmt.Errorf("unhandled inventory Service type %T", apiSvc)
		}
	}

	slices.SortStableFunc(res.Mongodb, func(a, b *inventoryv1.MongoDBService) int {
		return strings.Compare(a.ServiceName, b.ServiceName)
	})

	return res, nil
}

// ListSessions returns the list of currently running Real-Time Analytics Sessions (gRPC handler).
func (s *Service) ListSessions(ctx context.Context, req *rtav1.ListSessionsRequest) (*rtav1.ListSessionsResponse, error) {
	response := &rtav1.ListSessionsResponse{
		Sessions: []*rtav1.Session{},
	}

	for _, at := range models.GetRTAAgentTypes() {
		// fetch all RTA agents of this type
		agents, err := models.FindAgents(s.db.WithContext(ctx), models.AgentFilters{
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

			select {
			case <-ctx.Done():
				return nil, status.Error(codes.Canceled, "request canceled")
			default:
			}

			service, err := models.FindServiceByID(s.db.WithContext(ctx), *agent.ServiceID)
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

	slices.SortStableFunc(response.Sessions, func(a, b *rtav1.Session) int {
		return strings.Compare(a.ServiceName, b.ServiceName)
	})

	return response, nil
}

// StartSession starts Real-Time Analytics Session for a specified service (gRPC handler).
func (s *Service) StartSession(ctx context.Context, req *rtav1.StartSessionRequest) (*rtav1.StartSessionResponse, error) {
	var (
		err     error
		session *rtav1.Session
	)
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
		rtaAgentType, err = getRTAAgentTypeForServiceType(service.ServiceType)
		if err != nil {
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
			// Need to update CreatedAt to reflect the new session start time.
			rtaAgent.CreatedAt = time.Now()
			// Encrypt agent's sensitive data before updating it in the database.
			rtaAgent = pointer.To(models.EncryptAgent(*rtaAgent))

			err = tx.Update(rtaAgent)
			if err != nil {
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
				"Service %s of type %s doesn't have agents to retrieve credentials and pmm-agent ID",
				service.ServiceID, service.ServiceType)
		}

		if existingAgent.PMMAgentID == nil {
			return status.Errorf(codes.FailedPrecondition,
				"Existing %s agent for service %s has no pmm-agent ID",
				service.ServiceType, service.ServiceID)
		}

		// Check that a corresponding pmm-agent has version supporting RTA.
		pmmAgent, err := models.FindAgentByID(tx.Querier, *existingAgent.PMMAgentID)
		if err != nil {
			return err
		}

		if !isRtaFeatureSupported(*pmmAgent.Version) {
			return status.Errorf(codes.FailedPrecondition,
				"Service %s has pmm-agent with version not supporting Real-Time Analytics.", service.ServiceID)
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
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		var agentType models.AgentType
		// Check that service type supports RTA
		agentType, err = getRTAAgentTypeForServiceType(service.ServiceType)
		if err != nil {
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
		// Encrypt agent's sensitive data before updating it in the database.
		rtaAgent = pointer.To(models.EncryptAgent(*rtaAgent))

		err = tx.Update(rtaAgent)
		if err != nil {
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
func (s *Service) SearchQueries(ctx context.Context, req *rtav1.SearchQueriesRequest) (*rtav1.SearchQueriesResponse, error) {
	// Validate that all the requested services exist
	for _, serviceID := range req.ServiceIds {
		_, err := models.FindServiceByID(s.db.Querier, serviceID)
		if err != nil {
			return nil, err
		}
	}

	headers := metadata.Pairs("Cache-Control", "no-store")

	err := grpc.SetHeader(ctx, headers)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to set response headers: %v", err)
	}

	resp := &rtav1.SearchQueriesResponse{
		Queries: []*rtav1.QueryData{},
	}

	for _, serviceID := range req.ServiceIds {
		// Queries for a particular service requested
		resp.Queries = append(resp.Queries, s.store.Get(serviceID)...)
	}

	// Sort queries by query_execution_duration in descending order.
	slices.SortStableFunc(resp.Queries, func(a, b *rtav1.QueryData) int {
		var aD, bD int64
		if a.QueryExecutionDuration != nil {
			aD = a.QueryExecutionDuration.AsDuration().Nanoseconds()
		}

		if b.QueryExecutionDuration != nil {
			bD = b.QueryExecutionDuration.AsDuration().Nanoseconds()
		}

		if aD < bD {
			return 1
		} else if aD > bD {
			return -1
		}

		return 0
	})

	// Apply limit if specified to final list of queries after filtering by service and sorting.
	if req.Limit > 0 && int64(len(resp.Queries)) > req.Limit {
		resp.Queries = resp.Queries[:req.Limit]
	}

	return resp, nil
}

// Collect handles incoming streaming RTA query data from agents (gRPC handler).
func (s *Service) Collect(stream grpc.ClientStreamingServer[rtav1.CollectRequest, rtav1.CollectResponse]) error {
	streamCtx := stream.Context()
	l := logger.Get(streamCtx)

	agentMD, err := agentv1.ReceiveAgentConnectMetadata(stream)
	if err != nil {
		l.Warnf("Disconnecting client: authentication failed: %v", err)
		return status.Error(codes.Unauthenticated, "Failed to receive agent metadata")
	}

	// Validate that the pmm-agent exists
	agent, err := models.FindAgentByID(s.db.Querier, agentMD.ID)
	if err != nil {
		l.Warnf("Disconnecting client: agent validation failed: %v", err)
		return status.Error(codes.InvalidArgument, "Invalid Agent ID: "+agentMD.ID)
	}

	if agent.AgentType != models.PMMAgentType {
		return status.Errorf(codes.InvalidArgument, "Agent with ID %s is not a pmm-agent", agentMD.ID)
	}

	for {
		// Check if context is canceled before receiving
		select {
		case <-streamCtx.Done():
			return status.Error(codes.Canceled, "client disconnected")
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// client has closed it's side of stream.
				// just exit the loop and close our side.
				l.Info("Client closed the stream, closing our side.")
				return stream.SendAndClose(&rtav1.CollectResponse{})
			}

			return err // stream error
		}

		if l.Logger.IsLevelEnabled(logrus.DebugLevel) {
			// do not use default compact representation for large/complex messages
			if size := proto.Size(msg); size < 100 { //nolint:mnd
				l.Debugf("Received message (%d bytes): %s.", size, msg)
			} else {
				l.Debugf("Received message (%d bytes):\n%s\n", proto.Size(msg), prototext.Format(msg))
			}
		}

		if len(msg.Queries) == 0 || msg.Queries[0].ServiceId == "" {
			continue
		}

		// Store received queries into the in-memory storage.
		// All queries in the message belong to the same service.
		s.store.Set(msg.Queries[0].ServiceId, msg.Queries)
	}
}

// Helpers.

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
		ServiceId:       service.ServiceID,
		ServiceName:     service.ServiceName,
		ClusterName:     service.Cluster,
		StartTime:       timestamppb.New(agent.CreatedAt),
		CollectInterval: durationpb.New(*agent.RTAOptions.CollectInterval),
		Status:          status,
	}
}

func getRTAAgentTypeForServiceType(serviceType models.ServiceType) (models.AgentType, error) {
	switch serviceType {
	case models.MongoDBServiceType:
		return models.RTAMongoDBAgentType, nil
	default:
		return "", fmt.Errorf("service of type %s does not support Real-Time Analytics", serviceType)
	}
}

// isRtaFeatureSupported checks if the passed pmm-agent's version supporting RTA.
func isRtaFeatureSupported(pmmAgentVersion string) bool {
	versionParsed, versionParseErr := version.Parse(pmmAgentVersion)
	if versionParseErr != nil {
		return false
	}

	return versionParsed.IsFeatureSupported(version.MongoDBRtaAgentSupportVersion)
}

// check interfaces.
var (
	_ rtav1.RealtimeAnalyticsServiceServer = (*Service)(nil)
	_ rtav1.CollectorServiceServer         = (*Service)(nil)
)
