# PMM Server HA Request Forwarding Implementation

## Overview

Enable PMM Server instances in HA mode to forward all agent-related requests (actions, connection checks, state updates, jobs) to the correct PMM Server instance using direct gRPC communication with mTLS authentication and gossip protocol for real-time agent location discovery.

**Key Design Decisions:**

- Agent locations tracked exclusively via gossip (no database storage)
- Direct gRPC on existing port 8443 (service multiplexing)
- Pod IPs from memberlist for addressing
- Pass-through timeout handling
- Requester writes action results
- Buffer + compression for large responses
- Feature flag + protocol versioning for compatibility
- Trust mTLS with audit logging for security
- Full state sync on request for gossip scalability

## Current Architecture

- **HA Infrastructure**: Raft consensus for leader election, Gossip protocol (memberlist) for node discovery
- **Agent Registry**: `managed/services/agents/registry.go` tracks connected agents in-memory on each server
- **Request Flow**: Web UI → pmm-managed (gRPC) → pmm-agent
- **Error**: `registry.get()` returns `codes.FailedPrecondition` when agent not connected locally

## Solution Architecture

### 1. Agent Connection Tracking via Gossip Protocol (No Database)

**Why Gossip-Only:**

- Real-time updates (sub-second latency)
- Zero database overhead or schema changes
- Automatic consistency and failure detection
- Scales naturally with cluster size
- Ephemeral data - agents reconnect on server restart

**Gossip Message Types** (`managed/services/ha/gossip_messages.go` - new file):

```go
type GossipMessageType string

const (
    GossipAgentConnect    GossipMessageType = "agent_connect"
    GossipAgentDisconnect GossipMessageType = "agent_disconnect"
    GossipFullStateSync   GossipMessageType = "full_state_sync"
)

type GossipMessage struct {
    Type      GossipMessageType
    Timestamp time.Time
    Data      []byte
}

type AgentConnectionEvent struct {
    AgentID   string
    ServerID  string
    EventType string // "connect" or "disconnect"
    Timestamp time.Time
}

type FullStateSyncRequest struct {
    RequesterServerID string
}

type FullStateSyncResponse struct {
    AgentLocations map[string]string // agent_id -> server_id
    Timestamp      time.Time
}
```

**Gossip Integration** (`managed/services/ha/highavailability.go`):

- Implement `memberlist.Delegate` interface for custom gossip messages
- Add `agentLocations map[string]string` (agent_id → server_id) with RWMutex
- On receiving gossip: update shared map, expose via `GetAgentLocation(agentID) string`
- **Event-driven gossip** for connect/disconnect (efficient for <1000 agents)
- **On-demand full state sync** via direct RPC when node joins/restarts (scalable to 10K+ agents)
- Automatic cleanup when server leaves cluster (memberlist handles this)

**Registry Enhancement** (`managed/services/agents/registry.go`):

- On agent connect (`register()` line 156-217): Call `haService.BroadcastAgentConnect(agentID, serverID)`
- On agent disconnect (`unregister()` line 270+): Call `haService.BroadcastAgentDisconnect(agentID)`
- Add optional dependency on HA service (nil if HA disabled)
- **No database writes for agent connections**

### 2. Inter-Server Communication via Direct gRPC with mTLS

**Port Configuration: Reuse 8443 (Service Multiplexing)**

- Register `AgentForwarding` service alongside existing services on same gRPC server
- No additional port configuration needed
- mTLS authentication prevents unauthorized external access
- Simpler Kubernetes and firewall configuration

**Server Addressing: Pod IPs from Memberlist**

- Format: `172.20.0.5:8443` (from `PMM_HA_ADVERTISE_ADDRESS`)
- Works in any environment (K8s, Docker, VMs)
- Already available in memberlist
- Memberlist updates automatically on pod restart

**Protobuf Definitions** (`api/serverpb/forwarding.proto` - new file):

```protobuf
syntax = "proto3";
package serverpb;

import "agent/v1/agent.proto";
import "google/protobuf/any.proto";

service AgentForwarding {
  rpc ForwardAgentRequest(ForwardAgentRequestRequest) returns (ForwardAgentRequestResponse);
  rpc ForwardStateUpdate(ForwardStateUpdateRequest) returns (ForwardStateUpdateResponse);
  rpc ForwardJobRequest(ForwardJobRequestRequest) returns (ForwardJobRequestResponse);
  rpc SyncFullAgentState(FullStateSyncRequest) returns (FullStateSyncResponse); // For large clusters
}

message ForwardAgentRequestRequest {
  string agent_id = 1;
  string forwarded_by = 2;  // Source server ID (loop prevention)
  google.protobuf.Any request_payload = 3;
  string request_type = 4;
  int32 protocol_version = 5;  // For future compatibility
}

message ForwardAgentRequestResponse {
  google.protobuf.Any response_payload = 1;
  string error = 2;
}

message FullStateSyncRequest {
  string requester_server_id = 1;
}

message FullStateSyncResponse {
  map<string, string> agent_locations = 1; // agent_id -> server_id
  int64 timestamp = 2;
}
```

**gRPC Server** (`managed/services/agents/grpc/forwarding_server.go` - new file):

- Implements `AgentForwardingServer` interface
- Validates `forwarded_by` to prevent circular forwarding (max 1 hop)
- Checks `protocol_version` for compatibility
- Looks up agent in local registry and forwards request
- Returns response or "agent not found" error
- **Registered on existing port 8443** alongside other services

**gRPC Client** (`managed/services/agents/forwarding_client.go` - new file):

- Connects to other PMM servers using Pod IPs from memberlist
- Format: `<ip>:8443` (e.g., `172.20.0.5:8443`)
- mTLS authentication with cluster certificates from Kubernetes secrets
- Connection pooling for performance (one persistent connection per server)
- **Pass-through timeout** from original context (no adjustment)
- **Enable gRPC compression** (gzip) for large responses

**Authentication: Mutual TLS (mTLS)**

Certificate provisioning via Kubernetes secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pmm-cluster-cert
  namespace: monitoring
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-cert>
  tls.key: <base64-encoded-key>
  ca.crt: <base64-encoded-ca>
```

Mount to PMM server pods:

```yaml
volumeMounts:
  - name: cluster-cert
    mountPath: /srv/pmm-certs/cluster
    readOnly: true
volumes:
  - name: cluster-cert
    secret:
      secretName: pmm-cluster-cert
```

**Client mTLS Configuration:**

```go
func (c *ForwardingClient) Connect(ctx context.Context, serverAddr string) (*grpc.ClientConn, error) {
    tlsConfig, err := c.loadTLSConfig()
    if err != nil {
        return nil, err
    }
    
    return grpc.DialContext(ctx, serverAddr,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
        grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")), // Enable compression
        grpc.WithBlock(),
        grpc.WithTimeout(5*time.Second),
    )
}
```

**Certificate Generation Script** (`build/scripts/generate-cluster-cert.sh` - new file):

```bash
#!/bin/bash
# Generate CA, cluster certificate, and Kubernetes secret

openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
  -out ca.crt -subj "/CN=PMM Cluster CA/O=Percona"

openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -out tls.csr \
  -subj "/CN=pmm-cluster-internal/O=Percona"

cat > san.cnf <<EOF
[v3_req]
keyUsage = keyEncipherment, digitalSignature
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = *.pmm-cluster.svc.cluster.local
DNS.2 = pmm-server-active
DNS.3 = pmm-server-passive
DNS.4 = pmm-server-passive-2
EOF

openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out tls.crt -days 3650 -sha256 \
  -extensions v3_req -extfile san.cnf

kubectl create secret generic pmm-cluster-cert \
  --from-file=tls.crt=tls.crt \
  --from-file=tls.key=tls.key \
  --from-file=ca.crt=ca.crt \
  -n monitoring
```

### 3. Request Forwarding Logic with Retry

**Forwarder Service** (`managed/services/agents/forwarder.go` - new file):

Core algorithm (Try All Servers):

```go
func (f *Forwarder) ForwardRequest(ctx, agentID, request) (response, error) {
  // Check feature flag
  if !f.isEnabled() {
    return nil, ErrForwardingDisabled
  }
  
  // 1. Check local registry first
  if agent := f.registry.get(agentID); agent != nil {
    return agent.send(request)
  }
  
  // 2. Query gossip for agent location (no database query)
  serverID := f.haService.GetAgentLocation(agentID)
  if serverID == "" {
    return nil, ErrAgentNotFound
  }
  
  // 3. Try target server (pass through original context timeout)
  resp, err := f.client.Forward(ctx, serverID, agentID, request)
  if err == nil {
    // Requester writes action result to database
    return resp, nil
  }
  
  // 4. If failed, try ALL other servers in cluster
  servers := f.haService.GetPeerServers()
  for _, srv := range servers {
    if srv.ID == serverID { continue }
    resp, err := f.client.Forward(ctx, srv.ID, agentID, request)
    if err == nil {
      return resp, nil
    }
  }
  
  // 5. Final local retry (agent may have reconnected)
  if agent := f.registry.get(agentID); agent != nil {
    return agent.send(request)
  }
  
  f.logForwardingFailure(agentID, serverID)
  return nil, ErrAgentNotFound
}
```

**Graceful Shutdown Handling:**

- Set `shuttingDown` atomic bool on SIGTERM
- Reject new forwarding requests with error during shutdown
- Requesters automatically retry on other servers
- Simple implementation, graceful degradation

**Leader Election Behavior:**

- Followers **accept forwarding requests normally**
- Agent connections are independent of leadership
- No leader checks needed in forwarding logic

**Loop Prevention:**

- Check `forwarded_by` field in request
- If present, reject with error (max 1 hop)
- Add current server ID when forwarding

**Action Results Storage:**

- **Requester writes to database** (Server-2 in our example)
- Standard RPC pattern where caller handles response
- Create `action_results` entry after receiving forwarded response
- Consistent with non-HA behavior

### 4. Integration Points (All Agent Communication)

**Actions Service** (`managed/services/agents/actions.go`):

Modify ALL action methods (lines 458-586):

- `StartMySQLQueryShowAction()`
- `StartMongoDBQueryGetDiagnosticDataAction()`
- `StartPTSummaryAction()`
- `StartPTPgSummaryAction()`
- `StartPTMongoDBSummaryAction()`
- `StartPTMySQLSummaryAction()`
- `StopAction()`

Pattern:

```go
agent, err := s.r.get(pmmAgentID)
if err != nil {
  if s.forwarder != nil && s.forwarder.IsEnabled() {
    return s.forwarder.ForwardActionRequest(ctx, pmmAgentID, request)
  }
  return err
}
// ... existing code to send request to agent
```

**Connection Checker** (`managed/services/agents/connection_checker.go`):

- Modify `CheckConnectionToService()` at line 79-82
- Same pattern as actions

**Service Info Broker** (`managed/services/agents/service_info_broker.go`):

- Modify `GetInfoFromService()` at line 162-165
- Same pattern as actions

**State Manager** (`managed/services/agents/state.go`):

- Add forwarding to state update requests
- Ensure state changes propagate to correct server

### 5. HA Service Extensions

**New Methods** (`managed/services/ha/highavailability.go`):

```go
// GetAgentLocation returns server ID where agent is connected (from gossip map)
func (s *Service) GetAgentLocation(agentID string) string

// BroadcastAgentConnect notifies cluster of new agent connection via gossip
func (s *Service) BroadcastAgentConnect(agentID, serverID string)

// BroadcastAgentDisconnect notifies cluster of agent disconnect via gossip
func (s *Service) BroadcastAgentDisconnect(agentID string)

// GetPeerServers returns list of all active servers with IDs and addresses (Pod IPs)
func (s *Service) GetPeerServers() []PeerServer

type PeerServer struct {
    ID      string  // Node ID (e.g., "pmm-server-active")
    Address string  // Pod IP:Port (e.g., "172.20.0.5:8443")
}

// RequestFullStateSync requests full agent location map from another server
// Used when node joins or on-demand for large clusters
func (s *Service) RequestFullStateSync(ctx context.Context, targetServer string) error
```

**Memberlist Delegate Implementation** (`managed/services/ha/gossip_delegate.go` - new file):

- Implement `memberlist.Delegate` interface
- Serialize `GossipMessage` to bytes for gossip
- Handle `NodeMeta`, `NotifyMsg`, `GetBroadcasts`, `LocalState`, `MergeRemoteState`
- Event-driven gossip for connect/disconnect
- Full state sync via direct gRPC call (not gossip) for scalability

### 6. Feature Flag & Compatibility

**Feature Flag** (`managed/models/settings.go`):

```go
type Settings struct {
    // ... existing fields
    HAForwardingEnabled bool `json:"ha_forwarding_enabled"`
}
```

- Stored in database `settings` table
- Default: `false` for safety during rollout
- Enable via API: `PATCH /v1/Settings/Change` with `{"ha_forwarding_enabled": true}`
- Check flag in Forwarder before attempting forwarding

**Protocol Versioning:**

- Include `protocol_version: 1` in all forwarding requests
- Server validates version, rejects if incompatible
- Future-proofs for protocol changes

**Backward Compatibility Strategy:**

1. Deploy new PMM version to all servers (forwarding disabled)
2. Verify all servers healthy
3. Enable feature flag: `ha_forwarding_enabled = true`
4. Monitor forwarding metrics
5. Rollback: disable flag if issues arise

### 7. Security & Audit

**Authentication: mTLS Trust Model**

- If server has valid cluster certificate, it can access any agent
- Cluster servers are trusted peers (appropriate for HA architecture)
- No additional authorization checks beyond mTLS

**Audit Logging:**

- Log ALL forwarding requests with:
        - `agent_id`, `source_server`, `target_server`, `request_type`, `timestamp`
        - Request ID for correlation
        - Success/failure status
- Store in dedicated log file: `/srv/logs/pmm-forwarding-audit.log`
- Retention: 30 days
- Format: JSON for easy parsing

Example log entry:

```json
{
  "timestamp": "2025-10-17T10:30:45Z",
  "request_id": "uuid-1234",
  "agent_id": "pmm-agent-xyz",
  "source_server": "pmm-server-active",
  "target_server": "pmm-server-passive",
  "request_type": "pt_mysql_summary",
  "result": "success",
  "duration_ms": 156
}
```

### 8. Error Handling & Edge Cases

**Gossip Lag Handling:**

- If gossip says agent is on Server-1 but it's not there, try all servers
- Covers cases where gossip hasn't propagated yet
- Final local retry catches agent reconnection race

**Circular Forwarding Prevention:**

- Reject if `forwarded_by` header present
- Maximum 1 hop enforced at gRPC server level
- Log warning if loop detected

**Retry Strategy (Try All Servers):**

1. Try server from gossip location map
2. If failed, iterate all cluster members
3. Stop on first success
4. Return error if all fail
5. Final local retry before giving up

**Network Partition Handling:**

- If server unreachable, skip to next
- Gossip failure detection removes dead servers from memberlist
- Client retries handle transient failures

**Performance Optimization:**

- Cache gRPC connections per server (persistent connection pool)
- Reuse connections for multiple requests
- Close connections on server removal from cluster
- Enable gRPC compression for large responses (pt-summary can be 100KB+)

**Timeout Handling:**

- Pass through original context deadline unchanged
- If timeout expires during forwarding, that's a legitimate timeout
- No arbitrary timeout adjustments

### 9. Observability

**Metrics** (`managed/services/agents/forwarder.go`):

```go
pmm_managed_agents_requests_forwarded_total{target_server, success}
pmm_managed_agents_requests_forwarded_errors_total{error_type}
pmm_managed_agents_forwarding_duration_seconds{target_server}
pmm_managed_agents_gossip_events_total{event_type}
pmm_managed_agents_location_cache_size
pmm_managed_agents_forwarding_enabled{} // Feature flag status
```

**Logging:**

- Structured logs with `agent_id`, `source_server`, `target_server`, `request_type`, `request_id`
- Debug level: gossip events, cache updates
- Info level: forwarding attempts, successes
- Warn level: retry failures, loop detection
- Error level: complete forwarding failures
- **Audit logs**: separate file for all forwarding activity

**Request Correlation:**

- Generate unique `request_id` (UUID) for each forwarding request
- Include in logs at both source and target servers
- Propagate in gRPC metadata for distributed tracing
- Enables cross-server request tracking

## Implementation Files

**New Files:**

- `api/serverpb/forwarding.proto` - gRPC service definitions
- `managed/services/agents/grpc/forwarding_server.go` - gRPC server with mTLS (port 8443)
- `managed/services/agents/forwarding_client.go` - gRPC client with mTLS
- `managed/services/agents/forwarder.go` - forwarding coordinator
- `managed/services/ha/gossip_messages.go` - gossip message types
- `managed/services/ha/gossip_delegate.go` - memberlist delegate
- `build/scripts/generate-cluster-cert.sh` - certificate generation script

**Modified Files:**

- `managed/services/agents/actions.go` - add forwarding to all actions (7 methods)
- `managed/services/agents/connection_checker.go` - add forwarding
- `managed/services/agents/service_info_broker.go` - add forwarding
- `managed/services/agents/registry.go` - gossip broadcast on connect/disconnect (no DB writes)
- `managed/services/agents/state.go` - add forwarding to state updates
- `managed/services/ha/highavailability.go` - agent location tracking via gossip
- `managed/models/settings.go` - add HAForwardingEnabled field
- `managed/cmd/pmm-managed/main.go` - register forwarding gRPC service on port 8443

**No Database Schema Changes** - All agent location tracking is gossip-based (in-memory)

## Testing Strategy

**Unit Tests:**

- Forwarder retry logic with mocked client
- Gossip message serialization/deserialization
- Loop detection with forwarded_by header
- Error handling for all failure modes
- mTLS certificate loading and validation
- Feature flag checks
- Timeout propagation

**Integration Tests (3-node HA cluster):**

- Agent on server-1, request from server-2 via gossip lookup
- Agent disconnects during forwarding, verify retry
- Kill target server, verify try-all-servers logic
- Leader failover during active requests (verify followers accept forwarding)
- Concurrent requests to same agent from multiple servers
- Gossip propagation delay scenarios
- Certificate-based authentication between servers
- Feature flag enable/disable toggling
- Graceful shutdown during active forwarding
- Action result storage on requester server

**Performance Tests:**

- Forwarding overhead vs local request (<5ms acceptable with compression)
- Gossip broadcast latency across cluster
- Connection pool efficiency under load
- Memory usage of location cache with 10k agents
- mTLS handshake overhead
- gRPC compression effectiveness (expect 80% reduction for text)