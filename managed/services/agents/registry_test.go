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
	"database/sql/driver"
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
	"github.com/percona/pmm/utils/logger"
)

const testAgentID = "/agent_id/00000000-0000-4000-8000-000000000001"

// haServiceStub stands in for the HA service; nil params means HA is disabled.
type haServiceStub struct {
	params *models.HAParams
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

type haEnabledStub struct{}

func pmmAgentRow(agentID string, connected bool) []driver.Value {
	columns := models.AgentTable.Columns()
	values := make([]driver.Value, len(columns))
	for i, c := range columns {
		switch c {
		case "agent_id":
			values[i] = agentID
		case "agent_type":
			values[i] = string(models.PMMAgentType)
		case "created_at", "updated_at":
			values[i] = time.Now()
		case "disabled", "tls", "tls_skip_verify":
			values[i] = false
		case "is_connected":
			values[i] = connected
		case "status":
			values[i] = ""
		}
	}
	return values
}

func (haEnabledStub) Params() *models.HAParams { return &models.HAParams{Enabled: true} }

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
	r.haService = haEnabledStub{}
	r.db = reform.NewDB(sqlDB, postgresql.Dialect, nil)
	r.connectionCache = map[string]struct{}{testAgentID: {}}
	current := newTestConn()
	r.agents[testAgentID] = current

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .+ FROM "agents"`).
		WithArgs(testAgentID).
		WillReturnRows(sqlmock.NewRows(models.AgentTable.Columns()).AddRow(pmmAgentRow(testAgentID, true)...))
	mock.ExpectExec(`UPDATE "agents" SET "updated_at" = \$1, "is_connected" = \$2 WHERE "agent_id" = \$3`).
		WithArgs(sqlmock.AnyArg(), false, testAgentID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ctx, cancel := context.WithCancel(logger.SetEntry(t.Context(), logrus.WithField("test", t.Name())))
	cancel()

	assert.Same(t, current, r.unregister(ctx, testAgentID, "done", current))
	assert.Empty(t, r.connectionCache)
}

func TestIsConnectedInHA(t *testing.T) {
	t.Parallel()

	const (
		disconnectedID = "/agent_id/00000000-0000-4000-8000-000000000002"
		exporterID     = "/agent_id/00000000-0000-4000-8000-000000000003"
	)

	newRegistry := func(t *testing.T) (*Registry, sqlmock.Sqlmock) {
		t.Helper()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, mock.ExpectationsWereMet())
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		r := newTestRegistry()
		r.haService = haEnabledStub{}
		r.db = reform.NewDB(sqlDB, postgresql.Dialect, nil)
		r.connectionCache = make(map[string]struct{})
		return r, mock
	}

	t.Run("fresh cache answers misses without a query", func(t *testing.T) {
		t.Parallel()

		r, _ := newRegistry(t)
		r.connectionCache[testAgentID] = struct{}{}
		r.connectionCacheTTL = time.Now().Add(time.Minute)

		assert.False(t, r.IsConnected(exporterID))
		// An unexpected rebuild fails on sqlmock and empties the cache.
		assert.True(t, r.IsConnected(testAgentID))
	})

	t.Run("expired cache is rebuilt once for concurrent callers", func(t *testing.T) {
		t.Parallel()

		r, mock := newRegistry(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE agent_type = \$1`).
			WithArgs(string(models.PMMAgentType)).
			WillDelayFor(100 * time.Millisecond).
			WillReturnRows(sqlmock.NewRows(models.AgentTable.Columns()).
				AddRow(pmmAgentRow(testAgentID, true)...).
				AddRow(pmmAgentRow(disconnectedID, false)...))
		mock.ExpectCommit()

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				assert.True(t, r.IsConnected(testAgentID))
			})
		}
		wg.Wait()

		assert.False(t, r.IsConnected(disconnectedID))
		assert.False(t, r.IsConnected(exporterID))
		assert.True(t, r.IsConnected(testAgentID))
	})
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
