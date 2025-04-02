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
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

const (
	defaultNodeEventChanSize = 5
	defaultRaftRetries       = 3
	defaultLeaveTimeout      = 5 * time.Second
	defaultTickerInterval    = 5 * time.Second
	defaultApplyTimeout      = 3 * time.Second
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
	s.receivedMessages <- logEntry.Data
	return nil
}

// Snapshot returns a snapshot of the high-availability service.
func (s *Service) Snapshot() (raft.FSMSnapshot, error) { //nolint:ireturn
	return nil, nil //nolint:nilnil
}

// Restore restores the high availability service to a previous state.
func (s *Service) Restore(_ io.ReadCloser) error {
	return nil
}

// New provides a new instance of the high availability service.
func New(params *models.HAParams) *Service {
	return &Service{
		params:           params,
		bootstrapCluster: params.Bootstrap,
		services:         newServices(),
		nodeCh:           make(chan memberlist.NodeEvent, defaultNodeEventChanSize),
		leaderCh:         make(chan raft.Observation),
		receivedMessages: make(chan []byte),
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
		return nil
	}

	s.l.Infoln("Starting...")
	defer s.l.Infoln("Done.")

	// Create the Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(s.params.NodeID)
	raftConfig.LogLevel = "DEBUG"

	// Create a new Raft transport
	raa, err := net.ResolveTCPAddr("", net.JoinHostPort(s.params.AdvertiseAddress, strconv.Itoa(s.params.RaftPort)))
	if err != nil {
		return err
	}
	raftTrans, err := raft.NewTCPTransport(net.JoinHostPort("0.0.0.0", strconv.Itoa(s.params.RaftPort)), raa, defaultRaftRetries, 10*time.Second, nil)
	if err != nil {
		return err
	}

	// Create a new Raft node
	s.rw.Lock()
	s.raftNode, err = raft.NewRaft(raftConfig, s, raft.NewInmemStore(), raft.NewInmemStore(), raft.NewInmemSnapshotStore(), raftTrans)
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
			return fmt.Errorf("failed to bootstrap Raft cluster: %w", err)
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
			servers := s.raftNode.GetConfiguration().Configuration().Servers
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
			t.Stop()
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
func (s *Service) BroadcastMessage(message []byte) {
	if s.params.Enabled {
		s.rw.RLock()
		defer s.rw.RUnlock()
		s.raftNode.Apply(message, defaultApplyTimeout)
	} else {
		s.receivedMessages <- message
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
