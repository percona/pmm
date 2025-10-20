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

package grpc

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// maxProtocolVersion is the maximum protocol version we support.
	maxProtocolVersion = 1
	// maxForwardingHops prevents circular forwarding (we allow 0 hops = direct only).
	maxForwardingHops = 0
)

// AgentRegistry defines the interface for accessing agent connections.
type AgentRegistry interface {
	Get(agentID string) (AgentConnection, error)
}

// AgentConnection represents a connection to an agent.
type AgentConnection interface {
	SendRequest(request interface{}) (interface{}, error)
}

// ForwardingServer implements the AgentForwarding gRPC service.
type ForwardingServer struct {
	registry AgentRegistry
	nodeID   string
	l        *logrus.Entry

	// Uncomment when protobuf is generated:
	// serverpb.UnimplementedAgentForwardingServer
}

// NewForwardingServer creates a new forwarding server instance.
func NewForwardingServer(registry AgentRegistry, nodeID string) *ForwardingServer {
	return &ForwardingServer{
		registry: registry,
		nodeID:   nodeID,
		l:        logrus.WithField("component", "forwarding-server"),
	}
}

// ForwardAgentRequest handles forwarding of agent requests (placeholder until protobuf is generated).
func (s *ForwardingServer) ForwardAgentRequest(
	ctx context.Context,
	req interface{}, // Will be *serverpb.ForwardAgentRequestRequest once protobuf is generated
) (interface{}, error) {
	// This is a placeholder implementation until protobuf is generated
	// The actual implementation will be:
	/*
		func (s *ForwardingServer) ForwardAgentRequest(
			ctx context.Context,
			req *serverpb.ForwardAgentRequestRequest,
		) (*serverpb.ForwardAgentRequestResponse, error) {
			s.l.WithFields(logrus.Fields{
				"agent_id":    req.AgentId,
				"forwarded_by": req.ForwardedBy,
				"request_type": req.RequestType,
				"request_id":   req.RequestId,
			}).Debug("Received forwarding request")

			// Validate protocol version
			if req.ProtocolVersion > maxProtocolVersion {
				return &serverpb.ForwardAgentRequestResponse{
					Error: fmt.Sprintf("unsupported protocol version: %d (max: %d)",
						req.ProtocolVersion, maxProtocolVersion),
				}, nil
			}

			// Prevent circular forwarding (max 0 hops = must be original request)
			if req.ForwardedBy != "" {
				s.l.Warnf("Circular forwarding detected: request already forwarded by %s", req.ForwardedBy)
				return &serverpb.ForwardAgentRequestResponse{
					Error: "circular forwarding detected (max 0 hops allowed)",
				}, nil
			}

			// Look up agent in local registry
			agent, err := s.registry.Get(req.AgentId)
			if err != nil {
				s.l.Debugf("Agent %s not found locally: %v", req.AgentId, err)
				return &serverpb.ForwardAgentRequestResponse{
					Error: fmt.Sprintf("agent not connected to this server: %s", req.AgentId),
				}, nil
			}

			// Unpack the original request
			var startActionReq agentv1.StartActionRequest
			if err := req.RequestPayload.UnmarshalTo(&startActionReq); err != nil {
				return &serverpb.ForwardAgentRequestResponse{
					Error: fmt.Sprintf("failed to unpack request payload: %v", err),
				}, nil
			}

			// Apply timeout from request if provided
			if req.Timeout != nil {
				timeout := req.Timeout.AsDuration()
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			// Send request to local agent
			resp, err := agent.SendRequest(&startActionReq)
			if err != nil {
				s.l.Errorf("Failed to send request to local agent %s: %v", req.AgentId, err)
				return &serverpb.ForwardAgentRequestResponse{
					Error: fmt.Sprintf("agent request failed: %v", err),
				}, nil
			}

			// Pack response
			respPayload, err := anypb.New(resp)
			if err != nil {
				return &serverpb.ForwardAgentRequestResponse{
					Error: fmt.Sprintf("failed to pack response: %v", err),
				}, nil
			}

			s.l.WithFields(logrus.Fields{
				"agent_id":   req.AgentId,
				"request_id": req.RequestId,
			}).Debug("Successfully processed forwarded request")

			return &serverpb.ForwardAgentRequestResponse{
				ResponsePayload: respPayload,
				ProcessedBy:     s.nodeID,
			}, nil
		}
	*/

	s.l.Warn("ForwardAgentRequest called but protobuf not yet generated")
	return nil, status.Error(codes.Unimplemented, "forwarding not yet implemented (protobuf generation pending)")
}

// ForwardStateUpdate handles forwarding of state updates (placeholder).
func (s *ForwardingServer) ForwardStateUpdate(
	ctx context.Context,
	req interface{}, // Will be *serverpb.ForwardStateUpdateRequest
) (interface{}, error) {
	s.l.Debug("ForwardStateUpdate called")
	// TODO: Implement once protobuf is generated
	return nil, status.Error(codes.Unimplemented, "not yet implemented")
}

// ForwardJobRequest handles forwarding of job requests (placeholder).
func (s *ForwardingServer) ForwardJobRequest(
	ctx context.Context,
	req interface{}, // Will be *serverpb.ForwardJobRequestRequest
) (interface{}, error) {
	s.l.Debug("ForwardJobRequest called")
	// TODO: Implement once protobuf is generated
	return nil, status.Error(codes.Unimplemented, "not yet implemented")
}

// SyncFullAgentState handles full state synchronization requests (placeholder).
func (s *ForwardingServer) SyncFullAgentState(
	ctx context.Context,
	req interface{}, // Will be *serverpb.FullStateSyncRequest
) (interface{}, error) {
	s.l.Debug("SyncFullAgentState called")
	// TODO: Implement with HA service integration once protobuf is generated
	/*
		func (s *ForwardingServer) SyncFullAgentState(
			ctx context.Context,
			req *serverpb.FullStateSyncRequest,
		) (*serverpb.FullStateSyncResponse, error) {
			s.l.Infof("Full state sync requested by %s", req.RequesterServerId)

			// Get full agent location state from HA service
			state := s.haService.GetFullAgentState()

			return &serverpb.FullStateSyncResponse{
				AgentLocations: state.AgentLocations,
				Timestamp:      timestamppb.New(state.Timestamp),
				AgentCount:     int32(len(state.AgentLocations)),
			}, nil
		}
	*/
	return nil, status.Error(codes.Unimplemented, "not yet implemented")
}

// validateRequest performs common request validation.
func (s *ForwardingServer) validateRequest(agentID, requestID string, protocolVersion int32) error {
	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id is required")
	}

	if requestID == "" {
		return status.Error(codes.InvalidArgument, "request_id is required")
	}

	if protocolVersion < 1 || protocolVersion > maxProtocolVersion {
		return status.Errorf(codes.InvalidArgument,
			"unsupported protocol version: %d (supported: 1-%d)",
			protocolVersion, maxProtocolVersion)
	}

	return nil
}
