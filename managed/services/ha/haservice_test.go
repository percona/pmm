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
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		name       string
		voters     int
		members    int
		wantVoters int
	}{
		{name: "keeps a lost node in a three-voter cluster", voters: 3, members: 2, wantVoters: 3},
		{name: "removes nodes that left after a scale-down from five to three", voters: 5, members: 3, wantVoters: 3},
		{name: "never shrinks below three voters", voters: 5, members: 1, wantVoters: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			leader := newTestRaftCluster(t, tc.voters)
			_, leaderID := leader.LeaderWithID()
			ml := newTestMemberlist(t, string(leaderID))
			for _, server := range raftServers(t, leader) {
				if ml.NumMembers() == tc.members {
					break
				}
				if server.ID != leaderID {
					newTestMemberlist(t, string(server.ID), ml.LocalNode().Address())
				}
			}
			require.Eventually(t, func() bool { return ml.NumMembers() == tc.members }, 5*time.Second, 10*time.Millisecond)

			s := &Service{
				params:     &models.HAParams{Enabled: true, NodeID: string(leaderID)},
				l:          logrus.WithField("component", "ha-test"),
				raftNode:   leader,
				memberlist: ml,
			}
			s.reconcileRaftWithMemberlist(t.Context())

			servers := raftServers(t, leader)
			voters := make([]string, 0, len(servers))
			for _, server := range servers {
				voters = append(voters, string(server.ID))
			}
			assert.Len(t, voters, tc.wantVoters)
			for _, member := range ml.Members() {
				assert.Contains(t, voters, member.Name)
			}
		})
	}
}

func TestService_RejoinMemberlist(t *testing.T) {
	t.Parallel()

	local := newTestMemberlist(t, "pmm-ha-0")
	peer := newTestMemberlist(t, "pmm-ha-1")
	require.Equal(t, 1, local.NumMembers())

	s := &Service{
		params: &models.HAParams{
			Enabled: true,
			NodeID:  "pmm-ha-0",
			Nodes:   []string{local.LocalNode().Address(), peer.LocalNode().Address(), "127.0.0.1:1"},
		},
		l:          logrus.WithField("component", "ha-test"),
		memberlist: local,
	}
	s.rejoinMemberlist(t.Context())

	assert.Equal(t, 2, local.NumMembers())
	assert.Eventually(t, func() bool { return peer.NumMembers() == 2 }, 5*time.Second, 10*time.Millisecond)
}

// newTestRaftCluster starts n in-memory Raft voters named pmm-ha-<i> and returns the elected leader.
func newTestRaftCluster(t *testing.T, n int) *raft.Raft {
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
		cfg.HeartbeatTimeout = 50 * time.Millisecond
		cfg.ElectionTimeout = 50 * time.Millisecond
		cfg.LeaderLeaseTimeout = 50 * time.Millisecond
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
	require.Eventually(t, func() bool {
		for _, r := range nodes {
			if r.State() == raft.Leader {
				leader = r
				return true
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)

	return leader
}

// newTestMemberlist starts a loopback memberlist node with the given name, joined to join if set.
func newTestMemberlist(t *testing.T, name string, join ...string) *memberlist.Memberlist {
	t.Helper()

	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = name
	cfg.BindAddr = "127.0.0.1"
	cfg.BindPort = 0
	cfg.LogOutput = io.Discard

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
