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

// Package realtimeanalytics runs built-in Real-Time Analytics Agent for MySQL.
package realtimeanalytics

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	mysqlversion "github.com/percona/pmm/agent/utils/version"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	changesBufferSize = 10
	// Number of picoseconds per nanosecond, used to convert MySQL picosecond latencies into Go durations.
	picosecondsPerNanosecond = 1000
	// Interval used when the server sends none. A non-positive duration makes
	// time.NewTicker panic, and a panic here takes down the whole pmm-agent, so a
	// missing interval degrades to the server's own default instead.
	defaultCollectInterval = 2 * time.Second
	// MySQL's ER_NO_SUCH_TABLE. The performance_schema lock tables the blocking query needs
	// were added in 8.0, so this error means the server will never serve them.
	mysqlErrNoSuchTable = 1146
	// MySQL's ER_TABLEACCESS_DENIED_ERROR. The monitoring user was never granted access to the
	// lock tables, which stays true until someone grants it and restarts the agent.
	mysqlErrTableAccessDenied = 1142
)

// currentQueriesSQL fetches currently running queries from the sys schema.
// The sys.x$processlist view is the machine-readable (raw) version of sys.processlist
// (https://dev.mysql.com/doc/refman/8.4/en/sys-processlist.html); it exposes
// the same columns but with unformatted numeric latencies.
// We select all columns so the complete row is preserved in the raw payload
// (mirroring how the MongoDB RTA agent dumps the whole currentOp document), and
// exclude background threads, idle ("Sleep") connections, the RTA agent's own
// connection and rows without a current statement.
const currentQueriesSQL = `
SELECT *
FROM sys.x$processlist
WHERE conn_id IS NOT NULL
  AND conn_id <> CONNECTION_ID()
  AND current_statement IS NOT NULL
  AND command NOT IN ('Sleep', 'Daemon')`

// blockingTransactionsSQL reads the InnoDB lock-wait graph, so a statement that is stuck can
// be explained in the same payload as the statement itself rather than by a second round trip.
//
// The sys.innodb_lock_waits view would give the same graph in one shot, but the PMM monitoring user
// cannot read it: the sys views call sys stored functions, and PMM's documented grants carry
// SELECT without EXECUTE, so it fails with ERROR 1356. Reusing it would force
// GRANT EXECUTE ON sys.* onto every existing deployment, hence this join over the raw tables,
// which the documented grants already cover.
//
// STRAIGHT_JOIN is load-bearing, not cosmetic. It forces MySQL to drive from data_lock_waits,
// which is empty whenever nothing is blocked, letting the later tables be skipped entirely.
// Without it the optimizer materializes information_schema.innodb_trx first: measured on
// Percona Server 8.0.46 with 48 open transactions and no locks held, 68ms per collection
// against 8ms with the order forced.
//
// The processlist join is a LEFT JOIN on purpose: a blocking transaction whose thread has
// already gone (or an XA transaction with no connection) still explains the wait, so the
// relationship is kept and only the blocker's command/query/user come back empty.
//
// Rows are ordered so that repeated collections of an unchanged lock graph agree with one
// another: performance_schema does not promise an order, and one waiter/blocker pair can
// produce several rows (a record lock and a gap lock on the same row), so without an order
// the deduplicated row -- and the contended index it carries -- would flip between cycles.
//
// The blocking statement is read as current_statement, falling back to last_statement, because
// the head of a blocking chain is typically idle inside an open transaction and is running
// nothing at all -- its last statement is the one that took the lock.
//
// Latencies are taken in microseconds: a 2s collect interval routinely catches waits well
// under a second, and whole-second truncation would report those as "0s".
const blockingTransactionsSQL = `
SELECT STRAIGHT_JOIN
    r.trx_mysql_thread_id AS waiting_conn_id,
    b.trx_mysql_thread_id AS blocking_conn_id,
    TIMESTAMPDIFF(MICROSECOND, r.trx_wait_started, NOW(6)) AS wait_micros,
    TIMESTAMPDIFF(MICROSECOND, b.trx_started, NOW(6)) AS blocker_trx_micros,
    p.command AS blocking_command,
    p.user AS blocking_user,
    COALESCE(p.current_statement, p.last_statement) AS blocking_query,
    CONCAT(rl.OBJECT_SCHEMA, '.', rl.OBJECT_NAME) AS locked_table,
    rl.INDEX_NAME AS locked_index
FROM performance_schema.data_lock_waits w
JOIN information_schema.innodb_trx r ON r.trx_id = w.REQUESTING_ENGINE_TRANSACTION_ID
JOIN information_schema.innodb_trx b ON b.trx_id = w.BLOCKING_ENGINE_TRANSACTION_ID
JOIN performance_schema.data_locks rl ON rl.ENGINE_LOCK_ID = w.REQUESTING_ENGINE_LOCK_ID
LEFT JOIN sys.x$processlist p ON p.conn_id = b.trx_mysql_thread_id
ORDER BY waiting_conn_id, blocking_conn_id, locked_index IS NULL, locked_index`

// MySQLRTA extracts Real-Time Analytics data (currently running DB queries) from MySQL.
type MySQLRTA struct {
	agentID     string
	serviceID   string
	serviceName string
	l           *logrus.Entry

	// Channel to obtain data from this agent.
	changes chan agents.Change

	// dsn to connect to MySQL.
	dsn string
	// files holds TLS certificates to register for the MySQL connection.
	files map[string]string
	// tlsSkipVerify controls TLS certificate validation.
	tlsSkipVerify bool
	// collectInterval is how often to collect data from MySQL.
	collectInterval time.Duration

	// db is the open connection to MySQL, kept between collection cycles.
	db *sql.DB
	// dbInstanceAddress is the monitored instance address parsed from the DSN.
	dbInstanceAddress string
	// blockingUnavailable records that the lock-wait graph could not be read, so a
	// persistent problem is logged once instead of on every collection.
	blockingUnavailable bool
	// blockingUnsupported records that this server will never serve the lock-wait graph
	// (the performance_schema lock tables are 8.0+), so the query is abandoned rather than
	// re-issued every collect interval for the life of the agent.
	blockingUnsupported bool
}

// Params represent Agent parameters.
type Params struct {
	AgentID         string
	DSN             string             // DSN to connect to MySQL.
	ServiceID       string             // ServiceID shall be set in RTA queries to link them to the service.
	ServiceName     string             // ServiceName shall be set in RTA queries to link them to the service.
	CollectInterval time.Duration      // CollectInterval is how often to collect data from MySQL.
	TextFiles       *agentv1.TextFiles // TLS certificate files (optional).
	TLSSkipVerify   bool               // Skip TLS certificate validation.
}

// New creates new MySQLRTA service.
// The DSN is expected to be already rendered by the caller (the supervisor renders
// TLS file templates before constructing the agent).
func New(params *Params, l *logrus.Entry) *MySQLRTA {
	var files map[string]string
	if params.TextFiles != nil {
		files = params.TextFiles.Files
	}

	collectInterval := params.CollectInterval
	if collectInterval <= 0 {
		l.Warnf("No collect interval set for Real-Time Analytics, falling back to %s", defaultCollectInterval)
		collectInterval = defaultCollectInterval
	}

	return &MySQLRTA{
		agentID:         params.AgentID,
		serviceID:       params.ServiceID,
		serviceName:     params.ServiceName,
		dsn:             params.DSN,
		files:           files,
		tlsSkipVerify:   params.TLSSkipVerify,
		collectInterval: collectInterval,
		l:               l,
		changes:         make(chan agents.Change, changesBufferSize),
	}
}

// Run extracts currently running DB queries from MySQL
// and sends it to the channel until ctx is canceled.
func (m *MySQLRTA) Run(ctx context.Context) {
	m.l.Info("Starting MySQL RTA agent")

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	// collectors tracks in-flight collection goroutines so we can wait for them
	// before closing m.changes, avoiding a "send on closed channel" race on shutdown.
	var collectors sync.WaitGroup

	// collecting keeps one collection in flight at a time. Each collection runs on
	// its own pooled connection and the query only excludes its own conn_id, so two
	// overlapping collections would report each other's processlist query as a
	// running query.
	var collecting atomic.Bool

	// terminalStatus is reported just before the changes channel is closed. It stays
	// DONE for a normal stop and becomes INITIALIZATION_ERROR when the agent cannot
	// start (connection failure or unmet prerequisites), so the session surfaces a
	// clear error instead of sitting in RUNNING with no data.
	terminalStatus := inventoryv1.AgentStatus_AGENT_STATUS_DONE
	defer func() {
		collectors.Wait()

		m.changes <- agents.Change{Status: terminalStatus}

		close(m.changes)
	}()

	db, addr, err := createConnection(ctx, m.dsn, m.files, m.tlsSkipVerify)
	if err != nil {
		// A shutdown during initialization is a normal stop, not an initialization failure.
		if ctx.Err() != nil {
			return
		}
		m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)
		terminalStatus = inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR
		return
	}

	defer func() {
		_ = db.Close()
	}()

	m.db = db
	m.dbInstanceAddress = addr

	// Verify the instance can actually serve RTA (not MariaDB, performance_schema on,
	// sys.x$processlist readable) before reporting RUNNING.
	err = m.checkPrerequisites(ctx)
	if err != nil {
		// A shutdown during initialization is a normal stop, not an initialization failure.
		if ctx.Err() != nil {
			return
		}
		m.l.Errorf("Real-Time Analytics is not supported for this instance: %v", err)
		terminalStatus = inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR
		return
	}

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}

	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.l.Info("Stopping MySQL RTA agent")

			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			// m.changes channel will be closed in defer, so we don't need to close it here, just exit the function.
			return
		case <-ticker.C:
			// Skip the tick when the previous collection has not finished; the next
			// one is only a collect interval away and this is a live view.
			if !collecting.CompareAndSwap(false, true) {
				m.l.Debug("Previous processlist collection still running, skipping this tick")
				continue
			}

			// Run collection in a separate goroutine to avoid blocking the main loop
			// and allow timely execution of next ticks in case collection takes longer
			// than the collect interval.
			collectors.Add(1)
			go func(curCtx context.Context) {
				defer collectors.Done()
				defer collecting.Store(false)

				rtaQueryBucket, err := m.collectProcessList(curCtx)
				if err != nil {
					m.l.Warnf("processlist collection failed: %v", err)
					return
				}

				if len(rtaQueryBucket) == 0 {
					return
				}

				// Send and cancellation are selected together: the buffer can be full
				// while nothing drains it during shutdown, and a blocked send would
				// keep Run from ever returning.
				select {
				case <-curCtx.Done():
				case m.changes <- agents.Change{RTAQueriesBucket: rtaQueryBucket}:
				}
			}(ctx)
		}
	}
}

// checkPrerequisites verifies that the target instance can serve Real-Time Analytics:
//   - it must be Oracle MySQL or Percona Server. MariaDB's performance_schema/sys schema
//     differ (no sys.x$processlist with these columns) and are not supported.
//   - performance_schema must be enabled (sys.x$processlist is backed by it).
//   - sys.x$processlist must be readable by the monitoring user (the view is
//     SQL SECURITY INVOKER, so it requires SELECT on the underlying performance_schema tables).
//
// It returns a descriptive error otherwise, so the session reports a clear status
// instead of silently collecting nothing every cycle.
func (m *MySQLRTA) checkPrerequisites(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	_, vendor, err := mysqlversion.GetMySQLVersion(checkCtx, m.db)
	if err != nil {
		return fmt.Errorf("failed to detect MySQL version: %w", err)
	}
	if vendor == mysqlversion.MariaDBVendor {
		return errors.New("MariaDB is not supported by MySQL Real-Time Analytics")
	}

	var performanceSchema sql.NullInt64
	err = m.db.QueryRowContext(checkCtx, "SELECT @@performance_schema").Scan(&performanceSchema)
	if err != nil {
		return fmt.Errorf("failed to read @@performance_schema: %w", err)
	}
	if performanceSchema.Int64 != 1 {
		return errors.New("performance_schema is disabled; it is required for Real-Time Analytics")
	}

	// Probe the view that the collector uses so missing schema or privileges fail fast.
	rows, err := m.db.QueryContext(checkCtx, "SELECT 1 FROM sys.x$processlist LIMIT 1")
	if err != nil {
		return fmt.Errorf("sys.x$processlist is not accessible: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	return rows.Err()
}

// collectProcessList queries sys.x$processlist and parses the result into a slice of *QueryData.
func (m *MySQLRTA) collectProcessList(ctx context.Context) ([]*rtav1.QueryData, error) {
	queryCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	// An empty processlist is not an error: QueryContext does not return sql.ErrNoRows,
	// it simply yields no rows below, so we only get here on a real query failure.
	rows, err := m.db.QueryContext(queryCtx, currentQueriesSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query sys.x$processlist: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to read processlist columns: %w", err)
	}

	collectTime := timestamppb.New(time.Now())

	var scanned []map[string]any
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		row, err := scanRow(rows, columns)
		if err != nil {
			m.l.Warnf("Failed to scan processlist row: %v", err)
			continue
		}

		scanned = append(scanned, row)
	}

	err = rows.Err()
	if err != nil {
		m.l.Warnf("Failed to iterate processlist rows: %v", err)
		return nil, err
	}

	// Closed before the lock graph is read so the two queries never overlap: the statement
	// list excludes only its own connection, and a second connection querying at the same
	// time would show up in it as a running statement.
	_ = rows.Close()

	// Read second, on purpose. The two reads are milliseconds apart either way, but the order
	// decides which way a stale answer errs. Reading the graph first lets a wait that clears
	// in between attach its blockers to whatever the connection runs next -- a confident
	// "blocked by 409" on a statement that never waited. Reading it second can only miss a
	// wait that started in between, which the next collection picks up.
	graph := m.collectBlockingTransactionsOrWarn(ctx)

	results := make([]*rtav1.QueryData, 0, len(scanned))
	for _, row := range scanned {
		queryData := m.buildQueryData(row, graph)
		queryData.QueryCollectTime = collectTime

		results = append(results, queryData)
	}

	return results, nil
}

// blockingGraph is what one collection learned about which statements are waiting. A nil
// graph means the lock tables could not be read at all, which is deliberately different from
// a graph with no waits in it: the first is ignorance, the second is a verified healthy server.
type blockingGraph struct {
	// blockers holds the transactions holding up each waiting connection, keyed by the
	// waiting connection id.
	blockers map[string][]*rtav1.BlockingTransaction
	// lockedTable and lockedIndex describe the lock each waiting connection asked for. They
	// are a property of the waiter, not of any one blocker: every blocker of a statement is
	// contending over the same requested lock.
	lockedTable map[string]string
	lockedIndex map[string]string
}

// collectBlockingTransactionsOrWarn reads the lock-wait graph and returns nil when it cannot,
// so callers report the difference rather than passing an outage off as a healthy server.
func (m *MySQLRTA) collectBlockingTransactionsOrWarn(ctx context.Context) *blockingGraph {
	if m.blockingUnsupported {
		return nil
	}

	graph, err := m.collectBlockingTransactions(ctx)
	if err != nil {
		// Two failures never heal on their own: the performance_schema lock tables only exist
		// from 8.0, and a privilege the monitoring user was never granted stays ungranted
		// until someone changes it and restarts the agent. Retrying either every collect
		// interval would be tens of thousands of doomed round trips a day, so stop asking and
		// say what would make it work.
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && permanentBlockingError(mysqlErr.Number) {
			m.blockingUnsupported = true
			m.l.Warnf("Blocking transaction details cannot be collected from this instance and will not be retried "+
				"(grant the monitoring user SELECT on performance_schema and restart the agent if this is a privilege problem): %v", err)

			return nil
		}

		// Anything else may be transient, so collection keeps trying. The warning is logged
		// once per outage: repeating it every collect interval would bury the rest of the log.
		if !m.blockingUnavailable {
			m.blockingUnavailable = true
			m.l.Warnf("Blocking transaction details are unavailable: %v", err)
		}

		return nil
	}

	if m.blockingUnavailable {
		m.blockingUnavailable = false
		m.l.Info("Blocking transaction details are available again")
	}

	return graph
}

// permanentBlockingError reports whether a MySQL error means the lock graph will never become
// readable without operator action, making retries pointless.
func permanentBlockingError(number uint16) bool {
	switch number {
	case mysqlErrNoSuchTable, mysqlErrTableAccessDenied:
		return true
	default:
		return false
	}
}

// collectBlockingTransactions returns what the lock graph says about every waiting connection.
func (m *MySQLRTA) collectBlockingTransactions(ctx context.Context) (*blockingGraph, error) {
	queryCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	rows, err := m.db.QueryContext(queryCtx, blockingTransactionsSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query the lock wait graph: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	graph := &blockingGraph{
		blockers:    make(map[string][]*rtav1.BlockingTransaction),
		lockedTable: make(map[string]string),
		lockedIndex: make(map[string]string),
	}
	// waiting collects every connection that is itself waiting, so the transactions at the
	// head of a chain can be told apart from those queued in the middle of it.
	waiting := make(map[int64]struct{})
	// seen keeps one entry per (waiter, blocker) pair. A single pair can produce several rows
	// -- one per contended lock, e.g. a record lock and a gap lock on the same row -- and the
	// same blocker must not be reported twice for one statement. The query's ORDER BY makes
	// the surviving row the same one on every collection.
	seen := make(map[[2]int64]struct{})

	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var waitingConnID, blockingConnID int64
		var waitMicros, blockerTrxMicros sql.NullInt64
		var blockingCommand, blockingUser, blockingQuery, lockedTable, lockedIndex sql.NullString

		err = rows.Scan(&waitingConnID, &blockingConnID, &waitMicros, &blockerTrxMicros,
			&blockingCommand, &blockingUser, &blockingQuery, &lockedTable, &lockedIndex)
		if err != nil {
			m.l.Warnf("Failed to scan lock wait row: %v", err)
			continue
		}

		waiting[waitingConnID] = struct{}{}
		key := strconv.FormatInt(waitingConnID, 10)

		// The contended lock belongs to the waiter and is the same across its blockers, so it
		// is recorded once, from the first row the ORDER BY yields for that connection.
		if _, recorded := graph.lockedTable[key]; !recorded {
			graph.lockedTable[key] = lockedTable.String
			graph.lockedIndex[key] = lockedIndex.String
		}

		pair := [2]int64{waitingConnID, blockingConnID}
		if _, duplicate := seen[pair]; duplicate {
			continue
		}
		seen[pair] = struct{}{}

		graph.blockers[key] = append(graph.blockers[key], &rtav1.BlockingTransaction{
			BlockingConnId:             blockingConnID,
			BlockingQuery:              blockingQuery.String,
			BlockingCommand:            blockingCommand.String,
			BlockingUsername:           blockingUser.String,
			WaitDuration:               microsToDuration(waitMicros),
			BlockerTransactionDuration: microsToDuration(blockerTrxMicros),
		})
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate lock wait rows: %w", err)
	}

	markRootBlockers(graph.blockers, waiting)

	return graph, nil
}

// microsToDuration converts a microsecond column into a duration, leaving it unset when the
// column is NULL. A NULL means the server had no value -- a transaction that stopped waiting
// between the two reads inside the query -- which is not the same as a zero-length wait, and
// reporting it as "0s" would state something untrue.
func microsToDuration(micros sql.NullInt64) *durationpb.Duration {
	if !micros.Valid {
		return nil
	}

	return durationpb.New(time.Duration(micros.Int64) * time.Microsecond)
}

// markRootBlockers flags the blockers that are not themselves waiting for a lock. Those sit at
// the head of the chain, so resolving them is what actually frees the waiter.
//
// There can be more than one: a statement can be held up by several independent transactions
// at once (two holders of a shared lock, say). Every one of them is marked, and it is the
// caller's job not to present a single one as "the" culprit when several are flagged --
// resolving one of two independent holders leaves the statement blocked by the other.
//
// There can also be none, when the lock graph contains a cycle and every participant is
// waiting. Nothing is invented in that case.
//
// Each list is sorted by connection id so repeated collections of an unchanged lock graph
// agree with one another.
func markRootBlockers(blockers map[string][]*rtav1.BlockingTransaction, waiting map[int64]struct{}) {
	for _, list := range blockers {
		for _, blocker := range list {
			_, alsoWaiting := waiting[blocker.BlockingConnId]
			blocker.Root = !alsoWaiting
		}

		slices.SortFunc(list, func(a, b *rtav1.BlockingTransaction) int {
			return cmp.Compare(a.BlockingConnId, b.BlockingConnId)
		})
	}
}

// scanRow scans a single result row into a map keyed by column name. Values are
// coerced to int64/float64 when numeric and to nil for SQL NULLs, so the raw
// payload is human-readable JSON with native types.
func scanRow(rows *sql.Rows, columns []string) (map[string]any, error) {
	rawValues := make([]sql.RawBytes, len(columns))
	scanArgs := make([]any, len(columns))
	for i := range rawValues {
		scanArgs[i] = &rawValues[i]
	}

	err := rows.Scan(scanArgs...)
	if err != nil {
		return nil, err
	}

	row := make(map[string]any, len(columns))
	for i, col := range columns {
		row[col] = coerceValue(rawValues[i])
	}

	return row, nil
}

// coerceValue converts a raw column value into nil (NULL), int64, float64 or string
// so the raw payload renders as human-readable JSON with native types.
//
// It is tuned for the sys.x$processlist columns, whose numeric columns are plain
// integers/decimals. It will reinterpret any numeric-looking string as a number, so
// it is not a general-purpose converter: zero-padded identifiers or values wider than
// int64 would lose their original textual form. None of the processlist columns have
// that shape, but keep this in mind before reusing the helper elsewhere.
func coerceValue(b sql.RawBytes) any {
	if b == nil {
		return nil
	}

	s := string(b)

	i, intErr := strconv.ParseInt(s, 10, 64)
	if intErr == nil {
		return i
	}

	f, floatErr := strconv.ParseFloat(s, 64)
	if floatErr == nil {
		return f
	}

	return s
}

// buildQueryData converts a single sys.x$processlist row into a *QueryData.
// The complete row is preserved in QueryRawJson; a curated subset is exposed
// via the MySQL payload for the details view.
func (m *MySQLRTA) buildQueryData(row map[string]any, graph *blockingGraph) *rtav1.QueryData {
	execDuration := durationpb.New(time.Duration(mapFloat(row, "statement_latency")/picosecondsPerNanosecond) * time.Nanosecond)

	connID := mapString(row, "conn_id")

	// A nil graph means the lock tables could not be read. Reporting NOT_BLOCKED then would
	// dress a monitoring gap up as a healthy server, so the status stays unspecified and the
	// UI can say it does not know rather than that nothing is wrong.
	blockedStatus := rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED
	var blockedBy []*rtav1.BlockingTransaction
	var lockedTable, lockedIndex string

	if graph != nil {
		blockedBy = graph.blockers[connID]
		blockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED
		if len(blockedBy) > 0 {
			blockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED
			lockedTable = graph.lockedTable[connID]
			lockedIndex = graph.lockedIndex[connID]
		}
	}

	mysqlPayload := &rtav1.QueryMySQLData{
		DbInstanceAddress: m.dbInstanceAddress,
		ProgramName:       mapString(row, "program_name"),
		DatabaseName:      mapString(row, "db"),
		Command:           mapString(row, "command"),
		State:             mapString(row, "state"),
		Username:          mapString(row, "user"),
		RowsExamined:      mapInt(row, "rows_examined"),
		RowsSent:          mapInt(row, "rows_sent"),
		FullScan:          strings.EqualFold(mapString(row, "full_scan"), "YES"),
		BlockedStatus:     blockedStatus,
		BlockedBy:         blockedBy,
		LockedTable:       lockedTable,
		LockedIndex:       lockedIndex,
	}

	rawJSON, err := json.MarshalIndent(row, "", "    ")
	if err != nil {
		m.l.Warnf("Failed to marshal raw query data: %v", err)
	}

	return &rtav1.QueryData{
		ServiceId:              m.serviceID,
		ServiceName:            m.serviceName,
		QueryId:                connID,
		QueryText:              mapString(row, "current_statement"),
		QueryRawJson:           string(rawJSON),
		QueryExecutionDuration: execDuration,
		Payload: &rtav1.QueryData_MySqlPayload{
			MySqlPayload: mysqlPayload,
		},
	}
}

// mapString reads a column from the row as a string regardless of its scanned type.
func mapString(row map[string]any, key string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

// mapInt reads a column from the row as an int64.
func mapInt(row map[string]any, key string) int64 {
	switch v := row[key].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	default:
		return 0
	}
}

// mapFloat reads a column from the row as a float64.
func mapFloat(row map[string]any, key string) float64 {
	switch v := row[key].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// Changes returns channel that should be read until it is closed.
func (m *MySQLRTA) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector.
func (m *MySQLRTA) Describe(_ chan<- *prometheus.Desc) {
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (m *MySQLRTA) Collect(_ chan<- prometheus.Metric) {
	// This method is needed to satisfy interface.
}

// check interfaces.
var (
	_ prometheus.Collector = (*MySQLRTA)(nil)
)
