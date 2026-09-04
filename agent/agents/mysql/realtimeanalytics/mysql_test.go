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
	}, &blockingGraph{})
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
		lockedTable: map[string]string{"411": "sbtest.sbtest1"},
		lockedIndex: map[string]string{"411": "PRIMARY"},
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
	}, &blockingGraph{blockers: blockers})

	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED, qd.GetMySqlPayload().BlockedStatus)
	assert.Empty(t, qd.GetMySqlPayload().BlockedBy)
}

// blockingRows builds a result set shaped like blockingTransactionsSQL returns.
func blockingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"waiting_conn_id", "blocking_conn_id", "wait_micros", "blocker_trx_micros",
		"blocking_command", "blocking_user", "blocking_query", "locked_table", "locked_index",
	})
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
		AddRow(411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "GEN_CLUST_INDEX").
		AddRow(411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "PRIMARY"))

	graph, err := m.collectBlockingTransactions(t.Context())
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1, "the same blocking transaction must not be listed twice")
	assert.Equal(t, int64(409), graph.blockers["411"][0].BlockingConnId)
	// First row wins, and the ORDER BY makes which row that is the same on every collection.
	assert.Equal(t, "GEN_CLUST_INDEX", graph.lockedIndex["411"])
}

func TestCollectBlockingTransactionsSubSecondWait(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// 900ms: whole-second truncation would report this as a zero-length wait.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 900_000, 3_400_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY"))

	graph, err := m.collectBlockingTransactions(t.Context())
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
		AddRow(411, 409, nil, nil, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY"))

	graph, err := m.collectBlockingTransactions(t.Context())
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
		AddRow(411, 409, 1_000_000, 2_000_000, nil, nil, nil, "db.t", "PRIMARY"))

	graph, err := m.collectBlockingTransactions(t.Context())
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1, "a blocker with no processlist row must not drop the edge")
	assert.Empty(t, graph.blockers["411"][0].BlockingCommand)
	assert.Equal(t, "db.t", graph.lockedTable["411"])
	assert.True(t, graph.blockers["411"][0].Root)
}

func TestCollectBlockingTransactionsOrWarnStopsOnMissingTable(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// MySQL 5.7 has no performance_schema lock tables; retrying forever would never succeed.
	mock.ExpectQuery("data_lock_waits").WillReturnError(&mysql.MySQLError{
		Number:  mysqlErrNoSuchTable,
		Message: "Table 'performance_schema.data_lock_waits' doesn't exist",
	})

	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.True(t, m.blockingUnsupported)

	// No second query is expected: sqlmock fails on an unexpected call.
	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsOrWarnRetriesTransientErrors(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("data_lock_waits").WillReturnError(errors.New("connection reset"))
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY"))

	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.False(t, m.blockingUnsupported, "a transient failure must not disable collection")

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.Len(t, graph.blockers["411"], 1)
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

	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.True(t, m.blockingUnsupported)

	// sqlmock fails on an unexpected call, so a second query would fail this assertion.
	assert.Nil(t, m.collectBlockingTransactionsOrWarn(t.Context()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsRecordsWaiterLockOnce(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// Two blockers of one waiter contend over the same requested lock: it is a property of
	// the waiting statement, recorded once, not repeated per blocker.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY").
		AddRow(411, 410, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 2", "db.t", "PRIMARY"))

	graph, err := m.collectBlockingTransactions(t.Context())
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 2)
	assert.Equal(t, "db.t", graph.lockedTable["411"])
	assert.Equal(t, "PRIMARY", graph.lockedIndex["411"])
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
		AddRow(411, 410, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY").
		AddRow(411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 2", "db.t", nil))

	graph, err := m.collectBlockingTransactions(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "PRIMARY", graph.lockedIndex["411"], "a table-level lock must not erase the index")
	assert.Len(t, graph.blockers["411"], 2)
}

func TestPermanentBlockingError(t *testing.T) {
	t.Parallel()

	// Errors that never heal without an operator changing something, so retrying is pointless.
	for _, number := range []uint16{
		mysqlErrNoSuchTable, mysqlErrTableAccessDenied,
		mysqlErrSpecificAccessDenied, mysqlErrViewInvalid,
	} {
		assert.True(t, permanentBlockingError(number), "error %d must stop collection", number)
	}

	// A dropped connection or a lock-wait timeout may well succeed next cycle.
	for _, number := range []uint16{2006, 1205, 0} {
		assert.False(t, permanentBlockingError(number), "error %d must stay retryable", number)
	}
}
