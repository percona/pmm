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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestService_Apply(t *testing.T) {
	t.Parallel()

	s := &Service{
		l: logrus.WithField("component", "test"),
	}

	logEntry := &raft.Log{
		Index: 42,
		Data:  []byte("test data"),
	}

	result := s.Apply(logEntry)
	assert.Nil(t, result)
}

func TestService_Snapshot(t *testing.T) {
	t.Parallel()

	s := &Service{
		l: logrus.WithField("component", "test"),
	}

	snapshot, err := s.Snapshot()

	require.NoError(t, err)
	require.NotNil(t, snapshot)

	_, ok := snapshot.(*fsmSnapshot)
	assert.True(t, ok, "snapshot should be of type *fsmSnapshot")
}

func TestService_Restore(t *testing.T) {
	t.Parallel()

	t.Run("closes reader successfully", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			l: logrus.WithField("component", "test"),
		}

		data := []byte("test restore data")
		rc := io.NopCloser(bytes.NewReader(data))

		err := s.Restore(rc)

		require.NoError(t, err)
	})

	t.Run("handles empty reader", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			l: logrus.WithField("component", "test"),
		}

		rc := io.NopCloser(bytes.NewReader(nil))

		err := s.Restore(rc)

		require.NoError(t, err)
	})
}

func TestFSMSnapshot_Persist(t *testing.T) {
	t.Parallel()

	snapshot := &fsmSnapshot{}

	mockSink := &mockSnapshotSink{
		closed: false,
	}

	err := snapshot.Persist(mockSink)

	require.NoError(t, err)
	assert.True(t, mockSink.closed)
}

func TestFSMSnapshot_Release(t *testing.T) {
	t.Parallel()

	snapshot := &fsmSnapshot{}

	assert.NotPanics(t, func() {
		snapshot.Release()
	})
}

func TestSetupRaftStorage(t *testing.T) {
	t.Run("creates directory structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		raftDir := filepath.Join(tmpDir, "test-node-1")

		require.NoError(t, os.MkdirAll(raftDir, defaultRaftDataDirPerm))

		l := logrus.WithField("component", "test")

		logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
		require.NoError(t, err)
		require.NotNil(t, logStore)
		defer logStore.Close()

		stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
		require.NoError(t, err)
		require.NotNil(t, stableStore)
		defer stableStore.Close()

		snapshotStore, err := raft.NewFileSnapshotStore(raftDir, defaultSnapshotRetention, l.Logger.Out)
		require.NoError(t, err)
		require.NotNil(t, snapshotStore)

		assert.DirExists(t, raftDir)
		assert.FileExists(t, filepath.Join(raftDir, "raft-log.db"))
		assert.FileExists(t, filepath.Join(raftDir, "raft-stable.db"))
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	params := &models.HAParams{
		Enabled:          true,
		NodeID:           "node-1",
		AdvertiseAddress: "127.0.0.1",
		RaftPort:         7300,
		GossipPort:       7301,
		Nodes:            []string{"node-2"},
	}

	service := New(params)

	require.NotNil(t, service)
	assert.Equal(t, params, service.params)
	assert.True(t, service.bootstrapCluster)
	assert.NotNil(t, service.services)
	assert.NotNil(t, service.nodeCh)
	assert.NotNil(t, service.leaderCh)
	assert.NotNil(t, service.l)
	assert.NotNil(t, service.wg)

	assert.Equal(t, defaultNodeEventChanSize, cap(service.nodeCh))
}

func TestService_IsLeader(t *testing.T) {
	t.Parallel()

	t.Run("returns true when HA disabled", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			params: &models.HAParams{
				Enabled: false,
			},
		}

		assert.True(t, s.IsLeader())
	})

	t.Run("returns false when raftNode is nil", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			params: &models.HAParams{
				Enabled: true,
			},
			raftNode: nil,
		}

		assert.False(t, s.IsLeader())
	})
}

func TestService_Params(t *testing.T) {
	t.Parallel()

	params := &models.HAParams{
		Enabled:          true,
		NodeID:           "test-node",
		AdvertiseAddress: "192.168.1.1",
		RaftPort:         7300,
		GossipPort:       7301,
	}

	s := &Service{
		params: params,
	}

	result := s.Params()

	assert.Equal(t, params, result)
	assert.Equal(t, "test-node", result.NodeID)
	assert.Equal(t, "192.168.1.1", result.AdvertiseAddress)
	assert.Equal(t, 7300, result.RaftPort)
	assert.Equal(t, 7301, result.GossipPort)
}

func TestService_AddLeaderService(t *testing.T) {
	t.Parallel()

	t.Run("successfully adds service", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			services: newServices(),
			l:        logrus.WithField("component", "test"),
		}

		svc := &mockLeaderService{id: "test-service"}
		s.AddLeaderService(svc)

		assert.Len(t, s.services.all, 1)
		assert.Equal(t, svc, s.services.all["test-service"])
	})

	t.Run("logs error when add fails", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			services: newServices(),
			l:        logrus.WithField("component", "test"),
		}

		svc := &mockLeaderService{id: "duplicate"}
		s.AddLeaderService(svc)

		assert.NotPanics(t, func() {
			s.AddLeaderService(svc)
		})
	})
}

func TestService_BroadcastMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns error when HA disabled", func(t *testing.T) {
		t.Parallel()

		s := &Service{
			params: &models.HAParams{
				Enabled: false,
			},
		}

		err := s.BroadcastMessage([]byte("test message"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "HA is disabled")
	})
}

func TestService_ReconcileRaftWithMemberlist(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		voters      int
		members     int
		left        int
		unreachable bool
		refused     bool
		wantVoters  int
	}{
		{name: "keeps a failed node in a three-voter cluster", voters: 3, members: 2, wantVoters: 3},
		{name: "keeps failed nodes in a five-voter cluster", voters: 5, members: 3, wantVoters: 5},
		{name: "removes a node that left a three-voter cluster", voters: 3, members: 2, left: 1, wantVoters: 2},
		{name: "removes nodes that left after a scale-down from five to three", voters: 5, members: 3, left: 2, wantVoters: 3},
		{name: "keeps a node that left while another voter is down", voters: 3, members: 1, left: 1, wantVoters: 3},
		{name: "keeps a node that left while another voter is unreachable over Raft", voters: 3, members: 2, left: 1, unreachable: true, wantVoters: 3},
		{name: "keeps a node that left while another voter refuses Raft connections", voters: 3, members: 2, left: 1, refused: true, wantVoters: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			leader, leaderID, _ := newTestRaftCluster(t, tc.voters)
			ml := newTestMemberlist(t, leaderID)
			s := newTestService(leaderID, leader, ml)
			members, left := 1, 0
			for _, server := range raftServers(t, leader) {
				switch {
				case string(server.ID) == leaderID:
				case members < tc.members:
					newTestMemberlist(t, string(server.ID), ml.LocalNode().Address())
					members++
					if tc.unreachable {
						s.setUnreachable(server.ID, true)
					}
					if tc.refused {
						refused := server.Address
						s.raftReachable = func(_ context.Context, address raft.ServerAddress) bool { return address != refused }
					}
				case left < tc.left:
					s.setLeft(string(server.ID), true)
					left++
				}
			}
			require.Eventually(t, func() bool { return ml.NumMembers() == tc.members }, 5*time.Second, 10*time.Millisecond)

			s.reconcileRaftWithMemberlist(t.Context())

			voters := raftServerIDs(t, leader)
			assert.Len(t, voters, tc.wantVoters)
			for _, member := range ml.Members() {
				assert.Contains(t, voters, member.Name)
			}
		})
	}
}

func TestService_HandleNodeEvent(t *testing.T) {
	t.Parallel()

	t.Run("every node records graceful leaves only", func(t *testing.T) {
		t.Parallel()

		s := newTestService("pmm-ha-0", nil, nil)
		s.handleNodeEvent(t.Context(), memberlist.NodeEvent{Event: memberlist.NodeLeave, Node: &memberlist.Node{Name: "pmm-ha-1"}})
		assert.False(t, s.hasLeft("pmm-ha-1"))

		leaving := &memberlist.Node{Name: "pmm-ha-2", Meta: []byte(memberlistMetaLeaving)}
		s.handleNodeEvent(t.Context(), memberlist.NodeEvent{Event: memberlist.NodeLeave, Node: leaving})
		assert.True(t, s.hasLeft("pmm-ha-2"))

		s.handleNodeEvent(t.Context(), memberlist.NodeEvent{Event: memberlist.NodeJoin, Node: &memberlist.Node{Name: "pmm-ha-2"}})
		assert.False(t, s.hasLeft("pmm-ha-2"))
	})

	t.Run("the leader removes a node that left", func(t *testing.T) {
		t.Parallel()

		leader, leaderID, _ := newTestRaftCluster(t, 3)
		ml := newTestMemberlist(t, leaderID)
		s := newTestService(leaderID, leader, ml)
		var followers []string
		for _, id := range raftServerIDs(t, leader) {
			if id != leaderID {
				followers = append(followers, id)
			}
		}
		newTestMemberlist(t, followers[0], ml.LocalNode().Address())
		require.Eventually(t, func() bool { return ml.NumMembers() == 2 }, 5*time.Second, 10*time.Millisecond)

		s.handleNodeEvent(t.Context(), memberlist.NodeEvent{Event: memberlist.NodeLeave, Node: &memberlist.Node{Name: followers[1]}})
		assert.Len(t, raftServerIDs(t, leader), 3, "a failed node must stay a voter")

		leaving := &memberlist.Node{Name: followers[1], Meta: []byte(memberlistMetaLeaving)}
		s.handleNodeEvent(t.Context(), memberlist.NodeEvent{Event: memberlist.NodeLeave, Node: leaving})
		assert.ElementsMatch(t, []string{leaderID, followers[0]}, raftServerIDs(t, leader), "a node that left must be removed")
	})
}

func TestMemberlistDelegate(t *testing.T) {
	t.Parallel()

	events := make(chan memberlist.NodeEvent, 16)
	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = "observer"
	cfg.BindAddr = "127.0.0.1"
	cfg.BindPort = 0
	cfg.LogOutput = io.Discard
	cfg.Events = &memberlist.ChannelEventDelegate{Ch: events}
	cfg.SuspicionMaxTimeoutMult = 1
	observer, err := memberlist.Create(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, observer.Shutdown())
	})

	leaveMeta := func(name string, graceful bool) []byte {
		s := New(&models.HAParams{Enabled: true, NodeID: name})
		cfg := memberlist.DefaultLocalConfig()
		cfg.Name = name
		cfg.BindAddr = "127.0.0.1"
		cfg.BindPort = 0
		cfg.LogOutput = io.Discard
		cfg.Delegate = s.delegate
		m, err := memberlist.Create(cfg)
		require.NoError(t, err)
		s.memberlist = m
		_, err = m.Join([]string{observer.LocalNode().Address()})
		require.NoError(t, err)

		if graceful {
			s.leaveMemberlist()
		} else {
			require.NoError(t, m.Shutdown())
		}

		for {
			select {
			case event := <-events:
				if event.Event == memberlist.NodeLeave && event.Node.Name == name {
					return event.Node.Meta
				}
			case <-time.After(30 * time.Second):
				require.FailNow(t, "no leave event", name)
			}
		}
	}

	assert.Equal(t, []byte(memberlistMetaLeaving), leaveMeta("graceful", true))
	assert.Empty(t, leaveMeta("failed", false))

	alone := New(&models.HAParams{Enabled: true, NodeID: "alone"})
	alone.memberlist = newTestMemberlist(t, "alone")
	start := time.Now()
	alone.leaveMemberlist()
	assert.Less(t, time.Since(start), defaultLeaveAnnounceTimeout, "a node without peers has no one to announce leaving to")
}

func TestService_HeartbeatObserver(t *testing.T) {
	t.Parallel()

	leader, leaderID, isolate := newTestRaftCluster(t, 3)
	s := newTestService(leaderID, leader, nil)
	s.startHeartbeatObserver(t.Context())
	var follower raft.ServerID
	for _, id := range raftServerIDs(t, leader) {
		if id != leaderID {
			follower = raft.ServerID(id)
			break
		}
	}

	heal := isolate(string(follower))
	assert.Eventually(t, func() bool { return s.isUnreachable(follower) }, 10*time.Second, 10*time.Millisecond)

	heal()
	assert.Eventually(t, func() bool { return !s.isUnreachable(follower) }, 10*time.Second, 10*time.Millisecond)
}

func TestAliveQuorumWithout(t *testing.T) {
	t.Parallel()

	voters := func(ids ...string) []raft.Server {
		servers := make([]raft.Server, 0, len(ids))
		for _, id := range ids {
			servers = append(servers, raft.Server{Suffrage: raft.Voter, ID: raft.ServerID(id)})
		}
		return servers
	}
	alive := func(ids ...string) map[string]struct{} {
		m := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			m[id] = struct{}{}
		}
		return m
	}

	for _, tc := range []struct {
		name    string
		servers []raft.Server
		alive   map[string]struct{}
		want    bool
	}{
		{name: "two voters, the other one leaves", servers: voters("l", "x"), alive: alive("l"), want: true},
		{name: "three voters, the third is alive", servers: voters("l", "a", "x"), alive: alive("l", "a"), want: true},
		{name: "three voters, the third is down", servers: voters("l", "a", "x"), alive: alive("l"), want: false},
		{name: "five voters, three alive", servers: voters("l", "a", "b", "c", "x"), alive: alive("l", "a", "b"), want: true},
		{name: "five voters, two alive", servers: voters("l", "a", "b", "c", "x"), alive: alive("l", "a"), want: false},
		{
			name:    "non-voters don't count",
			servers: append(voters("l", "a", "x"), raft.Server{Suffrage: raft.Nonvoter, ID: "n"}),
			alive:   alive("l", "a"),
			want:    true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, aliveQuorumWithout(tc.servers, tc.alive, "x"))
		})
	}
}

func TestService_RejoinMemberlist(t *testing.T) {
	t.Parallel()

	local := newTestMemberlist(t, "pmm-ha-0")
	peer := newTestMemberlist(t, "pmm-ha-1")
	require.Equal(t, 1, local.NumMembers())

	s := newTestService("pmm-ha-0", nil, local)
	s.params.Nodes = []string{local.LocalNode().Address(), peer.LocalNode().Address(), "127.0.0.1:1"}
	s.rejoinMemberlist(t.Context())

	assert.Equal(t, 2, local.NumMembers())
	assert.Eventually(t, func() bool { return peer.NumMembers() == 2 }, 5*time.Second, 10*time.Millisecond)
}

func newTestService(nodeID string, r *raft.Raft, m *memberlist.Memberlist) *Service {
	s := New(&models.HAParams{Enabled: true, NodeID: nodeID})
	s.raftNode = r
	s.memberlist = m
	// In-memory Raft addresses can't be dialed.
	s.raftReachable = func(context.Context, raft.ServerAddress) bool { return true }
	return s
}

func TestService_HasOtherVoters(t *testing.T) {
	t.Parallel()

	alone, aloneID, _ := newTestRaftCluster(t, 1)
	assert.False(t, newTestService(aloneID, alone, nil).hasOtherVoters())

	leader, leaderID, _ := newTestRaftCluster(t, 3)
	assert.True(t, newTestService(leaderID, leader, nil).hasOtherVoters())
}

func TestBootstrapsCluster(t *testing.T) {
	t.Parallel()

	peers := []string{
		"pmm-ha-0.monitoring-service.pmm.svc.cluster.local",
		"pmm-ha-1.monitoring-service.pmm.svc.cluster.local",
		"pmm-ha-2.monitoring-service.pmm.svc.cluster.local",
	}
	for _, tc := range []struct {
		name   string
		nodeID string
		nodes  []string
		want   bool
	}{
		{name: "no peers", nodeID: "pmm-ha-0", want: true},
		{name: "first peer", nodeID: "pmm-ha-0", nodes: peers, want: true},
		{name: "other peer", nodeID: "pmm-ha-2", nodes: peers, want: false},
		{name: "peer with a port", nodeID: "pmm-ha-1", nodes: []string{"pmm-ha-0.svc:9096", "pmm-ha-1.svc:9096"}, want: false},
		{name: "IP peers", nodeID: "pmm-ha-1", nodes: []string{"10.0.0.1", "10.0.0.2"}, want: true},
		{name: "node ID not among peers", nodeID: "node-2", nodes: []string{"pmm-a.example.com", "pmm-b.example.com"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, bootstrapsCluster(&models.HAParams{NodeID: tc.nodeID, Nodes: tc.nodes}))
		})
	}
}

func TestRaftReachable(t *testing.T) {
	t.Parallel()

	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	assert.True(t, raftReachable(t.Context(), raft.ServerAddress(address)))

	require.NoError(t, listener.Close())
	assert.False(t, raftReachable(t.Context(), raft.ServerAddress(address)))
}

// newTestRaftCluster starts n in-memory Raft voters named pmm-ha-<i> and returns the leader once it has committed,
// with a function that cuts a node off from the others and returns the function that reconnects it.
func newTestRaftCluster(t *testing.T, n int) (*raft.Raft, string, func(id string) func()) {
	t.Helper()

	addrs := make([]raft.ServerAddress, n)
	transports := make([]*raft.InmemTransport, n)
	servers := make([]raft.Server, n)
	for i := range n {
		addrs[i], transports[i] = raft.NewInmemTransport("")
		servers[i] = raft.Server{Suffrage: raft.Voter, ID: raft.ServerID(fmt.Sprintf("pmm-ha-%d", i)), Address: addrs[i]}
	}
	for i, from := range transports {
		for j, to := range transports {
			if i != j {
				from.Connect(addrs[j], to)
			}
		}
	}

	nodes := make([]*raft.Raft, n)
	for i := range n {
		cfg := raft.DefaultConfig()
		cfg.LocalID = servers[i].ID
		cfg.LogOutput = io.Discard
		cfg.HeartbeatTimeout = 500 * time.Millisecond
		cfg.ElectionTimeout = 500 * time.Millisecond
		cfg.LeaderLeaseTimeout = 500 * time.Millisecond
		cfg.CommitTimeout = 5 * time.Millisecond

		store := raft.NewInmemStore()
		snapshots := raft.NewInmemSnapshotStore()
		err := raft.BootstrapCluster(cfg, store, store, snapshots, transports[i], raft.Configuration{Servers: servers})
		require.NoError(t, err)

		r, err := raft.NewRaft(cfg, &Service{l: logrus.WithField("component", "ha-test")}, store, store, snapshots, transports[i])
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, r.Shutdown().Error())
		})
		nodes[i] = r
	}

	var leader *raft.Raft
	var leaderID string
	require.Eventually(t, func() bool {
		for i, r := range nodes {
			if r.State() == raft.Leader && r.Barrier(time.Second).Error() == nil {
				leader, leaderID = r, string(servers[i].ID)
				return true
			}
		}
		return false
	}, 10*time.Second, 10*time.Millisecond)

	isolate := func(id string) func() {
		i := slices.IndexFunc(servers, func(server raft.Server) bool { return string(server.ID) == id })
		transports[i].DisconnectAll()
		for j := range n {
			if j != i {
				transports[j].Disconnect(addrs[i])
			}
		}
		return func() {
			for j := range n {
				if j != i {
					transports[i].Connect(addrs[j], transports[j])
					transports[j].Connect(addrs[i], transports[i])
				}
			}
		}
	}

	return leader, leaderID, isolate
}

// newTestMemberlist starts a loopback memberlist node with the given name, joined to join if set.
func newTestMemberlist(t *testing.T, name string, join ...string) *memberlist.Memberlist {
	t.Helper()

	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = name
	cfg.BindAddr = "127.0.0.1"
	cfg.BindPort = 0
	cfg.LogOutput = io.Discard
	// Failure detection isn't under test here, and on a loaded host it marks live peers dead.
	cfg.ProbeInterval = time.Minute

	m, err := memberlist.Create(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, m.Shutdown())
	})

	if len(join) != 0 {
		_, err = m.Join(join)
		require.NoError(t, err)
	}

	return m
}

func raftServerIDs(t *testing.T, r *raft.Raft) []string {
	t.Helper()

	servers := raftServers(t, r)
	ids := make([]string, 0, len(servers))
	for _, server := range servers {
		ids = append(ids, string(server.ID))
	}
	return ids
}

func raftServers(t *testing.T, r *raft.Raft) []raft.Server {
	t.Helper()

	future := r.GetConfiguration()
	require.NoError(t, future.Error())
	return future.Configuration().Servers
}

type mockSnapshotSink struct {
	closed bool
}

func (m *mockSnapshotSink) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockSnapshotSink) Close() error {
	m.closed = true
	return nil
}

func (m *mockSnapshotSink) ID() string {
	return "mock-snapshot"
}

func (m *mockSnapshotSink) Cancel() error {
	return nil
}
