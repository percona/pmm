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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

func TestCoerceValue(t *testing.T) {
	t.Parallel()

	assert.Nil(t, coerceValue(nil), "NULL must become nil")
	assert.Equal(t, int64(123), coerceValue(sql.RawBytes("123")))
	assert.Equal(t, int64(-5), coerceValue(sql.RawBytes("-5")))
	assert.Equal(t, int64(2648724198000), coerceValue(sql.RawBytes("2648724198000")))
	assert.InEpsilon(t, 1.5, coerceValue(sql.RawBytes("1.5")), 0.0001)
	assert.Equal(t, "COMMIT", coerceValue(sql.RawBytes("COMMIT")))
	assert.Equal(t, "ACTIVE", coerceValue(sql.RawBytes("ACTIVE")))
	// non-nil empty value stays an empty string (not nil)
	emptyValue := coerceValue(sql.RawBytes(""))
	assert.NotNil(t, emptyValue)
	assert.Empty(t, emptyValue)
}

func TestMapHelpers(t *testing.T) {
	t.Parallel()

	row := map[string]any{
		"i":        int64(7),
		"f":        2.5,
		"s":        "text",
		"numStr":   "9",
		"floatStr": "3.5",
		"null":     nil,
	}

	assert.Equal(t, "7", mapString(row, "i"))
	assert.Equal(t, "text", mapString(row, "s"))
	assert.Empty(t, mapString(row, "missing"))
	assert.Empty(t, mapString(row, "null"))

	assert.Equal(t, int64(7), mapInt(row, "i"))
	assert.Equal(t, int64(2), mapInt(row, "f")) // truncates
	assert.Equal(t, int64(9), mapInt(row, "numStr"))
	assert.Equal(t, int64(0), mapInt(row, "missing"))

	assert.InDelta(t, 2.5, mapFloat(row, "f"), 0)
	assert.InDelta(t, float64(7), mapFloat(row, "i"), 0)
	assert.InDelta(t, 3.5, mapFloat(row, "floatStr"), 0)
	assert.InDelta(t, float64(0), mapFloat(row, "missing"), 0)
}

func TestBuildQueryData(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{
		serviceID:         "svc-1",
		serviceName:       "rta-mysql",
		dbInstanceAddress: "127.0.0.1:3306",
	}

	row := map[string]any{
		"conn_id":           int64(42),
		"user":              "sbtest@localhost",
		"db":                "sbtest",
		"command":           "Query",
		"state":             "executing",
		"statement_latency": int64(2_000_000_000), // 2ms expressed in picoseconds
		"current_statement": "SELECT 1",
		"rows_examined":     int64(200),
		"rows_sent":         int64(100),
		"full_scan":         "YES",
		"program_name":      "mysql",
		"trx_state":         "ACTIVE",
		"pid":               nil,
	}

	qd := m.buildQueryData(row, &blockingGraph{})
	require.NotNil(t, qd)

	assert.Equal(t, "svc-1", qd.ServiceId)
	assert.Equal(t, "rta-mysql", qd.ServiceName)
	assert.Equal(t, "42", qd.QueryId)
	assert.Equal(t, "SELECT 1", qd.QueryText)
	// 2_000_000_000 ps / 1000 = 2_000_000 ns = 2ms
	assert.Equal(t, 2*time.Millisecond, qd.QueryExecutionDuration.AsDuration())

	p := qd.GetMySqlPayload()
	require.NotNil(t, p)
	assert.Equal(t, "127.0.0.1:3306", p.DbInstanceAddress)
	assert.Equal(t, "sbtest", p.DatabaseName)
	assert.Equal(t, "Query", p.Command)
	assert.Equal(t, "executing", p.State)
	assert.Equal(t, "sbtest@localhost", p.Username)
	assert.Equal(t, int64(200), p.RowsExamined)
	assert.Equal(t, int64(100), p.RowsSent)
	assert.True(t, p.FullScan)
	assert.Equal(t, "mysql", p.ProgramName)

	// Raw payload is pretty-printed (multi-line) and preserves the whole row, NULLs included.
	assert.Contains(t, qd.QueryRawJson, "\n")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(qd.QueryRawJson), &parsed))
	assert.Contains(t, parsed, "current_statement")
	assert.Contains(t, parsed, "statement_latency")
	assert.Contains(t, parsed, "trx_state")
	assert.Nil(t, parsed["pid"], "NULL columns are preserved as JSON null")
}

func TestBuildQueryDataFullScanAndMissing(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}

	// full_scan "NO" -> false, and a missing statement_latency -> zero duration.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(1),
		"current_statement": "SELECT 2",
		"full_scan":         "NO",
	}, &blockingGraph{complete: true})
	require.NotNil(t, qd)
	assert.False(t, qd.GetMySqlPayload().FullScan)
	assert.Equal(t, time.Duration(0), qd.QueryExecutionDuration.AsDuration())
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED, qd.GetMySqlPayload().BlockedStatus)
	assert.Empty(t, qd.GetMySqlPayload().BlockedBy)
}

func TestNewCollectInterval(t *testing.T) {
	t.Parallel()

	l := logrus.NewEntry(logrus.New())

	assert.Equal(t, 5*time.Second, New(&Params{CollectInterval: 5 * time.Second}, l).collectInterval)

	// A missing or non-positive interval must not reach time.NewTicker, which
	// panics on it and would take the whole pmm-agent down.
	assert.Equal(t, defaultCollectInterval, New(&Params{}, l).collectInterval)
	assert.Equal(t, defaultCollectInterval, New(&Params{CollectInterval: -1}, l).collectInterval)
}

// The lock graph observed on Percona Server 8.0.46 for a three-connection pile-up: 409 is idle
// inside an open transaction and blocks both waiters, and 412 blocks 411 while waiting itself.
func blockingGraphFixture() (map[string][]*rtav1.BlockingTransaction, map[int64]struct{}) {
	blockers := map[string][]*rtav1.BlockingTransaction{
		"411": {
			{BlockingConnId: 412, BlockingCommand: "Query", BlockingQuery: "UPDATE sbtest1 SET k=k+1 WHERE id=1"},
			{BlockingConnId: 409, BlockingCommand: "Sleep", BlockingQuery: "SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE"},
		},
		"412": {
			{BlockingConnId: 409, BlockingCommand: "Sleep", BlockingQuery: "SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE"},
		},
	}
	waiting := map[int64]struct{}{411: {}, 412: {}}

	return blockers, waiting
}

func TestMarkRootBlockers(t *testing.T) {
	t.Parallel()

	blockers, waiting := blockingGraphFixture()
	markRootBlockers(blockers, waiting)

	// 409 waits for nothing, so it is the head of the chain; 412 is queued in the middle of it.
	require.Len(t, blockers["411"], 2)
	assert.Equal(t, int64(409), blockers["411"][0].BlockingConnId, "sorted by connection id")
	assert.True(t, blockers["411"][0].Root, "409 is not itself waiting")
	assert.Equal(t, int64(412), blockers["411"][1].BlockingConnId)
	assert.False(t, blockers["411"][1].Root, "412 is itself waiting on 409")

	require.Len(t, blockers["412"], 1)
	assert.True(t, blockers["412"][0].Root)
}

func TestMarkRootBlockersEveryBlockerWaiting(t *testing.T) {
	t.Parallel()

	// A cycle has no head. Nothing may be reported as root rather than picking one arbitrarily.
	blockers := map[string][]*rtav1.BlockingTransaction{
		"1": {{BlockingConnId: 2}},
		"2": {{BlockingConnId: 1}},
	}
	markRootBlockers(blockers, map[int64]struct{}{1: {}, 2: {}})

	assert.False(t, blockers["1"][0].Root)
	assert.False(t, blockers["2"][0].Root)
}

func TestBuildQueryDataBlocked(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}
	graph := &blockingGraph{
		blockers: map[string][]*rtav1.BlockingTransaction{
			"411": {{
				BlockingConnId:             409,
				BlockingCommand:            "Sleep",
				BlockingQuery:              "SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE",
				BlockingUsername:           "sbtest@172.17.0.1",
				WaitDuration:               durationpb.New(134 * time.Second),
				BlockerTransactionDuration: durationpb.New(154 * time.Second),
				Root:                       true,
			}},
		},
		waiters: map[string]waiterLock{"411": {
			lockType:      rtav1.LockType_LOCK_TYPE_ROW,
			lockedTable:   "sbtest.sbtest1",
			lockedIndex:   "PRIMARY",
			requestedMode: "X,REC_NOT_GAP",
		}},
		complete: true,
	}

	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(411),
		"current_statement": "UPDATE sbtest1 SET k=k+1 WHERE id=1",
		"command":           "Query",
		"state":             "updating",
	}, graph)

	payload := qd.GetMySqlPayload()
	require.NotNil(t, payload)
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED, payload.BlockedStatus)
	require.Len(t, payload.BlockedBy, 1)
	assert.Equal(t, int64(409), payload.BlockedBy[0].BlockingConnId)
	assert.Equal(t, "Sleep", payload.BlockedBy[0].BlockingCommand, "the head of a chain is idle in a transaction")
	assert.Equal(t, 134*time.Second, payload.BlockedBy[0].WaitDuration.AsDuration())
	assert.True(t, payload.BlockedBy[0].Root)
	// The contended lock describes the waiting statement, not any one of its blockers.
	assert.Equal(t, "sbtest.sbtest1", payload.LockedTable)
	assert.Equal(t, "PRIMARY", payload.LockedIndex)
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_ROW, payload.LockType)
	assert.Equal(t, "X,REC_NOT_GAP", payload.RequestedLockMode)
}

func TestBuildQueryDataNotBlockedWhenGraphHasOtherConnections(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}
	blockers, waiting := blockingGraphFixture()
	markRootBlockers(blockers, waiting)

	// Connection 999 is running while others are blocked; it must not inherit their blockers.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(999),
		"current_statement": "SELECT 1",
	}, &blockingGraph{blockers: blockers, waiters: map[string]waiterLock{}, complete: true})

	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED, qd.GetMySqlPayload().BlockedStatus)
	assert.Empty(t, qd.GetMySqlPayload().BlockedBy)
}

// blockingRows builds a result set shaped like blockingTransactionsSQL returns.
func blockingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"waiting_conn_id", "blocking_conn_id", "wait_micros", "blocker_trx_micros",
		"blocking_command", "blocking_user", "blocking_query", "locked_table", "locked_index",
		"requested_mode", "blocking_mode",
	})
}

// metadataRows builds a result set shaped like metadataLockWaitsSQL returns. Its column order
// differs from the row-lock query's and it carries no wait duration and no index, which is the
// point of keeping the two scanners apart.
func metadataRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"waiting_conn_id", "blocking_conn_id", "requested_mode", "blocking_mode",
		"blocking_command", "blocking_user", "blocker_trx_picos", "blocking_query", "locked_table",
	})
}

// readRowLocks reads just the row-lock source into a fresh graph, so the tests below can pin
// how one query is parsed without standing up the other.
func readRowLocks(t *testing.T, m *MySQLRTA) (*blockingGraph, error) {
	t.Helper()

	graph := newBlockingGraph()
	waiting := make(map[int64]struct{})

	found, err := m.readLockEdges(t.Context(), rowLockSource)
	if err != nil {
		return graph, err
	}

	graph.merge(found, waiting)
	markRootBlockers(graph.blockers, waiting)

	return graph, nil
}

// expectNoMetadataLocks makes the metadata-lock query part of a collection return nothing, for
// tests whose subject is the row-lock half.
func expectNoMetadataLocks(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows())
}

func newMockedRTA(t *testing.T) (*MySQLRTA, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return &MySQLRTA{db: db, l: logrus.NewEntry(logrus.New())}, mock
}

func TestCollectBlockingTransactionsDeduplicates(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// One waiter/blocker pair contending on both a record lock and a gap lock produces two
	// rows for a single relationship; the blocker must be reported once.
	//
	// The rows are fed in the order the production query's ORDER BY yields them --
	// "GEN_CLUST_INDEX" sorts before "PRIMARY" -- so the assertion below pins the value the
	// server would really deliver rather than an arbitrary one.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "GEN_CLUST_INDEX", "X,REC_NOT_GAP", "X,REC_NOT_GAP").
		AddRow(411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1, "the same blocking transaction must not be listed twice")
	assert.Equal(t, int64(409), graph.blockers["411"][0].BlockingConnId)
	// First row wins, and the ORDER BY makes which row that is the same on every collection.
	assert.Equal(t, "GEN_CLUST_INDEX", graph.waiters["411"].lockedIndex)
}

func TestCollectBlockingTransactionsSubSecondWait(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// 900ms: whole-second truncation would report this as a zero-length wait.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 900_000, 3_400_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1)
	assert.Equal(t, 900*time.Millisecond, graph.blockers["411"][0].WaitDuration.AsDuration())
	assert.Equal(t, 3400*time.Millisecond, graph.blockers["411"][0].BlockerTransactionDuration.AsDuration())
}

func TestCollectBlockingTransactionsNullDurations(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// The wait ended between the two reads inside the query: no value is not zero seconds.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, nil, nil, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1)
	assert.Nil(t, graph.blockers["411"][0].WaitDuration, "a NULL duration must stay unset, not become 0s")
	assert.Nil(t, graph.blockers["411"][0].BlockerTransactionDuration)
}

func TestCollectBlockingTransactionsKeepsEdgeWithoutProcesslistRow(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// The blocking thread is gone, so the LEFT JOIN yields NULL columns. The relationship
	// still explains the wait and must survive.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, nil, nil, nil, "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1, "a blocker with no processlist row must not drop the edge")
	assert.Empty(t, graph.blockers["411"][0].BlockingCommand)
	assert.Equal(t, "db.t", graph.waiters["411"].lockedTable)
	assert.True(t, graph.blockers["411"][0].Root)
}

func TestCollectBlockingTransactionsOrWarnStopsOnMissingTable(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// MySQL 5.7 has no performance_schema.data_lock_waits; retrying forever would never
	// succeed. It does have metadata_locks, so the other source must carry on: the two are
	// disabled independently or a 5.7 server would lose the half of the graph it can serve.
	mock.ExpectQuery("data_lock_waits").WillReturnError(&mysql.MySQLError{
		Number:  mysqlErrNoSuchTable,
		Message: "Table 'performance_schema.data_lock_waits' doesn't exist",
	})
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(411, 409, "EXCLUSIVE", "SHARED_READ", "Sleep", "u@h", nil, "SELECT 1", "db.t"))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph, "one dead source must not discard what the other found")
	assert.True(t, m.rowLocks.unsupported)
	assert.False(t, m.metadataLocks.unsupported)
	assert.Len(t, graph.blockers["411"], 1)
	assert.False(t, graph.complete, "with row locks unreadable, silence is not proof of health")

	// Only the metadata-lock query is expected the second time round: sqlmock fails on an
	// unexpected call, so a retry of the dead source would fail the assertion below.
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows())
	assert.NotNil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsOrWarnRetriesTransientErrors(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("data_lock_waits").WillReturnError(errors.New("connection reset"))
	expectNoMetadataLocks(mock)
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))
	expectNoMetadataLocks(mock)

	partial := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, partial, "the surviving source still answered")
	assert.False(t, partial.complete, "a failed source leaves the graph unable to prove health")
	assert.False(t, m.rowLocks.unsupported, "a transient failure must not disable collection")

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.Len(t, graph.blockers["411"], 1)
	assert.True(t, graph.complete, "both sources answered on the retry")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildQueryDataUnknownWhenGraphUnavailable(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}
	// A nil graph means the lock tables could not be read. Reporting NOT_BLOCKED here would
	// let a monitoring gap read as a verified healthy server during a real pile-up.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(411),
		"current_statement": "UPDATE sbtest1 SET k=k+1 WHERE id=1",
	}, nil)

	payload := qd.GetMySqlPayload()
	require.NotNil(t, payload)
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED, payload.BlockedStatus)
	assert.Empty(t, payload.BlockedBy)
}

func TestCollectBlockingTransactionsOrWarnStopsOnAccessDenied(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// A privilege the monitoring user was never granted stays ungranted until an operator
	// changes it, so retrying every 2s would be tens of thousands of doomed round trips a day.
	mock.ExpectQuery("data_lock_waits").WillReturnError(&mysql.MySQLError{
		Number:  mysqlErrTableAccessDenied,
		Message: "SELECT command denied to user 'pmm'@'%' for table 'data_lock_waits'",
	})
	mock.ExpectQuery("metadata_locks").WillReturnError(&mysql.MySQLError{
		Number:  mysqlErrTableAccessDenied,
		Message: "SELECT command denied to user 'pmm'@'%' for table 'metadata_locks'",
	})

	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()), "with both sources denied nothing is known")
	assert.True(t, m.rowLocks.unsupported)
	assert.True(t, m.metadataLocks.unsupported)

	// sqlmock fails on an unexpected call, so any retry would fail this assertion.
	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsRecordsWaiterLockOnce(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// Two blockers of one waiter contend over the same requested lock: it is a property of
	// the waiting statement, recorded once, not repeated per blocker.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP").
		AddRow(411, 410, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 2", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 2)
	assert.Equal(t, "db.t", graph.waiters["411"].lockedTable)
	assert.Equal(t, "PRIMARY", graph.waiters["411"].lockedIndex)
	// Both hold the statement up independently, so both are roots and neither is "the" cause.
	assert.True(t, graph.blockers["411"][0].Root)
	assert.True(t, graph.blockers["411"][1].Root)
}

func TestCollectBlockingTransactionsPrefersARowThatNamesAnIndex(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// Waiter 411 contends with conn 409 over a table-level lock, which reports no index, and
	// with conn 410 over a record lock on PRIMARY. The query orders the row that names an
	// index first precisely so the lower-numbered blocker cannot leave the waiter with none.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 410, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP").
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 2", "db.t", nil, "X,REC_NOT_GAP", "X,REC_NOT_GAP"))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	assert.Equal(t, "PRIMARY", graph.waiters["411"].lockedIndex, "a table-level lock must not erase the index")
	assert.Len(t, graph.blockers["411"], 2)
}

func TestCollectMetadataLockWaits(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	// The ALTER pile-up: 409 holds SHARED_READ inside an open transaction, the ALTER on 411
	// wants EXCLUSIVE and cannot have it, and 412's plain SELECT is queued behind the ALTER.
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(411, 409, "EXCLUSIVE", "SHARED_READ", "Sleep", "u@h", 154_000_000_000_000, "SELECT COUNT(*) FROM t", "db.t").
		AddRow(412, 409, "SHARED_READ", "SHARED_READ", "Sleep", "u@h", 154_000_000_000_000, "SELECT COUNT(*) FROM t", "db.t").
		AddRow(412, 411, "SHARED_READ", "SHARED_UPGRADABLE", "Query", "u@h", 20_000_000_000_000, "ALTER TABLE t ADD COLUMN c INT", "db.t"))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.True(t, graph.complete)

	// The waiter's own request, taken once from the first row for that connection.
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_METADATA, graph.waiters["411"].lockType)
	assert.Equal(t, "EXCLUSIVE", graph.waiters["411"].requestedMode)
	assert.Equal(t, "db.t", graph.waiters["411"].lockedTable)
	assert.Empty(t, graph.waiters["411"].lockedIndex, "a metadata lock is taken on the table, not an index")

	// performance_schema.metadata_locks records no timestamp, so no wait can be derived; the
	// blocker's open transaction is timed in picoseconds and must survive the conversion.
	require.Len(t, graph.blockers["411"], 1)
	assert.Nil(t, graph.blockers["411"][0].WaitDuration, "MDL waits have no recorded start time")
	assert.Equal(t, 154*time.Second, graph.blockers["411"][0].BlockerTransactionDuration.AsDuration())
	assert.Equal(t, "SHARED_READ", graph.blockers["411"][0].BlockingLockMode)

	// 412 is held up by the transaction at the head of the chain and by the ALTER queued in
	// front of it. Only the former is a root: ending it is what actually frees the queue.
	require.Len(t, graph.blockers["412"], 2)
	assert.Equal(t, int64(409), graph.blockers["412"][0].BlockingConnId)
	assert.True(t, graph.blockers["412"][0].Root)
	assert.Equal(t, int64(411), graph.blockers["412"][1].BlockingConnId)
	assert.False(t, graph.blockers["412"][1].Root, "the ALTER is itself waiting, so it is not the cause")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsKeepsLockTypesApart(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// The same connection appears in both sources -- a stale metadata_locks row read moments
	// after the row-lock graph, say. A connection waits on one thing at a time, so the second
	// source must not append blockers held under a mechanism the reported lock type does not
	// describe, which would send the reader after the wrong remedy.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP"))
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(411, 500, "EXCLUSIVE", "SHARED_READ", "Sleep", "u@h", nil, "SELECT 2", "db.t"))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	require.Len(t, graph.blockers["411"], 1, "the second source must not add to a claimed waiter")
	assert.Equal(t, int64(409), graph.blockers["411"][0].BlockingConnId)
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_ROW, graph.waiters["411"].lockType)
	assert.Equal(t, "PRIMARY", graph.waiters["411"].lockedIndex)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildQueryDataUnknownWhenGraphIncomplete(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}
	// One source failed, so this statement's absence from the graph proves nothing: it may be
	// waiting on whichever source did not answer. Reporting NOT_BLOCKED would be a guess
	// wearing the same badge as a verified result.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(411),
		"current_statement": "ALTER TABLE t ADD COLUMN c INT",
	}, newBlockingGraph())

	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED, qd.GetMySqlPayload().BlockedStatus)
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_UNSPECIFIED, qd.GetMySqlPayload().LockType)
}

func TestBuildCurrentQueriesSQLIncludesOnlyColumnsTheServerHas(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// EXECUTION_ENGINE arrived in 8.0.24 and CPU_TIME in 8.0.28. Naming a column the server
	// does not have fails the whole collection, so each is probed and only then selected.
	mock.ExpectQuery("information_schema.COLUMNS").
		WithArgs("threads", "EXECUTION_ENGINE").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("information_schema.COLUMNS").
		WithArgs("events_statements_current", "CPU_TIME").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	expectConsumerProbes(mock, false)

	query, err := m.buildCurrentQueriesSQL(t.Context())
	require.NoError(t, err)
	assert.Contains(t, query, "execution_engine", "a column the server has must be selected")
	assert.NotContains(t, query, "cpu_latency", "a column the server lacks must be left out")
	// The rest of the payload is version-independent and must always be there.
	assert.Contains(t, query, "AS conn_id")
	assert.NotContains(t, query, "memory_summary_by_thread_by_event_name",
		"the per-row memory lookup was the query's largest cost and is no longer collected")
	assert.NotContains(t, query, "sys.x$processlist", "the sys view is what this replaces")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildCurrentQueriesSQLFailsLoudly(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// A failed probe must not silently yield a query missing columns: the agent reports the
	// problem at startup rather than serving a payload that quietly lost fields.
	mock.ExpectQuery("information_schema.COLUMNS").WillReturnError(errors.New("connection reset"))

	_, err := m.buildCurrentQueriesSQL(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXECUTION_ENGINE")
}

// expectConsumerProbes answers the setup_consumers probes the query builder makes, one per
// optional source, in the order it asks.
func expectConsumerProbes(mock sqlmock.Sqlmock, enabled bool) {
	state := "NO"
	if enabled {
		state = "YES"
	}

	for _, source := range optionalProcesslistSources {
		mock.ExpectQuery("setup_consumers").WithArgs(source.consumer).
			WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow(state))
	}
}

func TestOptionalSourcesFollowTheirConsumers(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, consumersOn bool) string {
		t.Helper()

		m, mock := newMockedRTA(t)
		for _, column := range optionalProcesslistColumns {
			mock.ExpectQuery("information_schema.COLUMNS").WithArgs(column.table, column.column).
				WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
		}
		expectConsumerProbes(mock, consumersOn)

		query, err := m.buildCurrentQueriesSQL(t.Context())
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())

		return query
	}

	// Off is the default. The tables are then empty, so joining them buys nothing but costs a
	// lookup per row on every collection.
	off := build(t, false)
	assert.NotContains(t, off, "events_waits_current")
	assert.NotContains(t, off, "events_stages_current")
	assert.NotContains(t, off, "last_wait")
	assert.NotContains(t, off, "progress")

	// Someone who has deliberately enabled them still gets the columns.
	on := build(t, true)
	assert.Contains(t, on, "LEFT JOIN performance_schema.events_waits_current ewc")
	assert.Contains(t, on, "LEFT JOIN performance_schema.events_stages_current estc")
	assert.Contains(t, on, "AS last_wait")
	assert.Contains(t, on, "AS source")
	assert.Contains(t, on, "AS progress")
}

func TestStatementColumnsComeFromOneJoin(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	for _, column := range optionalProcesslistColumns {
		mock.ExpectQuery("information_schema.COLUMNS").WithArgs(column.table, column.column).
			WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	}
	expectConsumerProbes(mock, false)

	query, err := m.buildCurrentQueriesSQL(t.Context())
	require.NoError(t, err)

	// events_statements_current has to be joined inside the derived table for the sort key, so
	// the statement columns are carried out of it rather than joined a second time outside.
	assert.Equal(t, 1, strings.Count(query, "performance_schema.events_statements_current"),
		"one join, not two")
	assert.Contains(t, query, "s.SQL_TEXT", "the derived table carries the statement columns")
	assert.Contains(t, query, "sel.SQL_TEXT", "and the outer query reads them from it")
	assert.NotContains(t, query, "esc.")
}

func TestUnreadableConsumerDoesNotStopTheAgent(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	for _, column := range optionalProcesslistColumns {
		mock.ExpectQuery("information_schema.COLUMNS").WithArgs(column.table, column.column).
			WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	}
	// Reading setup_consumers needs a grant the rest of the query does not. A user without it
	// must lose the optional columns only -- refusing to start would throw away everything else
	// the agent can still collect.
	for _, source := range optionalProcesslistSources {
		mock.ExpectQuery("setup_consumers").WithArgs(source.consumer).
			WillReturnError(&mysql.MySQLError{
				Number:  mysqlErrTableAccessDenied,
				Message: "SELECT command denied to user 'pmm'@'%' for table 'setup_consumers'",
			})
	}

	query, err := m.buildCurrentQueriesSQL(t.Context())
	require.NoError(t, err, "the agent must still start")
	assert.NotContains(t, query, "events_waits_current")
	assert.Contains(t, query, "AS conn_id", "the rest of the query is unaffected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCurrentQueriesSQLIsBounded(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("information_schema.COLUMNS").WithArgs("threads", "EXECUTION_ENGINE").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("information_schema.COLUMNS").WithArgs("events_statements_current", "CPU_TIME").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	expectConsumerProbes(mock, false)

	query, err := m.buildCurrentQueriesSQL(t.Context())
	require.NoError(t, err)

	// Bounded at all: otherwise the row count is capped only by max_connections.
	assert.Contains(t, query, fmt.Sprintf("LIMIT %d", processlistRowLimit))
	// Bounded inside the derived table, not as a trailing ORDER BY ... LIMIT on the whole query:
	// that form runs the current_memory subquery for every candidate before discarding any, and
	// measured slower than no limit at all. The limit must therefore close before the derived
	// table does.
	limitAt := strings.Index(query, fmt.Sprintf("LIMIT %d", processlistRowLimit))
	selAt := strings.Index(query, ") sel")
	require.Positive(t, limitAt, "the query must carry a limit")
	require.Positive(t, selAt, "the bound belongs to a derived table")
	assert.Less(t, limitAt, selAt, "the limit must sit inside the derived table, not after the whole query")
	assert.Equal(t, 1, strings.Count(query, "LIMIT "), "exactly one limit, and it is the derived table's")
	// Ordered by statement latency, not PROCESSLIST_TIME: whole seconds tie in a pile-up and
	// which rows survived truncation would flip between collections.
	assert.Contains(t, query, "ORDER BY COALESCE(s.TIMER_WAIT, 0) DESC, t.PROCESSLIST_ID")
	// The outer query repeats the filter, so a statement finishing between the two reads cannot
	// come back as a row with no query text.
	assert.Contains(t, query, "WHERE pps.PROCESSLIST_INFO IS NOT NULL")
}

func TestProcesslistTruncationIsReportedOncePerEpisode(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	m.currentQueriesSQL = "SELECT 1"

	full := func() *sqlmock.Rows {
		rows := sqlmock.NewRows([]string{"conn_id", "current_statement"})
		for i := range processlistRowLimit {
			rows.AddRow(i, "SELECT 1")
		}

		return rows
	}
	// Two truncated collections then one that is not, so both edges of the episode are covered.
	for range 2 {
		mock.ExpectQuery("SELECT 1").WillReturnRows(full())
		mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
		expectNoMetadataLocks(mock)
	}
	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"conn_id", "current_statement"}).AddRow(1, "SELECT 1"))
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	expectNoMetadataLocks(mock)

	_, err := m.collectProcessList(t.Context())
	require.NoError(t, err)
	assert.True(t, m.processlistTruncated, "the first truncated collection starts the episode")

	_, err = m.collectProcessList(t.Context())
	require.NoError(t, err)
	assert.True(t, m.processlistTruncated, "still truncated, and it must not be announced again")

	_, err = m.collectProcessList(t.Context())
	require.NoError(t, err)
	assert.False(t, m.processlistTruncated, "the episode ends when the list fits again")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLockSourceRowLimitsMatchTheirQueries(t *testing.T) {
	t.Parallel()

	// Truncation is detected by counting rows against this number, so a LIMIT edited in the SQL
	// without the constant would silently stop the detection working.
	for _, source := range []lockSource{rowLockSource, metadataLockSource} {
		assert.Equal(t, lockGraphRowLimit, source.rowLimit, "%s rowLimit", source.name)
		assert.Contains(t, source.sql, fmt.Sprintf("LIMIT %d", lockGraphRowLimit), "%s SQL", source.name)
	}
}

func TestTruncatedLockGraphIsNotCalledComplete(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// Contention is quadratic in the waiters on one object, so a busy table behind a DDL can
	// produce more edges than the query may return. The waiters past the cut are simply absent,
	// and calling the graph complete would publish every one of them as NOT_BLOCKED.
	rows := blockingRows()
	for i := range lockGraphRowLimit {
		rows.AddRow(1000+i, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X", "X")
	}
	mock.ExpectQuery("data_lock_waits").WillReturnRows(rows)
	expectNoMetadataLocks(mock)

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.False(t, graph.complete, "a truncated graph cannot prove anything about the waiters it left out")

	// A statement the truncated graph says nothing about must stay unknown, not healthy.
	qd := m.buildQueryData(map[string]any{"conn_id": int64(999999), "current_statement": "UPDATE t SET a=1"}, graph)
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED, qd.GetMySqlPayload().BlockedStatus)
}

func TestBlockerWithoutConnectionIdStillReportsTheWait(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// A metadata lock held by a background thread has no connection id to report. Dropping the
	// row would leave the waiter looking unblocked, which is the wait dressed up as health.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(411, nil, "EXCLUSIVE", "SHARED_READ", "Daemon", "sql/main", nil, nil, "db.t"))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.Empty(t, graph.blockers["411"], "a blocker with no connection id cannot be named")

	qd := m.buildQueryData(map[string]any{"conn_id": int64(411), "current_statement": "ALTER TABLE t"}, graph)
	payload := qd.GetMySqlPayload()
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED, payload.BlockedStatus, "the wait is real even with no holder to name")
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_METADATA, payload.LockType)
	assert.Equal(t, "db.t", payload.LockedTable)
}

func TestFailedSourceLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// The row-lock source streams one edge and then fails. Those rows must not claim waiter 411
	// and lock it out of the metadata source, which has the answer that is actually current.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X", "X").
		AddRow(412, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X", "X").
		RowError(1, errors.New("connection reset")))
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(411, 500, "EXCLUSIVE", "SHARED_READ", "Query", "u@h", nil, "ALTER TABLE t", "db.t"))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	require.Len(t, graph.blockers["411"], 1)
	assert.Equal(t, int64(500), graph.blockers["411"][0].BlockingConnId, "the failed source must not have claimed this waiter")
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_METADATA, graph.waiters["411"].lockType)
	assert.NotContains(t, graph.waiters, "412", "no waiter may survive from a source that failed")
	assert.False(t, graph.complete)
}

func TestMetadataLockInstrumentDisabledDisablesTheSource(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// With the instrument off, metadata_locks is empty and the query succeeds. Believing it
	// would publish every statement queued behind a DDL as NOT_BLOCKED.
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("NO"))

	m.checkMetadataLockInstrument(t.Context())
	assert.True(t, m.metadataLocks.unsupported, "the source must be disabled, not trusted")

	// Only the row-lock source runs now, so the graph can no longer call anything healthy.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.False(t, graph.complete)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMetadataLockInstrumentUnreadableDoesNotStopTheAgent(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// Reading setup_instruments needs a grant the lock tables themselves do not. A user without
	// it must lose the metadata source only -- statements and row locks are still collectable,
	// and failing startup here would throw away everything the agent can still do.
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnError(&mysql.MySQLError{
			Number:  mysqlErrTableAccessDenied,
			Message: "SELECT command denied to user 'pmm'@'%' for table 'setup_instruments'",
		})

	m.checkMetadataLockInstrument(t.Context())
	assert.True(t, m.metadataLocks.unsupported)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMetadataLockInstrumentMissingDoesNotStopTheAgent(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// A server old enough not to have the instrument at all.
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}))

	m.checkMetadataLockInstrument(t.Context())
	assert.True(t, m.metadataLocks.unsupported)
}

func TestMetadataLockInstrumentEnabledKeepsTheSource(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))

	m.checkMetadataLockInstrument(t.Context())
	assert.False(t, m.metadataLocks.unsupported)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPermanentBlockingError(t *testing.T) {
	t.Parallel()

	// Errors that never heal without an operator changing something, so retrying is pointless.
	for _, number := range []uint16{
		mysqlErrNoSuchTable, mysqlErrTableAccessDenied, mysqlErrSpecificAccessDenied,
	} {
		assert.True(t, permanentBlockingError(number), "error %d must stop collection", number)
	}

	// A dropped connection or a lock-wait timeout may well succeed next cycle.
	for _, number := range []uint16{2006, 1205, 0} {
		assert.False(t, permanentBlockingError(number), "error %d must stay retryable", number)
	}
}
