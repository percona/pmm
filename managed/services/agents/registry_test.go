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

package agents

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
)

const testAgentID = "/agent_id/00000000-0000-4000-8000-000000000001"

// haServiceStub stands in for the HA service; nil params means HA is disabled, and nil members
// that every node is a member.
type haServiceStub struct {
	params  *models.HAParams
	members map[string]struct{}
}

func (s haServiceStub) IsMember(nodeID string) bool {
	if s.members == nil {
		return true
	}
	_, ok := s.members[nodeID]
	return ok
}

func (s haServiceStub) Params() *models.HAParams {
	if s.params == nil {
		return &models.HAParams{}
	}
	return s.params
}

func newTestConn() *pmmAgentInfo {
	return &pmmAgentInfo{
		id:              testAgentID,
		stateChangeChan: make(chan struct{}, 1),
		kickChan:        make(chan struct{}),
	}
}

func isKicked(conn *pmmAgentInfo) bool {
	select {
	case <-conn.kickChan:
		return true
	default:
		return false
	}
}

func newTestRegistry() *Registry {
	return &Registry{
		agents:    make(map[string]*pmmAgentInfo),
		roster:    newRoster(nil),
		haService: haServiceStub{},
		mDisconnects: prom.NewCounterVec(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "disconnects_total",
			Help:      "A total number of pmm-agent disconnects.",
		}, []string{"reason"}),
	}
}

func TestUnregister(t *testing.T) {
	t.Parallel()

	ctx := logger.SetEntry(t.Context(), logrus.WithField("test", t.Name()))

	t.Run("removes the current connection", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()
		current := &pmmAgentInfo{id: testAgentID}
		r.agents[testAgentID] = current

		assert.Same(t, current, r.unregister(ctx, testAgentID, "done", current))
		assert.Empty(t, r.agents)
	})

	t.Run("removes any connection when none is given", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()
		current := &pmmAgentInfo{id: testAgentID}
		r.agents[testAgentID] = current

		assert.Same(t, current, r.unregister(ctx, testAgentID, "kick", nil))
		assert.Empty(t, r.agents)
	})

	t.Run("keeps the connection that superseded a stale one", func(t *testing.T) {
		// A silently dropped connection can take minutes to die. By then the agent has
		// reconnected and registered again, and the stale handler must not evict it,
		// otherwise the agent stays connected while pmm-managed reports it as
		// disconnected forever. See PMM-15310.
		t.Parallel()

		r := newTestRegistry()
		stale := &pmmAgentInfo{id: testAgentID}
		current := &pmmAgentInfo{id: testAgentID}
		r.agents[testAgentID] = current

		assert.Nil(t, r.unregister(ctx, testAgentID, "done", stale))
		assert.Same(t, current, r.agents[testAgentID])
	})

	t.Run("does nothing for an unknown agent", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()

		assert.Nil(t, r.unregister(ctx, testAgentID, "done", &pmmAgentInfo{id: testAgentID}))
		assert.Empty(t, r.agents)
	})
}

// TestUnregisterPersistsDisconnectInHA guards against the connection status staying true after a
// disconnect: the handler unregisters with the context of the stream that just ended, which is
// already canceled.
func TestUnregisterPersistsDisconnectInHA(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	r := newTestRegistry()
	r.haService = haServiceStub{params: &models.HAParams{Enabled: true}}
	r.db = reform.NewDB(sqlDB, postgresql.Dialect, nil)
	r.connectionCache = map[string]string{testAgentID: "pmm-ha-0"}
	current := newTestConn()
	current.connectionID = "pmm-ha-0/1"
	r.agents[testAgentID] = current

	// Only the connection being unregistered: the agent may have connected again meanwhile.
	mock.ExpectExec(`UPDATE agents SET is_connected = false, updated_at = \$1 WHERE agent_id = \$2 AND connection_id = \$3`).
		WithArgs(sqlmock.AnyArg(), testAgentID, "pmm-ha-0/1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx, cancel := context.WithCancel(logger.SetEntry(t.Context(), logrus.WithField("test", t.Name())))
	cancel()

	assert.Same(t, current, r.unregister(ctx, testAgentID, "done", current))
	assert.Empty(t, r.connectionCache)
}

func TestKickConn(t *testing.T) {
	t.Parallel()

	ctx := logger.SetEntry(t.Context(), logrus.WithField("test", t.Name()))

	t.Run("kicks the connection it probed", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()
		current := newTestConn()
		r.agents[testAgentID] = current

		r.kickConn(ctx, current)

		assert.Empty(t, r.agents)
		assert.True(t, isKicked(current))
	})

	t.Run("leaves the connection that superseded the probed one", func(t *testing.T) {
		// Two registrations can probe the same stale connection concurrently. The one that
		// loses the race must not disconnect the connection that already replaced it,
		// otherwise it takes down a healthy agent. See PMM-15310.
		t.Parallel()

		r := newTestRegistry()
		stale := newTestConn()
		current := newTestConn()
		r.agents[testAgentID] = current

		r.kickConn(ctx, stale)

		assert.Same(t, current, r.agents[testAgentID])
		assert.False(t, isKicked(current))
		assert.False(t, isKicked(stale))
	})

	t.Run("concurrent kicks of the same ID hit only the registered connection", func(t *testing.T) {
		t.Parallel()

		const conns = 8

		r := newTestRegistry()
		probed := make([]*pmmAgentInfo, conns)
		for i := range probed {
			probed[i] = newTestConn()
		}
		r.agents[testAgentID] = probed[0]

		var wg sync.WaitGroup
		for _, conn := range probed {
			wg.Go(func() {
				r.kickConn(ctx, conn)
			})
		}
		wg.Wait()

		assert.Empty(t, r.agents)
		assert.True(t, isKicked(probed[0]))
		for _, conn := range probed[1:] {
			assert.False(t, isKicked(conn))
		}
	})
}

// newHATestRegistry returns an HA-mode registry with PMM Server's pmm-agent connected, as persisted
// in the database, and that connection.
func newHATestRegistry(t *testing.T) (*Registry, *reform.DB, *pmmAgentInfo) {
	t.Helper()

	const connectionID = "connection-1"

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	agent, err := models.FindAgentByID(db.Querier, models.PMMServerAgentID)
	require.NoError(t, err)
	agent.IsConnected = true
	agent.ConnectionID = new(connectionID)
	require.NoError(t, db.Update(agent))

	r := newTestRegistry()
	r.db = db
	r.haService = haServiceStub{params: &models.HAParams{Enabled: true}}
	r.connectionCache = map[string]string{models.PMMServerAgentID: ""}
	conn := &pmmAgentInfo{id: models.PMMServerAgentID, connectionID: connectionID}
	r.agents[models.PMMServerAgentID] = conn

	return r, db, conn
}

func isConnectedInDB(t *testing.T, db *reform.DB) bool {
	t.Helper()

	agent, err := models.FindAgentByID(db.Querier, models.PMMServerAgentID)
	require.NoError(t, err)

	return agent.IsConnected
}

// TestUnregisterDoesNotHoldTheRegistryWhilePersisting covers a slow database: the disconnect of one
// agent must not stop the registry from serving every other agent while it is written.
func TestUnregisterDoesNotHoldTheRegistryWhilePersisting(t *testing.T) {
	r, db, conn := newHATestRegistry(t)
	ctx := logger.SetEntry(t.Context(), logrus.WithField("test", t.Name()))

	// Hold the row, so that persisting the disconnect waits for it.
	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec("SELECT 1 FROM agents WHERE agent_id = $1 FOR UPDATE", models.PMMServerAgentID)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.unregister(ctx, models.PMMServerAgentID, "done", conn)
	}()

	require.Eventually(t, func() bool {
		var waiting int
		err := db.QueryRow("SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND query LIKE 'UPDATE%agents%'").
			Scan(&waiting)
		return err == nil && waiting > 0
	}, 5*time.Second, 10*time.Millisecond, "the disconnect is not waiting for the row")

	locked := r.rw.TryLock()
	if locked {
		r.rw.Unlock()
	}
	assert.True(t, locked, "the registry is locked while the disconnect is persisted")

	require.NoError(t, tx.Rollback())
	<-done
	assert.False(t, isConnectedInDB(t, db))
}

// TestUnregisterKeepsANewerConnection covers an agent which connects again, to this PMM Server or
// another one, before the disconnect of its previous connection is written: the newer connection is
// persisted by then, and the late disconnect must not overwrite it.
func TestUnregisterKeepsANewerConnection(t *testing.T) {
	r, db, conn := newHATestRegistry(t)
	ctx := logger.SetEntry(t.Context(), logrus.WithField("test", t.Name()))

	agent, err := models.FindAgentByID(db.Querier, models.PMMServerAgentID)
	require.NoError(t, err)
	agent.ConnectionID = new("connection-2")
	require.NoError(t, db.Update(agent))

	assert.Same(t, conn, r.unregister(ctx, models.PMMServerAgentID, "done", conn))

	assert.True(t, isConnectedInDB(t, db))
	assert.True(t, r.IsConnected(models.PMMServerAgentID))
}

// TestIsConnectedIgnoresConnectionsOfLostReplicas covers a replica that is lost with its Kubernetes
// node, or scaled away after crashing: it never persists the disconnects of its agents, so their
// connections count only while it is a member of the cluster.
func TestIsConnectedIgnoresConnectionsOfLostReplicas(t *testing.T) {
	r, db, _ := newHATestRegistry(t)
	r.haService = haServiceStub{params: &models.HAParams{Enabled: true}, members: map[string]struct{}{"pmm-ha-0": {}}}

	for connectionID, expected := range map[string]bool{
		"pmm-ha-0/1": true,
		"pmm-ha-2/1": false,
		// An ID written by an earlier version names no owner to check.
		"connection-1": true,
	} {
		agent, err := models.FindAgentByID(db.Querier, models.PMMServerAgentID)
		require.NoError(t, err)
		agent.ConnectionID = new(connectionID)
		require.NoError(t, db.Update(agent))

		r.connectionCacheTTL = time.Time{}
		assert.Equal(t, expected, r.IsConnected(models.PMMServerAgentID), connectionID)
	}
}

// TestIsConnectedKeepsStatusesOnDatabaseError covers a failed refresh of the connection statuses:
// reporting every agent as disconnected would let the Nodes protected while it is connected go.
func TestIsConnectedKeepsStatusesOnDatabaseError(t *testing.T) {
	r, db, _ := newHATestRegistry(t)
	r.connectionCacheTTL = time.Time{}
	require.True(t, r.IsConnected(models.PMMServerAgentID))

	// A closed pool fails every query without touching the test database.
	closedDB, err := sql.Open("postgres", "host=127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, closedDB.Close())
	r.db = reform.NewDB(closedDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r.connectionCacheTTL = time.Time{}

	assert.True(t, r.IsConnected(models.PMMServerAgentID))
	assert.True(t, isConnectedInDB(t, db))

	// Every failed refresh moves the retry, so an unchanged one means the database was not queried:
	// a lookup missing from the cache is answered from it until the retry is due.
	retryAt := r.connectionCacheRetryAt
	require.False(t, retryAt.IsZero())
	assert.False(t, r.IsConnected("/agent_id/missing"))
	assert.Equal(t, retryAt, r.connectionCacheRetryAt)

	r.connectionCacheRetryAt = time.Now().Add(-time.Second)
	assert.False(t, r.IsConnected("/agent_id/missing"))
	assert.True(t, r.connectionCacheRetryAt.After(retryAt), "the refresh is not retried once due")
}

// TestGetNamesTheReplicaHoldingTheConnection covers a request for a pmm-agent connected to another
// replica: it can't be served, and the error must say why (PMM-15684).
func TestGetNamesTheReplicaHoldingTheConnection(t *testing.T) {
	t.Parallel()

	const notConnected = "rpc error: code = FailedPrecondition desc = pmm-agent with ID %s is not currently connected"

	for _, tc := range []struct {
		name      string
		haEnabled bool
		agentID   string
		expected  string
	}{
		{
			name: "connected to another replica", haEnabled: true, agentID: "/agent_id/on-pmm-ha-0",
			expected: "rpc error: code = FailedPrecondition desc = pmm-agent with ID /agent_id/on-pmm-ha-0 is connected to " +
				"PMM Server replica pmm-ha-0, not to pmm-ha-1; in HA mode, actions run only on pmm-agents connected to the leader",
		},
		{
			name: "connection ID names no replica", haEnabled: true, agentID: "/agent_id/no-owner",
			expected: "rpc error: code = FailedPrecondition desc = pmm-agent with ID /agent_id/no-owner is connected to " +
				"another PMM Server replica, not to pmm-ha-1; in HA mode, actions run only on pmm-agents connected to the leader",
		},
		// Persisted as connected to this replica, which no longer holds it: see PMM-15669.
		{name: "stale connection of this replica", haEnabled: true, agentID: "/agent_id/on-pmm-ha-1"},
		{name: "not connected", haEnabled: true, agentID: "/agent_id/missing"},
		{name: "HA disabled", agentID: "/agent_id/on-pmm-ha-0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newTestRegistry()
			r.haService = haServiceStub{params: &models.HAParams{Enabled: tc.haEnabled, NodeID: "pmm-ha-1"}}
			// A fresh cache, and no database: a miss must not query it.
			r.connectionCache = map[string]string{
				"/agent_id/on-pmm-ha-0": "pmm-ha-0",
				"/agent_id/no-owner":    "",
				"/agent_id/on-pmm-ha-1": "pmm-ha-1",
			}
			r.connectionCacheTTL = time.Now().Add(time.Hour)

			expected := tc.expected
			if expected == "" {
				expected = fmt.Sprintf(notConnected, tc.agentID)
			}
			_, err := r.get(tc.agentID)
			assert.EqualError(t, err, expected)
		})
	}

	t.Run("connected to this replica", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()
		r.haService = haServiceStub{params: &models.HAParams{Enabled: true, NodeID: "pmm-ha-1"}}
		conn := newTestConn()
		r.agents[testAgentID] = conn

		actual, err := r.get(testAgentID)
		require.NoError(t, err)
		assert.Same(t, conn, actual)
	})
}

// TestGetReadsTheReplicaFromTheDatabase covers the owner coming from the persisted connection ID.
func TestGetReadsTheReplicaFromTheDatabase(t *testing.T) {
	r, db, _ := newHATestRegistry(t)
	r.haService = haServiceStub{params: &models.HAParams{Enabled: true, NodeID: "pmm-ha-1"}}
	delete(r.agents, models.PMMServerAgentID)

	agent, err := models.FindAgentByID(db.Querier, models.PMMServerAgentID)
	require.NoError(t, err)
	agent.ConnectionID = new("pmm-ha-0/1")
	require.NoError(t, db.Update(agent))
	r.connectionCacheTTL = time.Time{}

	_, err = r.get(models.PMMServerAgentID)
	assert.ErrorContains(t, err, "is connected to PMM Server replica pmm-ha-0, not to pmm-ha-1")
}
