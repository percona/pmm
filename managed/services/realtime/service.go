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

// Package realtime provides real-time query analytics service.
package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"gopkg.in/reform.v1"

	realtimev1 "github.com/percona/pmm/api/realtime/v1"
	"github.com/percona/pmm/managed/models"
)

// LBACHeaderName is an http header name used for LBAC (same as qan-api2).
const LBACHeaderName = "X-Proxy-Filter"

// Service provides real-time analytics functionality.
type Service struct {
	l  *logrus.Entry
	db *reform.DB

	// Dependencies
	agentsRegistry    agentsRegistry
	connectionChecker connectionChecker
	stateUpdater      stateUpdater

	// Data management
	mu          sync.RWMutex
	dataBuffer  map[string][]*realtimev1.RealTimeQueryData // serviceID -> current queries
	historyData map[string][]*realtimev1.RealTimeQueryData // serviceID -> historical data (2-minute buffer)

	// Configuration
	bufferDuration time.Duration
}

// NewService creates a new real-time analytics service.
func NewService(
	db *reform.DB,
	agentsRegistry agentsRegistry,
	connectionChecker connectionChecker,
	stateUpdater stateUpdater,
) *Service {
	return &Service{
		l:                 logrus.WithField("component", "realtime-analytics"),
		db:                db,
		agentsRegistry:    agentsRegistry,
		connectionChecker: connectionChecker,
		stateUpdater:      stateUpdater,
		dataBuffer:        make(map[string][]*realtimev1.RealTimeQueryData),
		historyData:       make(map[string][]*realtimev1.RealTimeQueryData),
		bufferDuration:    2 * time.Minute, // 2-minute sliding window
	}
}

// SendRealTimeData handles incoming real-time data from agents.
func (s *Service) SendRealTimeData(ctx context.Context, req *realtimev1.RealTimeAnalyticsRequest) (*realtimev1.RealTimeAnalyticsResponse, error) {
	// Process and store the data
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Group queries by service ID (each query now has its own service metadata)
	serviceGroups := make(map[string][]*realtimev1.RealTimeQueryData)
	for _, data := range req.Queries {
		serviceID := data.ServiceId
		if serviceID == "" {
			s.l.Warn("Received query without service ID")
			continue
		}
		serviceGroups[serviceID] = append(serviceGroups[serviceID], data)
	}

	// Store data for each service
	for serviceID, queries := range serviceGroups {
		// Add to current data buffer
		if s.dataBuffer[serviceID] == nil {
			s.dataBuffer[serviceID] = make([]*realtimev1.RealTimeQueryData, 0)
		}
		s.dataBuffer[serviceID] = queries

		// Add to history buffer
		if s.historyData[serviceID] == nil {
			s.historyData[serviceID] = make([]*realtimev1.RealTimeQueryData, 0)
		}
		s.historyData[serviceID] = append(s.historyData[serviceID], queries...)

		// Clean old data from history (keep only data within buffer duration)
		cutoff := now.Add(-s.bufferDuration)
		filtered := make([]*realtimev1.RealTimeQueryData, 0)

		for _, historical := range s.historyData[serviceID] {
			if historical.Timestamp != nil && historical.Timestamp.AsTime().After(cutoff) {
				filtered = append(filtered, historical)
			}
		}
		s.historyData[serviceID] = filtered
	}

	// Enhanced logging with opid information
	totalQueries := len(req.Queries)
	opIds := make([]int64, 0, totalQueries)
	for _, query := range req.Queries {
		if query.Mongodb != nil && query.Mongodb.Opid != 0 {
			opIds = append(opIds, query.Mongodb.Opid)
		}
	}

	s.l.Debugf("Processed %d real-time data points across %d services (opids: %v)",
		totalQueries, len(serviceGroups), opIds)

	return &realtimev1.RealTimeAnalyticsResponse{}, nil
}

// GetRealTimeData retrieves current real-time data for the UI.
func (s *Service) GetRealTimeData(ctx context.Context, req *realtimev1.RealTimeDataRequest) (*realtimev1.RealTimeDataResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []*realtimev1.RealTimeQueryData
	serviceIDs := req.ServiceIds

	// If no specific services requested, get all accessible services
	if len(serviceIDs) == 0 {
		for serviceID := range s.dataBuffer {
			serviceIDs = append(serviceIDs, serviceID)
		}
	}

	// Get LBAC filters from context (same logic as qan-api2)
	lbacSelectors, err := s.headersToLBACFilters(ctx)
	if err != nil {
		s.l.Errorf("Failed to get LBAC filters: %v", err)
		return &realtimev1.RealTimeDataResponse{}, nil
	}

	// Process each service
	for _, serviceID := range serviceIDs {
		// Check if user has access to this service based on LBAC filters
		if !s.hasServiceAccess(serviceID, lbacSelectors) {
			continue
		}

		if req.IncludeHistory {
			// If historical data is requested, return the history buffer
			if historical, exists := s.historyData[serviceID]; exists {
				data = append(data, historical...)
			}
		} else {
			// By default, only return current data (most recent batch of running queries)
			if currentData, exists := s.dataBuffer[serviceID]; exists {
				data = append(data, currentData...)
			}
		}
	}

	// Apply limit
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if len(data) > limit {
		data = data[:limit]
	}

	return &realtimev1.RealTimeDataResponse{
		Queries: data,
	}, nil
}

// EnableRealTimeAnalytics enables real-time analytics for a service.
func (s *Service) EnableRealTimeAnalytics(ctx context.Context, req *realtimev1.EnableRealTimeAnalyticsRequest) (*realtimev1.ConfigResponse, error) {
	var pmmAgentID string

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		// Get the service and validate service type - only MongoDB is supported in MVP
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return errors.Wrap(err, "failed to find service")
		}

		if service.ServiceType != models.MongoDBServiceType {
			return errors.Errorf("real-time analytics is only supported for MongoDB services, got %s", service.ServiceType)
		}

		// Find the PMM agent for this service to trigger state update
		agents, err := models.FindAgents(tx.Querier, models.AgentFilters{
			ServiceID: req.ServiceId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to find agents for service")
		}

		var rtaAgent *models.Agent
		// Find the MongoDB RTA agent for this service
		for _, agent := range agents {
			if agent.AgentType == models.MongoDBRealtimeAnalyticsAgentType {
				rtaAgent = agent
				if agent.PMMAgentID != nil {
					pmmAgentID = *agent.PMMAgentID
				}
				break
			}
		}

		if rtaAgent == nil {
			return errors.New("no MongoDB real-time analytics agent found for this service")
		}

		if pmmAgentID == "" {
			return errors.New("MongoDB real-time analytics agent has no PMM agent ID")
		}

		// Update the RTA agent configuration
		rtaOptions := models.RealTimeAnalyticsOptions{
			CollectionIntervalSeconds: uint32(req.Config.CollectionIntervalSeconds),
			DisableExamples:           req.Config.DisableExamples,
		}

		// Set default values if not provided
		if rtaOptions.CollectionIntervalSeconds <= 0 {
			rtaOptions.CollectionIntervalSeconds = 1
		}

		rtaAgent.RealTimeAnalyticsOptions = rtaOptions
		rtaAgent.Disabled = false // Enable the agent

		return tx.Update(rtaAgent)
	})

	if errTx != nil {
		return nil, errors.Wrap(errTx, "failed to enable real-time analytics")
	}

	// Trigger agent configuration update
	if pmmAgentID != "" {
		s.requestAgentConfigUpdate(ctx, pmmAgentID)
	}

	s.l.Infof("Enabled real-time analytics for service %s", req.ServiceId)

	return &realtimev1.ConfigResponse{
		Success: true,
		Message: "enabled",
	}, nil
}

// DisableRealTimeAnalytics disables real-time analytics for a service.
func (s *Service) DisableRealTimeAnalytics(ctx context.Context, req *realtimev1.DisableRealTimeAnalyticsRequest) (*realtimev1.ConfigResponse, error) {
	var pmmAgentID string

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		// Find the PMM agent for this service to trigger state update
		agents, err := models.FindAgents(tx.Querier, models.AgentFilters{
			ServiceID: req.ServiceId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to find agents for service")
		}

		var rtaAgent *models.Agent
		// Find the MongoDB RTA agent for this service
		for _, agent := range agents {
			if agent.AgentType == models.MongoDBRealtimeAnalyticsAgentType {
				rtaAgent = agent
				if agent.PMMAgentID != nil {
					pmmAgentID = *agent.PMMAgentID
				}
				break
			}
		}

		if rtaAgent == nil {
			return errors.New("no MongoDB real-time analytics agent found for this service")
		}

		if pmmAgentID == "" {
			return errors.New("MongoDB real-time analytics agent has no PMM agent ID")
		}

		// Disable the RTA agent
		rtaAgent.Disabled = true

		return tx.Update(rtaAgent)
	})

	if errTx != nil {
		return nil, errors.Wrap(errTx, "failed to disable real-time analytics")
	}

	// Trigger agent configuration update
	if pmmAgentID != "" {
		s.requestAgentConfigUpdate(ctx, pmmAgentID)
	}

	// Clean up in-memory data for this service
	s.mu.Lock()
	delete(s.dataBuffer, req.ServiceId)
	delete(s.historyData, req.ServiceId)
	s.mu.Unlock()

	s.l.Infof("Disabled real-time analytics for service %s", req.ServiceId)

	return &realtimev1.ConfigResponse{
		Success: true,
		Message: "disabled",
	}, nil
}

// GetEnabledServices returns a list of services with real-time analytics enabled.
func (s *Service) GetEnabledServices(ctx context.Context) ([]*models.Service, error) {
	var services []*models.Service

	err := s.db.InTransaction(func(tx *reform.TX) error {
		// Find all RTA agents that are not disabled
		agentType := models.MongoDBRealtimeAnalyticsAgentType
		agents, err := models.FindAgents(tx.Querier, models.AgentFilters{
			AgentType: &agentType,
		})
		if err != nil {
			return errors.Wrap(err, "failed to find RTA agents")
		}

		// Get unique service IDs from enabled RTA agents
		serviceIDMap := make(map[string]bool)
		for _, agent := range agents {
			if !agent.Disabled && agent.ServiceID != nil {
				serviceIDMap[*agent.ServiceID] = true
			}
		}

		// Get the services
		for serviceID := range serviceIDMap {
			service, err := models.FindServiceByID(tx.Querier, serviceID)
			if err != nil {
				s.l.Warnf("Failed to find service %s: %v", serviceID, err)
				continue
			}
			services = append(services, service)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get enabled services")
	}

	return services, nil
}

// CleanupOldData periodically cleans up old data from the buffers.
func (s *Service) CleanupOldData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.bufferDuration)

	for serviceID, historical := range s.historyData {
		filtered := make([]*realtimev1.RealTimeQueryData, 0)

		for _, data := range historical {
			if data.Timestamp != nil && data.Timestamp.AsTime().After(cutoff) {
				filtered = append(filtered, data)
			}
		}

		s.historyData[serviceID] = filtered

		// If no data remains, remove the service from buffers
		if len(filtered) == 0 {
			delete(s.historyData, serviceID)
			delete(s.dataBuffer, serviceID)
		}
	}
}

// StartCleanupRoutine starts a background routine to clean up old data.
func (s *Service) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Cleanup every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.CleanupOldData()
		}
	}
}

// headersToLBACFilters extracts filters from the context (same logic as qan-api2).
func (s *Service) headersToLBACFilters(ctx context.Context) ([]string, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// No metadata in context means no LBAC filtering (full access)
		return []string{}, nil
	}

	filters := headers.Get(strings.ToLower(LBACHeaderName))
	if len(filters) == 0 {
		// No LBAC filters means full access
		return []string{}, nil
	}

	return s.parseFilters(filters)
}

// parseFilters decodes and unmarshals the filters (same logic as qan-api2).
func (s *Service) parseFilters(filters []string) ([]string, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(filters[0])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode filters %s", filters[0])
	}

	var parsed []string
	if err := json.Unmarshal(decoded, &parsed); err != nil {
		return nil, errors.Wrap(err, "failed to parse JSON")
	}

	return parsed, nil
}

// hasServiceAccess checks if a user has access to a service based on LBAC selectors.
// This follows the same logic as qan-api2 Prometheus selector matching.
func (s *Service) hasServiceAccess(serviceID string, selectors []string) bool {
	// Empty selectors means full access (same as qan-api2 logic)
	if len(selectors) == 0 {
		return true
	}

	// Get service to check its labels
	service, err := models.FindServiceByID(s.db.Querier, serviceID)
	if err != nil {
		s.l.Warnf("Failed to find service %s: %v", serviceID, err)
		return false
	}

	// Get service labels for matching
	serviceLabels, err := service.UnifiedLabels()
	if err != nil {
		s.l.Warnf("Failed to get labels for service %s: %v", serviceID, err)
		return false
	}

	// Check if any selector matches (OR logic, same as qan-api2)
	for _, selector := range selectors {
		if s.matchesSelector(serviceLabels, selector) {
			return true
		}
	}

	return false
}

// matchesSelector checks if service labels match a Prometheus selector.
// This implements the same logic as qan-api2 using Prometheus parser.
func (s *Service) matchesSelector(serviceLabels map[string]string, selector string) bool {
	if selector == "" {
		return true
	}

	// Parse Prometheus selector using same logic as qan-api2
	matchers, err := parser.ParseMetricSelector(selector)
	if err != nil {
		s.l.Errorf("Failed to parse metric selector: %v", err)
		return false // Conservative: deny access on parse error
	}

	// Check if all matchers match (AND logic within a selector)
	for _, matcher := range matchers {
		if !s.matchesMatcher(serviceLabels, matcher) {
			return false
		}
	}

	return true
}

// requestAgentConfigUpdate triggers a configuration update for the PMM agent.
// This will cause the agent to receive new real-time analytics configuration.
func (s *Service) requestAgentConfigUpdate(ctx context.Context, pmmAgentID string) {
	s.l.Debugf("Requesting agent config update for PMM agent %s", pmmAgentID)
	s.stateUpdater.RequestStateUpdate(ctx, pmmAgentID)
}

// matchesMatcher checks if service labels match a single Prometheus matcher.
func (s *Service) matchesMatcher(serviceLabels map[string]string, matcher *labels.Matcher) bool {
	serviceValue, exists := serviceLabels[matcher.Name]

	switch matcher.Type {
	case labels.MatchEqual:
		return exists && serviceValue == matcher.Value
	case labels.MatchNotEqual:
		return !exists || serviceValue != matcher.Value
	case labels.MatchRegexp:
		return exists && matcher.Matches(serviceValue)
	case labels.MatchNotRegexp:
		return !exists || !matcher.Matches(serviceValue)
	default:
		s.l.Debugf("Unknown matcher type %v, allowing access", matcher.Type)
		return true // Conservative: allow access for unknown matcher types
	}
}
