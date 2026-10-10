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

// Package realtimeanalytics runs built-in Real-Time Analytics Agent for PostgreSQL.
package realtimeanalytics

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// activityQuery returns the non-idle client sessions, longest-running first, and every session blocking any session.
// Its CTE is named agents.RTAQueryTag so the QAN agents skip it; the tag comes first so it survives
// pg_stat_monitor's pgsm_query_max_len.
//
//   - pg_blocking_pids() is called only for sessions waiting on a heavyweight lock: it takes the lock
//     manager's locks, so calling it for every backend every collect interval would be load of its own.
//   - The second column says whether a row is a listed session; the others are only blockers, which
//     blockingChains needs to follow a chain through idle sessions and background workers.
//   - Blockers are matched to waiters in Go: done here, the match grows with the cube of a lock queue.
//   - pg_locks is read once, and only while a session waits on a lock: each read copies the whole lock table.
//   - query_text_truncated allows for a multibyte character cut short (up to 3 bytes) below track_activity_query_size.
//   - Parallel workers are left out; their leader is listed and carries the query.
//   - query_id (14+) and pg_locks.waitstart (14+) are read through to_jsonb, so one query serves 12 and 13 too.
//   - For sessions idle in transaction the duration is the transaction's, not the last query's.
//   - Without pg_read_all_stats, other users' sessions have a NULL backend_type, so they are left out;
//     Run reports that in the session status.
//   - The whole pg_stat_activity row goes to the raw data, pretty-printed as for MySQL; its columns differ
//     between versions. It is converted before blk is added, which can hold every session queued ahead.
//
//nolint:unqueryvet
const activityQuery = `WITH ` + agents.RTAQueryTag + ` AS (SELECT *, to_jsonb(a) AS j,
  CASE WHEN wait_event_type = 'Lock' THEN pg_blocking_pids(pid) END AS blk FROM pg_stat_activity a),
listed AS (SELECT pid, row_number() OVER (ORDER BY CASE WHEN state LIKE 'idle in transaction%' THEN xact_start ELSE query_start END NULLS LAST, pid) AS n
  FROM ` + agents.RTAQueryTag + ` WHERE backend_type = 'client backend' AND state IS DISTINCT FROM 'idle' AND pid <> pg_backend_pid()
  ORDER BY n LIMIT 1000),
waits AS (SELECT pid, min((to_jsonb(lk)->>'waitstart')::timestamptz) AS since FROM pg_locks lk
  WHERE NOT granted AND EXISTS (SELECT FROM ` + agents.RTAQueryTag + ` WHERE blk IS NOT NULL) GROUP BY pid)
SELECT w.pid, l.pid IS NOT NULL, CASE WHEN l.pid IS NOT NULL THEN jsonb_pretty(w.j) END,
  COALESCE(w.datname, ''), COALESCE(w.usename, ''), COALESCE(w.application_name, ''),
  COALESCE(w.state, ''), COALESCE(w.backend_type, ''), COALESCE(w.wait_event_type, ''), COALESCE(w.wait_event, ''),
  COALESCE(host(w.client_addr) || ':' || w.client_port, ''), COALESCE(w.query, ''),
  COALESCE(w.j->>'query_id', ''), w.xact_start, w.query_start,
  EXTRACT(EPOCH FROM clock_timestamp() - CASE WHEN w.state LIKE 'idle in transaction%' THEN w.xact_start ELSE w.query_start END),
  EXTRACT(EPOCH FROM clock_timestamp() - w.xact_start),
  COALESCE(octet_length(w.query) >= s.size - 4, false),
  w.blk, EXTRACT(EPOCH FROM clock_timestamp() - wt.since)
FROM ` + agents.RTAQueryTag + ` w LEFT JOIN listed l ON l.pid = w.pid LEFT JOIN waits wt ON wt.pid = w.pid,
  (SELECT setting::int AS size FROM pg_settings WHERE name = 'track_activity_query_size') s
WHERE l.pid IS NOT NULL OR w.pid IN (SELECT unnest(blk) FROM ` + agents.RTAQueryTag + `)
ORDER BY l.n NULLS LAST`

const (
	defaultCollectInterval = 2 * time.Second
	// The LIMIT in activityQuery, as processlistRowLimit is for MySQL.
	sessionLimit = 1000
	// The blocked_by entries of one collection are bounded, as lockGraphRowLimit bounds MySQL's: a session
	// queued on a row lists every session queued ahead of it, so the entries grow with the square of the queue.
	blockerLimit = 5000
	// The pid pg_blocking_pids() gives a prepared transaction holding the lock.
	preparedTransaction = 0
)

// PostgreSQLRTA extracts Real-Time Analytics data (currently running queries) from PostgreSQL.
type PostgreSQLRTA struct {
	db                *sql.DB
	dbInstanceAddress string
	serviceID         string
	serviceName       string
	collectInterval   time.Duration
	l                 *logrus.Entry
	changes           chan agents.Change
	// sessionsTruncated is set while activityQuery hits sessionLimit, so that is logged once per episode.
	sessionsTruncated bool
}

// Params represent Agent parameters.
type Params struct {
	DSN             string
	ServiceID       string
	ServiceName     string
	CollectInterval time.Duration
}

// New creates new PostgreSQLRTA agent.
func New(params *Params, l *logrus.Entry) (*PostgreSQLRTA, error) {
	db, err := sql.Open("postgres", params.DSN)
	if err != nil {
		return nil, err
	}
	// a single connection makes pg_backend_pid() exclude all of our own queries
	db.SetMaxOpenConns(1)

	collectInterval := params.CollectInterval
	if collectInterval <= 0 {
		l.Warnf("No collect interval set for Real-Time Analytics, falling back to %s", defaultCollectInterval)
		collectInterval = defaultCollectInterval
	}

	return &PostgreSQLRTA{
		db:                db,
		dbInstanceAddress: instanceAddress(params.DSN),
		serviceID:         params.ServiceID,
		serviceName:       params.ServiceName,
		collectInterval:   collectInterval,
		l:                 l,
		changes:           make(chan agents.Change, 10), //nolint:mnd
	}, nil
}

// Run collects currently running queries and sends them to the channel until ctx is canceled.
func (m *PostgreSQLRTA) Run(ctx context.Context) {
	terminalStatus := inventoryv1.AgentStatus_AGENT_STATUS_DONE
	var terminalMessage string
	defer func() {
		_ = m.db.Close()
		m.changes <- agents.Change{Status: terminalStatus, StatusMessage: terminalMessage}
		close(m.changes)
	}()

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	var canReadAllStats bool
	err := m.db.QueryRowContext(ctx, "SELECT pg_has_role('pg_read_all_stats', 'USAGE')").Scan(&canReadAllStats)
	if err != nil {
		if ctx.Err() == nil {
			m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)
			terminalStatus = inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR
			terminalMessage = fmt.Sprintf("Cannot connect to PostgreSQL: %v", err)
		}
		return
	}

	err = m.probe(ctx)
	if err != nil {
		if ctx.Err() == nil {
			m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)
			terminalStatus = inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR
			terminalMessage = fmt.Sprintf("Cannot run the Real-Time Analytics query on this instance: %v", err)
		}
		return
	}

	var warning string
	if !canReadAllStats {
		warning = "The monitoring user is not a member of pg_read_all_stats, so other users' sessions are not " +
			"shown. Grant it pg_monitor or pg_read_all_stats."
	}
	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, StatusMessage: warning}

	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			return
		case <-ticker.C:
			queries, err := m.collect(ctx)
			if err != nil {
				m.l.Warnf("pg_stat_activity collection failed: %v", err)
				continue
			}

			if len(queries) == 0 {
				continue
			}

			select {
			case <-ctx.Done():
			case m.changes <- agents.Change{RTAQueriesBucket: queries}:
			}
		}
	}
}

// probe plans and starts activityQuery without fetching a row, so a server that cannot run it fails at startup.
func (m *PostgreSQLRTA) probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, m.collectInterval)
	defer cancel()

	rows, err := m.db.QueryContext(ctx, activityQuery+"\nLIMIT 0")
	if err != nil {
		return err
	}
	defer rows.Close() //nolint:errcheck

	return rows.Err()
}

func (m *PostgreSQLRTA) collect(ctx context.Context) ([]*rtav1.QueryData, error) {
	ctx, cancel := context.WithTimeout(ctx, m.collectInterval)
	defer cancel()

	rows, err := m.db.QueryContext(ctx, activityQuery)
	if err != nil {
		return nil, fmt.Errorf("pg_stat_activity not available or permission denied: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	now := timestamppb.Now()
	var res []*rtav1.QueryData
	// blk and blockers hold every row, listed or not, so chains can be followed through all of them.
	blk := make(map[int64][]int64)
	blockers := make(map[int64]*rtav1.BlockingTransaction)
	waited := make(map[int64]*durationpb.Duration)

	for rows.Next() {
		var (
			p                          rtav1.QueryPostgreSQLData
			q                          rtav1.QueryData
			listed                     bool
			rawJSON                    sql.NullString
			backendType                string
			xactStart, qStart          sql.NullTime
			duration, xactAge, waitAge *float64
			blockedBy                  pq.Int64Array
		)

		err = rows.Scan(&p.Pid, &listed, &rawJSON, &p.DatabaseName, &p.Username, &p.ApplicationName,
			&p.State, &backendType, &p.WaitEventType, &p.WaitEvent, &q.ClientAddress, &q.QueryText, &p.QueryId,
			&xactStart, &qStart, &duration, &xactAge, &p.QueryTextTruncated, &blockedBy, &waitAge)
		if err != nil {
			return nil, err
		}

		pid := int64(p.Pid)
		blk[pid] = blockedBy
		blockers[pid] = &rtav1.BlockingTransaction{
			BlockingConnId:             pid,
			BlockingQuery:              q.QueryText,
			BlockingCommand:            cmp.Or(p.State, backendType),
			BlockingUsername:           p.Username,
			BlockerTransactionDuration: seconds(xactAge),
			Root:                       len(blockedBy) == 0,
			BlockingQueryTruncated:     p.QueryTextTruncated,
		}
		if !listed {
			continue
		}

		waited[pid] = seconds(waitAge)
		p.BlockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED
		if xactStart.Valid {
			p.TransactionStartTime = timestamppb.New(xactStart.Time)
		}
		if qStart.Valid {
			p.QueryStartTime = timestamppb.New(qStart.Time)
		}

		q.QueryRawJson = rawJSON.String
		q.QueryExecutionDuration = seconds(duration)
		q.ServiceId = m.serviceID
		q.ServiceName = m.serviceName
		q.QueryId = strconv.Itoa(int(p.Pid))
		q.QueryCollectTime = now
		p.DbInstanceAddress = m.dbInstanceAddress
		q.Payload = &rtav1.QueryData_PostgresqlPayload{PostgresqlPayload: &p}
		res = append(res, &q)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	switch truncated := len(res) >= sessionLimit; {
	case truncated && !m.sessionsTruncated:
		m.sessionsTruncated = true
		m.l.Warnf("More than %d sessions are active, so only the %d longest-running are collected", sessionLimit, sessionLimit)
	case !truncated && m.sessionsTruncated:
		m.sessionsTruncated = false
		m.l.Infof("Fewer than %d sessions are active again, so all of them are collected", sessionLimit)
	}

	m.setBlockers(res, blk, blockers, waited)

	return res, nil
}

// setBlockers sets the blocked status and blockers of the listed sessions in res; the maps cover every row of activityQuery.
func (m *PostgreSQLRTA) setBlockers(
	res []*rtav1.QueryData,
	blk map[int64][]int64,
	blockers map[int64]*rtav1.BlockingTransaction,
	waited map[int64]*durationpb.Duration,
) {
	var waiters []int64
	for _, q := range res {
		pid := int64(q.GetPostgresqlPayload().Pid)
		if len(blk[pid]) != 0 {
			waiters = append(waiters, pid)
		}
	}

	chains, complete := blockingChains(blk, waiters, blockerLimit)
	if !complete {
		m.l.Warnf("The lock waits need more than %d blocker entries, so some blocked sessions are shown "+
			"with only the sessions at the head of their chain, or without their blockers", blockerLimit)
	}

	for _, q := range res {
		p := q.GetPostgresqlPayload()
		pid := int64(p.Pid)
		if len(blk[pid]) == 0 {
			continue
		}

		// Blocked, but not described within blockerLimit: as for MySQL, unknown rather than a guess.
		if len(chains[pid]) == 0 {
			p.BlockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_UNSPECIFIED
			continue
		}

		p.BlockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED
		for _, b := range chains[pid] {
			blocker, ok := blockers[b]
			if !ok {
				blocker = &rtav1.BlockingTransaction{BlockingConnId: b, Root: true}
				if b == preparedTransaction {
					blocker.BlockingCommand = "prepared transaction"
				}
			}

			blocker = proto.CloneOf(blocker)
			blocker.WaitDuration = waited[pid]
			p.BlockedBy = append(p.BlockedBy, blocker)
		}
	}
}

// instanceAddress returns the host:port, or the socket directory, that a pmm-managed PostgreSQL DSN connects to.
func instanceAddress(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	if u.Host != "" {
		return u.Host
	}
	return u.Query().Get("host")
}

// blockingChains returns the sessions each waiter waits for, directly or not, ordered by pid, in at most limit
// entries in all. Every waiter gets its roots first, the sessions holding up the whole chain; then whole chains
// are filled in the order of waiters until one no longer fits. The second result reports whether all of them did.
func blockingChains(blk map[int64][]int64, waiters []int64, limit int) (map[int64][]int64, bool) {
	waitingFor := make(map[int64][]int64)
	for pid, blockers := range blk {
		for _, b := range blockers {
			waitingFor[b] = append(waitingFor[b], pid)
		}
	}

	// Walking back from each root once finds every session's roots without walking every chain.
	roots := make(map[int64][]int64)
	for root := range waitingFor {
		if len(blk[root]) != 0 {
			continue
		}

		seen := map[int64]bool{root: true}
		for queue := []int64{root}; len(queue) != 0; queue = queue[1:] {
			for _, pid := range waitingFor[queue[0]] {
				if !seen[pid] {
					seen[pid] = true
					roots[pid] = append(roots[pid], root)
					queue = append(queue, pid)
				}
			}
		}
	}

	chains := make(map[int64][]int64, len(waiters))
	entries := 0
	complete := true
	for _, w := range waiters {
		if entries+len(roots[w]) > limit {
			complete = false
			break
		}

		slices.Sort(roots[w])
		chains[w] = roots[w]
		entries += len(roots[w])
	}

	for _, w := range waiters {
		have, ok := chains[w]
		if !ok {
			break
		}

		chain, ok := waitsFor(blk, w, limit-entries+len(have))
		if !ok {
			complete = false
			break
		}

		chains[w] = chain
		entries += len(chain) - len(have)
	}

	return chains, complete
}

// waitsFor returns the sessions pid waits for, directly or not, ordered by pid, or false if there are more than limit.
func waitsFor(blk map[int64][]int64, pid int64, limit int) ([]int64, bool) {
	seen := map[int64]bool{pid: true}
	var res []int64
	for queue := []int64{pid}; len(queue) != 0; queue = queue[1:] {
		for _, b := range blk[queue[0]] {
			if seen[b] {
				continue
			}
			if len(res) == limit {
				return nil, false
			}

			seen[b] = true
			res = append(res, b)
			queue = append(queue, b)
		}
	}

	slices.Sort(res)

	return res, true
}

func seconds(s *float64) *durationpb.Duration {
	if s == nil {
		return nil
	}

	return durationpb.New(time.Duration(*s * float64(time.Second)))
}

// Changes returns channel that should be read until it is closed.
func (m *PostgreSQLRTA) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector.
func (m *PostgreSQLRTA) Describe(_ chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector.
func (m *PostgreSQLRTA) Collect(_ chan<- prometheus.Metric) {}

// check interfaces.
var _ prometheus.Collector = (*PostgreSQLRTA)(nil)
