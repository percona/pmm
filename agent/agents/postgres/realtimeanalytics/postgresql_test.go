// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package realtimeanalytics

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

func TestActivityQueryIsTagged(t *testing.T) {
	t.Parallel()

	assert.True(t, strings.HasPrefix(activityQuery, "WITH "+agents.RTAQueryTag+" "))
}

func TestNewCollectInterval(t *testing.T) {
	t.Parallel()

	l := logrus.NewEntry(logrus.New())
	for _, interval := range []time.Duration{0, -1} {
		m, err := New(&Params{CollectInterval: interval}, l)
		require.NoError(t, err)
		assert.Equal(t, defaultCollectInterval, m.collectInterval)
	}
}

func TestInstanceAddress(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "pg-1:5432", instanceAddress("postgres://pmm:secret@pg-1:5432/postgres?sslmode=disable"))
	assert.Equal(t, "[::1]:5433", instanceAddress("postgres://pmm@[::1]:5433/postgres"))
	assert.Equal(t, "/var/run/postgresql", instanceAddress("postgres://pmm:secret@/postgres?host=%2Fvar%2Frun%2Fpostgresql&sslmode=disable"))
	assert.Empty(t, instanceAddress("postgres://%zz"))
}

func TestActivityQuerySessionLimit(t *testing.T) {
	t.Parallel()

	assert.Contains(t, activityQuery, "LIMIT "+strconv.Itoa(sessionLimit)+")")
}

func TestBlockingChains(t *testing.T) {
	t.Parallel()

	t.Run("Chain", func(t *testing.T) {
		t.Parallel()

		// 7 is idle in transaction; 8 waits for 7, 9 waits for 8.
		blk := map[int64][]int64{7: nil, 8: {7}, 9: {8}}
		chains, complete := blockingChains(blk, []int64{8, 9}, blockerLimit)
		assert.True(t, complete)
		assert.Equal(t, map[int64][]int64{8: {7}, 9: {7, 8}}, chains)
	})

	t.Run("Cycle", func(t *testing.T) {
		t.Parallel()

		// A deadlock not yet broken by deadlock_timeout: neither session lists itself.
		blk := map[int64][]int64{5: {6}, 6: {5}}
		chains, complete := blockingChains(blk, []int64{5, 6}, blockerLimit)
		assert.True(t, complete)
		assert.Equal(t, map[int64][]int64{5: {6}, 6: {5}}, chains)
	})

	t.Run("PreparedTransaction", func(t *testing.T) {
		t.Parallel()

		// pg_blocking_pids() names a prepared transaction 0, a pid no row has.
		blk := map[int64][]int64{3: {preparedTransaction}, 4: {3}}
		chains, complete := blockingChains(blk, []int64{3, 4}, blockerLimit)
		assert.True(t, complete)
		assert.Equal(t, map[int64][]int64{3: {0}, 4: {0, 3}}, chains)
	})

	t.Run("BackgroundWorkerInChain", func(t *testing.T) {
		t.Parallel()

		// 2 waits for autovacuum 50, which waits for 7; 50 is not listed but is still followed.
		blk := map[int64][]int64{2: {50}, 50: {7}, 7: nil}
		chains, complete := blockingChains(blk, []int64{2}, blockerLimit)
		assert.True(t, complete)
		assert.Equal(t, map[int64][]int64{2: {7, 50}}, chains)
	})

	t.Run("QueueOnOneRow", func(t *testing.T) {
		t.Parallel()

		// 100 holds the row; 1 waits for it and the rest queue behind 1, each listing everyone ahead.
		blk := map[int64][]int64{100: nil, 1: {100}}
		waiters := []int64{1}
		for pid := int64(2); pid <= 6; pid++ {
			for ahead := int64(1); ahead < pid; ahead++ {
				blk[pid] = append(blk[pid], ahead)
			}
			waiters = append(waiters, pid)
		}

		// Roots take 6 entries, the chains of 1, 2 and 3 another 3; the chain of 4 does not fit.
		chains, complete := blockingChains(blk, waiters, 10)
		assert.False(t, complete)
		assert.Equal(t, map[int64][]int64{
			1: {100},
			2: {1, 100},
			3: {1, 2, 100},
			4: {100},
			5: {100},
			6: {100},
		}, chains)

		chains, complete = blockingChains(blk, waiters, blockerLimit)
		assert.True(t, complete)
		assert.Equal(t, []int64{1, 2, 3, 4, 5, 100}, chains[6])
	})

	t.Run("RootsDoNotFit", func(t *testing.T) {
		t.Parallel()

		// 1 and 2 each wait for the same three holders; only 1 fits.
		blk := map[int64][]int64{1: {10, 11, 12}, 2: {10, 11, 12}, 10: nil, 11: nil, 12: nil}
		chains, complete := blockingChains(blk, []int64{1, 2}, 4)
		assert.False(t, complete)
		assert.Equal(t, map[int64][]int64{1: {10, 11, 12}}, chains)
	})
}

func TestCollect(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	columns := []string{
		"pid", "listed", "raw", "datname", "usename", "application_name", "state", "backend_type",
		"wait_event_type", "wait_event", "client", "query", "query_id", "xact_start", "query_start",
		"duration", "xact_age", "truncated", "blk", "waited",
	}
	started := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(columns).
		// idle in transaction, holding the row 8 waits for
		AddRow(7, true, `{"pid": 7}`, "db", "app", "", "idle in transaction", "client backend",
			"Client", "ClientRead", "10.0.0.1:5000", "UPDATE t SET v = 1", "", started, started,
			30.0, 30.0, false, nil, nil).
		AddRow(8, true, `{"pid": 8}`, "db", "app", "", "active", "client backend",
			"Lock", "transactionid", "10.0.0.2:5000", "UPDATE t SET v = 2", "", started, started,
			5.0, 5.0, false, "{7}", 4.0).
		// waits for a prepared transaction
		AddRow(9, true, `{"pid": 9}`, "db", "app", "", "active", "client backend",
			"Lock", "transactionid", "10.0.0.3:5000", "UPDATE t SET v = 3", "", started, started,
			2.0, 2.0, false, "{0}", 1.5).
		// waits for autovacuum, which is not listed
		AddRow(10, true, `{"pid": 10}`, "db", "app", "", "active", "client backend",
			"Lock", "relation", "10.0.0.4:5000", "ALTER TABLE t ADD c int", "", started, started,
			1.0, 1.0, false, "{50}", 0.5).
		AddRow(50, false, nil, "db", "", "", "", "autovacuum worker",
			"", "", "", "autovacuum: VACUUM public.t", "", started, started,
			nil, 60.0, false, nil, nil)
	mock.ExpectQuery(activityQuery).WillReturnRows(rows)

	m := &PostgreSQLRTA{db: db, l: logrus.NewEntry(logrus.New()), collectInterval: time.Second, serviceID: "svc"}
	res, err := m.collect(t.Context())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, res, 4, "the autovacuum worker is only a blocker")

	holder := res[0].GetPostgresqlPayload()
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED, holder.BlockedStatus)
	assert.Equal(t, `{"pid": 7}`, res[0].QueryRawJson)

	waiter := res[1].GetPostgresqlPayload()
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED, waiter.BlockedStatus)
	require.Len(t, waiter.BlockedBy, 1)
	assert.Equal(t, int64(7), waiter.BlockedBy[0].BlockingConnId)
	assert.Equal(t, "UPDATE t SET v = 1", waiter.BlockedBy[0].BlockingQuery)
	assert.Equal(t, "idle in transaction", waiter.BlockedBy[0].BlockingCommand)
	assert.Equal(t, 30*time.Second, waiter.BlockedBy[0].BlockerTransactionDuration.AsDuration())
	assert.Equal(t, 4*time.Second, waiter.BlockedBy[0].WaitDuration.AsDuration())
	assert.True(t, waiter.BlockedBy[0].Root)

	prepared := res[2].GetPostgresqlPayload()
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED, prepared.BlockedStatus)
	require.Len(t, prepared.BlockedBy, 1)
	assert.Equal(t, int64(0), prepared.BlockedBy[0].BlockingConnId)
	assert.Equal(t, "prepared transaction", prepared.BlockedBy[0].BlockingCommand)
	assert.True(t, prepared.BlockedBy[0].Root)

	behindWorker := res[3].GetPostgresqlPayload()
	require.Len(t, behindWorker.BlockedBy, 1)
	assert.Equal(t, int64(50), behindWorker.BlockedBy[0].BlockingConnId)
	assert.Equal(t, "autovacuum worker", behindWorker.BlockedBy[0].BlockingCommand)
	assert.True(t, behindWorker.BlockedBy[0].Root)
}

func TestRunProbe(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, probeErr error) []agents.Change {
		t.Helper()

		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)

		mock.ExpectQuery("SELECT pg_has_role('pg_read_all_stats', 'USAGE')").
			WillReturnRows(sqlmock.NewRows([]string{"pg_has_role"}).AddRow(true))
		probe := mock.ExpectQuery(activityQuery + "\nLIMIT 0")
		if probeErr != nil {
			probe.WillReturnError(probeErr)
		} else {
			probe.WillReturnRows(sqlmock.NewRows([]string{"pid"}))
		}

		m := &PostgreSQLRTA{
			db:              db,
			l:               logrus.NewEntry(logrus.New()),
			collectInterval: time.Hour,
			changes:         make(chan agents.Change, 10),
		}
		ctx, cancel := context.WithCancel(t.Context())
		go m.Run(ctx)

		var changes []agents.Change
		for c := range m.Changes() {
			changes = append(changes, c)
			if c.Status == inventoryv1.AgentStatus_AGENT_STATUS_RUNNING {
				cancel()
			}
		}
		cancel()
		require.NoError(t, mock.ExpectationsWereMet())

		return changes
	}

	t.Run("QueryFails", func(t *testing.T) {
		t.Parallel()

		changes := run(t, errors.New("function pg_blocking_pids(integer) does not exist"))
		require.Len(t, changes, 2)
		assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_STARTING, changes[0].Status)
		assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, changes[1].Status)
		assert.Contains(t, changes[1].StatusMessage, "pg_blocking_pids(integer) does not exist")
	})

	t.Run("QueryRuns", func(t *testing.T) {
		t.Parallel()

		changes := run(t, nil)
		statuses := make([]inventoryv1.AgentStatus, 0, len(changes))
		for _, c := range changes {
			statuses = append(statuses, c.Status)
		}
		assert.Equal(t, []inventoryv1.AgentStatus{
			inventoryv1.AgentStatus_AGENT_STATUS_STARTING,
			inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
			inventoryv1.AgentStatus_AGENT_STATUS_STOPPING,
			inventoryv1.AgentStatus_AGENT_STATUS_DONE,
		}, statuses)
	})
}
