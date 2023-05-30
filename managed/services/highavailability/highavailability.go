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

// Package highavailability contains everything related to high availability.
package highavailability

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	transport "github.com/Jille/raft-grpc-transport"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/percona/pmm/api/hapb"
)

type Params struct {
	Enabled          bool
	Bootstrap        bool
	NodeID           string
	AdvertiseAddress string
	Nodes            []string
	RaftPort         int
	GossipPort       int
}

type Service struct {
	params           *Params
	bootstrapCluster bool

	services *services

	receivedMessages chan []byte
	nodeCh           chan memberlist.NodeEvent
	leaderCh         chan raft.Observation

	l  *logrus.Entry
	wg *sync.WaitGroup

	raftNode   *raft.Raft
	memberlist *memberlist.Memberlist
	tm         *transport.Manager
}

func (s *Service) Apply(logEntry *raft.Log) interface{} {
	s.l.Printf("raft: got a message: %s", string(logEntry.Data))
	var m hapb.ClusterMessage
	err := proto.Unmarshal(logEntry.Data, &m)
	if err != nil {
		return err
	}
	switch m.Payload.(type) {
	case *hapb.ClusterMessage_ServerMessage:
	case *hapb.ClusterMessage_AgentMessage:

	}
	s.receivedMessages <- logEntry.Data
	return nil
}

func (s *Service) Snapshot() (raft.FSMSnapshot, error) {
	// TODO implement me
	return nil, nil
}

func (s *Service) Restore(snapshot io.ReadCloser) error {
	// TODO implement me
	return nil
}

func New(params *Params) *Service {
	return &Service{
		params:           params,
		bootstrapCluster: params.Bootstrap,
		services:         newServices(),
		nodeCh:           make(chan memberlist.NodeEvent, 5),
		leaderCh:         make(chan raft.Observation),
		receivedMessages: make(chan []byte),
		l:                logrus.WithField("component", "ha"),
		wg:               &sync.WaitGroup{},
		tm:               transport.New(raft.ServerAddress(fmt.Sprintf("%s:443", params.AdvertiseAddress)), []grpc.DialOption{grpc.WithInsecure()}),
	}
}

func (s *Service) Run(ctx context.Context) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.services.ServiceAdded():
				if s.IsLeader() {
					s.services.StartAllServices(ctx)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	if !s.params.Enabled {
		return nil
	}

	// Create the Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(s.params.NodeID)
	raftConfig.LogLevel = "INFO"

	// Create a new Raft transport
	raa, err := net.ResolveTCPAddr("", fmt.Sprintf("%s:%d", s.params.AdvertiseAddress, s.params.RaftPort))
	if err != nil {
		return err
	}
	raftTrans, err := raft.NewTCPTransport(fmt.Sprintf("0.0.0.0:%d", s.params.RaftPort), raa, 3, 10*time.Second, nil)
	if err != nil {
		return err
	}

	// Create a new Raft node
	s.raftNode, err = raft.NewRaft(raftConfig, s, raft.NewInmemStore(), raft.NewInmemStore(), raft.NewInmemSnapshotStore(), raftTrans)
	if err != nil {
		return err
	}
	defer s.raftNode.Shutdown().Error()

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
		return fmt.Errorf("failed to create memberlist: %q", err)
	}
	defer s.memberlist.Leave(5 * time.Second)

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
			return fmt.Errorf("failed to bootstrap Raft cluster: %q", err)
		}
	} else {
		_, err := s.memberlist.Join(s.params.Nodes)
		if err != nil {
			return fmt.Errorf("failed to join memberlist cluster: %q", err)
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
	for {
		select {
		case event := <-s.nodeCh:
			if !s.IsLeader() {
				continue
			}
			node := event.Node
			switch event.Event {
			case memberlist.NodeJoin:
				s.raftNode.AddVoter(raft.ServerID(node.Name), raft.ServerAddress(fmt.Sprintf("%s:%d", node.Addr.String(), s.params.RaftPort)), s.raftNode.AppliedIndex(), 10*time.Second).Error()
			case memberlist.NodeLeave:
				s.raftNode.RemoveServer(raft.ServerID(node.Name), 0, 10*time.Second).Error()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) runLeaderObserver(ctx context.Context) {
	for {
		select {
		case isLeader := <-s.raftNode.LeaderCh():
			if isLeader {
				s.services.StartAllServices(ctx)
				// This node is the leader
				s.l.Printf("I am a leader!")
				peers := s.memberlist.Members()
				for _, peer := range peers {
					if peer.Name == s.params.NodeID {
						continue
					}
					if err := s.raftNode.AddVoter(raft.ServerID(peer.Name), raft.ServerAddress(fmt.Sprintf("%s:%d", peer.Addr.String(), s.params.RaftPort)), s.raftNode.AppliedIndex(), 10*time.Second).Error(); err != nil {
						s.l.Warnf("Failed to add Raft member: %v", err)
					}
				}
			} else {
				s.l.Printf("I am not a leader!")
				s.services.StopRunningServices()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) AddLeaderService(leaderService LeaderService) {
	err := s.services.Add(leaderService)
	if err != nil {
		s.l.Errorf("couldn't add HA service: +%v", err)
	}
}

func (s *Service) BroadcastMessage(message []byte) {
	if s.params.Enabled {
		s.raftNode.Apply(message, 3*time.Second)
	} else {
		s.receivedMessages <- message
	}
}

func (s *Service) IsLeader() bool {
	return (s.raftNode != nil && s.raftNode.State() == raft.Leader) || !s.params.Enabled
}

func (s *Service) Bootstrap() bool {
	return s.params.Bootstrap || !s.params.Enabled
}

func (s *Service) RegisterGRPC(server *grpc.Server) {
	if s.params.Enabled {
		s.tm.Register(server)
	}
}
