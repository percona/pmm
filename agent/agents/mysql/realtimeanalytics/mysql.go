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
	// ER_SPECIFIC_ACCESS_DENIED_ERROR: information_schema.innodb_trx needs the PROCESS
	// privilege, which the monitoring user either has or does not.
	mysqlErrSpecificAccessDenied = 1227
	// The statement query is bounded by processlistRowLimit. Its result is capped only by
	// max_connections otherwise -- thousands on a busy server -- and every row costs a
	// per-thread memory lookup and a slice of raw JSON on the wire, every collect interval.
	processlistRowLimit = 1000
	// Each lock query is bounded by lockGraphRowLimit. Contention is quadratic in the number of
	// waiters on one object -- 51 open readers behind one DDL already produce 2600 edges -- so
	// a limit is needed, and hitting it is reported rather than passed off as a complete graph.
	lockGraphRowLimit = 5000
)

// currentQueriesSQLTemplate fetches currently running queries from the performance_schema
// tables sys.x$processlist is built on. The row is preserved in the raw payload, mirroring how
// the MongoDB RTA agent dumps the whole currentOp document, and background threads, idle
// ("Sleep") connections, the agent's own connection and rows without a current statement are
// excluded.
//
// The view itself is not used because it is far too expensive to run every collect interval.
// Among its six joins is sys.x$memory_by_thread_by_current_bytes, which groups the whole of
// performance_schema.memory_summary_by_thread_by_event_name and then orders the result, all to
// produce one column. Measured on Percona Server 8.0.46 under a 64-thread sysbench run: 89.6ms
// per collection for the view against 8.7ms for an equivalent query. Selecting a narrower column
// list does not help: MySQL does not eliminate the unused join, and the same measurement gives
// 67ms.
//
// The view's current_memory column is not collected at all. It was the one column costing a
// lookup per row -- summing one row per enabled memory instrument, 383 on a default 8.0 server
// -- and measured 41-49% of the whole query, its single largest cost, growing with both the row
// count and the server's thread count. What it buys is slight: the manual defines it as "the
// number of bytes allocated by the thread", but attribution follows whichever thread performed
// the allocation, so on a real server it is dominated by InnoDB internals (13.6MB of 19.9MB on a
// sampled connection) rather than by anything the statement is doing, and the manual notes the
// per-thread values can even go negative when memory ownership moves between threads.
//
// The row count is bounded, because otherwise it is capped only by max_connections and every row
// carries a slice of raw JSON on the wire, every interval. The limit rarely binds -- only running
// statements are returned, and threads_running sits far below max_connections on a healthy
// server -- but it bounds the pile-up this feature exists for.
//
// The bound is applied in a derived table rather than as a plain ORDER BY ... LIMIT on the whole
// query, so the outer joins run only for the rows that survive truncation. The derived table also
// carries the statement columns, which keeps events_statements_current to one lookup per row
// instead of two: it has to be joined inside for the sort key either way.
//
// Ordering is by statement latency, not PROCESSLIST_TIME. That column counts whole seconds, so in
// a pile-up hundreds of rows tie on the same value and which of them survived truncation would
// flip between collections; TIMER_WAIT is picosecond-grained and is what the UI already shows as
// elapsed time. The connection id breaks any remaining tie so repeated collections of an
// unchanged server agree with one another.
//
// The outer query repeats the PROCESSLIST_INFO filter. The derived table and the join read
// threads at slightly different moments, so a statement that finishes in between would otherwise
// come back with a NULL current_statement and surface as a row with no query text.
//
// Row order beyond that is not depended on: the raw payload is a JSON object keyed by column name
// and the collector maps rows by connection id.
const currentQueriesSQLTemplate = `
SELECT
    pps.THREAD_ID AS thd_id,
    pps.PROCESSLIST_ID AS conn_id,
    IF(pps.NAME IN ('thread/sql/one_connection', 'thread/thread_pool/tp_one_connection'),
       CONCAT(pps.PROCESSLIST_USER, '@', CONVERT(pps.PROCESSLIST_HOST USING utf8mb4)),
       REPLACE(pps.NAME, 'thread/', '')) AS user,
    pps.PROCESSLIST_DB AS db,
    pps.PROCESSLIST_COMMAND AS command,
    pps.PROCESSLIST_STATE AS state,
    pps.PROCESSLIST_TIME AS time,
    pps.PROCESSLIST_INFO AS current_statement,
    IF(sel.END_EVENT_ID IS NULL, sel.TIMER_WAIT, NULL) AS statement_latency,
    sel.LOCK_TIME AS lock_latency,
    sel.ROWS_EXAMINED AS rows_examined,
    sel.ROWS_SENT AS rows_sent,
    sel.ROWS_AFFECTED AS rows_affected,
    sel.CREATED_TMP_TABLES AS tmp_tables,
    sel.CREATED_TMP_DISK_TABLES AS tmp_disk_tables,
    IF(sel.NO_GOOD_INDEX_USED > 0 OR sel.NO_INDEX_USED > 0, 'YES', 'NO') AS full_scan,
    IF(sel.END_EVENT_ID IS NOT NULL, sel.SQL_TEXT, NULL) AS last_statement,
    IF(sel.END_EVENT_ID IS NOT NULL, sel.TIMER_WAIT, NULL) AS last_statement_latency,
    etc.TIMER_WAIT AS trx_latency,
    etc.STATE AS trx_state,
    etc.AUTOCOMMIT AS trx_autocommit,
    conattr_pid.ATTR_VALUE AS pid,
    conattr_progname.ATTR_VALUE AS program_name{{outer_columns}}
FROM (
    SELECT t.THREAD_ID, s.END_EVENT_ID, s.TIMER_WAIT, s.LOCK_TIME, s.ROWS_EXAMINED, s.ROWS_SENT,
           s.ROWS_AFFECTED, s.CREATED_TMP_TABLES, s.CREATED_TMP_DISK_TABLES,
           s.NO_GOOD_INDEX_USED, s.NO_INDEX_USED, s.SQL_TEXT{{inner_columns}}
    FROM performance_schema.threads t
    LEFT JOIN performance_schema.events_statements_current s ON s.THREAD_ID = t.THREAD_ID
    WHERE t.PROCESSLIST_ID IS NOT NULL
      AND t.PROCESSLIST_ID <> CONNECTION_ID()
      AND t.PROCESSLIST_INFO IS NOT NULL
      AND t.PROCESSLIST_COMMAND NOT IN ('Sleep', 'Daemon')
    ORDER BY COALESCE(s.TIMER_WAIT, 0) DESC, t.PROCESSLIST_ID
    LIMIT {{limit}}
) sel
JOIN performance_schema.threads pps ON pps.THREAD_ID = sel.THREAD_ID
LEFT JOIN performance_schema.events_transactions_current etc ON pps.THREAD_ID = etc.THREAD_ID
LEFT JOIN performance_schema.session_connect_attrs conattr_pid
       ON conattr_pid.PROCESSLIST_ID = pps.PROCESSLIST_ID AND conattr_pid.ATTR_NAME = '_pid'
LEFT JOIN performance_schema.session_connect_attrs conattr_progname
       ON conattr_progname.PROCESSLIST_ID = pps.PROCESSLIST_ID AND conattr_progname.ATTR_NAME = 'program_name'{{joins}}
WHERE pps.PROCESSLIST_INFO IS NOT NULL`

// optionalProcesslistColumns are columns sys.x$processlist exposes on some servers and not
// others: EXECUTION_ENGINE arrived in MySQL 8.0.24 and CPU_TIME in 8.0.28. Naming one on a
// server that lacks it fails the whole collection, and the view degrades by simply not having
// the column, so each is probed once at startup and included only when it is really there.
//
// The check is against information_schema rather than the version string because forks and
// distributions report versions inconsistently, while the column either exists or it does not.
var optionalProcesslistColumns = []struct {
	table  string
	column string
	// outer is the SELECT-list entry. inner is what the derived table must carry for it, empty
	// when the column comes from a table the outer query joins directly.
	outer string
	inner string
}{
	{
		table:  "threads",
		column: "EXECUTION_ENGINE",
		outer:  ",\n    pps.EXECUTION_ENGINE AS execution_engine",
	},
	{
		table:  "events_statements_current",
		column: "CPU_TIME",
		outer:  ",\n    sel.CPU_TIME AS cpu_latency",
		inner:  ", s.CPU_TIME",
	},
}

// optionalProcesslistSources are joins whose performance_schema consumer is off by default, in
// which case the table holds no rows at all and every column it feeds comes back NULL. Joining
// them regardless costs a lookup per row for nothing -- measured at about 11% of the query -- so
// each is probed once at startup and joined only when its consumer is on, which keeps the columns
// available to anyone who has deliberately enabled them.
var optionalProcesslistSources = []struct {
	consumer string
	outer    string
	join     string
}{
	{
		consumer: "events_waits_current",
		outer: ",\n    ewc.EVENT_NAME AS last_wait," +
			"\n    IF(ewc.END_EVENT_ID IS NULL AND ewc.EVENT_NAME IS NOT NULL, 'Still Waiting', ewc.TIMER_WAIT) AS last_wait_latency," +
			"\n    ewc.SOURCE AS source",
		join: "\nLEFT JOIN performance_schema.events_waits_current ewc ON pps.THREAD_ID = ewc.THREAD_ID",
	},
	{
		consumer: "events_stages_current",
		outer:    ",\n    IF(sel.END_EVENT_ID IS NULL, ROUND(100 * (estc.WORK_COMPLETED / estc.WORK_ESTIMATED), 2), NULL) AS progress",
		join:     "\nLEFT JOIN performance_schema.events_stages_current estc ON pps.THREAD_ID = estc.THREAD_ID",
	},
}

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
// The processlist joins are LEFT JOINs on purpose: a blocking transaction whose thread has
// already gone (or an XA transaction with no connection) still explains the wait, so the
// relationship is kept and only the blocker's command/query/user come back empty.
//
// The blocker's details come from performance_schema.threads directly rather than through
// sys.x$processlist, for the reason given on currentQueriesSQLTemplate: that view aggregates
// every thread's memory to produce a column nothing here reads. Measured on the same server
// with a lock pile-up live, 12.7ms per collection through the view against 0.2ms for the join
// below, returning the same values.
//
// Rows are ordered so that repeated collections of an unchanged lock graph agree with one
// another: performance_schema does not promise an order, and one waiter/blocker pair can
// produce several rows (a record lock and a gap lock on the same row), so without an order
// the deduplicated row -- and the contended index it carries -- would flip between cycles.
//
// The null-index tiebreak deliberately sorts ahead of blocking_conn_id. The waiter's lock is
// recorded from the first row seen for that connection, and a blocker holding a table-level
// lock reports no index at all; ordering by connection first would let such a blocker, purely
// by having the lower id, leave the waiter with no index recorded.
//
// LIMIT bounds the worst case. The graph is largest during exactly the pile-up this feature
// exists for -- many-to-many contention is quadratic in the number of waiters -- and every
// edge costs two innodb_trx joins and two processlist lookups. Ordering by waiter means a
// truncated result still describes the waiters it does include completely.
//
// The blocking statement is read as the thread's live statement, falling back to the last one
// performance_schema recorded for it, because the head of a blocking chain is typically idle
// inside an open transaction and is running nothing at all -- its last statement is the one
// that took the lock.
//
// The two LOCK_MODE columns say what was asked for and what is held ("X,REC_NOT_GAP",
// "S,GAP", ...). They are what separates a wait on the row itself from a wait on the gap
// before it, which is the distinction that makes an apparently impossible deadlock explicable.
// The blocking lock is looked up by its own ENGINE_LOCK_ID, a primary-key lookup into a table
// the query already reads, measured at 0.8ms on top of the query's other work.
//
// Latencies are taken in microseconds rather than seconds so a wait shorter than a second is
// not truncated to "0s". The precision is not real, though: innodb_trx.trx_wait_started and
// trx_started are second-granularity DATETIME columns, so the fractional part comes from NOW(6)
// alone and a wait is over-reported by up to one second. Treat these as "about this long",
// accurate to a second, not as microsecond measurements.
const blockingTransactionsSQL = `
SELECT STRAIGHT_JOIN
    r.trx_mysql_thread_id AS waiting_conn_id,
    b.trx_mysql_thread_id AS blocking_conn_id,
    TIMESTAMPDIFF(MICROSECOND, r.trx_wait_started, NOW(6)) AS wait_micros,
    TIMESTAMPDIFF(MICROSECOND, b.trx_started, NOW(6)) AS blocker_trx_micros,
    bt.PROCESSLIST_COMMAND AS blocking_command,
    IF(bt.PROCESSLIST_USER IS NULL,
       REPLACE(bt.NAME, 'thread/', ''),
       CONCAT(bt.PROCESSLIST_USER, '@', CONVERT(bt.PROCESSLIST_HOST USING utf8mb4))) AS blocking_user,
    COALESCE(bt.PROCESSLIST_INFO, bs.SQL_TEXT) AS blocking_query,
    CONCAT(rl.OBJECT_SCHEMA, '.', rl.OBJECT_NAME) AS locked_table,
    rl.INDEX_NAME AS locked_index,
    rl.LOCK_MODE AS requested_mode,
    bl.LOCK_MODE AS blocking_mode
FROM performance_schema.data_lock_waits w
JOIN information_schema.innodb_trx r ON r.trx_id = w.REQUESTING_ENGINE_TRANSACTION_ID
JOIN information_schema.innodb_trx b ON b.trx_id = w.BLOCKING_ENGINE_TRANSACTION_ID
JOIN performance_schema.data_locks rl ON rl.ENGINE_LOCK_ID = w.REQUESTING_ENGINE_LOCK_ID
JOIN performance_schema.data_locks bl ON bl.ENGINE_LOCK_ID = w.BLOCKING_ENGINE_LOCK_ID
LEFT JOIN performance_schema.threads bt ON bt.PROCESSLIST_ID = b.trx_mysql_thread_id
LEFT JOIN performance_schema.events_statements_current bs ON bs.THREAD_ID = bt.THREAD_ID
ORDER BY waiting_conn_id, locked_index IS NULL, locked_index, blocking_conn_id
LIMIT 5000`

// metadataLockWaitsSQL reads the metadata-lock (MDL) wait graph. Metadata locks are a wholly
// separate mechanism from InnoDB row locks -- different tables, different lifetimes, different
// remedies -- and a statement waiting on one shows up nowhere in data_lock_waits. Without this
// query the most visible stall MySQL produces, a DDL parked behind an open transaction with
// every later statement on that table queued behind the DDL, is reported as "not blocked".
//
// The sys.schema_table_lock_waits view would give the graph in one shot and is unusable for
// exactly the reason sys.innodb_lock_waits is: it calls sys stored functions, PMM's documented grants carry
// SELECT without EXECUTE, and it fails with ERROR 1356. The raw tables below need nothing the
// monitoring user does not already have.
//
// There is no metadata_locks equivalent of data_lock_waits -- performance_schema records the
// locks but not the waits between them -- so the edges are derived by self-joining PENDING
// requests against GRANTED holders of the same object. STRAIGHT_JOIN drives from the PENDING
// side, which is empty on a healthy server, so the rest of the join is skipped entirely.
//
// The object columns are compared NULL-safely because only TABLE locks carry a schema and name;
// GLOBAL, COMMIT and BACKUP LOCK rows leave both NULL, and plain equality would drop them --
// losing precisely the FLUSH TABLES WITH READ LOCK stall a backup causes.
//
// A granted lock is reported as blocking even when the two modes look compatible (a SHARED_READ
// holder against a SHARED_READ request, say). That is not a bug: MDL grants are queued fairly,
// so once a request for EXCLUSIVE is pending, later SHARED requests queue behind it and cannot
// be granted until the holders ahead of them let go. The SHARED_READ holder really is what the
// whole queue is waiting on, and marking it as such is what points at the transaction to end.
//
// Blocker details come from performance_schema directly rather than through sys.x$processlist.
// That view joins six sources including a per-thread memory aggregate, and on the same server
// and scenario costs 13.6ms against 0.19ms for the form below -- for the same values, since
// threads is already joined here to map thread ids onto connection ids.
//
// No wait duration is reported: performance_schema.metadata_locks records no timestamp of any
// kind, so how long a request has been pending is simply not available. Reporting the waiting
// statement's own elapsed time in its place would be a different number wearing this one's
// label, so the field is left unset and the UI omits it.
//
// The blocker's transaction duration is only reported while that transaction is ACTIVE. The
// events_transactions_current row survives commit with its final timer intact, and reporting
// that would age a finished transaction as though it were still open.
const metadataLockWaitsSQL = `
SELECT STRAIGHT_JOIN
    tw.PROCESSLIST_ID AS waiting_conn_id,
    tg.PROCESSLIST_ID AS blocking_conn_id,
    w.LOCK_TYPE AS requested_mode,
    g.LOCK_TYPE AS blocking_mode,
    tg.PROCESSLIST_COMMAND AS blocking_command,
    IF(tg.PROCESSLIST_USER IS NULL,
       REPLACE(tg.NAME, 'thread/', ''),
       CONCAT(tg.PROCESSLIST_USER, '@', CONVERT(tg.PROCESSLIST_HOST USING utf8mb4))) AS blocking_user,
    IF(tx.STATE = 'ACTIVE', tx.TIMER_WAIT, NULL) AS blocker_trx_picos,
    COALESCE(tg.PROCESSLIST_INFO, st.SQL_TEXT) AS blocking_query,
    CASE
        WHEN w.OBJECT_NAME IS NOT NULL THEN CONCAT(w.OBJECT_SCHEMA, '.', w.OBJECT_NAME)
        WHEN w.OBJECT_SCHEMA IS NOT NULL THEN w.OBJECT_SCHEMA
        ELSE w.OBJECT_TYPE
    END AS locked_table
FROM performance_schema.metadata_locks w
JOIN performance_schema.metadata_locks g
  ON g.OBJECT_TYPE <=> w.OBJECT_TYPE
  AND g.OBJECT_SCHEMA <=> w.OBJECT_SCHEMA
  AND g.OBJECT_NAME <=> w.OBJECT_NAME
  AND g.LOCK_STATUS = 'GRANTED'
  AND g.OWNER_THREAD_ID <> w.OWNER_THREAD_ID
JOIN performance_schema.threads tw ON tw.THREAD_ID = w.OWNER_THREAD_ID
JOIN performance_schema.threads tg ON tg.THREAD_ID = g.OWNER_THREAD_ID
LEFT JOIN performance_schema.events_statements_current st ON st.THREAD_ID = tg.THREAD_ID
LEFT JOIN performance_schema.events_transactions_current tx ON tx.THREAD_ID = tg.THREAD_ID
WHERE w.LOCK_STATUS = 'PENDING'
  AND tw.PROCESSLIST_ID IS NOT NULL
ORDER BY waiting_conn_id, blocking_conn_id
LIMIT 5000`

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
	// currentQueriesSQL is the statement query built for this server, with the columns it
	// actually has. Built once at startup because the answer cannot change while the agent is
	// connected, and probing it per collection would cost a round trip every interval.
	currentQueriesSQL string
	// processlistTruncated remembers that the statement list is being cut short, so a pile-up
	// that lasts hours is reported when it starts and when it clears rather than every two
	// seconds for its whole duration.
	processlistTruncated bool
	// rowLocks and metadataLocks track the two lock sources separately because they fail
	// independently: performance_schema.data_lock_waits arrived in 8.0 while metadata_locks
	// goes back to 5.7, so a server that can never serve one may serve the other perfectly.
	rowLocks      lockSourceState
	metadataLocks lockSourceState
}

// lockSourceState remembers why one lock source stopped answering, so a permanent problem is
// not retried every collect interval and a transient one is not logged every time.
type lockSourceState struct {
	// unavailable records a failure that may yet heal, so the warning is logged once per
	// outage rather than on every collection.
	unavailable bool
	// unsupported records a failure that never heals without operator action -- the table does
	// not exist on this version, or the monitoring user was never granted it -- so the query is
	// abandoned for the life of the agent instead of costing a doomed round trip every tick.
	unsupported bool
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
	// the performance_schema processlist readable) before reporting RUNNING.
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
//   - it must be Oracle MySQL or Percona Server. MariaDB's performance_schema differs and is
//     not supported.
//   - performance_schema must be enabled.
//   - the statement query must run, which needs SELECT on the performance_schema tables it
//     reads. It is assembled first, for the columns this server actually has, and then run.
//   - the metadata lock instrument must be on, or that lock source is disabled rather than
//     left to report an empty table as a healthy server.
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

	// Assemble the statement query for the columns this server has, then run it, so missing
	// schema or privileges fail fast at startup instead of once per collection.
	m.currentQueriesSQL, err = m.buildCurrentQueriesSQL(checkCtx)
	if err != nil {
		return err
	}

	err = m.probeCurrentQueries(checkCtx)
	if err != nil {
		return err
	}

	// Checked last, and it returns nothing: everything it can conclude disables one lock source
	// rather than the agent, so a server that cannot report metadata locks still collects
	// statements and row locks.
	m.checkMetadataLockInstrument(checkCtx)

	return nil
}

// probeCurrentQueries runs the statement query once so missing schema or privileges fail at
// startup instead of once per collection.
//
// It is a function of its own so the rows are closed before the caller runs anything else: the
// pool is capped at a single connection, so a query issued while these rows are still open
// waits on the connection they hold and gets nothing but the context deadline.
func (m *MySQLRTA) probeCurrentQueries(ctx context.Context) error {
	// LIMIT 1 because this only has to prove the query runs: without it the probe pulls every
	// running statement, and pays the per-row memory subquery for each, only to discard them.
	rows, err := m.db.QueryContext(ctx, m.currentQueriesSQL+"\nLIMIT 1")
	if err != nil {
		return fmt.Errorf("the performance_schema processlist is not accessible: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	return rows.Err()
}

// metadataLockInstrument is what records metadata locks. With it disabled -- the default before
// MySQL 8.0 -- performance_schema.metadata_locks is simply empty, so the wait query succeeds and
// returns nothing, which is indistinguishable from a server where nothing is waiting.
const metadataLockInstrument = "wait/lock/metadata/sql/mdl"

// checkMetadataLockInstrument disables the metadata source when the instrument that feeds it is
// off. Without this the source reports success on an empty table and every statement queued
// behind a DDL is published as NOT_BLOCKED -- a monitoring gap wearing the badge of a verified
// healthy server, which is the one outcome this feature must never produce.
//
// The state is read once at startup, like the privilege and version checks around it: turning a
// performance_schema instrument on is a deliberate operator action, and the agent says in its
// log what to change and that a restart picks it up.
//
// It never fails the agent. Everything it can conclude only ever disables one of the two lock
// sources, so an instance that cannot answer still collects statements and row locks.
func (m *MySQLRTA) checkMetadataLockInstrument(ctx context.Context) {
	var enabled string
	err := m.db.QueryRowContext(ctx,
		"SELECT ENABLED FROM performance_schema.setup_instruments WHERE NAME = ?", metadataLockInstrument).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		// The instrument does not exist on this server, so metadata locks are never recorded.
		m.metadataLocks.unsupported = true
		m.l.Warnf("This server has no %s instrument, so metadata lock waits cannot be detected", metadataLockInstrument)

		return
	}
	if err != nil {
		// Never fatal. This check only decides whether one lock source can be trusted, so a
		// monitoring user that cannot read setup_instruments -- a narrower grant than the lock
		// tables themselves need -- keeps collecting statements and row locks instead of
		// having the agent refuse to start.
		m.metadataLocks.unsupported = true
		m.l.Warnf("Could not read the %s instrument state, so metadata lock waits will not be collected: %v",
			metadataLockInstrument, err)

		return
	}

	if !strings.EqualFold(enabled, "YES") {
		m.metadataLocks.unsupported = true
		m.l.Warnf("The %s instrument is disabled, so metadata lock waits cannot be detected. "+
			"Enable it (UPDATE performance_schema.setup_instruments SET ENABLED='YES', TIMED='YES' WHERE NAME='%s', "+
			"or performance_schema_instrument='%s=ON' in the config) and restart the agent",
			metadataLockInstrument, metadataLockInstrument, metadataLockInstrument)
	}
}

// buildCurrentQueriesSQL fills in the columns that only some servers have, so one agent build
// serves every supported version without naming a column that would fail the whole collection.
func (m *MySQLRTA) buildCurrentQueriesSQL(ctx context.Context) (string, error) {
	var outer, inner, joins strings.Builder

	for _, column := range optionalProcesslistColumns {
		var present int
		err := m.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = 'performance_schema' AND TABLE_NAME = ? AND COLUMN_NAME = ?",
			column.table, column.column).Scan(&present)
		if err != nil {
			return "", fmt.Errorf("failed to check for performance_schema.%s.%s: %w", column.table, column.column, err)
		}

		if present == 0 {
			m.l.Debugf("This server has no performance_schema.%s.%s; the column is omitted from the raw payload",
				column.table, column.column)

			continue
		}

		outer.WriteString(column.outer)
		inner.WriteString(column.inner)
	}

	for _, source := range optionalProcesslistSources {
		var enabled string
		err := m.db.QueryRowContext(ctx,
			"SELECT ENABLED FROM performance_schema.setup_consumers WHERE NAME = ?", source.consumer).Scan(&enabled)
		// Any failure to read the consumer -- it does not exist, or the monitoring user cannot
		// select from setup_consumers, which is a narrower grant than the rest of the query needs
		// -- is treated as off. The columns it feeds are extras, so losing them costs far less
		// than refusing to collect anything at all.
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			m.l.Warnf("Could not read the %s consumer state, so its columns are omitted from the raw payload: %v",
				source.consumer, err)

			continue
		}

		if !strings.EqualFold(enabled, "YES") {
			m.l.Debugf("The %s consumer is off, so its columns are omitted from the raw payload", source.consumer)

			continue
		}

		outer.WriteString(source.outer)
		joins.WriteString(source.join)
	}

	return strings.NewReplacer(
		"{{outer_columns}}", outer.String(),
		"{{inner_columns}}", inner.String(),
		"{{joins}}", joins.String(),
		"{{limit}}", strconv.Itoa(processlistRowLimit),
	).Replace(currentQueriesSQLTemplate), nil
}

// collectProcessList queries the performance_schema processlist and parses the result into a
// slice of *QueryData. It relies on checkPrerequisites having built currentQueriesSQL, which
// Run guarantees by returning before the collection loop starts if that fails.
func (m *MySQLRTA) collectProcessList(ctx context.Context) ([]*rtav1.QueryData, error) {
	// One budget for the whole cycle. Giving each query its own would let a stalled server hold
	// the collector's single pooled connection for twice the timeout, dropping twice as many
	// ticks as intended.
	cycleCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	// An empty processlist is not an error: QueryContext does not return sql.ErrNoRows,
	// it simply yields no rows below, so we only get here on a real query failure.
	rows, err := m.db.QueryContext(cycleCtx, m.currentQueriesSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query the performance_schema processlist: %w", err)
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
		case <-cycleCtx.Done():
			return nil, cycleCtx.Err()
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

	// With no statements there is nothing for the lock graph to explain, and Run discards an
	// empty bucket anyway, so an idle instance is spared the join entirely.
	if len(scanned) == 0 {
		return nil, nil
	}

	// Said out loud rather than left for someone to notice a short list: past this point the
	// view shows the longest-running statements and not every statement. Unlike a truncated
	// lock graph this cannot turn into a false "not blocked" -- the rows that are here are
	// still described completely -- so it is reported and collection carries on.
	//
	// Once per episode, not once per collection: this condition lasts as long as the load does,
	// and at a two-second interval saying it every time would bury everything else in the log.
	switch truncated := len(scanned) >= processlistRowLimit; {
	case truncated && !m.processlistTruncated:
		m.processlistTruncated = true
		m.l.Warnf("More than %d statements are running, so only the %d longest-running are collected",
			processlistRowLimit, processlistRowLimit)
	case !truncated && m.processlistTruncated:
		m.processlistTruncated = false
		m.l.Infof("Fewer than %d statements are running again, so all of them are collected", processlistRowLimit)
	}

	// Read second, on purpose. The two reads are milliseconds apart either way, but the order
	// decides which way a stale answer errs. Reading the graph first lets a wait that clears
	// in between attach its blockers to whatever the connection runs next -- a confident
	// "blocked by 409" on a statement that never waited. Reading it second can only miss a
	// wait that started in between, which the next collection picks up.
	graph := m.collectBlockingTransactionsOrWarn(cycleCtx)

	results := make([]*rtav1.QueryData, 0, len(scanned))
	for _, row := range scanned {
		queryData := m.buildQueryData(row, graph)
		queryData.QueryCollectTime = collectTime

		results = append(results, queryData)
	}

	return results, nil
}

// waiterLock describes the lock one waiting connection asked for. Every blocker of a statement
// contends over the same requested lock, so it is recorded once per waiter rather than repeated
// per blocker, and kept as a single value so the four fields cannot drift apart.
type waiterLock struct {
	lockType      rtav1.LockType
	lockedTable   string
	lockedIndex   string
	requestedMode string
}

// blockingGraph is what one collection learned about which statements are waiting. A nil
// graph means no lock source could be read at all, which is deliberately different from a
// graph with no waits in it: the first is ignorance, the second is a verified healthy server.
type blockingGraph struct {
	// blockers holds the transactions holding up each waiting connection, keyed by the
	// waiting connection id.
	blockers map[string][]*rtav1.BlockingTransaction
	// waiters records what each waiting connection asked for. Presence here, not a non-empty
	// blockers entry, is what makes a statement blocked: a holder can be a thread with no
	// connection id to report, and dropping the waiter with it would report the wait as health.
	waiters map[string]waiterLock
	// complete is false when a lock source could not be read, or read only in part. The
	// blockers that were found are still reported, but a connection absent from the graph can
	// no longer be called "not blocked" -- it may be waiting on whatever was missed.
	complete bool
}

func newBlockingGraph() *blockingGraph {
	return &blockingGraph{
		blockers: make(map[string][]*rtav1.BlockingTransaction),
		waiters:  make(map[string]waiterLock),
	}
}

// lockEdge is one "this waiter is held up by this blocker" relationship, in the single shape
// both lock queries reduce to so the graph is assembled once rather than once per source.
//
// Both connection ids are nullable. A waiting XA transaction has no connection, and a metadata
// lock can be held by a background thread, which performance_schema reports with a NULL
// PROCESSLIST_ID. Scanning either into a plain int64 fails the whole row.
type lockEdge struct {
	waitingConnID      sql.NullInt64
	blockingConnID     sql.NullInt64
	waitDuration       *durationpb.Duration
	blockerTrxDuration *durationpb.Duration
	blockingCommand    string
	blockingUser       string
	blockingQuery      string
	lockedTable        string
	lockedIndex        string
	requestedMode      string
	blockingMode       string
}

// lockSource is one of the two independent ways a MySQL statement can be stuck.
type lockSource struct {
	// name identifies the source in log messages, which is what tells an operator which of the
	// two stopped working.
	name string
	// lockType is what a wait found by this source is a wait on.
	lockType rtav1.LockType
	// sql is the query that yields this source's edges.
	sql string
	// rowLimit is the LIMIT the query carries. A result of exactly this many rows describes
	// only some waiters, and TestLockSourceRowLimits keeps the two in step.
	rowLimit int
	// scan reads one result row into the common edge shape.
	scan func(*sql.Rows) (*lockEdge, error)
}

// rowLockSource reads InnoDB row-lock waits: a transaction holding a row the waiter needs.
var rowLockSource = lockSource{
	name:     "row lock",
	lockType: rtav1.LockType_LOCK_TYPE_ROW,
	sql:      blockingTransactionsSQL,
	rowLimit: lockGraphRowLimit,
	scan: func(rows *sql.Rows) (*lockEdge, error) {
		var edge lockEdge
		var waitMicros, blockerTrxMicros sql.NullInt64
		var blockingCommand, blockingUser, blockingQuery sql.NullString
		var lockedTable, lockedIndex, requestedMode, blockingMode sql.NullString

		err := rows.Scan(&edge.waitingConnID, &edge.blockingConnID, &waitMicros, &blockerTrxMicros,
			&blockingCommand, &blockingUser, &blockingQuery, &lockedTable, &lockedIndex,
			&requestedMode, &blockingMode)
		if err != nil {
			return nil, err
		}

		edge.waitDuration = microsToDuration(waitMicros)
		edge.blockerTrxDuration = microsToDuration(blockerTrxMicros)
		edge.blockingCommand = blockingCommand.String
		edge.blockingUser = blockingUser.String
		edge.blockingQuery = blockingQuery.String
		edge.lockedTable = lockedTable.String
		edge.lockedIndex = lockedIndex.String
		edge.requestedMode = requestedMode.String
		edge.blockingMode = blockingMode.String

		return &edge, nil
	},
}

// metadataLockSource reads table metadata-lock waits: the DDL-behind-an-open-transaction stall,
// and everything queued behind that DDL. It reports no wait duration because
// performance_schema.metadata_locks records no timestamp to derive one from, and no locked
// index because a metadata lock is taken on the table as a whole.
var metadataLockSource = lockSource{
	name:     "metadata lock",
	lockType: rtav1.LockType_LOCK_TYPE_METADATA,
	sql:      metadataLockWaitsSQL,
	rowLimit: lockGraphRowLimit,
	scan: func(rows *sql.Rows) (*lockEdge, error) {
		var edge lockEdge
		var blockerTrxPicos sql.NullInt64
		var requestedMode, blockingMode sql.NullString
		var blockingCommand, blockingUser, blockingQuery, lockedTable sql.NullString

		err := rows.Scan(&edge.waitingConnID, &edge.blockingConnID, &requestedMode, &blockingMode,
			&blockingCommand, &blockingUser, &blockerTrxPicos, &blockingQuery, &lockedTable)
		if err != nil {
			return nil, err
		}

		edge.blockerTrxDuration = picosToDuration(blockerTrxPicos)
		edge.blockingCommand = blockingCommand.String
		edge.blockingUser = blockingUser.String
		edge.blockingQuery = blockingQuery.String
		edge.lockedTable = lockedTable.String
		edge.requestedMode = requestedMode.String
		edge.blockingMode = blockingMode.String

		return &edge, nil
	},
}

// collectBlockingTransactionsOrWarn reads both lock sources into one graph and returns nil when
// neither could be read, so callers report the difference rather than passing an outage off as
// a healthy server.
func (m *MySQLRTA) collectBlockingTransactionsOrWarn(ctx context.Context) *blockingGraph {
	graph := newBlockingGraph()
	// waiting collects every connection that is itself waiting, across both sources, so the
	// transactions at the head of a chain can be told apart from those queued in the middle of
	// it. A metadata-lock waiter can be what a row-lock waiter is queued behind and vice versa,
	// so this has to be shared rather than computed per source.
	waiting := make(map[int64]struct{})

	rowOK := m.readLockSource(ctx, rowLockSource, &m.rowLocks, graph, waiting)
	metadataOK := m.readLockSource(ctx, metadataLockSource, &m.metadataLocks, graph, waiting)

	if !rowOK && !metadataOK {
		return nil
	}

	// Only a graph built from both sources can support "this statement is not waiting". With
	// one source missing, the blockers found are still reported and everything else is left
	// unknown rather than declared healthy.
	graph.complete = rowOK && metadataOK

	markRootBlockers(graph.blockers, waiting)

	return graph
}

// readLockSource runs one lock query and merges its edges into the graph, reporting whether the
// source answered completely. A source that fails permanently is never asked again.
func (m *MySQLRTA) readLockSource(ctx context.Context, source lockSource, state *lockSourceState, graph *blockingGraph, waiting map[int64]struct{}) bool {
	if state.unsupported {
		return false
	}

	found, err := m.readLockEdges(ctx, source)
	if err != nil {
		// Two failures never heal on their own: a performance_schema table that does not exist
		// on this server version, and a privilege the monitoring user was never granted, which
		// stays ungranted until someone changes it and restarts the agent. Retrying either
		// every collect interval would be tens of thousands of doomed round trips a day, so
		// stop asking and say what would make it work.
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && permanentBlockingError(mysqlErr.Number) {
			state.unsupported = true
			m.l.Warnf("%s details cannot be collected from this instance and will not be retried "+
				"(grant the monitoring user SELECT on performance_schema and restart the agent if this is a privilege problem): %v",
				source.name, err)

			return false
		}

		// Anything else may be transient, so collection keeps trying. The warning is logged
		// once per outage: repeating it every collect interval would bury the rest of the log.
		if !state.unavailable {
			state.unavailable = true
			m.l.Warnf("%s details are unavailable: %v", source.name, err)
		}

		return false
	}

	if state.unavailable {
		state.unavailable = false
		m.l.Infof("%s details are available again", source.name)
	}

	graph.merge(found, waiting)

	if found.truncated {
		// Logged every time it happens rather than once: unlike a privilege problem this is a
		// property of the current load, so it says something about the incident in progress.
		m.l.Warnf("The %s wait graph hit its %d row limit, so some waiting statements are not described in this collection",
			source.name, source.rowLimit)

		return false
	}

	return true
}

// permanentBlockingError reports whether a MySQL error means a lock source will never become
// readable without operator action, making retries pointless.
func permanentBlockingError(number uint16) bool {
	switch number {
	case mysqlErrNoSuchTable, mysqlErrTableAccessDenied, mysqlErrSpecificAccessDenied:
		return true
	default:
		return false
	}
}

// sourceEdges is one source's contribution, staged before it is merged. A source that fails
// part-way must leave nothing behind: partial rows would claim waiters and lock them out of the
// other source, which may hold the real answer for them.
type sourceEdges struct {
	blockers map[string][]*rtav1.BlockingTransaction
	waiters  map[string]waiterLock
	waiting  map[int64]struct{}
	// truncated records that the query returned every row it was allowed to. Past that point
	// the result describes only some waiters, so silence about the rest is not evidence.
	truncated bool
}

// readLockEdges runs one lock query and returns what it found, without touching the shared
// graph -- the caller merges only after the source has finished successfully.
func (m *MySQLRTA) readLockEdges(ctx context.Context, source lockSource) (*sourceEdges, error) {
	// ctx already carries the collection's remaining budget; a second timeout here would extend
	// the cycle rather than bound it.
	rows, err := m.db.QueryContext(ctx, source.sql)
	if err != nil {
		return nil, fmt.Errorf("failed to query the %s wait graph: %w", source.name, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	found := &sourceEdges{
		blockers: make(map[string][]*rtav1.BlockingTransaction),
		waiters:  make(map[string]waiterLock),
		waiting:  make(map[int64]struct{}),
	}
	// seen keeps one entry per (waiter, blocker) pair. A single pair can produce several rows
	// -- one per contended lock, e.g. a record lock and a gap lock on the same row, or several
	// metadata locks on one table -- and the same blocker must not be reported twice for one
	// statement. Each query's ORDER BY makes the surviving row the same one on every collection.
	seen := make(map[[2]int64]struct{})
	read := 0

	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		read++

		edge, err := source.scan(rows)
		if err != nil {
			m.l.Warnf("Failed to scan %s wait row: %v", source.name, err)
			continue
		}

		// A waiter with no connection id cannot be matched to a statement -- an XA transaction
		// with no session, say -- so there is nothing for the row to explain.
		if !edge.waitingConnID.Valid {
			continue
		}

		key := strconv.FormatInt(edge.waitingConnID.Int64, 10)
		found.waiting[edge.waitingConnID.Int64] = struct{}{}

		// The contended lock belongs to the waiter and is the same across its blockers, so it
		// is recorded once, from the first row the ORDER BY yields for that connection. It is
		// recorded even when the blocker below turns out to be unnameable, because the waiting
		// is a fact about this statement either way.
		if _, ok := found.waiters[key]; !ok {
			found.waiters[key] = waiterLock{
				lockType:      source.lockType,
				lockedTable:   edge.lockedTable,
				lockedIndex:   edge.lockedIndex,
				requestedMode: edge.requestedMode,
			}
		}

		// A metadata lock can be held by a background thread, which has no connection id to
		// report. The wait is still real, so the waiter stays recorded above and only the
		// blocker is left out -- the pane already has a state for "held by something not in
		// this snapshot", which is the honest rendering.
		if !edge.blockingConnID.Valid {
			continue
		}

		pair := [2]int64{edge.waitingConnID.Int64, edge.blockingConnID.Int64}
		if _, duplicate := seen[pair]; duplicate {
			continue
		}
		seen[pair] = struct{}{}

		found.blockers[key] = append(found.blockers[key], &rtav1.BlockingTransaction{
			BlockingConnId:             edge.blockingConnID.Int64,
			BlockingQuery:              edge.blockingQuery,
			BlockingCommand:            edge.blockingCommand,
			BlockingUsername:           edge.blockingUser,
			WaitDuration:               edge.waitDuration,
			BlockerTransactionDuration: edge.blockerTrxDuration,
			BlockingLockMode:           edge.blockingMode,
		})
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate %s wait rows: %w", source.name, err)
	}

	// Contention is quadratic in the number of waiters on one object, so a busy table behind a
	// DDL can produce far more edges than the query is allowed to return. The rows that did
	// arrive are still true; what is lost is the guarantee that every waiter is represented.
	found.truncated = source.rowLimit > 0 && read >= source.rowLimit

	return found, nil
}

// merge folds one source's edges into the shared graph.
func (g *blockingGraph) merge(found *sourceEdges, waiting map[int64]struct{}) {
	for key, lock := range found.waiters {
		// A connection waits on one thing at a time, so a waiter another source already claimed
		// is left to that source. Appending these blockers would build a list whose entries are
		// held under different mechanisms, described by a single lock type that fits only some
		// of them -- and would point the reader at the wrong remedy for the rest.
		if _, claimed := g.waiters[key]; claimed {
			continue
		}

		g.waiters[key] = lock
		if blockers := found.blockers[key]; len(blockers) > 0 {
			g.blockers[key] = blockers
		}
	}

	for connID := range found.waiting {
		waiting[connID] = struct{}{}
	}
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

// picosToDuration converts a picosecond column into a duration, leaving it unset when the
// column is NULL. The performance_schema timers are in picoseconds; a NULL means the server
// had no value to report -- for the blocker's transaction timer, that it has no transaction
// open -- which is not the same as a zero-length one.
func picosToDuration(picos sql.NullInt64) *durationpb.Duration {
	if !picos.Valid {
		return nil
	}

	return durationpb.New(time.Duration(picos.Int64/picosecondsPerNanosecond) * time.Nanosecond)
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
// It is tuned for the processlist columns, whose numeric columns are plain
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

// buildQueryData converts a single processlist row into a *QueryData.
// The complete row is preserved in QueryRawJson; a curated subset is exposed
// via the MySQL payload for the details view.
func (m *MySQLRTA) buildQueryData(row map[string]any, graph *blockingGraph) *rtav1.QueryData {
	execDuration := picosToDuration(sql.NullInt64{Int64: int64(mapFloat(row, "statement_latency")), Valid: true})

	connID := mapString(row, "conn_id")

	// A nil graph means no lock source could be read. Reporting NOT_BLOCKED then would dress a
	// monitoring gap up as a healthy server, so the status stays unspecified and the UI can say
	// it does not know rather than that nothing is wrong. The same applies, per statement, when
	// only one of the two sources answered: blockers that were found are reported, but silence
	// from an incomplete graph is not evidence of health.
	blockedStatus := rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED
	lockType := rtav1.LockType_LOCK_TYPE_UNSPECIFIED
	var blockedBy []*rtav1.BlockingTransaction
	var lockedTable, lockedIndex, requestedLockMode string

	if graph != nil {
		// The waiter record, not the blocker list, is what says this statement is waiting: a
		// holder can be a thread with no connection id, and reading "no blockers" as "not
		// waiting" would turn that into a clean bill of health.
		lock, waiting := graph.waiters[connID]
		switch {
		case waiting:
			blockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED
			blockedBy = graph.blockers[connID]
			lockType = lock.lockType
			lockedTable = lock.lockedTable
			lockedIndex = lock.lockedIndex
			requestedLockMode = lock.requestedMode
		case graph.complete:
			blockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED
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
		LockType:          lockType,
		RequestedLockMode: requestedLockMode,
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
