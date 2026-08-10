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
	"io"
	"os"
	"path/filepath"
	"testing"

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
