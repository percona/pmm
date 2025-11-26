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

// Package ha contains everything related to high availability.
package ha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

const (
	defaultNodeEventChanSize  = 5
	defaultRaftRetries        = 3
	defaultTransportTimeout   = 10 * time.Second
	defaultLeaveTimeout       = 5 * time.Second
	defaultTickerInterval     = 5 * time.Second
	defaultApplyTimeout       = 3 * time.Second
	defaultRaftDataDir        = "/srv/ha"
	defaultRaftDataDirPerm    = 0o750
	defaultSnapshotRetention  = 3
	defaultSnapshotThreshold  = 8192
	defaultTrailingLogs       = 10240
	defaultHeartbeatTimeout   = 1000 * time.Millisecond
	defaultElectionTimeout    = 1000 * time.Millisecond
	defaultCommitTimeout      = 50 * time.Millisecond
	defaultLeaderLeaseTimeout = 500 * time.Millisecond
	defaultSnapshotInterval   = 120 * time.Second
	defaultServerOpTimeout    = 10 * time.Second
)

// Service represents the high-availability service.
type Service struct {
	params           *models.HAParams
	bootstrapCluster bool

	services *services

	nodeCh   chan memberlist.NodeEvent
	leaderCh chan raft.Observation

	l  *logrus.Entry
	wg *sync.WaitGroup

	rw         sync.RWMutex
	raftNode   *raft.Raft
	memberlist *memberlist.Memberlist

	// Agent connection status tracking (distributed state)
	connectionsMu sync.RWMutex
	connections   map[string]bool // agentID -> connected status
}

// Apply applies a log entry to the high-availability service.
// Processes commands to maintain distributed agent connection state.
func (s *Service) Apply(logEntry *raft.Log) interface{} {
	s.l.Debugf("raft: applying log entry: index=%d, type=%d", logEntry.Index, logEntry.Type)

	// Skip non-data entries
	if logEntry.Type != raft.LogCommand {
		return nil
	}

	cmd, err := DecodeCommand(logEntry.Data)
	if err != nil {
		s.l.Errorf("failed to decode command: %v", err)
		return err
	}

	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	switch cmd.Type {
	case CommandTypeSetAgentConnection:
		var payload SetAgentConnectionPayload
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			s.l.Errorf("failed to unmarshal SetAgentConnection payload: %v", err)
			return err
		}
		s.connections[payload.AgentID] = payload.Connected
		s.l.Debugf("Set agent %s connection status to %v", payload.AgentID, payload.Connected)
		return nil

	case CommandTypeDeleteAgentConnection:
		var payload DeleteAgentConnectionPayload
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			s.l.Errorf("failed to unmarshal DeleteAgentConnection payload: %v", err)
			return err
		}
		delete(s.connections, payload.AgentID)
		s.l.Debugf("Deleted agent %s connection status", payload.AgentID)
		return nil

	default:
		s.l.Warnf("unknown command type: %s", cmd.Type)
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Snapshot returns a snapshot of the high-availability service.
// Captures the current state of agent connections for persistence.
func (s *Service) Snapshot() (raft.FSMSnapshot, error) { //nolint:ireturn
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	// Create a deep copy of the connections map
	connectionsCopy := make(map[string]bool, len(s.connections))
	for k, v := range s.connections {
		connectionsCopy[k] = v
	}

	s.l.Infof("Creating snapshot with %d agent connection statuses", len(connectionsCopy))
	return &fsmSnapshot{connections: connectionsCopy}, nil
}

// Restore restores the high availability service to a previous state.
// Restores agent connection status from a snapshot (called on node restart).
func (s *Service) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var connections map[string]bool
	if err := json.NewDecoder(rc).Decode(&connections); err != nil {
		// If snapshot is empty or invalid, start with empty state
		if err == io.EOF {
			s.l.Info("Empty snapshot, starting with no agent connections")
			s.connectionsMu.Lock()
			s.connections = make(map[string]bool)
			s.connectionsMu.Unlock()
			return nil
		}
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	s.connectionsMu.Lock()
	s.connections = connections
	s.connectionsMu.Unlock()

	s.l.Infof("Restored %d agent connection statuses from snapshot", len(connections))
	return nil
}

// fsmSnapshot implements raft.FSMSnapshot for PMM HA.
type fsmSnapshot struct {
	connections map[string]bool
}

// Persist writes the snapshot to the sink.
// Persists agent connection state for restoration on node restart.
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(f.connections)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}
	return sink.Close()
}

// Release is called when we are finished with the snapshot.
func (f *fsmSnapshot) Release() {
	// Nothing to release
}

// setupRaftStorage sets up persistent storage for Raft.
func setupRaftStorage(nodeID string, l *logrus.Entry) (*raftboltdb.BoltStore, *raftboltdb.BoltStore, *raft.FileSnapshotStore, error) {
	// Create the Raft data directory for this node
	raftDir := filepath.Join(defaultRaftDataDir, nodeID)
	if err := os.MkdirAll(raftDir, defaultRaftDataDirPerm); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Raft data directory: %w", err)
	}
	l.Infof("Using Raft data directory: %s", raftDir)

	// Create BoltDB-based log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BoltDB log store: %w", err)
	}

	// Create BoltDB-based stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
	if err != nil {
		if cerr := logStore.Close(); cerr != nil {
			l.Errorf("failed to close logStore after stableStore error: %v", cerr)
		}
		return nil, nil, nil, fmt.Errorf("failed to create BoltDB stable store: %w", err)
	}

	// Create file-based snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, defaultSnapshotRetention, os.Stderr)
	if err != nil {
		if cerr := logStore.Close(); cerr != nil {
			l.Errorf("failed to close logStore after snapshotStore error: %v", cerr)
		}
		if cerr := stableStore.Close(); cerr != nil {
			l.Errorf("failed to close stableStore after snapshotStore error: %v", cerr)
		}
		return nil, nil, nil, fmt.Errorf("failed to create file snapshot store: %w", err)
	}

	return logStore, stableStore, snapshotStore, nil
}

// New provides a new instance of the high availability service.
func New(params *models.HAParams) *Service {
	return &Service{
		params:           params,
		bootstrapCluster: params.Bootstrap,
		services:         newServices(),
		nodeCh:           make(chan memberlist.NodeEvent, defaultNodeEventChanSize),
		leaderCh:         make(chan raft.Observation),
		l:                logrus.WithField("component", "ha"),
		wg:               &sync.WaitGroup{},
		connections:      make(map[string]bool),
	}
}

// Run runs the high availability service.
func (s *Service) Run(ctx context.Context) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.services.Refresh():
				if s.IsLeader() {
					s.services.StartAllServices(ctx)
				}
			case <-ctx.Done():
				s.services.StopAllServices()
				return
			}
		}
	}()

	if !s.params.Enabled {
		s.l.Infoln("High availability is disabled")
		s.services.Wait()
		s.wg.Wait()
		return nil
	}

	s.l.Infoln("Starting...")
	defer s.l.Infoln("Done.")

	// Create the Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(s.params.NodeID)
	raftConfig.LogOutput = s.l.Logger.Out

	// Set log level based on environment
	if os.Getenv("PMM_DEBUG") == "1" {
		raftConfig.LogLevel = "DEBUG"
	} else {
		raftConfig.LogLevel = "WARN"
	}

	// Configure timeouts for better cluster stability
	raftConfig.HeartbeatTimeout = defaultHeartbeatTimeout
	raftConfig.ElectionTimeout = defaultElectionTimeout
	raftConfig.CommitTimeout = defaultCommitTimeout
	raftConfig.LeaderLeaseTimeout = defaultLeaderLeaseTimeout

	// Configure snapshots for log compaction
	// Since PMM HA is stateless (leader election only), snapshots are minimal
	raftConfig.SnapshotInterval = defaultSnapshotInterval
	raftConfig.SnapshotThreshold = defaultSnapshotThreshold
	raftConfig.TrailingLogs = defaultTrailingLogs

	// Create a new Raft transport
	raa, err := net.ResolveTCPAddr("", net.JoinHostPort(s.params.AdvertiseAddress, strconv.Itoa(s.params.RaftPort)))
	if err != nil {
		return err
	}
	raftTrans, err := raft.NewTCPTransport(net.JoinHostPort("0.0.0.0", strconv.Itoa(s.params.RaftPort)), raa, defaultRaftRetries, defaultTransportTimeout, nil)
	if err != nil {
		return err
	}

	// Set up persistent storage for Raft
	logStore, stableStore, snapshotStore, err := setupRaftStorage(s.params.NodeID, s.l)
	if err != nil {
		return err
	}

	defer func() {
		if logStore != nil {
			if closeErr := logStore.Close(); closeErr != nil {
				s.l.Errorf("error closing log store: %v", closeErr)
			}
		}
		if stableStore != nil {
			if closeErr := stableStore.Close(); closeErr != nil {
				s.l.Errorf("error closing stable store: %v", closeErr)
			}
		}
	}()

	// Create a new Raft node with persistent storage
	s.rw.Lock()
	s.raftNode, err = raft.NewRaft(raftConfig, s, logStore, stableStore, snapshotStore, raftTrans)
	s.rw.Unlock()
	if err != nil {
		return err
	}
	defer func() {
		if s.IsLeader() {
			s.raftNode.LeadershipTransfer()
		}
		err := s.raftNode.Shutdown().Error()
		if err != nil {
			s.l.Errorf("error during the shutdown of raft node: %q", err)
		}
	}()

	// Create the memberlist configuration
	memberlistConfig := memberlist.DefaultWANConfig()
	memberlistConfig.Name = s.params.NodeID
	memberlistConfig.BindAddr = "0.0.0.0"
	memberlistConfig.BindPort = s.params.GossipPort
	memberlistConfig.AdvertiseAddr = raa.IP.String()
	memberlistConfig.AdvertisePort = s.params.GossipPort
	memberlistConfig.Events = &memberlist.ChannelEventDelegate{Ch: s.nodeCh}

	// Create the memberlist
	s.memberlist, err = memberlist.Create(memberlistConfig)
	if err != nil {
		return fmt.Errorf("failed to create memberlist: %w", err)
	}
	defer func() {
		err := s.memberlist.Leave(defaultLeaveTimeout)
		if err != nil {
			s.l.Errorf("couldn't leave memberlist cluster: %q", err)
		}
		err = s.memberlist.Shutdown()
		if err != nil {
			s.l.Errorf("couldn't shutdown memberlist listeners: %q", err)
		}
	}()

	if s.bootstrapCluster {
		// Start the Raft node
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(s.params.NodeID),
					Address:  raft.ServerAddress(raa.String()),
				},
			},
		}
		if err := s.raftNode.BootstrapCluster(cfg).Error(); err != nil {
			// Cluster might already be bootstrapped with persistent storage
			if !errors.Is(err, raft.ErrCantBootstrap) {
				return fmt.Errorf("failed to bootstrap Raft cluster: %w", err)
			}
			s.l.Info("Cluster already bootstrapped, skipping")
		}
	}
	if len(s.params.Nodes) != 0 {
		_, err := s.memberlist.Join(s.params.Nodes)
		if err != nil {
			return fmt.Errorf("failed to join memberlist cluster: %w", err)
		}
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runLeaderObserver(ctx)
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runRaftNodesSynchronizer(ctx)
	}()

	<-ctx.Done()

	s.services.Wait()
	s.wg.Wait()

	return nil
}

func (s *Service) runRaftNodesSynchronizer(ctx context.Context) {
	t := time.NewTicker(defaultTickerInterval)
	defer t.Stop()

	for {
		select {
		case event := <-s.nodeCh:
			if !s.IsLeader() {
				continue
			}
			node := event.Node
			switch event.Event {
			case memberlist.NodeJoin:
				s.addMemberlistNodeToRaft(node)
			case memberlist.NodeLeave:
				s.removeMemberlistNodeFromRaft(node)
			case memberlist.NodeUpdate:
				continue
			}
		case <-t.C:
			if !s.IsLeader() {
				continue
			}

			// Get Raft configuration with error handling
			configFuture := s.raftNode.GetConfiguration()
			if err := configFuture.Error(); err != nil {
				s.l.Errorf("failed to get raft configuration: %v", err)
				continue
			}

			servers := configFuture.Configuration().Servers
			raftServers := make(map[string]struct{})
			for _, server := range servers {
				raftServers[string(server.ID)] = struct{}{}
			}
			members := s.memberlist.Members()
			s.l.Infof("HA memberlist: %v", members)
			for _, node := range members {
				if _, ok := raftServers[node.Name]; !ok {
					s.addMemberlistNodeToRaft(node)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) removeMemberlistNodeFromRaft(node *memberlist.Node) {
	s.rw.RLock()
	defer s.rw.RUnlock()
	err := s.raftNode.RemoveServer(raft.ServerID(node.Name), 0, defaultServerOpTimeout).Error()
	if err != nil {
		s.l.Errorln(err)
	}
}

func (s *Service) addMemberlistNodeToRaft(node *memberlist.Node) {
	s.rw.RLock()
	defer s.rw.RUnlock()
	serverAddress := raft.ServerAddress(fmt.Sprintf("%s:%d", node.Addr.String(), s.params.RaftPort))
	err := s.raftNode.AddVoter(raft.ServerID(node.Name), serverAddress, 0, defaultServerOpTimeout).Error()
	if err != nil {
		s.l.Errorf("couldn't add a server node %s: %q", node.Name, err)
	}
}

func (s *Service) runLeaderObserver(ctx context.Context) {
	t := time.NewTicker(defaultTickerInterval)
	defer t.Stop()

	for {
		s.rw.RLock()
		node := s.raftNode
		s.rw.RUnlock()
		select {
		case isLeader := <-node.LeaderCh():
			if isLeader {
				s.services.StartAllServices(ctx)
				s.l.Info("I am the leader!")
				peers := s.memberlist.Members()
				for _, peer := range peers {
					if peer.Name == s.params.NodeID {
						continue
					}
					s.addMemberlistNodeToRaft(peer)
				}
			} else {
				s.l.Info("I am not a leader!")
				s.services.StopAllServices()
			}
		case <-t.C:
			address, serverID := s.raftNode.LeaderWithID()
			if serverID != "" {
				s.l.Infof("Leader is %s on %s", serverID, address)
			}
		case <-ctx.Done():
			return
		}
	}
}

// AddLeaderService adds a leader service to the high availability service.
func (s *Service) AddLeaderService(leaderService LeaderService) {
	err := s.services.Add(leaderService)
	if err != nil {
		s.l.Errorf("couldn't add HA service: %+v", err)
	}
}

// BroadcastMessage broadcasts a message from the high availability service.
// Used for distributing agent connection state updates across the cluster.
func (s *Service) BroadcastMessage(message []byte) error {
	if !s.params.Enabled {
		return fmt.Errorf("HA is disabled")
	}

	s.rw.RLock()
	raftNode := s.raftNode
	s.rw.RUnlock()

	if raftNode == nil {
		return fmt.Errorf("raft node is not initialized")
	}

	future := raftNode.Apply(message, defaultApplyTimeout)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply log to raft: %w", err)
	}
	return nil
}

// IsLeader checks if the current instance of HA service is the leader.
func (s *Service) IsLeader() bool {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return !s.params.Enabled || (s.raftNode != nil && s.raftNode.State() == raft.Leader)
}

// Bootstrap returns true if HA service should bootstrap (true in non-HA setups).
func (s *Service) Bootstrap() bool {
	return s.params.Bootstrap || !s.params.Enabled
}

// GetParams returns HA parameters.
func (s *Service) GetParams() *models.HAParams {
	return s.params
}

// SetAgentConnection sets the connection status for an agent via Raft.
// In HA mode, this replicates the state across all nodes.
func (s *Service) SetAgentConnection(agentID string, connected bool) error {
	if !s.params.Enabled {
		// In non-HA mode, just store locally
		s.connectionsMu.Lock()
		s.connections[agentID] = connected
		s.connectionsMu.Unlock()
		return nil
	}

	payload := SetAgentConnectionPayload{
		AgentID:   agentID,
		Connected: connected,
	}

	data, err := EncodeCommand(CommandTypeSetAgentConnection, payload)
	if err != nil {
		return fmt.Errorf("failed to encode command: %w", err)
	}

	return s.BroadcastMessage(data)
}

// DeleteAgentConnection removes the connection status for an agent via Raft.
// In HA mode, this replicates the deletion across all nodes.
func (s *Service) DeleteAgentConnection(agentID string) error {
	if !s.params.Enabled {
		// In non-HA mode, just delete locally
		s.connectionsMu.Lock()
		delete(s.connections, agentID)
		s.connectionsMu.Unlock()
		return nil
	}

	payload := DeleteAgentConnectionPayload{
		AgentID: agentID,
	}

	data, err := EncodeCommand(CommandTypeDeleteAgentConnection, payload)
	if err != nil {
		return fmt.Errorf("failed to encode command: %w", err)
	}

	return s.BroadcastMessage(data)
}

// IsAgentConnected checks if an agent is connected (reads from distributed state).
func (s *Service) IsAgentConnected(agentID string) bool {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()
	return s.connections[agentID]
}

// GetAllConnections returns a copy of all connection statuses.
func (s *Service) GetAllConnections() map[string]bool {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	result := make(map[string]bool, len(s.connections))
	for k, v := range s.connections {
		result[k] = v
	}
	return result
}
