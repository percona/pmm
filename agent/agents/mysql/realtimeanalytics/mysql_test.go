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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
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
	assert.Equal(t, int64(2), mapInt(row, "f"))
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
		"statement_latency": int64(2_000_000_000),
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
	// statement_latency is in picoseconds.
	assert.Equal(t, 2*time.Millisecond, qd.QueryExecutionDuration.AsDuration())

	p := qd.GetMySqlPayload()
	require.NotNil(t, p)
	assert.Equal(t, "127.0.0.1:3306", p.DbInstanceAddress)
	assert.Equal(t, "sbtest", p.DatabaseName)
	assert.Equal(t, "Query", p.Command)
	assert.Equal(t, "executing", p.State)
	assert.Equal(t, "sbtest@localhost", p.Username)
	assert.Equal(t, int64(200), p.GetRowsExamined())
	assert.Equal(t, int64(100), p.GetRowsSent())
	assert.True(t, p.GetFullScan())
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

	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(1),
		"current_statement": "SELECT 2",
		"full_scan":         "NO",
	}, &blockingGraph{complete: true})
	require.NotNil(t, qd)
	assert.False(t, qd.GetMySqlPayload().GetFullScan())
	assert.Equal(t, time.Duration(0), qd.QueryExecutionDuration.AsDuration())
	assert.Equal(t, rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED, qd.GetMySqlPayload().BlockedStatus)
	assert.Empty(t, qd.GetMySqlPayload().BlockedBy)
}

func TestNewCollectInterval(t *testing.T) {
	t.Parallel()

	l := logrus.NewEntry(logrus.New())

	assert.Equal(t, 5*time.Second, New(&Params{CollectInterval: 5 * time.Second}, l).collectInterval)

	// time.NewTicker panics on a non-positive interval, which would take pmm-agent down.
	assert.Equal(t, defaultCollectInterval, New(&Params{}, l).collectInterval)
	assert.Equal(t, defaultCollectInterval, New(&Params{CollectInterval: -1}, l).collectInterval)
}

// blockingGraphFixture returns a pile-up: idle 409 blocks 411 and 412, and 412 also blocks 411.
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

	// A cycle has no head, so no blocker may be reported as root.
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
		"pmm_agent_rta", "waiting_conn_id", "blocking_conn_id", "wait_micros", "blocker_trx_micros",
		"blocking_command", "blocking_user", "blocking_query", "locked_table", "locked_index",
		"requested_mode", "blocking_mode", "waiting_event_id", "waiting_query",
	})
}

// metadataRows builds a result set shaped like metadataLockWaitsSQL returns.
func metadataRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"pmm_agent_rta", "waiting_conn_id", "blocking_conn_id", "requested_mode", "blocking_mode",
		"blocking_command", "blocking_user", "blocker_trx_picos", "blocking_query", "locked_table",
		"waiting_event_id", "waiting_query",
	})
}

// readRowLocks reads only the data_lock_waits source into a fresh graph.
func readRowLocks(t *testing.T, m *MySQLRTA) (*blockingGraph, error) {
	t.Helper()

	graph := newBlockingGraph()
	waiting := make(map[int64]struct{})

	found, err := m.readLockEdges(t.Context(), dataLockWaitsSource)
	if err != nil {
		return graph, err
	}

	graph.merge(found, waiting)
	markRootBlockers(graph.blockers, waiting)

	return graph, nil
}

// expectNoMetadataLocks makes the metadata-lock query of a collection return no rows.
func expectNoMetadataLocks(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows())
}

// newMockedRTA builds an agent on a mocked connection that reads row locks from data_lock_waits.
func newMockedRTA(t *testing.T) (*MySQLRTA, sqlmock.Sqlmock) {
	t.Helper()

	return newMockedRTAWithSource(t, dataLockWaitsSource)
}

func newMockedRTAWithSource(t *testing.T, source lockSource) (*MySQLRTA, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return &MySQLRTA{db: db, l: logrus.NewEntry(logrus.New()), rowLockSource: source}, mock
}

func TestCollectBlockingTransactionsDeduplicates(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// A record lock and a gap lock between one pair yield two rows for a single relationship.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "GEN_CLUST_INDEX", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil).
		AddRow(nil, 411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1, "the same blocking transaction must not be listed twice")
	assert.Equal(t, int64(409), graph.blockers["411"][0].BlockingConnId)
	// First row wins; the rows are fed in the query's ORDER BY order, as the server returns them.
	assert.Equal(t, "GEN_CLUST_INDEX", graph.waiters["411"].lockedIndex)
}

func TestCollectBlockingTransactionsSubSecondWait(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// 900ms: whole-second truncation would report this as a zero-length wait.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 900_000, 3_400_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1)
	assert.Equal(t, 900*time.Millisecond, graph.blockers["411"][0].WaitDuration.AsDuration())
	assert.Equal(t, 3400*time.Millisecond, graph.blockers["411"][0].BlockerTransactionDuration.AsDuration())
}

func TestCollectBlockingTransactionsNullDurations(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// The durations are NULL when the wait ends between the query's two reads.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, nil, nil, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1)
	assert.Nil(t, graph.blockers["411"][0].WaitDuration, "a NULL duration must stay unset, not become 0s")
	assert.Nil(t, graph.blockers["411"][0].BlockerTransactionDuration)
}

func TestCollectBlockingTransactionsKeepsEdgeWithoutProcesslistRow(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// The blocking thread is gone, so the LEFT JOIN yields NULL processlist columns.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_000_000, 2_000_000, nil, nil, nil, "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))

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
	// MySQL 5.7 has metadata_locks but no data_lock_waits: the sources are disabled independently.
	mock.ExpectQuery("data_lock_waits").WillReturnError(&mysql.MySQLError{
		Number:  mysqlErrNoSuchTable,
		Message: "Table 'performance_schema.data_lock_waits' doesn't exist",
	})
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(nil, 411, 409, "EXCLUSIVE", "SHARED_READ", "Sleep", "u@h", nil, "SELECT 1", "db.t", nil, nil))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph, "one dead source must not discard what the other found")
	assert.True(t, m.rowLocks.unsupported)
	assert.False(t, m.metadataLocks.unsupported)
	assert.Len(t, graph.blockers["411"], 1)
	assert.False(t, graph.complete, "with row locks unreadable, silence is not proof of health")

	// sqlmock fails on an unexpected query, so a retry of the dead source would fail below.
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
		AddRow(nil, 411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))
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
	// A nil graph means the lock tables were unreadable, which must not read as NOT_BLOCKED.
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
	// A missing grant stays missing until an operator acts, so retrying is pointless.
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
	// The requested lock belongs to the waiting statement: recorded once, not per blocker.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil).
		AddRow(nil, 411, 410, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 2", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 2)
	assert.Equal(t, "db.t", graph.waiters["411"].lockedTable)
	assert.Equal(t, "PRIMARY", graph.waiters["411"].lockedIndex)
	// Each blocker holds the statement up independently, so both are roots.
	assert.True(t, graph.blockers["411"][0].Root)
	assert.True(t, graph.blockers["411"][1].Root)
}

func TestCollectBlockingTransactionsPrefersARowThatNamesAnIndex(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// 409's table-level lock names no index; the query orders rows that name one first.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 410, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil).
		AddRow(nil, 411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 2", "db.t", nil, "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	assert.Equal(t, "PRIMARY", graph.waiters["411"].lockedIndex, "a table-level lock must not erase the index")
	assert.Len(t, graph.blockers["411"], 2)
}

func TestCollectMetadataLockWaits(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	// 409 holds SHARED_READ in an open transaction, 411's ALTER waits on it, 412 queues behind.
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(nil, 411, 409, "EXCLUSIVE", "SHARED_READ", "Sleep", "u@h", 154_000_000_000_000, "SELECT COUNT(*) FROM t", "db.t", nil, nil).
		AddRow(nil, 412, 409, "SHARED_READ", "SHARED_READ", "Sleep", "u@h", 154_000_000_000_000, "SELECT COUNT(*) FROM t", "db.t", nil, nil).
		AddRow(nil, 412, 411, "SHARED_READ", "SHARED_UPGRADABLE", "Query", "u@h", 20_000_000_000_000, "ALTER TABLE t ADD COLUMN c INT", "db.t", nil, nil))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.True(t, graph.complete)

	// The waiter's own request, taken once from the first row for that connection.
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_METADATA, graph.waiters["411"].lockType)
	assert.Equal(t, "EXCLUSIVE", graph.waiters["411"].requestedMode)
	assert.Equal(t, "db.t", graph.waiters["411"].lockedTable)
	assert.Empty(t, graph.waiters["411"].lockedIndex, "a metadata lock is taken on the table, not an index")

	// metadata_locks has no timestamps; the blocker's transaction time arrives in picoseconds.
	require.Len(t, graph.blockers["411"], 1)
	assert.Nil(t, graph.blockers["411"][0].WaitDuration, "MDL waits have no recorded start time")
	assert.Equal(t, 154*time.Second, graph.blockers["411"][0].BlockerTransactionDuration.AsDuration())
	assert.Equal(t, "SHARED_READ", graph.blockers["411"][0].BlockingLockMode)

	// Only the head of the chain is a root: ending it is what frees the queue.
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
	// A connection waits on one thing at a time, so a second source must not add blockers
	// under a lock type other than the one reported.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X,REC_NOT_GAP", "X,REC_NOT_GAP", nil, nil))
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(nil, 411, 500, "EXCLUSIVE", "SHARED_READ", "Sleep", "u@h", nil, "SELECT 2", "db.t", nil, nil))

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
	// With a source missing, absence from the graph proves nothing, so the status is unknown.
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
	// Selecting a missing column fails the query: EXECUTION_ENGINE is 8.0.24+, CPU_TIME 8.0.28+.
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
	// Version-independent columns are always selected.
	assert.Contains(t, query, "AS conn_id")
	assert.NotContains(t, query, "memory_summary_by_thread_by_event_name",
		"the per-row memory lookup was the query's largest cost and is no longer collected")
	assert.NotContains(t, query, "sys.x$processlist", "the sys view is what this replaces")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildCurrentQueriesSQLFailsLoudly(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// A failed probe must fail startup rather than silently drop columns.
	mock.ExpectQuery("information_schema.COLUMNS").WillReturnError(errors.New("connection reset"))

	_, err := m.buildCurrentQueriesSQL(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXECUTION_ENGINE")
}

// expectConsumerProbes answers the setup_consumers probe for each optional source, in order.
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

	// Off by default, the tables are empty and joining them only costs a lookup per row.
	off := build(t, false)
	assert.NotContains(t, off, "events_waits_current")
	assert.NotContains(t, off, "events_stages_current")
	assert.NotContains(t, off, "last_wait")
	assert.NotContains(t, off, "progress")

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

	// events_statements_current is joined once, in the derived table that sorts by it.
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
	// setup_consumers needs its own grant; without it only the optional columns are lost.
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

	// Without a limit the row count is capped only by max_connections.
	assert.Contains(t, query, fmt.Sprintf("LIMIT %d", processlistRowLimit))
	// A LIMIT on the whole query would run the outer joins for every candidate row, so it must
	// sit inside the derived table.
	limitAt := strings.Index(query, fmt.Sprintf("LIMIT %d", processlistRowLimit))
	selAt := strings.Index(query, ") sel")
	require.Positive(t, limitAt, "the query must carry a limit")
	require.Positive(t, selAt, "the bound belongs to a derived table")
	assert.Less(t, limitAt, selAt, "the limit must sit inside the derived table, not after the whole query")
	assert.Equal(t, 1, strings.Count(query, "LIMIT "), "exactly one limit, and it is the derived table's")
	// Latency is NULL on stock MariaDB, so PROCESSLIST_TIME must come before the connection id:
	// sorting by id alone would truncate away the newest connections of a pile-up.
	assert.Contains(t, query,
		"ORDER BY COALESCE(s.TIMER_WAIT, 0) DESC, COALESCE(t.PROCESSLIST_TIME, 0) DESC, t.PROCESSLIST_ID")
	// The outer query repeats the filter in case a statement finishes between the two reads.
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

	// Truncation is detected by counting rows against rowLimit, so it must match the LIMIT.
	for _, source := range []lockSource{dataLockWaitsSource, innodbLockWaitsSource, metadataLockSource} {
		assert.Equal(t, lockGraphRowLimit, source.rowLimit, "%s rowLimit", source.name)
		assert.Contains(t, source.sql, fmt.Sprintf("LIMIT %d", lockGraphRowLimit), "%s SQL", source.name)
	}
}

func TestTruncatedLockGraphIsNotCalledComplete(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	rows := blockingRows()
	for i := range lockGraphRowLimit {
		rows.AddRow(nil, 1000+i, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X", "X", nil, nil)
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
	// A metadata lock held by a background thread has no connection id to report.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(nil, 411, nil, "EXCLUSIVE", "SHARED_READ", "Daemon", "sql/main", nil, nil, "db.t", nil, nil))

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
	// Rows read before a source fails must not claim a waiter from the other source.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X", "X", nil, nil).
		AddRow(nil, 412, 409, 1_000_000, 2_000_000, "Sleep", "u@h", "SELECT 1", "db.t", "PRIMARY", "X", "X", nil, nil).
		RowError(1, errors.New("connection reset")))
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(nil, 411, 500, "EXCLUSIVE", "SHARED_READ", "Query", "u@h", nil, "ALTER TABLE t", "db.t", nil, nil))

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
	// With the instrument off, metadata_locks is empty but the query still succeeds.
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("NO"))

	m.checkMetadataLockInstrument(t.Context())
	assert.True(t, m.metadataLocks.unsupported, "the source must be disabled, not trusted")

	// With only the row-lock source left, the graph cannot be complete.
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows())
	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.False(t, graph.complete)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMetadataLockInstrumentUnreadableDoesNotStopTheAgent(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	// setup_instruments needs its own grant; without it only the metadata source is lost.
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

// readRowLocksFrom reads one row-lock source into a fresh graph.
func readRowLocksFrom(t *testing.T, m *MySQLRTA, source lockSource) (*blockingGraph, error) {
	t.Helper()

	graph := newBlockingGraph()
	waiting := make(map[int64]struct{})

	found, err := m.readLockEdges(t.Context(), source)
	if err != nil {
		return graph, err
	}

	graph.merge(found, waiting)
	markRootBlockers(graph.blockers, waiting)

	return graph, nil
}

// selectListAliases returns the column aliases a query's SELECT list assigns, in order.
func selectListAliases(query string) []string {
	selectList, _, _ := strings.Cut(query, "\nFROM ")

	return regexp.MustCompile(`AS (\w+)`).FindAllString(selectList, -1)
}

func TestRowLockSourcesReturnTheSameColumns(t *testing.T) {
	t.Parallel()

	// scanRowLockEdge reads both queries by position, so a mismatch silently misassigns fields.
	assert.Equal(t,
		selectListAliases(blockingTransactionsSQL),
		selectListAliases(innodbLockWaitsSQL),
		"the two row-lock queries share one scanner, so their SELECT lists must match")
}

func TestInnodbLockWaitsSourceParsesTheSameEdges(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTAWithSource(t, innodbLockWaitsSource)
	// MariaDB reports plain lock modes, and the query unquotes its backticked lock_table.
	mock.ExpectQuery("INNODB_LOCK_WAITS").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "UPDATE t SET v=1", "db.t", "PRIMARY", "X", "X", nil, nil))

	graph, err := readRowLocksFrom(t, m, innodbLockWaitsSource)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1)
	assert.Equal(t, int64(409), graph.blockers["411"][0].BlockingConnId)
	assert.Equal(t, "UPDATE t SET v=1", graph.blockers["411"][0].BlockingQuery)
	assert.Equal(t, "db.t", graph.waiters["411"].lockedTable)
	assert.Equal(t, "PRIMARY", graph.waiters["411"].lockedIndex)
	assert.Equal(t, rtav1.LockType_LOCK_TYPE_ROW, innodbLockWaitsSource.lockType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// expectRowLockProbe queues selectRowLockSource's information_schema.TABLES lookups.
func expectRowLockProbe(mock sqlmock.Sqlmock, dataLockWaits, innodbLockWaits int) {
	mock.ExpectQuery("information_schema.TABLES").
		WithArgs("performance_schema", "data_lock_waits").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(dataLockWaits))

	if dataLockWaits > 0 {
		return
	}

	mock.ExpectQuery("information_schema.TABLES").
		WithArgs("information_schema", "INNODB_LOCK_WAITS").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(innodbLockWaits))
}

func TestSelectRowLockSourcePrefersPerformanceSchema(t *testing.T) {
	t.Parallel()

	// data_locks keeps the gap/record distinction that information_schema drops.
	m, mock := newMockedRTAWithSource(t, lockSource{})
	expectRowLockProbe(mock, 1, 0)

	m.selectRowLockSource(t.Context())

	assert.Equal(t, blockingTransactionsSQL, m.rowLockSource.sql)
	assert.False(t, m.rowLocks.unsupported)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSelectRowLockSourceFallsBackToInformationSchema(t *testing.T) {
	t.Parallel()

	// MariaDB and MySQL before 8.0 have INNODB_LOCK_WAITS but no data_lock_waits.
	m, mock := newMockedRTAWithSource(t, lockSource{})
	expectRowLockProbe(mock, 0, 1)

	m.selectRowLockSource(t.Context())

	assert.Equal(t, innodbLockWaitsSQL, m.rowLockSource.sql)
	assert.False(t, m.rowLocks.unsupported, "a server with the information_schema tables can report row locks")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSelectRowLockSourceWithNeitherTableDisablesRowLocks(t *testing.T) {
	t.Parallel()

	// With no source, row locks are unsupported rather than read as "nothing is blocked".
	m, mock := newMockedRTAWithSource(t, lockSource{})
	expectRowLockProbe(mock, 0, 0)

	m.selectRowLockSource(t.Context())

	assert.True(t, m.rowLocks.unsupported)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSelectRowLockSourceTreatsAProbeFailureAsAbsent(t *testing.T) {
	t.Parallel()

	// A failed probe for one table must not cost the source that can be read.
	m, mock := newMockedRTAWithSource(t, lockSource{})
	mock.ExpectQuery("information_schema.TABLES").
		WithArgs("performance_schema", "data_lock_waits").
		WillReturnError(errors.New("probe failed"))
	mock.ExpectQuery("information_schema.TABLES").
		WithArgs("information_schema", "INNODB_LOCK_WAITS").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))

	m.selectRowLockSource(t.Context())

	assert.Equal(t, innodbLockWaitsSQL, m.rowLockSource.sql)
	assert.False(t, m.rowLocks.unsupported)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// expectStatementConsumerProbes answers each checkStatementConsumers lookup with enabled.
func expectStatementConsumerProbes(mock sqlmock.Sqlmock, enabled string) {
	for _, consumer := range statementConsumers {
		mock.ExpectQuery("setup_consumers").
			WithArgs(consumer.name).
			WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow(enabled))
	}
}

func TestStatementConsumersDisabledDoNotStopCollection(t *testing.T) {
	t.Parallel()

	// Stock MariaDB ships both consumers off; threads still provides the query text, user,
	// state and elapsed time, so the agent warns and carries on.
	m, mock := newMockedRTA(t)
	expectStatementConsumerProbes(mock, "NO")

	m.checkStatementConsumers(t.Context())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatementConsumersSurviveAnUnreadableSetupTable(t *testing.T) {
	t.Parallel()

	// setup_consumers needs its own grant, so failing to read it must not stop collection.
	m, mock := newMockedRTA(t)
	for _, consumer := range statementConsumers {
		mock.ExpectQuery("setup_consumers").
			WithArgs(consumer.name).
			WillReturnError(errors.New("access denied"))
	}

	m.checkStatementConsumers(t.Context())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatementConsumersAreAllProbed(t *testing.T) {
	t.Parallel()

	// Both consumers feed columns the statement query always selects, so both are probed.
	m, mock := newMockedRTA(t)
	expectStatementConsumerProbes(mock, "YES")

	m.checkStatementConsumers(t.Context())

	assert.Len(t, statementConsumers, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestElapsedTimeFallsBackToProcesslistTime(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}

	// Stock MariaDB has events_statements_current off, leaving only PROCESSLIST_TIME.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(7),
		"current_statement": "SELECT 1",
		"time":              int64(42),
	}, &blockingGraph{})
	require.NotNil(t, qd)
	assert.Equal(t, 42*time.Second, qd.QueryExecutionDuration.AsDuration())
}

func TestStatementLatencyWinsOverProcesslistTime(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}

	// PROCESSLIST_TIME is whole seconds; the picosecond statement_latency is more accurate.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(7),
		"current_statement": "SELECT 1",
		"statement_latency": int64(2_500_000_000),
		"time":              int64(42),
	}, &blockingGraph{})
	require.NotNil(t, qd)
	assert.Equal(t, 2500*time.Microsecond, qd.QueryExecutionDuration.AsDuration())
}

func TestStatementCountsAreUnsetWhenNotMeasured(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}

	// Stock MariaDB has events_statements_current off, so the counts come back NULL.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(7),
		"current_statement": "SELECT 1",
		"rows_examined":     nil,
		"rows_sent":         nil,
		"full_scan":         nil,
		"time":              int64(9),
	}, &blockingGraph{})
	require.NotNil(t, qd)

	p := qd.GetMySqlPayload()
	require.NotNil(t, p)
	assert.Nil(t, p.RowsExamined, "an unmeasured count must be absent, not zero")
	assert.Nil(t, p.RowsSent, "an unmeasured count must be absent, not zero")
	assert.Nil(t, p.FullScan, "an unmeasured flag must be absent, not a clean bill of health")
}

func TestFullScanSQLYieldsNullWhenNothingWasMeasured(t *testing.T) {
	t.Parallel()

	// Two NULL counters must give a NULL full_scan, not 'NO', so unmeasured reads as unknown.
	assert.Contains(t, currentQueriesSQLTemplate,
		"WHEN sel.NO_GOOD_INDEX_USED IS NULL AND sel.NO_INDEX_USED IS NULL THEN NULL",
		"the query must preserve NULL rather than reporting 'NO' for an unmeasured statement")
	assert.NotContains(t, currentQueriesSQLTemplate,
		"IF(sel.NO_GOOD_INDEX_USED > 0 OR sel.NO_INDEX_USED > 0, 'YES', 'NO')",
		"the old expression fabricated 'NO' from NULLs")
}

func TestFullScanIsUnsetWhenTheServerDidNotSayEither(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}

	// Anything but 'YES'/'NO' is unknown, which the UI shows as unavailable rather than "No".
	for _, value := range []any{nil, ""} {
		qd := m.buildQueryData(map[string]any{
			"conn_id":           int64(7),
			"current_statement": "SELECT 1",
			"full_scan":         value,
		}, &blockingGraph{})
		require.NotNil(t, qd)
		assert.Nil(t, qd.GetMySqlPayload().FullScan, "full_scan %v must be reported as unknown", value)
	}

	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(7),
		"current_statement": "SELECT 1",
		"full_scan":         "YES",
		"rows_examined":     int64(500),
	}, &blockingGraph{})
	require.NotNil(t, qd.GetMySqlPayload().FullScan)
	assert.True(t, *qd.GetMySqlPayload().FullScan)
	assert.Equal(t, int64(500), qd.GetMySqlPayload().GetRowsExamined())
}

func TestSelectRowLockSourceDoesNotDisableOnProbeFailure(t *testing.T) {
	t.Parallel()

	// A timed-out probe says nothing about which tables exist, so it must not disable row locks.
	m, mock := newMockedRTAWithSource(t, lockSource{})
	for range rowLockSources {
		mock.ExpectQuery("information_schema.TABLES").WillReturnError(context.DeadlineExceeded)
	}

	assert.False(t, m.selectRowLockSource(t.Context()), "no source could be chosen")
	assert.False(t, m.rowLocks.unsupported, "a failed probe is not evidence the table is absent")
	assert.Empty(t, m.rowLockSource.sql)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsRetriesSourceSelection(t *testing.T) {
	t.Parallel()

	// With no source chosen yet, collection probes again and reads row locks in the same cycle.
	m, mock := newMockedRTAWithSource(t, lockSource{})
	expectRowLockProbe(mock, 0, 1)
	mock.ExpectQuery("INNODB_LOCK_WAITS").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "UPDATE t SET v=1", "db.t", "PRIMARY", "X", "X", nil, nil))
	expectNoMetadataLocks(mock)

	graph := m.collectBlockingTransactionsOrWarn(t.Context())

	require.NotNil(t, graph)
	assert.Equal(t, innodbLockWaitsSQL, m.rowLockSource.sql, "the retry settled on a source")
	require.Len(t, graph.blockers["411"], 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectBlockingTransactionsDoesNotRetryASettledVerdict(t *testing.T) {
	t.Parallel()

	// A settled "neither table" answer is not re-probed; sqlmock fails on any unexpected query.
	m, mock := newMockedRTAWithSource(t, lockSource{})
	m.rowLocks.unsupported = true
	expectNoMetadataLocks(mock)

	m.collectBlockingTransactionsOrWarn(t.Context())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatementConsumerNeedsItsInstrumentToo(t *testing.T) {
	t.Parallel()

	// events_transactions_current also needs the transaction instrument, off on stock MariaDB.
	m, mock := newMockedRTA(t)
	mock.ExpectQuery("setup_consumers").WithArgs("events_statements_current").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_consumers").WithArgs("events_transactions_current").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_instruments").WithArgs("transaction").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("NO"))

	m.checkStatementConsumers(t.Context())

	// sqlmock fails on an unmet expectation, so the instrument must be consulted.
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatementConsumerInstrumentIsOnlyCheckedWhenDeclared(t *testing.T) {
	t.Parallel()

	// A consumer without an instrument, or one already off, must not probe setup_instruments.
	for _, consumer := range statementConsumers {
		if consumer.name == "events_statements_current" {
			assert.Empty(t, consumer.instrument, "the statement instruments are on by default everywhere")
		}
	}

	m, mock := newMockedRTA(t)
	expectStatementConsumerProbes(mock, "NO")

	m.checkStatementConsumers(t.Context())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDescribeDisabledNamesTheHalfThatIsOff(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{l: logrus.NewEntry(logrus.New())}

	assert.Contains(t, m.describeDisabled("c", false, "i", true), "The c consumer is disabled")
	assert.Contains(t, m.describeDisabled("c", true, "i", false), "The i instrument is disabled")
	both := m.describeDisabled("c", false, "i", false)
	assert.Contains(t, both, "c consumer")
	assert.Contains(t, both, "i instrument")
}

func TestStartupWarningsAreKeptForTheRunningStatus(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("setup_consumers").WithArgs("events_statements_current").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("NO"))
	mock.ExpectQuery("setup_consumers").WithArgs("events_transactions_current").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_instruments").WithArgs("transaction").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("NO"))

	m.checkStatementConsumers(t.Context())
	m.checkMetadataLockInstrument(t.Context())

	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, m.warnings, 2)
	assert.Contains(t, m.warnings[0], "The events_statements_current consumer is disabled")
	assert.Contains(t, m.warnings[1], "The "+metadataLockInstrument+" instrument is disabled")
}

func TestStartupWarningsAreEmptyWhenEverythingIsEnabled(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("setup_consumers").WithArgs("events_statements_current").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_consumers").WithArgs("events_transactions_current").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_instruments").WithArgs("transaction").
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))
	mock.ExpectQuery("setup_instruments").WithArgs(metadataLockInstrument).
		WillReturnRows(sqlmock.NewRows([]string{"ENABLED"}).AddRow("YES"))

	m.checkStatementConsumers(t.Context())
	m.checkMetadataLockInstrument(t.Context())

	require.NoError(t, mock.ExpectationsWereMet())
	assert.Empty(t, m.warnings)
}

func TestConnectionFailureIsReportedWithItsReason(t *testing.T) {
	t.Parallel()

	// Nothing listens on port 1, so the ping fails at once with "connection refused".
	m := New(&Params{
		AgentID:         "agent-id",
		DSN:             "pmm:secret@tcp(127.0.0.1:1)/?timeout=2s",
		CollectInterval: time.Second,
	}, logrus.NewEntry(logrus.New()))

	go m.Run(t.Context())

	var changes []agents.Change
	for change := range m.Changes() {
		changes = append(changes, change)
	}

	require.Len(t, changes, 2)
	assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_STARTING, changes[0].Status)
	assert.Empty(t, changes[0].StatusMessage)

	last := changes[1]
	assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, last.Status)
	assert.Contains(t, last.StatusMessage, "Cannot connect to MySQL")
	assert.Contains(t, last.StatusMessage, "127.0.0.1:1")
	assert.NotContains(t, last.StatusMessage, "secret", "the password must never reach the UI")
}

func TestStatementTextTakesTheLongerStatementText(t *testing.T) {
	t.Parallel()

	statement := "SELECT '" + strings.Repeat("x", 3000) + "'"
	processlist := statement[:processlistInfoMaxLength]

	m := &MySQLRTA{sqlTextMaxLength: 4096}
	text, truncated := m.statementText(processlist, statement)
	assert.Equal(t, statement, text, "SQL_TEXT carries the statement past the processlist's fixed limit")
	assert.False(t, truncated)

	// With the server default both sources stop at 1024 bytes.
	m = &MySQLRTA{sqlTextMaxLength: 1024}
	text, truncated = m.statementText(processlist, statement[:1024])
	assert.Equal(t, processlist, text)
	assert.True(t, truncated)

	// The events_statements_current consumer is off: only the processlist text is there.
	text, truncated = (&MySQLRTA{}).statementText(processlist, "")
	assert.Equal(t, processlist, text)
	assert.True(t, truncated, "a processlist text at its fixed limit is cut, whatever the variable says")
}

func TestStatementTextIgnoresAnUnrelatedStatementText(t *testing.T) {
	t.Parallel()

	// In a stored program, SQL_TEXT is the inner statement, not the CALL the processlist shows.
	call := "CALL p('" + strings.Repeat("x", 1100) + "')"
	inner := "UPDATE t SET a = 1 WHERE b = '" + strings.Repeat("y", 2000) + "'"

	m := &MySQLRTA{sqlTextMaxLength: 4096}
	text, truncated := m.statementText(call[:processlistInfoMaxLength], inner)
	assert.Equal(t, call[:processlistInfoMaxLength], text)
	assert.True(t, truncated)
}

func TestShortStatementTextIsNotTruncated(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{sqlTextMaxLength: 1024}
	text, truncated := m.statementText("SELECT 1", "SELECT 1")
	assert.Equal(t, "SELECT 1", text)
	assert.False(t, truncated)
}

func TestQueryTextIsBoundedOnARuneBoundary(t *testing.T) {
	t.Parallel()

	// An odd-length prefix and two-byte runes put maxQueryTextLength in the middle of one.
	statement := "SELECT 'a" + strings.Repeat("é", maxQueryTextLength) + "'"
	require.False(t, utf8.RuneStart(statement[maxQueryTextLength]))
	m := &MySQLRTA{sqlTextMaxLength: 1048576}

	text, truncated := m.statementText(statement[:processlistInfoMaxLength], statement)
	assert.True(t, truncated)
	assert.LessOrEqual(t, len(text), maxQueryTextLength)
	assert.Greater(t, len(text), maxQueryTextLength-utf8.UTFMax)
	assert.True(t, utf8.ValidString(text), "the text must not end in half a character")
	assert.True(t, strings.HasPrefix(statement, text))
}

func TestBuildQueryDataReportsTruncatedText(t *testing.T) {
	t.Parallel()

	statement := "SELECT '" + strings.Repeat("x", 2000) + "'"
	m := &MySQLRTA{sqlTextMaxLength: 1024}
	row := map[string]any{
		"conn_id":           int64(7),
		"current_statement": statement[:processlistInfoMaxLength],
		"current_sql_text":  statement[:1024],
	}

	qd := m.buildQueryData(row, nil)
	assert.Len(t, qd.QueryText, processlistInfoMaxLength)
	assert.True(t, qd.GetMySqlPayload().GetQueryTextTruncated())

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(qd.QueryRawJson), &parsed))
	assert.NotContains(t, parsed, "current_sql_text", "the helper column must not duplicate the statement in the raw payload")
	assert.Contains(t, parsed, "current_statement")
}

func TestBlockingQueryTruncationIsReported(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	m.sqlTextMaxLength = 1024
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_500_000, 2_000_000, "Query", "u@h", strings.Repeat("x", processlistInfoMaxLength), "db.t", "PRIMARY", "X", "X", nil, nil).
		AddRow(nil, 412, 410, 1_500_000, 2_000_000, "Query", "u@h", "SELECT 1 FOR UPDATE", "db.t", "PRIMARY", "X", "X", nil, nil))

	graph, err := readRowLocks(t, m)
	require.NoError(t, err)
	require.Len(t, graph.blockers["411"], 1)
	assert.True(t, graph.blockers["411"][0].BlockingQueryTruncated)
	require.Len(t, graph.blockers["412"], 1)
	assert.False(t, graph.blockers["412"][0].BlockingQueryTruncated)
}

func TestLockTimeIsConvertedFromPicoseconds(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{}
	for _, tc := range []struct {
		picos    any
		expected time.Duration
	}{
		// LOCK_TIME is reported in picoseconds but measured in whole microseconds.
		{picos: int64(3_000_000), expected: 3 * time.Microsecond},
		{picos: int64(1_250_000_000), expected: 1250 * time.Microsecond},
		{picos: int64(2_000_000_000_000), expected: 2 * time.Second},
		{picos: int64(0), expected: 0},
	} {
		qd := m.buildQueryData(map[string]any{"conn_id": int64(1), "current_statement": "SELECT 1", "lock_latency": tc.picos}, nil)
		lockTime := qd.GetMySqlPayload().GetLockTime()
		require.NotNil(t, lockTime, "a measured zero is a value, not an absence")
		assert.Equal(t, tc.expected, lockTime.AsDuration())
	}
}

func TestLockTimeIsUnsetWhenNotMeasured(t *testing.T) {
	t.Parallel()

	qd := (&MySQLRTA{}).buildQueryData(map[string]any{"conn_id": int64(1), "current_statement": "SELECT 1", "lock_latency": nil}, nil)
	assert.Nil(t, qd.GetMySqlPayload().GetLockTime())
}

// waitOnConnection411 returns a graph where 411 waits on a row lock held by 409.
func waitOnConnection411(waitingEventID sql.NullInt64, waitingQuery string) *blockingGraph {
	return &blockingGraph{
		blockers: map[string][]*rtav1.BlockingTransaction{
			"411": {{BlockingConnId: 409, BlockingCommand: "Sleep", Root: true}},
		},
		waiters: map[string]waiterLock{"411": {
			lockType:       rtav1.LockType_LOCK_TYPE_ROW,
			lockedTable:    "sbtest.sbtest1",
			lockedIndex:    "PRIMARY",
			requestedMode:  "X,REC_NOT_GAP",
			waitingEventID: waitingEventID,
			waitingQuery:   waitingQuery,
		}},
		complete: true,
	}
}

func incomplete(graph *blockingGraph) *blockingGraph {
	graph.complete = false
	return graph
}

func TestWaitIsAttachedOnlyToTheStatementThatWaited(t *testing.T) {
	t.Parallel()

	const update = "UPDATE sbtest1 SET k=k+1 WHERE id=1"

	for _, tc := range []struct {
		name     string
		row      map[string]any
		graph    *blockingGraph
		expected rtav1.BlockedStatus
	}{
		{
			// The connection moved on from the SELECT to the UPDATE between the two reads.
			name:     "LaterStatementByEventID",
			row:      map[string]any{"conn_id": int64(411), "current_statement": "SELECT c FROM sbtest2 WHERE id=7", "statement_event_id": int64(850)},
			graph:    waitOnConnection411(sql.NullInt64{Int64: 900, Valid: true}, update),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_UNATTRIBUTED,
		},
		{
			// The same text run again is still a different statement event.
			name:     "SameTextDifferentEventID",
			row:      map[string]any{"conn_id": int64(411), "current_statement": update, "statement_event_id": int64(850)},
			graph:    waitOnConnection411(sql.NullInt64{Int64: 900, Valid: true}, update),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_UNATTRIBUTED,
		},
		{
			name:     "SameEventID",
			row:      map[string]any{"conn_id": int64(411), "current_statement": update, "statement_event_id": int64(900)},
			graph:    waitOnConnection411(sql.NullInt64{Int64: 900, Valid: true}, update),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED,
		},
		{
			// MariaDB ships events_statements_current off, so only the texts can be compared.
			name:     "LaterStatementByText",
			row:      map[string]any{"conn_id": int64(411), "current_statement": "SELECT c FROM sbtest2 WHERE id=7"},
			graph:    waitOnConnection411(sql.NullInt64{}, update),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_UNATTRIBUTED,
		},
		{
			name:     "SameText",
			row:      map[string]any{"conn_id": int64(411), "current_statement": update},
			graph:    waitOnConnection411(sql.NullInt64{}, update),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED,
		},
		{
			// The processlist and INNODB_TRX cut a long statement at different lengths.
			name:     "TruncatedText",
			row:      map[string]any{"conn_id": int64(411), "current_statement": update},
			graph:    waitOnConnection411(sql.NullInt64{}, update[:12]),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED,
		},
		{
			// The source that answered saw this connection wait, even with the other one missing.
			name:     "LaterStatementIncompleteGraph",
			row:      map[string]any{"conn_id": int64(411), "current_statement": "SELECT c FROM sbtest2 WHERE id=7", "statement_event_id": int64(850)},
			graph:    incomplete(waitOnConnection411(sql.NullInt64{Int64: 900, Valid: true}, update)),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_UNATTRIBUTED,
		},
		{
			// Nothing to compare: the wait is kept rather than discarded on no evidence.
			name:     "NoIdentifiers",
			row:      map[string]any{"conn_id": int64(411), "current_statement": update},
			graph:    waitOnConnection411(sql.NullInt64{}, ""),
			expected: rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := (&MySQLRTA{}).buildQueryData(tc.row, tc.graph).GetMySqlPayload()
			assert.Equal(t, tc.expected, payload.BlockedStatus)

			if tc.expected == rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED {
				require.Len(t, payload.BlockedBy, 1)
				assert.Equal(t, "sbtest.sbtest1", payload.LockedTable)

				return
			}

			assert.Empty(t, payload.BlockedBy, "a statement that never waited must not name a blocker")
			assert.Empty(t, payload.LockedTable)
			assert.Equal(t, rtav1.LockType_LOCK_TYPE_UNSPECIFIED, payload.LockType)
		})
	}
}

func TestLockEdgesCarryTheWaitingStatement(t *testing.T) {
	t.Parallel()

	m, mock := newMockedRTA(t)
	mock.ExpectQuery("data_lock_waits").WillReturnRows(blockingRows().
		AddRow(nil, 411, 409, 1_500_000, 2_000_000, "Sleep", "u@h", "SELECT 1 FOR UPDATE", "db.t", "PRIMARY", "X", "X", 900, "UPDATE t SET v=1"))
	mock.ExpectQuery("metadata_locks").WillReturnRows(metadataRows().
		AddRow(nil, 412, 409, "SHARED_WRITE", "SHARED_READ", "Sleep", "u@h", nil, "SELECT 1", "db.t", 950, "INSERT INTO t VALUES (1)"))

	graph := m.collectBlockingTransactionsOrWarn(t.Context())
	require.NotNil(t, graph)
	assert.Equal(t, sql.NullInt64{Int64: 900, Valid: true}, graph.waiters["411"].waitingEventID)
	assert.Equal(t, "UPDATE t SET v=1", graph.waiters["411"].waitingQuery)
	assert.Equal(t, sql.NullInt64{Int64: 950, Valid: true}, graph.waiters["412"].waitingEventID)
	assert.Equal(t, "INSERT INTO t VALUES (1)", graph.waiters["412"].waitingQuery)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEveryLockQueryReturnsTheWaitingStatement(t *testing.T) {
	t.Parallel()

	for _, query := range []string{blockingTransactionsSQL, innodbLockWaitsSQL, metadataLockWaitsSQL} {
		aliases := selectListAliases(query)
		require.GreaterOrEqual(t, len(aliases), 2)
		assert.Equal(t, []string{"AS waiting_event_id", "AS waiting_query"}, aliases[len(aliases)-2:],
			"the scanners read the waiting statement from the last two columns")
	}

	assert.Contains(t, currentQueriesSQLTemplate, "AS statement_event_id")
}

func TestEveryPollingQueryIsTagged(t *testing.T) {
	t.Parallel()

	// perfschema digests strip comments, so the tag must survive as an identifier.
	digestLike := regexp.MustCompile(`/\*.*?\*/`)

	for name, query := range map[string]string{
		"statements": currentQueriesSQLTemplate,
		"row locks":  blockingTransactionsSQL,
		"innodb":     innodbLockWaitsSQL,
		"metadata":   metadataLockWaitsSQL,
	} {
		assert.Contains(t, query, "/* pmm-agent:rta */", name)
		assert.True(t, agents.IsRTAQuery(digestLike.ReplaceAllString(query, "")), "%s must stay recognisable without its comment", name)
	}

	assert.False(t, agents.IsRTAQuery("SELECT THREAD_ID FROM performance_schema.threads t"))
}

func TestQueryTagIsTheFirstColumn(t *testing.T) {
	t.Parallel()

	// DIGEST_TEXT stops at max_digest_length (1024 bytes by default); these queries are longer.
	for name, query := range map[string]string{
		"statements": currentQueriesSQLTemplate,
		"row locks":  blockingTransactionsSQL,
		"innodb":     innodbLockWaitsSQL,
		"metadata":   metadataLockWaitsSQL,
	} {
		assert.Equal(t, "AS "+agents.RTAQueryTag, selectListAliases(query)[0], name)
	}

	// The tag column carries nothing and must not reach the raw payload.
	qd := (&MySQLRTA{}).buildQueryData(map[string]any{agents.RTAQueryTag: nil, "conn_id": int64(1), "current_statement": "SELECT 1"}, nil)
	assert.NotContains(t, qd.QueryRawJson, agents.RTAQueryTag)
}
