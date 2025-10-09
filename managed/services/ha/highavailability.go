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
	defaultNodeEventChanSize = 5
	defaultMessageChanSize   = 100
	defaultRaftRetries       = 3
	defaultTransportTimeout  = 10 * time.Second
	defaultLeaveTimeout      = 5 * time.Second
	defaultTickerInterval    = 5 * time.Second
	defaultApplyTimeout      = 3 * time.Second
	defaultRaftDataDir       = "/srv/ha"
	defaultSnapshotRetention = 3
	defaultSnapshotThreshold = 8192
	defaultTrailingLogs      = 10240
)

// Service represents the high-availability service.
type Service struct {
	params           *models.HAParams
	bootstrapCluster bool

	services *services

	receivedMessages chan []byte
	nodeCh           chan memberlist.NodeEvent
	leaderCh         chan raft.Observation

	l  *logrus.Entry
	wg *sync.WaitGroup

	rw         sync.RWMutex
	raftNode   *raft.Raft
	memberlist *memberlist.Memberlist
}

// Apply applies a log entry to the high-availability service.
func (s *Service) Apply(logEntry *raft.Log) interface{} {
	s.l.Printf("raft: got a message: %s", string(logEntry.Data))

	// Non-blocking send to prevent blocking the Raft state machine
	select {
	case s.receivedMessages <- logEntry.Data:
		return nil
	default:
		s.l.Warn("receivedMessages channel full, dropping message")
		return fmt.Errorf("channel full, message dropped")
	}
}

// Snapshot returns a snapshot of the high-availability service.
func (s *Service) Snapshot() (raft.FSMSnapshot, error) { //nolint:ireturn
	return nil, nil //nolint:nilnil
}

// Restore restores the high availability service to a previous state.
func (s *Service) Restore(_ io.ReadCloser) error {
	return nil
}

// setupRaftStorage sets up persistent storage for Raft.
func setupRaftStorage(nodeID string, l *logrus.Entry) (*raftboltdb.BoltStore, *raftboltdb.BoltStore, raft.SnapshotStore, error) {
	// Create the Raft data directory for this node
	raftDir := filepath.Join(defaultRaftDataDir, nodeID)
	if err := os.MkdirAll(raftDir, 0o755); err != nil {
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
		logStore.Close() //nolint:errcheck
		return nil, nil, nil, fmt.Errorf("failed to create BoltDB stable store: %w", err)
	}

	// Create file-based snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, defaultSnapshotRetention, os.Stderr)
	if err != nil {
		logStore.Close()    //nolint:errcheck
		stableStore.Close() //nolint:errcheck
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
		receivedMessages: make(chan []byte, defaultMessageChanSize),
		l:                logrus.WithField("component", "ha"),
		wg:               &sync.WaitGroup{},
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
				s.services.StopRunningServices()
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

	// Set log level based on environment
	if os.Getenv("PMM_DEBUG") == "1" {
		raftConfig.LogLevel = "DEBUG"
	} else {
		raftConfig.LogLevel = "WARN"
	}

	// Configure timeouts for better cluster stability
	raftConfig.HeartbeatTimeout = 1000 * time.Millisecond
	raftConfig.ElectionTimeout = 1000 * time.Millisecond
	raftConfig.CommitTimeout = 50 * time.Millisecond
	raftConfig.LeaderLeaseTimeout = 500 * time.Millisecond

	// Disable automatic snapshots since FSM doesn't implement them properly
	// This saves disk space and avoids creating empty snapshot files
	raftConfig.SnapshotInterval = 0
	raftConfig.SnapshotThreshold = 0
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

	if len(s.params.Nodes) == 0 && s.bootstrapCluster {
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
			if err != raft.ErrCantBootstrap {
				return fmt.Errorf("failed to bootstrap Raft cluster: %w", err)
			}
			s.l.Info("Cluster already bootstrapped, skipping")
		}
	}
	if len(s.params.Nodes) != 0 {
		_, err := s.memberlist.Join(s.params.Nodes)
		if err != nil {
			if s.bootstrapCluster {
				s.l.WithError(err).Warn("failed to join memberlist cluster, trying to bootstrap Raft cluster")
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
					if err != raft.ErrCantBootstrap {
						return fmt.Errorf("failed to bootstrap Raft cluster: %w", err)
					}
					s.l.Info("Cluster already bootstrapped, skipping")
				}
			} else {
				return fmt.Errorf("failed to join memberlist cluster: %w", err)
			}
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
			s.l.Infof("memberlist members: %v", members)
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
	err := s.raftNode.RemoveServer(raft.ServerID(node.Name), 0, 10*time.Second).Error()
	if err != nil {
		s.l.Errorln(err)
	}
}

func (s *Service) addMemberlistNodeToRaft(node *memberlist.Node) {
	s.rw.RLock()
	defer s.rw.RUnlock()
	err := s.raftNode.AddVoter(raft.ServerID(node.Name), raft.ServerAddress(fmt.Sprintf("%s:%d", node.Addr.String(), s.params.RaftPort)), 0, 10*time.Second).Error()
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
				// This node is the leader
				s.l.Printf("I am the leader!")
				peers := s.memberlist.Members()
				for _, peer := range peers {
					if peer.Name == s.params.NodeID {
						continue
					}
					s.addMemberlistNodeToRaft(peer)
				}
			} else {
				s.l.Printf("I am not a leader!")
				s.services.StopRunningServices()
			}
		case <-t.C:
			address, serverID := s.raftNode.LeaderWithID()
			s.l.Infof("Leader is %s on %s", serverID, address)
		case <-ctx.Done():
			return
		}
	}
}

// AddLeaderService adds a leader service to the high availability service.
func (s *Service) AddLeaderService(leaderService LeaderService) {
	err := s.services.Add(leaderService)
	if err != nil {
		s.l.Errorf("couldn't add HA service: +%v", err)
	}
}

// BroadcastMessage broadcasts a message from the high availability service.
func (s *Service) BroadcastMessage(message []byte) error {
	if s.params.Enabled {
		s.rw.RLock()
		defer s.rw.RUnlock()

		future := s.raftNode.Apply(message, defaultApplyTimeout)
		if err := future.Error(); err != nil {
			return fmt.Errorf("failed to apply log to raft: %w", err)
		}
		return nil
	}

	select {
	case s.receivedMessages <- message:
		return nil
	case <-time.After(defaultApplyTimeout):
		return fmt.Errorf("timeout sending message to channel")
	}
}

// IsLeader checks if the current instance of the high availability service is the leader.
func (s *Service) IsLeader() bool {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return (s.raftNode != nil && s.raftNode.State() == raft.Leader) || !s.params.Enabled
}

// Bootstrap performs the necessary steps to initialize the high availability service.
func (s *Service) Bootstrap() bool {
	return s.params.Bootstrap || !s.params.Enabled
}
