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
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	changesBufferSize        = 10
	picosecondsPerNanosecond = 1000
	// Fallback when the server sends no interval: time.NewTicker panics on a non-positive one.
	defaultCollectInterval = 2 * time.Second
	// ER_NO_SUCH_TABLE.
	mysqlErrNoSuchTable = 1146
	// ER_TABLEACCESS_DENIED_ERROR.
	mysqlErrTableAccessDenied = 1142
	// ER_SPECIFIC_ACCESS_DENIED_ERROR, e.g. no PROCESS privilege for information_schema.innodb_trx.
	mysqlErrSpecificAccessDenied = 1227
	// Caps the statement list, which is otherwise bounded only by max_connections.
	processlistRowLimit = 1000
	// Caps each lock query: edges grow quadratically with the waiters on one object.
	lockGraphRowLimit = 5000
	// PROCESSLIST_INFO is a fixed 1024 bytes, whatever performance_schema_max_sql_text_length says.
	processlistInfoMaxLength = 1024
	// The server allows SQL text up to 1MB, too much to resend every interval.
	maxQueryTextLength = 64 * 1024
)

// currentQueriesSQLTemplate selects running statements from the tables behind sys.x$processlist,
// skipping background threads, idle connections and the agent's own connection.
//
// The view is not used, nor its current_memory column: the per-thread memory aggregate costs ~10x.
// LIMIT sits in the derived table so the outer joins run only for the rows kept.
// PROCESSLIST_TIME orders rows when statement latency is NULL (events_statements_current off, the
// MariaDB default); the connection id keeps the order stable between collections.
// The outer PROCESSLIST_INFO filter drops statements that finished between the two threads reads.
const currentQueriesSQLTemplate = `
SELECT /* pmm-agent:rta */
    NULL AS pmm_agent_rta,
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
    IF(sel.END_EVENT_ID IS NULL, sel.SQL_TEXT, NULL) AS current_sql_text,
    IF(sel.END_EVENT_ID IS NULL, sel.EVENT_ID, NULL) AS statement_event_id,
    IF(sel.END_EVENT_ID IS NULL, sel.TIMER_WAIT, NULL) AS statement_latency,
    sel.LOCK_TIME AS lock_latency,
    sel.ROWS_EXAMINED AS rows_examined,
    sel.ROWS_SENT AS rows_sent,
    sel.ROWS_AFFECTED AS rows_affected,
    sel.CREATED_TMP_TABLES AS tmp_tables,
    sel.CREATED_TMP_DISK_TABLES AS tmp_disk_tables,
    CASE
        WHEN sel.NO_GOOD_INDEX_USED IS NULL AND sel.NO_INDEX_USED IS NULL THEN NULL
        WHEN sel.NO_GOOD_INDEX_USED > 0 OR sel.NO_INDEX_USED > 0 THEN 'YES'
        ELSE 'NO'
    END AS full_scan,
    IF(sel.END_EVENT_ID IS NOT NULL, sel.SQL_TEXT, NULL) AS last_statement,
    IF(sel.END_EVENT_ID IS NOT NULL, sel.TIMER_WAIT, NULL) AS last_statement_latency,
    etc.TIMER_WAIT AS trx_latency,
    etc.STATE AS trx_state,
    etc.AUTOCOMMIT AS trx_autocommit,
    conattr_pid.ATTR_VALUE AS pid,
    conattr_progname.ATTR_VALUE AS program_name{{outer_columns}}
FROM (
    SELECT t.THREAD_ID, s.EVENT_ID, s.END_EVENT_ID, s.TIMER_WAIT, s.LOCK_TIME, s.ROWS_EXAMINED, s.ROWS_SENT,
           s.ROWS_AFFECTED, s.CREATED_TMP_TABLES, s.CREATED_TMP_DISK_TABLES,
           s.NO_GOOD_INDEX_USED, s.NO_INDEX_USED, s.SQL_TEXT{{inner_columns}}
    FROM performance_schema.threads t
    LEFT JOIN performance_schema.events_statements_current s ON s.THREAD_ID = t.THREAD_ID
    WHERE t.PROCESSLIST_ID IS NOT NULL
      AND t.PROCESSLIST_ID <> CONNECTION_ID()
      AND t.PROCESSLIST_INFO IS NOT NULL
      AND t.PROCESSLIST_COMMAND NOT IN ('Sleep', 'Daemon')
    ORDER BY COALESCE(s.TIMER_WAIT, 0) DESC, COALESCE(t.PROCESSLIST_TIME, 0) DESC, t.PROCESSLIST_ID
    LIMIT {{limit}}
) sel
JOIN performance_schema.threads pps ON pps.THREAD_ID = sel.THREAD_ID
LEFT JOIN performance_schema.events_transactions_current etc ON pps.THREAD_ID = etc.THREAD_ID
LEFT JOIN performance_schema.session_connect_attrs conattr_pid
       ON conattr_pid.PROCESSLIST_ID = pps.PROCESSLIST_ID AND conattr_pid.ATTR_NAME = '_pid'
LEFT JOIN performance_schema.session_connect_attrs conattr_progname
       ON conattr_progname.PROCESSLIST_ID = pps.PROCESSLIST_ID AND conattr_progname.ATTR_NAME = 'program_name'{{joins}}
WHERE pps.PROCESSLIST_INFO IS NOT NULL`

// optionalProcesslistColumns exist only on newer servers (EXECUTION_ENGINE since 8.0.24, CPU_TIME
// since 8.0.28) and are probed at startup, since naming a missing column fails the whole query.
var optionalProcesslistColumns = []struct {
	table  string
	column string
	// outer is the SELECT-list entry; inner is what the derived table must carry for it, if anything.
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

// optionalProcesslistSources are joined only when their consumer (off by default) is on: an empty
// table still costs a lookup per row (~11% of the query).
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

// blockingTransactionsSQL reads the InnoDB row-lock wait graph from performance_schema (MySQL 8.0+).
//
// The sys.innodb_lock_waits view is not used: it needs EXECUTE on sys, which PMM's grants lack.
// STRAIGHT_JOIN drives from data_lock_waits, empty when nothing waits; without it the optimizer
// materializes innodb_trx first, ~8x slower.
// The thread joins are LEFT so a blocker with no thread (gone, or XA) still explains the wait.
// The blocker's last statement stands in when it runs none: a chain's head is often idle in its trx.
// ORDER BY keeps the deduplicated row stable between collections; rows with an index sort before
// blocking_conn_id so the waiter's recorded lock keeps its index.
// Durations are in microseconds but accurate only to a second: innodb_trx timestamps are DATETIME.
const blockingTransactionsSQL = `
SELECT /* pmm-agent:rta */ STRAIGHT_JOIN
    NULL AS pmm_agent_rta,
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
    bl.LOCK_MODE AS blocking_mode,
    IF(ws.END_EVENT_ID IS NULL, ws.EVENT_ID, NULL) AS waiting_event_id,
    r.trx_query AS waiting_query
FROM performance_schema.data_lock_waits w
JOIN information_schema.innodb_trx r ON r.trx_id = w.REQUESTING_ENGINE_TRANSACTION_ID
JOIN information_schema.innodb_trx b ON b.trx_id = w.BLOCKING_ENGINE_TRANSACTION_ID
JOIN performance_schema.data_locks rl ON rl.ENGINE_LOCK_ID = w.REQUESTING_ENGINE_LOCK_ID
JOIN performance_schema.data_locks bl ON bl.ENGINE_LOCK_ID = w.BLOCKING_ENGINE_LOCK_ID
LEFT JOIN performance_schema.threads bt ON bt.PROCESSLIST_ID = b.trx_mysql_thread_id
LEFT JOIN performance_schema.events_statements_current bs ON bs.THREAD_ID = bt.THREAD_ID
LEFT JOIN performance_schema.events_statements_current ws ON ws.THREAD_ID = w.REQUESTING_THREAD_ID
ORDER BY waiting_conn_id, locked_index IS NULL, locked_index, blocking_conn_id
LIMIT 5000`

// innodbLockWaitsSQL reads the same graph from the information_schema tables of MariaDB and MySQL
// before 8.0; it must return blockingTransactionsSQL's columns in the same order (scanRowLockEdge).
//
// The lock_table column arrives quoted (`db`.`tbl`), so its backticks are stripped to match.
// STRAIGHT_JOIN drives from INNODB_LOCK_WAITS for the same reason as in blockingTransactionsSQL.
const innodbLockWaitsSQL = `
SELECT /* pmm-agent:rta */ STRAIGHT_JOIN
    NULL AS pmm_agent_rta,
    r.trx_mysql_thread_id AS waiting_conn_id,
    b.trx_mysql_thread_id AS blocking_conn_id,
    TIMESTAMPDIFF(MICROSECOND, r.trx_wait_started, NOW(6)) AS wait_micros,
    TIMESTAMPDIFF(MICROSECOND, b.trx_started, NOW(6)) AS blocker_trx_micros,
    bt.PROCESSLIST_COMMAND AS blocking_command,
    IF(bt.PROCESSLIST_USER IS NULL,
       REPLACE(bt.NAME, 'thread/', ''),
       CONCAT(bt.PROCESSLIST_USER, '@', CONVERT(bt.PROCESSLIST_HOST USING utf8mb4))) AS blocking_user,
    COALESCE(bt.PROCESSLIST_INFO, bs.SQL_TEXT) AS blocking_query,
    REPLACE(rl.lock_table, '` + "`" + `', '') AS locked_table,
    rl.lock_index AS locked_index,
    rl.lock_mode AS requested_mode,
    bl.lock_mode AS blocking_mode,
    IF(ws.END_EVENT_ID IS NULL, ws.EVENT_ID, NULL) AS waiting_event_id,
    r.trx_query AS waiting_query
FROM information_schema.INNODB_LOCK_WAITS w
JOIN information_schema.INNODB_TRX r ON r.trx_id = w.requesting_trx_id
JOIN information_schema.INNODB_TRX b ON b.trx_id = w.blocking_trx_id
JOIN information_schema.INNODB_LOCKS rl ON rl.lock_id = w.requested_lock_id
JOIN information_schema.INNODB_LOCKS bl ON bl.lock_id = w.blocking_lock_id
LEFT JOIN performance_schema.threads bt ON bt.PROCESSLIST_ID = b.trx_mysql_thread_id
LEFT JOIN performance_schema.events_statements_current bs ON bs.THREAD_ID = bt.THREAD_ID
LEFT JOIN performance_schema.threads wt ON wt.PROCESSLIST_ID = r.trx_mysql_thread_id
LEFT JOIN performance_schema.events_statements_current ws ON ws.THREAD_ID = wt.THREAD_ID
ORDER BY waiting_conn_id, locked_index IS NULL, locked_index, blocking_conn_id
LIMIT 5000`

// metadataLockWaitsSQL reads the metadata-lock wait graph (e.g. a DDL behind an open transaction),
// which data_lock_waits never shows; sys.schema_table_lock_waits needs EXECUTE on sys too.
//
// MDL waits are not recorded as such, so edges pair PENDING requests with GRANTED holders of
// the same object; STRAIGHT_JOIN drives from the PENDING side, empty on a healthy server.
// Objects compare NULL-safely: GLOBAL, COMMIT and BACKUP LOCK rows have no schema or name.
// A holder with a compatible mode is still a blocker: MDL grants queue behind a pending EXCLUSIVE.
// The transaction timer is read only while ACTIVE: its row keeps the final value after commit.
const metadataLockWaitsSQL = `
SELECT /* pmm-agent:rta */ STRAIGHT_JOIN
    NULL AS pmm_agent_rta,
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
    END AS locked_table,
    IF(ws.END_EVENT_ID IS NULL, ws.EVENT_ID, NULL) AS waiting_event_id,
    tw.PROCESSLIST_INFO AS waiting_query
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
LEFT JOIN performance_schema.events_statements_current ws ON ws.THREAD_ID = w.OWNER_THREAD_ID
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

	changes chan agents.Change

	dsn string
	// files holds TLS certificates to register for the MySQL connection.
	files           map[string]string
	tlsSkipVerify   bool
	collectInterval time.Duration

	db                *sql.DB
	dbInstanceAddress string
	// currentQueriesSQL is built once at startup for the columns this server has.
	currentQueriesSQL string
	// processlistTruncated keeps the truncation warning to once per episode.
	processlistTruncated bool
	// The two lock sources fail independently, so each keeps its own state.
	rowLocks      lockSourceState
	metadataLocks lockSourceState
	// sqlTextMaxLength is performance_schema_max_sql_text_length, zero when it could not be read.
	sqlTextMaxLength int
	// warnings are startup findings reported in the RUNNING status message.
	warnings []string
	// rowLockSource stays empty until selectRowLockSource finds this server's wait table.
	rowLockSource lockSource
}

// lockSourceState remembers why a lock source stopped answering.
type lockSourceState struct {
	// unavailable marks a possibly transient outage, warned about once until it clears.
	unavailable bool
	// unsupported marks a failure that needs operator action, so the source is not queried again.
	unsupported bool
}

// Params represent Agent parameters.
type Params struct {
	AgentID         string
	DSN             string
	ServiceID       string
	ServiceName     string
	CollectInterval time.Duration
	TextFiles       *agentv1.TextFiles
	TLSSkipVerify   bool
}

// New creates new MySQLRTA service from a DSN whose TLS file templates are already rendered.
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

	// Waited for before m.changes is closed, so no collection sends on a closed channel.
	var collectors sync.WaitGroup

	// One collection at a time: each excludes only its own connection, so two would list each other.
	var collecting atomic.Bool

	// Reported last, so a failed start surfaces as an error instead of RUNNING with no data.
	terminalStatus := inventoryv1.AgentStatus_AGENT_STATUS_DONE
	var terminalMessage string
	defer func() {
		collectors.Wait()

		m.changes <- agents.Change{Status: terminalStatus, StatusMessage: terminalMessage}

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
		terminalMessage = fmt.Sprintf("Cannot connect to MySQL: %v", err)
		return
	}

	defer func() {
		_ = db.Close()
	}()

	m.db = db
	m.dbInstanceAddress = addr

	err = m.checkPrerequisites(ctx)
	if err != nil {
		// A shutdown during initialization is a normal stop, not an initialization failure.
		if ctx.Err() != nil {
			return
		}
		m.l.Errorf("Real-Time Analytics is not supported for this instance: %v", err)
		terminalStatus = inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR
		terminalMessage = fmt.Sprintf("Real-Time Analytics is not supported for this instance: %v", err)
		return
	}

	m.changes <- agents.Change{
		Status:        inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
		StatusMessage: strings.Join(m.warnings, "\n"),
	}

	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.l.Info("Stopping MySQL RTA agent")

			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			return
		case <-ticker.C:
			if !collecting.CompareAndSwap(false, true) {
				m.l.Debug("Previous processlist collection still running, skipping this tick")
				continue
			}

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

				// Nothing drains m.changes during shutdown, so a send that ignored ctx could hang Run.
				select {
				case <-curCtx.Done():
				case m.changes <- agents.Change{RTAQueriesBucket: rtaQueryBucket}:
				}
			}(ctx)
		}
	}
}

// checkPrerequisites fails when RTA cannot run at all and warns about what only narrows collection.
// Checks probe for tables, columns and consumers, not versions: forks report those inconsistently.
func (m *MySQLRTA) checkPrerequisites(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	var performanceSchema sql.NullInt64
	err := m.db.QueryRowContext(checkCtx, "SELECT @@performance_schema").Scan(&performanceSchema)
	if err != nil {
		return fmt.Errorf("failed to read @@performance_schema: %w", err)
	}
	if performanceSchema.Int64 != 1 {
		return errors.New("performance_schema is disabled; it is required for Real-Time Analytics")
	}

	m.readSQLTextMaxLength(checkCtx)

	m.currentQueriesSQL, err = m.buildCurrentQueriesSQL(checkCtx)
	if err != nil {
		return err
	}

	err = m.probeCurrentQueries(checkCtx)
	if err != nil {
		return err
	}

	m.checkStatementConsumers(checkCtx)
	m.selectRowLockSource(checkCtx)
	m.checkMetadataLockInstrument(checkCtx)

	return nil
}

// statementConsumers feed the statement and transaction columns; with one off the query still
// succeeds but those columns are NULL, the stock MariaDB state.
var statementConsumers = []struct {
	name string
	// instrument must be on as well, empty when the consumer alone decides: a consumer whose
	// instrument is off records nothing.
	instrument string
	// lost names what goes uncollected, for the warning.
	lost string
}{
	{
		name: "events_statements_current",
		// statement/sql/% is on by default everywhere, MariaDB included, so the consumer alone decides.
		lost: "statement latency, lock time, row counts, temporary table counts and full-scan detection",
	},
	{
		name:       "events_transactions_current",
		instrument: "transaction",
		lost:       "transaction state, duration and autocommit",
	},
}

// checkStatementConsumers warns about disabled statement consumers without failing the agent: the
// running statements themselves come from performance_schema.threads, which no consumer gates.
func (m *MySQLRTA) checkStatementConsumers(ctx context.Context) {
	for _, consumer := range statementConsumers {
		consumerOn, known := m.switchEnabled(ctx,
			"SELECT ENABLED FROM performance_schema.setup_consumers WHERE NAME = ?",
			consumer.name, "consumer", consumer.lost)
		if !known {
			continue
		}

		instrumentOn := true
		if consumer.instrument != "" {
			instrumentOn, known = m.switchEnabled(ctx,
				"SELECT ENABLED FROM performance_schema.setup_instruments WHERE NAME = ?",
				consumer.instrument, "instrument", consumer.lost)
			if !known {
				continue
			}
		}

		if consumerOn && instrumentOn {
			continue
		}

		// The remedy names both switches: enabling only one still records nothing.
		remedy := fmt.Sprintf("UPDATE performance_schema.setup_consumers SET ENABLED='YES' WHERE NAME='%s'",
			consumer.name)
		config := fmt.Sprintf("performance_schema_consumer_%s=ON", consumer.name)
		if consumer.instrument != "" {
			remedy += fmt.Sprintf("; UPDATE performance_schema.setup_instruments SET ENABLED='YES', TIMED='YES' WHERE NAME='%s'",
				consumer.instrument)
			config += fmt.Sprintf(" and performance_schema_instrument='%s=ON'", consumer.instrument)
		}

		m.warnUserf("%s, so %s are not collected. Enable it (%s, or %s in the config) and restart the Real-Time Analytics session",
			m.describeDisabled(consumer.name, consumerOn, consumer.instrument, instrumentOn), consumer.lost, remedy, config)
	}
}

// warnUserf logs a startup finding and keeps it for the RUNNING status message.
func (m *MySQLRTA) warnUserf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	m.l.Warn(message)
	m.warnings = append(m.warnings, message)
}

// describeDisabled names whichever half of the switch is off.
func (m *MySQLRTA) describeDisabled(consumer string, consumerOn bool, instrument string, instrumentOn bool) string {
	switch {
	case !consumerOn && !instrumentOn:
		return fmt.Sprintf("The %s consumer and the %s instrument are disabled", consumer, instrument)
	case !consumerOn:
		return fmt.Sprintf("The %s consumer is disabled", consumer)
	default:
		return fmt.Sprintf("The %s instrument is disabled and the %s consumer therefore records nothing",
			instrument, consumer)
	}
}

// switchEnabled reads one performance_schema switch; the second result is false when it is unknown.
func (m *MySQLRTA) switchEnabled(ctx context.Context, query, name, kind, lost string) (bool, bool) {
	var enabled string
	err := m.db.QueryRowContext(ctx, query, name).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		m.warnUserf("This server has no %s %s, so %s are not collected", name, kind, lost)

		return false, false
	}
	if err != nil {
		// The setup tables need their own grant, so this says nothing about whether collection works.
		m.l.Warnf("Could not read the %s %s state: %v", name, kind, err)

		return false, false
	}

	return strings.EqualFold(enabled, "YES"), true
}

// selectRowLockSource picks the row-lock query by which wait table this server has (data_lock_waits
// on MySQL 8.0+, INNODB_LOCK_WAITS on MariaDB and older MySQL) and reports whether it chose one.
func (m *MySQLRTA) selectRowLockSource(ctx context.Context) bool {
	probeFailed := false

	for _, candidate := range rowLockSources {
		var present int
		err := m.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?",
			candidate.schema, candidate.table).Scan(&present)
		if err != nil {
			// An error is not evidence of absence, so it must not disable row locks for good.
			probeFailed = true
			m.l.Warnf("Could not check for %s.%s: %v", candidate.schema, candidate.table, err)

			continue
		}

		if present > 0 {
			m.rowLockSource = candidate.source
			m.l.Debugf("Reading row-lock waits from %s.%s", candidate.schema, candidate.table)

			return true
		}
	}

	if probeFailed {
		m.l.Warn("Could not determine where this server records InnoDB row-lock waits; will try again")

		return false
	}

	m.rowLocks.unsupported = true
	m.warnUserf("This server records InnoDB row-lock waits in neither performance_schema.data_lock_waits " +
		"nor information_schema.INNODB_LOCK_WAITS, so row-lock waits cannot be detected")

	return false
}

// readSQLTextMaxLength reads performance_schema_max_sql_text_length, leaving zero on failure.
func (m *MySQLRTA) readSQLTextMaxLength(ctx context.Context) {
	var length sql.NullInt64
	err := m.db.QueryRowContext(ctx, "SELECT @@performance_schema_max_sql_text_length").Scan(&length)
	if err != nil {
		m.l.Debugf("Could not read performance_schema_max_sql_text_length: %v", err)

		return
	}

	m.sqlTextMaxLength = int(length.Int64)
}

// statementText extends a capped PROCESSLIST_INFO with SQL_TEXT when SQL_TEXT starts with it (in a
// stored program it holds the inner statement, not the CALL) and reports if it is still truncated.
func (m *MySQLRTA) statementText(processlistText, statementText string) (string, bool) {
	text := processlistText
	if len(text) >= processlistInfoMaxLength && len(statementText) > len(text) && strings.HasPrefix(statementText, text) {
		text = statementText
	}

	return m.boundedQueryText(text)
}

// boundedQueryText cuts text to maxQueryTextLength and reports whether it is truncated; MySQL marks
// no cut, so a text exactly at a server limit counts as truncated.
func (m *MySQLRTA) boundedQueryText(text string) (string, bool) {
	truncated := len(text) == processlistInfoMaxLength || (m.sqlTextMaxLength > 0 && len(text) >= m.sqlTextMaxLength)

	if len(text) > maxQueryTextLength {
		cut := maxQueryTextLength
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}

		return text[:cut], true
	}

	return text, truncated
}

// probeCurrentQueries runs the statement query once so missing schema or privileges fail at startup.
// It closes its rows before returning: until then they hold the pool's only connection.
func (m *MySQLRTA) probeCurrentQueries(ctx context.Context) error {
	rows, err := m.db.QueryContext(ctx, m.currentQueriesSQL+"\nLIMIT 1")
	if err != nil {
		return fmt.Errorf("the performance_schema processlist is not accessible: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	return rows.Err()
}

// metadataLockInstrument fills performance_schema.metadata_locks; when off (the default before
// MySQL 8.0) the table is empty, which looks like a server with no waits.
const metadataLockInstrument = "wait/lock/metadata/sql/mdl"

// checkMetadataLockInstrument disables the metadata lock source unless its instrument is on, so an
// empty table is never reported as NOT_BLOCKED.
func (m *MySQLRTA) checkMetadataLockInstrument(ctx context.Context) {
	var enabled string
	err := m.db.QueryRowContext(ctx,
		"SELECT ENABLED FROM performance_schema.setup_instruments WHERE NAME = ?", metadataLockInstrument).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		m.metadataLocks.unsupported = true
		m.warnUserf("This server has no %s instrument, so metadata lock waits cannot be detected", metadataLockInstrument)

		return
	}
	if err != nil {
		m.metadataLocks.unsupported = true
		m.warnUserf("Could not read the %s instrument state, so metadata lock waits will not be collected: %v",
			metadataLockInstrument, err)

		return
	}

	if !strings.EqualFold(enabled, "YES") {
		m.metadataLocks.unsupported = true
		m.warnUserf("The %s instrument is disabled, so metadata lock waits cannot be detected. "+
			"Enable it (UPDATE performance_schema.setup_instruments SET ENABLED='YES', TIMED='YES' WHERE NAME='%s', "+
			"or performance_schema_instrument='%s=ON' in the config) and restart the Real-Time Analytics session",
			metadataLockInstrument, metadataLockInstrument, metadataLockInstrument)
	}
}

// buildCurrentQueriesSQL fills in the optional columns and joins this server supports.
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
		// A consumer that is missing or unreadable counts as off: its columns are extras.
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

// collectProcessList reads the running statements and their lock waits.
func (m *MySQLRTA) collectProcessList(ctx context.Context) ([]*rtav1.QueryData, error) {
	// One budget for the whole cycle, so a stalled server holds the only connection for one timeout.
	cycleCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

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

	// Closed before the lock query, which needs the pool's only connection and must not overlap.
	_ = rows.Close()

	if len(scanned) == 0 {
		return nil, nil
	}

	// Warned once per episode rather than every collection, since it lasts as long as the load.
	switch truncated := len(scanned) >= processlistRowLimit; {
	case truncated && !m.processlistTruncated:
		m.processlistTruncated = true
		m.l.Warnf("More than %d statements are running, so only the %d longest-running are collected",
			processlistRowLimit, processlistRowLimit)
	case !truncated && m.processlistTruncated:
		m.processlistTruncated = false
		m.l.Infof("Fewer than %d statements are running again, so all of them are collected", processlistRowLimit)
	}

	// Read after the statement list: the reverse order lets a wait that clears in between attach to
	// the connection's next statement, while this order can only miss a wait the next cycle sees.
	graph := m.collectBlockingTransactionsOrWarn(cycleCtx)

	results := make([]*rtav1.QueryData, 0, len(scanned))
	for _, row := range scanned {
		queryData := m.buildQueryData(row, graph)
		queryData.QueryCollectTime = collectTime

		results = append(results, queryData)
	}

	return results, nil
}

// waiterLock is the lock a waiting connection asked for, shared by all of its blockers.
type waiterLock struct {
	lockType      rtav1.LockType
	lockedTable   string
	lockedIndex   string
	requestedMode string
	// waitingEventID and waitingQuery identify the statement that waited; see waitingStatementMatches.
	waitingEventID sql.NullInt64
	waitingQuery   string
}

// blockingGraph holds one collection's lock waits; a nil graph means no source could be read, which
// must not be reported like a graph with no waits.
type blockingGraph struct {
	// blockers is keyed by the waiting connection id.
	blockers map[string][]*rtav1.BlockingTransaction
	// waiters, not blockers, marks a statement as blocked: a holder may have no connection id.
	waiters map[string]waiterLock
	// complete is false when a source was unread or truncated, so absence proves nothing.
	complete bool
}

func newBlockingGraph() *blockingGraph {
	return &blockingGraph{
		blockers: make(map[string][]*rtav1.BlockingTransaction),
		waiters:  make(map[string]waiterLock),
	}
}

// lockEdge is one waiter-blocker pair in the shape both lock sources scan into. Connection ids are
// nullable: an XA transaction or a background thread has none.
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
	waitingEventID     sql.NullInt64
	waitingQuery       string
}

// lockSource is one of the two independent ways a MySQL statement can be stuck.
type lockSource struct {
	// name identifies the source in log messages.
	name     string
	lockType rtav1.LockType
	sql      string
	// rowLimit must equal the query's LIMIT; TestLockSourceRowLimitsMatchTheirQueries checks it.
	rowLimit int
	scan     func(*sql.Rows) (*lockEdge, error)
}

// dataLockWaitsSource reads InnoDB row-lock waits from performance_schema (MySQL 8.0+).
var dataLockWaitsSource = lockSource{
	name:     "row lock",
	lockType: rtav1.LockType_LOCK_TYPE_ROW,
	sql:      blockingTransactionsSQL,
	rowLimit: lockGraphRowLimit,
	scan:     scanRowLockEdge,
}

// innodbLockWaitsSource reads InnoDB row-lock waits from information_schema (MariaDB, MySQL 5.7).
var innodbLockWaitsSource = lockSource{
	name:     "row lock",
	lockType: rtav1.LockType_LOCK_TYPE_ROW,
	sql:      innodbLockWaitsSQL,
	rowLimit: lockGraphRowLimit,
	scan:     scanRowLockEdge,
}

// rowLockSources are probed in order; performance_schema goes first as it tells gap from record locks.
var rowLockSources = []struct {
	source lockSource
	schema string
	table  string
}{
	{dataLockWaitsSource, "performance_schema", "data_lock_waits"},
	{innodbLockWaitsSource, "information_schema", "INNODB_LOCK_WAITS"},
}

// scanRowLockEdge reads a row of either row-lock query, skipping the agents.RTAQueryTag column.
func scanRowLockEdge(rows *sql.Rows) (*lockEdge, error) {
	var edge lockEdge
	var waitMicros, blockerTrxMicros sql.NullInt64
	var blockingCommand, blockingUser, blockingQuery sql.NullString
	var lockedTable, lockedIndex, requestedMode, blockingMode, waitingQuery sql.NullString

	err := rows.Scan(new(sql.RawBytes), &edge.waitingConnID, &edge.blockingConnID, &waitMicros, &blockerTrxMicros,
		&blockingCommand, &blockingUser, &blockingQuery, &lockedTable, &lockedIndex,
		&requestedMode, &blockingMode, &edge.waitingEventID, &waitingQuery)
	if err != nil {
		return nil, err
	}

	edge.waitingQuery = waitingQuery.String

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
}

// metadataLockSource reads metadata-lock waits, with no wait duration (metadata_locks has no
// timestamp) and no locked index (the lock covers the whole table).
var metadataLockSource = lockSource{
	name:     "metadata lock",
	lockType: rtav1.LockType_LOCK_TYPE_METADATA,
	sql:      metadataLockWaitsSQL,
	rowLimit: lockGraphRowLimit,
	scan: func(rows *sql.Rows) (*lockEdge, error) {
		var edge lockEdge
		var blockerTrxPicos sql.NullInt64
		var requestedMode, blockingMode sql.NullString
		var blockingCommand, blockingUser, blockingQuery, lockedTable, waitingQuery sql.NullString

		err := rows.Scan(new(sql.RawBytes), &edge.waitingConnID, &edge.blockingConnID, &requestedMode, &blockingMode,
			&blockingCommand, &blockingUser, &blockerTrxPicos, &blockingQuery, &lockedTable,
			&edge.waitingEventID, &waitingQuery)
		if err != nil {
			return nil, err
		}

		edge.waitingQuery = waitingQuery.String

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

// collectBlockingTransactionsOrWarn reads both lock sources into one graph, nil if neither answered.
func (m *MySQLRTA) collectBlockingTransactionsOrWarn(ctx context.Context) *blockingGraph {
	graph := newBlockingGraph()
	// Shared across sources, since one chain of waits can mix row and metadata locks.
	waiting := make(map[int64]struct{})

	// Retry a startup probe that could not answer rather than losing row locks for good.
	if m.rowLockSource.sql == "" && !m.rowLocks.unsupported {
		m.selectRowLockSource(ctx)
	}

	rowOK := m.rowLockSource.sql != "" &&
		m.readLockSource(ctx, m.rowLockSource, &m.rowLocks, graph, waiting)
	metadataOK := m.readLockSource(ctx, metadataLockSource, &m.metadataLocks, graph, waiting)

	if !rowOK && !metadataOK {
		return nil
	}

	// Only both sources together can show that a statement is not waiting.
	graph.complete = rowOK && metadataOK

	markRootBlockers(graph.blockers, waiting)

	return graph
}

// readLockSource merges one source's edges into graph and reports whether it answered completely.
func (m *MySQLRTA) readLockSource(ctx context.Context, source lockSource, state *lockSourceState, graph *blockingGraph, waiting map[int64]struct{}) bool {
	if state.unsupported {
		return false
	}

	found, err := m.readLockEdges(ctx, source)
	if err != nil {
		// A missing table or privilege needs operator action, so stop querying instead of failing.
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && permanentBlockingError(mysqlErr.Number) {
			state.unsupported = true
			m.l.Warnf("%s details cannot be collected from this instance and will not be retried "+
				"(grant the monitoring user SELECT on performance_schema and restart the agent if this is a privilege problem): %v",
				source.name, err)

			return false
		}

		// Anything else may be transient: keep trying, but warn once per outage.
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
		// Warned every time: it reflects the current load, not a lasting state.
		m.l.Warnf("The %s wait graph hit its %d row limit, so some waiting statements are not described in this collection",
			source.name, source.rowLimit)

		return false
	}

	return true
}

// permanentBlockingError reports whether a lock source needs operator action to become readable.
func permanentBlockingError(number uint16) bool {
	switch number {
	case mysqlErrNoSuchTable, mysqlErrTableAccessDenied, mysqlErrSpecificAccessDenied:
		return true
	default:
		return false
	}
}

// sourceEdges stages one source's result: a source failing part-way must not claim waiters the
// other source may explain.
type sourceEdges struct {
	blockers map[string][]*rtav1.BlockingTransaction
	waiters  map[string]waiterLock
	waiting  map[int64]struct{}
	// truncated means the query hit its row limit, so waiters missing from it prove nothing.
	truncated bool
}

// readLockEdges runs one lock query without touching the shared graph.
func (m *MySQLRTA) readLockEdges(ctx context.Context, source lockSource) (*sourceEdges, error) {
	// No timeout of its own: ctx carries what is left of the cycle's budget.
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
	// seen dedupes (waiter, blocker) pairs, which repeat once per contended lock.
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

		// A waiter with no connection id, such as an XA transaction, has no statement to explain.
		if !edge.waitingConnID.Valid {
			continue
		}

		key := strconv.FormatInt(edge.waitingConnID.Int64, 10)
		found.waiting[edge.waitingConnID.Int64] = struct{}{}

		// Taken from the waiter's first row, and kept even when the blocker has no connection id.
		if _, ok := found.waiters[key]; !ok {
			found.waiters[key] = waiterLock{
				lockType:       source.lockType,
				lockedTable:    edge.lockedTable,
				lockedIndex:    edge.lockedIndex,
				requestedMode:  edge.requestedMode,
				waitingEventID: edge.waitingEventID,
				waitingQuery:   edge.waitingQuery,
			}
		}

		// A background thread holding a metadata lock has no connection id; the waiter still counts.
		if !edge.blockingConnID.Valid {
			continue
		}

		pair := [2]int64{edge.waitingConnID.Int64, edge.blockingConnID.Int64}
		if _, duplicate := seen[pair]; duplicate {
			continue
		}
		seen[pair] = struct{}{}

		blockingQuery, blockingQueryTruncated := m.boundedQueryText(edge.blockingQuery)
		found.blockers[key] = append(found.blockers[key], &rtav1.BlockingTransaction{
			BlockingConnId:             edge.blockingConnID.Int64,
			BlockingQuery:              blockingQuery,
			BlockingQueryTruncated:     blockingQueryTruncated,
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

	found.truncated = source.rowLimit > 0 && read >= source.rowLimit

	return found, nil
}

// merge folds one source's edges into the shared graph.
func (g *blockingGraph) merge(found *sourceEdges, waiting map[int64]struct{}) {
	for key, lock := range found.waiters {
		// A connection waits on one lock at a time, so a waiter the other source claimed stays with it.
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

// microsToDuration converts a microsecond column, leaving NULL unset rather than reporting 0s.
func microsToDuration(micros sql.NullInt64) *durationpb.Duration {
	if !micros.Valid {
		return nil
	}

	return durationpb.New(time.Duration(micros.Int64) * time.Microsecond)
}

// picosToDuration converts a picosecond timer, leaving NULL unset rather than reporting 0s.
func picosToDuration(picos sql.NullInt64) *durationpb.Duration {
	if !picos.Valid {
		return nil
	}

	return durationpb.New(time.Duration(picos.Int64/picosecondsPerNanosecond) * time.Nanosecond)
}

// markRootBlockers flags blockers that are not waiting themselves (several, or none in a cycle) and
// sorts each list by connection id so repeated collections agree.
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

// scanRow scans one row into a map keyed by column name, with values coerced by coerceValue.
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

// coerceValue turns a raw value into nil, int64, float64 or string for native JSON types. Any
// numeric-looking string becomes a number, so it is unfit for zero-padded ids or wide integers.
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

// buildQueryData converts one processlist row into QueryData, keeping the whole row as raw JSON.
func (m *MySQLRTA) buildQueryData(row map[string]any, graph *blockingGraph) *rtav1.QueryData {
	// statement_latency is NULL with events_statements_current off (MariaDB's default), so fall back
	// to PROCESSLIST_TIME, whole seconds but always populated.
	execDuration := picosToDuration(sql.NullInt64{Int64: int64(mapFloat(row, "statement_latency")), Valid: true})
	if execDuration.AsDuration() == 0 {
		execDuration = durationpb.New(time.Duration(mapInt(row, "time")) * time.Second)
	}

	connID := mapString(row, "conn_id")

	// Without a complete graph a statement absent from it is unknown, never NOT_BLOCKED.
	blockedStatus := rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED
	lockType := rtav1.LockType_LOCK_TYPE_UNSPECIFIED
	var blockedBy []*rtav1.BlockingTransaction
	var lockedTable, lockedIndex, requestedLockMode string

	if graph != nil {
		lock, waiting := graph.waiters[connID]
		// The lock graph is read separately, so the wait may belong to a later statement on this
		// connection; that is UNATTRIBUTED rather than BLOCKED.
		otherStatement := waiting && !waitingStatementMatches(row, lock)
		switch {
		case otherStatement:
			blockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_UNATTRIBUTED
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

	queryText, queryTextTruncated := m.statementText(mapString(row, "current_statement"), mapString(row, "current_sql_text"))
	// current_sql_text only completes current_statement; keeping it would double the raw payload.
	delete(row, "current_sql_text")
	// The tag column carries nothing; it is there for the statement digest only.
	delete(row, agents.RTAQueryTag)

	mysqlPayload := &rtav1.QueryMySQLData{
		DbInstanceAddress:  m.dbInstanceAddress,
		QueryTextTruncated: queryTextTruncated,
		ProgramName:        mapString(row, "program_name"),
		DatabaseName:       mapString(row, "db"),
		Command:            mapString(row, "command"),
		State:              mapString(row, "state"),
		Username:           mapString(row, "user"),
		RowsExamined:       mapOptionalInt(row, "rows_examined"),
		RowsSent:           mapOptionalInt(row, "rows_sent"),
		FullScan:           mapOptionalFullScan(row),
		LockTime:           mapOptionalPicos(row, "lock_latency"),
		BlockedStatus:      blockedStatus,
		BlockedBy:          blockedBy,
		LockedTable:        lockedTable,
		LockedIndex:        lockedIndex,
		LockType:           lockType,
		RequestedLockMode:  requestedLockMode,
	}

	rawJSON, err := json.MarshalIndent(row, "", "    ")
	if err != nil {
		m.l.Warnf("Failed to marshal raw query data: %v", err)
	}

	return &rtav1.QueryData{
		ServiceId:              m.serviceID,
		ServiceName:            m.serviceName,
		QueryId:                connID,
		QueryText:              queryText,
		QueryRawJson:           string(rawJSON),
		QueryExecutionDuration: execDuration,
		Payload: &rtav1.QueryData_MySqlPayload{
			MySqlPayload: mysqlPayload,
		},
	}
}

// waitingStatementMatches reports whether row is the statement the lock graph saw waiting: by event
// id when both have one, else by text prefix (the sources truncate differently), else assumed true.
func waitingStatementMatches(row map[string]any, lock waiterLock) bool {
	if row["statement_event_id"] != nil && lock.waitingEventID.Valid {
		return mapInt(row, "statement_event_id") == lock.waitingEventID.Int64
	}

	statement := mapString(row, "current_statement")
	if statement == "" || lock.waitingQuery == "" {
		return true
	}

	return strings.HasPrefix(statement, lock.waitingQuery) || strings.HasPrefix(lock.waitingQuery, statement)
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

// mapOptionalInt returns nil for NULL: a count nobody measured (consumer off) is not zero.
func mapOptionalInt(row map[string]any, key string) *int64 {
	if row[key] == nil {
		return nil
	}

	value := mapInt(row, key)

	return &value
}

// mapOptionalPicos reads a picosecond timer, nil for NULL like mapOptionalInt.
func mapOptionalPicos(row map[string]any, key string) *durationpb.Duration {
	if row[key] == nil {
		return nil
	}

	return picosToDuration(sql.NullInt64{Int64: mapInt(row, key), Valid: true})
}

// mapOptionalFullScan returns nil unless full_scan is YES or NO, so unmeasured is not reported as NO.
func mapOptionalFullScan(row map[string]any) *bool {
	switch value := mapString(row, "full_scan"); {
	case strings.EqualFold(value, "YES"):
		yes := true

		return &yes
	case strings.EqualFold(value, "NO"):
		no := false

		return &no
	default:
		return nil
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
