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

package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
)

var (
	// ErrForwardingDisabled is returned when forwarding is disabled by feature flag.
	ErrForwardingDisabled = status.Error(codes.FailedPrecondition, "HA forwarding is disabled")
	// ErrAgentNotFound is returned when agent is not found in cluster.
	ErrAgentNotFound = status.Error(codes.NotFound, "agent not found in cluster")
	// ErrForwardingLoop is returned when circular forwarding is detected.
	ErrForwardingLoop = status.Error(codes.FailedPrecondition, "circular forwarding detected")
	// ErrAgentMayHaveReconnected is returned to signal caller to retry locally.
	ErrAgentMayHaveReconnected = status.Error(codes.Unavailable, "forwarding failed, agent may have reconnected locally")
)

// Forwarder coordinates request forwarding between PMM servers in HA mode.
type Forwarder struct {
	haService       *Service
	client          *ForwardingClient
	settingsService settingsService
	nodeID          string

	// Graceful shutdown
	shuttingDown atomic.Bool

	// Audit logging
	auditLogger *logrus.Logger

	// Metrics
	requestsForwardedTotal  *prometheus.CounterVec
	requestsForwardedErrors *prometheus.CounterVec
	forwardingDuration      *prometheus.HistogramVec
	gossipEventsTotal       *prometheus.CounterVec
	locationCacheSize       prometheus.Gauge
	forwardingEnabledGauge  prometheus.Gauge

	l *logrus.Entry
}

// settingsService defines interface for accessing PMM settings.
type settingsService interface {
	GetSettings(ctx context.Context) (*models.Settings, error)
}

// NewForwarder creates a new Forwarder instance.
func NewForwarder(
	haService *Service,
	client *ForwardingClient,
	settingsService settingsService,
	nodeID string,
) *Forwarder {
	f := &Forwarder{
		haService:       haService,
		client:          client,
		settingsService: settingsService,
		nodeID:          nodeID,
		l:               logrus.WithField("component", "agent-forwarder"),
	}

	// Initialize audit logger
	f.initAuditLogger()

	// Initialize metrics
	f.initMetrics()

	return f
}

// initAuditLogger initializes the audit logger for forwarding requests.
func (f *Forwarder) initAuditLogger() {
	f.auditLogger = logrus.New()
	f.auditLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Open audit log file
	auditFile, err := os.OpenFile("/srv/logs/pmm-forwarding-audit.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		f.l.Warnf("Failed to open audit log file: %v, using stdout", err)
		f.auditLogger.SetOutput(os.Stdout)
	} else {
		f.auditLogger.SetOutput(auditFile)
	}
}

// initMetrics initializes Prometheus metrics for forwarding.
func (f *Forwarder) initMetrics() {
	f.requestsForwardedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pmm_managed",
			Subsystem: "agents",
			Name:      "requests_forwarded_total",
			Help:      "Total number of agent requests forwarded to other servers",
		},
		[]string{"target_server", "success"},
	)

	f.requestsForwardedErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pmm_managed",
			Subsystem: "agents",
			Name:      "requests_forwarded_errors_total",
			Help:      "Total number of forwarding errors by error type",
		},
		[]string{"error_type"},
	)

	f.forwardingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pmm_managed",
			Subsystem: "agents",
			Name:      "forwarding_duration_seconds",
			Help:      "Time taken to forward requests to other servers",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0},
		},
		[]string{"target_server"},
	)

	f.gossipEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pmm_managed",
			Subsystem: "agents",
			Name:      "gossip_events_total",
			Help:      "Total number of gossip events processed",
		},
		[]string{"event_type"},
	)

	f.locationCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "pmm_managed",
			Subsystem: "agents",
			Name:      "location_cache_size",
			Help:      "Current number of agent locations tracked in cache",
		},
	)

	f.forwardingEnabledGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "pmm_managed",
			Subsystem: "agents",
			Name:      "forwarding_enabled",
			Help:      "Whether HA forwarding is currently enabled (1 = enabled, 0 = disabled)",
		},
	)

	// Register metrics
	prometheus.MustRegister(
		f.requestsForwardedTotal,
		f.requestsForwardedErrors,
		f.forwardingDuration,
		f.gossipEventsTotal,
		f.locationCacheSize,
		f.forwardingEnabledGauge,
	)
}

// IsEnabled checks if HA forwarding is enabled via feature flag.
func (f *Forwarder) IsEnabled(ctx context.Context) bool {
	if f.shuttingDown.Load() {
		return false
	}

	settings, err := f.settingsService.GetSettings(ctx)
	if err != nil {
		f.l.Warnf("Failed to get settings: %v, assuming forwarding disabled", err)
		f.forwardingEnabledGauge.Set(0)
		return false
	}

	enabled := settings.IsHAForwardingEnabled()
	if enabled {
		f.forwardingEnabledGauge.Set(1)
	} else {
		f.forwardingEnabledGauge.Set(0)
	}

	return enabled
}

// ForwardServerMessage forwards a ServerMessage to the PMM server where the agent is connected.
// Returns ErrAgentMayHaveReconnected if all forwarding attempts fail, signaling caller to retry locally.
func (f *Forwarder) ForwardServerMessage(
	ctx context.Context,
	pmmAgentID string,
	message *agentv1.ServerMessage,
) (*agentv1.AgentMessage, error) {
	// Check feature flag
	if !f.IsEnabled(ctx) {
		return nil, ErrForwardingDisabled
	}

	// Generate request ID for tracing
	requestID := uuid.New().String()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		f.l.WithFields(logrus.Fields{
			"agent_id":   pmmAgentID,
			"request_id": requestID,
			"duration":   duration,
		}).Debug("Forwarding request completed")
	}()

	// Step 1: Query gossip for agent location
	serverID := f.haService.GetAgentLocation(pmmAgentID)
	if serverID == "" {
		f.l.Warnf("Agent %s location unknown in gossip", pmmAgentID)
		f.requestsForwardedErrors.WithLabelValues("agent_not_found").Inc()
		f.logAuditEvent(requestID, pmmAgentID, "", "lookup_failed", "agent not found", 0)
		return nil, ErrAgentNotFound
	}

	f.l.Debugf("Agent %s is on server %s (from gossip)", pmmAgentID, serverID)

	// Step 2: Try target server from gossip
	resp, err := f.forwardMessageToServer(ctx, requestID, pmmAgentID, serverID, message)
	if err == nil {
		return resp, nil
	}

	f.l.Warnf("Failed to forward to gossip target %s: %v, trying all servers", serverID, err)

	// Step 3: Try ALL other servers in cluster (gossip may be stale)
	peers := f.haService.GetPeerServers()
	for _, peer := range peers {
		// Skip the server we already tried
		if peer.ID == serverID {
			continue
		}

		f.l.Debugf("Trying peer server %s (%s)", peer.ID, peer.Address)
		resp, err := f.forwardMessageToServer(ctx, requestID, pmmAgentID, peer.ID, message)
		if err == nil {
			f.l.Infof("Successfully forwarded to %s after gossip target failed", peer.ID)
			return resp, nil
		}

		f.l.Debugf("Failed to forward to %s: %v", peer.ID, err)
	}

	// All attempts failed - signal caller to retry locally (agent may have reconnected)
	f.requestsForwardedErrors.WithLabelValues("all_servers_failed").Inc()
	f.logAuditEvent(requestID, pmmAgentID, "", "failed", "all servers failed", 0)
	f.l.Warnf("Failed to forward request for agent %s to any server, caller should retry locally", pmmAgentID)

	return nil, ErrAgentMayHaveReconnected
}

// forwardMessageToServer forwards a ServerMessage to a specific server.
func (f *Forwarder) forwardMessageToServer(
	ctx context.Context,
	requestID string,
	pmmAgentID string,
	serverID string,
	message *agentv1.ServerMessage,
) (*agentv1.AgentMessage, error) {
	startTime := time.Now()

	serverAddr := f.haService.GetServerAddress(serverID)
	if serverAddr == "" {
		f.l.Warnf("Server %s address not found", serverID)
		return nil, fmt.Errorf("server %s address not found", serverID)
	}

	f.l.WithFields(logrus.Fields{
		"request_id": requestID,
		"agent_id":   pmmAgentID,
		"target":     serverAddr,
	}).Debug("Forwarding request to remote server")

	// Forward via gRPC client
	resp, err := f.client.ForwardServerMessage(ctx, serverAddr, pmmAgentID, message, requestID, f.nodeID)

	duration := time.Since(startTime)
	f.forwardingDuration.WithLabelValues(serverID).Observe(duration.Seconds())

	if err != nil {
		f.requestsForwardedTotal.WithLabelValues(serverID, "false").Inc()
		f.requestsForwardedErrors.WithLabelValues("grpc_error").Inc()
		f.logAuditEvent(requestID, pmmAgentID, serverID, "failed", err.Error(), duration.Milliseconds())
		return nil, err
	}

	f.requestsForwardedTotal.WithLabelValues(serverID, "true").Inc()
	f.logAuditEvent(requestID, pmmAgentID, serverID, "success", "", duration.Milliseconds())

	return resp, nil
}

// logAuditEvent logs a forwarding event to the audit log.
func (f *Forwarder) logAuditEvent(
	requestID string,
	agentID string,
	targetServer string,
	result string,
	errorMsg string,
	durationMs int64,
) {
	auditEntry := map[string]interface{}{
		"request_id":    requestID,
		"agent_id":      agentID,
		"source_server": f.nodeID,
		"target_server": targetServer,
		"result":        result,
		"duration_ms":   durationMs,
	}

	if errorMsg != "" {
		auditEntry["error"] = errorMsg
	}

	auditJSON, err := json.Marshal(auditEntry)
	if err != nil {
		f.l.Errorf("Failed to marshal audit entry: %v", err)
		return
	}

	f.auditLogger.Info(string(auditJSON))
}

// Shutdown sets the shutdown flag to reject new forwarding requests.
func (f *Forwarder) Shutdown() {
	f.shuttingDown.Store(true)
	f.l.Info("Forwarder shutdown initiated, rejecting new requests")

	// Close client connections
	if f.client != nil {
		f.client.Close()
	}
}
