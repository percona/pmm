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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/durationpb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/api/serverpb"
)

const (
	// protocolVersion is the current version of the forwarding protocol.
	protocolVersion = 1
	// connectionTimeout is the timeout for establishing gRPC connections.
	connectionTimeout = 5 * time.Second
	// certPath is the base path for cluster certificates.
	certBasePath = "/srv/pmm-certs/cluster"
)

// ForwardingClient handles gRPC connections to other PMM servers for forwarding.
type ForwardingClient struct {
	// Connection pool
	mu          sync.RWMutex
	connections map[string]*grpc.ClientConn

	// TLS configuration
	tlsConfig *tls.Config

	l *logrus.Entry
}

// NewForwardingClient creates a new forwarding client with mTLS support.
func NewForwardingClient() (*ForwardingClient, error) {
	client := &ForwardingClient{
		connections: make(map[string]*grpc.ClientConn),
		l:           logrus.WithField("component", "forwarding-client"),
	}

	// Load TLS configuration
	if err := client.loadTLSConfig(); err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	return client, nil
}

// loadTLSConfig loads cluster certificates for mTLS authentication.
func (c *ForwardingClient) loadTLSConfig() error {
	// Check if cluster certificates exist
	certFile := certBasePath + "/tls.crt"
	keyFile := certBasePath + "/tls.key"
	caFile := certBasePath + "/ca.crt"

	// If certificates don't exist, mTLS is not configured (non-HA mode)
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		c.l.Debug("Cluster certificates not found, mTLS disabled")
		return nil
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Create CA pool
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to add CA certificate to pool")
	}

	// Create TLS configuration
	c.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
		// ServerName can be left empty as we're using IP addresses
		// and the certificate has IP SANs
	}

	c.l.Info("mTLS configuration loaded successfully")
	return nil
}

// getConnection returns or creates a gRPC connection to the specified server.
func (c *ForwardingClient) getConnection(ctx context.Context, serverAddr string) (*grpc.ClientConn, error) {
	// Check if we already have a connection
	c.mu.RLock()
	conn, exists := c.connections[serverAddr]
	c.mu.RUnlock()

	if exists && conn.GetState().String() != "SHUTDOWN" {
		return conn, nil
	}

	// Create new connection
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	conn, exists = c.connections[serverAddr]
	if exists && conn.GetState().String() != "SHUTDOWN" {
		return conn, nil
	}

	c.l.Debugf("Creating new gRPC connection to %s", serverAddr)

	// Build dial options
	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")), // Enable compression
	}

	// Add TLS credentials if configured
	if c.tlsConfig != nil {
		creds := credentials.NewTLS(c.tlsConfig)
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
		c.l.Debugf("Using mTLS for connection to %s", serverAddr)
	} else {
		// Fallback to insecure for local development
		dialOpts = append(dialOpts, grpc.WithInsecure())
		c.l.Warnf("Using insecure connection to %s (mTLS not configured)", serverAddr)
	}

	// Create connection with timeout
	connCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(connCtx, serverAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", serverAddr, err)
	}

	// Store connection in pool
	c.connections[serverAddr] = conn
	c.l.Infof("Connected to %s", serverAddr)

	return conn, nil
}

// ForwardServerMessage forwards a ServerMessage to another PMM server where the agent is connected.
func (c *ForwardingClient) ForwardServerMessage(
	ctx context.Context,
	serverAddr string,
	agentID string,
	message *agentv1.ServerMessage,
	requestID string,
	sourceServerID string,
) (*agentv1.AgentMessage, error) {
	// Get or create connection
	conn, err := c.getConnection(ctx, serverAddr)
	if err != nil {
		return nil, err
	}

	// Create gRPC client
	client := serverpb.NewAgentForwardingClient(conn)

	// Get timeout from context or use default
	var timeout *durationpb.Duration
	if deadline, ok := ctx.Deadline(); ok {
		timeout = durationpb.New(time.Until(deadline))
	} else {
		timeout = durationpb.New(30 * time.Second)
	}

	// Create forwarding request
	forwardReq := &serverpb.ForwardRequestRequest{
		AgentId:         agentID,
		ForwardedBy:     sourceServerID,
		Message:         message,
		ProtocolVersion: protocolVersion,
		RequestId:       requestID,
		Timeout:         timeout,
	}

	// Forward request
	forwardResp, err := client.ForwardRequest(ctx, forwardReq)
	if err != nil {
		return nil, fmt.Errorf("forwarding failed: %w", err)
	}

	if forwardResp.Error != "" {
		return nil, fmt.Errorf("remote server error: %s", forwardResp.Error)
	}

	return forwardResp.Message, nil
}

// ForwardActionRequest forwards an action request to another PMM server.
// This is a convenience wrapper around ForwardServerMessage.
func (c *ForwardingClient) ForwardActionRequest(
	ctx context.Context,
	serverAddr string,
	agentID string,
	request *agentv1.StartActionRequest,
	requestID string,
	sourceServerID string,
) (*agentv1.StartActionResponse, error) {
	// Wrap in ServerMessage
	serverMsg := &agentv1.ServerMessage{
		Id: 0, // Will be set by the receiving server
		Payload: &agentv1.ServerMessage_StartAction{
			StartAction: request,
		},
	}

	// Forward the message
	agentMsg, err := c.ForwardServerMessage(ctx, serverAddr, agentID, serverMsg, requestID, sourceServerID)
	if err != nil {
		return nil, err
	}

	// Check for errors in the agent response
	if agentMsg.Status != nil && agentMsg.Status.Code != 0 {
		return nil, fmt.Errorf("agent returned error: %s", agentMsg.Status.Message)
	}

	// StartActionResponse is empty, but we return it for protocol consistency
	return &agentv1.StartActionResponse{}, nil
}

// Close closes all connections in the pool.
func (c *ForwardingClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for addr, conn := range c.connections {
		if err := conn.Close(); err != nil {
			c.l.Errorf("Failed to close connection to %s: %v", addr, err)
		} else {
			c.l.Debugf("Closed connection to %s", addr)
		}
	}

	c.connections = make(map[string]*grpc.ClientConn)
}

// removeConnection removes a connection from the pool (called when connection fails).
func (c *ForwardingClient) removeConnection(serverAddr string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conn, exists := c.connections[serverAddr]; exists {
		_ = conn.Close()
		delete(c.connections, serverAddr)
		c.l.Debugf("Removed connection to %s from pool", serverAddr)
	}
}
