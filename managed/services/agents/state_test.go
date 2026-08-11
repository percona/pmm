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
	"sync"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

type testLimiter struct{}

func (testLimiter) TryAcquire() bool {
	return true
}

func (testLimiter) Release() {}

func TestRequestStateUpdateQueuesUpdateForConnectedAgent(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	r.agentsCache.Store("agent-1", pmmAgentInfo{id: "agent-1", stateChangeChan: make(chan struct{}, 1)})
	u := NewStateUpdater(nil, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	u.RequestStateUpdate(ctx, "agent-1")

	agent, ok := r.agentsCache.Load("agent-1")
	require.True(t, ok)
	select {
	case <-agent.stateChangeChan:
	default:
		t.Fatal("expected state update signal")
	}
}

func TestRequestStateUpdateDoesNothingForMissingAgent(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	u := NewStateUpdater(nil, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	assert.NotPanics(t, func() {
		u.RequestStateUpdate(ctx, "missing-agent")
	})
}

func TestRequestStateUpdateDoesNotBlockWhenUpdateIsAlreadyQueued(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	agent := pmmAgentInfo{id: "agent-1", stateChangeChan: make(chan struct{}, 1)}
	agent.stateChangeChan <- struct{}{}
	r.agentsCache.Store("agent-1", agent)
	u := NewStateUpdater(nil, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	done := make(chan struct{})
	go func() {
		u.RequestStateUpdate(ctx, "agent-1")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RequestStateUpdate should not block when update is already queued")
	}

	assert.Len(t, agent.stateChangeChan, 1)
}

func TestRequestStateUpdateConcurrentCallsQueueSingleUpdate(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	agent := pmmAgentInfo{id: "agent-1", stateChangeChan: make(chan struct{}, 1)}
	r.agentsCache.Store("agent-1", agent)
	u := NewStateUpdater(nil, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	const callers = 32
	var wg sync.WaitGroup
	wg.Add(callers)

	for range callers {
		go func() {
			defer wg.Done()
			u.RequestStateUpdate(ctx, "agent-1")
		}()
	}
	wg.Wait()

	assert.Len(t, agent.stateChangeChan, 1)
}

func TestUpdateAgentsStateQueuesUpdatesForAllConnectedAgents(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	r.agentsCache.Store("agent-1", pmmAgentInfo{id: "agent-1", stateChangeChan: make(chan struct{}, 1)})
	r.agentsCache.Store("agent-2", pmmAgentInfo{id: "agent-2", stateChangeChan: make(chan struct{}, 1)})
	u := NewStateUpdater(db, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE agent_type = \$1 ORDER BY agent_id`).
		WithArgs(string(models.PMMAgentType)).
		WillReturnRows(newAgentRows(
			agentRow{id: "agent-1", connected: true},
			agentRow{id: "agent-2", connected: true},
		))

	err = u.UpdateAgentsState(ctx)
	require.NoError(t, err)

	for _, agentID := range []string{"agent-1", "agent-2"} {
		agent, ok := r.agentsCache.Load(agentID)
		require.True(t, ok)
		select {
		case <-agent.stateChangeChan:
		default:
			t.Fatalf("expected state update signal for %s", agentID)
		}
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAgentsStateReturnsErrorWhenFetchingAgentsFails(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	u := NewStateUpdater(db, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE agent_type = \$1 ORDER BY agent_id`).
		WithArgs(string(models.PMMAgentType)).
		WillReturnError(assert.AnError)

	err = u.UpdateAgentsState(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot find pmmAgentsIDs for AgentsState update")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAgentsStateIgnoresAgentsThatAreNotInRegistry(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	r.agentsCache.Store("agent-1", pmmAgentInfo{id: "agent-1", stateChangeChan: make(chan struct{}, 1)})
	u := NewStateUpdater(db, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE agent_type = \$1 ORDER BY agent_id`).
		WithArgs(string(models.PMMAgentType)).
		WillReturnRows(newAgentRows(
			agentRow{id: "agent-1", connected: true},
			agentRow{id: "agent-missing", connected: true},
		))

	err = u.UpdateAgentsState(ctx)
	require.NoError(t, err)

	agent, ok := r.agentsCache.Load("agent-1")
	require.True(t, ok)
	select {
	case <-agent.stateChangeChan:
	default:
		t.Fatal("expected state update signal for agent-1")
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAgentsStateSucceedsWhenThereAreNoPMMAgents(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	u := NewStateUpdater(db, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE agent_type = \$1 ORDER BY agent_id`).
		WithArgs(string(models.PMMAgentType)).
		WillReturnRows(newAgentRows())

	err = u.UpdateAgentsState(ctx)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAgentsStateDoesNotQueueDuplicateSignalWhenAlreadyQueued(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	agent := pmmAgentInfo{id: "agent-1", stateChangeChan: make(chan struct{}, 1)}
	agent.stateChangeChan <- struct{}{}
	r.agentsCache.Store("agent-1", agent)
	u := NewStateUpdater(db, r, nil, nil, nil, testLimiter{})
	ctx := logger.Set(context.Background(), "test-request")

	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE agent_type = \$1 ORDER BY agent_id`).
		WithArgs(string(models.PMMAgentType)).
		WillReturnRows(newAgentRows(agentRow{id: "agent-1", connected: true}))

	err = u.UpdateAgentsState(ctx)
	require.NoError(t, err)
	assert.Len(t, agent.stateChangeChan, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
