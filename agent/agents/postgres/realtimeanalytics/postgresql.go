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
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	_ "github.com/lib/pq" // register SQL driver
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// activityQuery returns the non-idle client sessions. Its CTE is named agents.RTAQueryTag so the QAN agents skip
// it; the tag comes first so it survives pg_stat_monitor's pgsm_query_max_len.
//
//   - pg_blocking_pids() is called only for sessions waiting on a heavyweight lock: it takes the lock
//     manager's locks, so calling it for every backend every collect interval would be load of its own.
//   - query_text_truncated allows for a multibyte character cut short (up to 3 bytes) below track_activity_query_size.
//   - Parallel workers are left out; their leader is listed and carries the query.
//   - query_id (14+) and pg_locks.waitstart (14+) are read through to_jsonb, so one query serves 12 and 13 too.
//   - For sessions idle in transaction the duration is the transaction's, not the last query's.
//   - Without pg_read_all_stats, other users' sessions have a NULL backend_type, so they are left out;
//     Run reports that in the session status.
//   - The whole pg_stat_activity row goes to the raw data, pretty-printed as for MySQL and without the blk
//     helper column; its columns differ between versions.
//
//nolint:unqueryvet
const activityQuery = `WITH ` + agents.RTAQueryTag + ` AS (SELECT *, CASE WHEN wait_event_type = 'Lock' THEN pg_blocking_pids(pid) END AS blk FROM pg_stat_activity)
SELECT w.pid, jsonb_pretty(to_jsonb(w) - 'blk'), COALESCE(w.datname, ''), COALESCE(w.usename, ''), COALESCE(w.application_name, ''),
  COALESCE(w.state, ''), COALESCE(w.wait_event_type, ''), COALESCE(w.wait_event, ''),
  COALESCE(host(w.client_addr) || ':' || w.client_port, ''), COALESCE(w.query, ''),
  COALESCE(to_jsonb(w)->>'query_id', ''), w.xact_start, w.query_start,
  EXTRACT(EPOCH FROM clock_timestamp() - CASE WHEN w.state LIKE 'idle in transaction%' THEN w.xact_start ELSE w.query_start END),
  COALESCE(octet_length(w.query) >= s.size - 4, false),
  COALESCE(cardinality(w.blk) > 0, false),
  (SELECT EXTRACT(EPOCH FROM clock_timestamp() - min((to_jsonb(l)->>'waitstart')::timestamptz))
     FROM pg_locks l WHERE w.blk IS NOT NULL AND l.pid = w.pid AND NOT l.granted),
  (SELECT json_agg(json_build_object(
       'pid', b.pid, 'query', COALESCE(b.query, ''), 'state', COALESCE(b.state, b.backend_type, ''),
       'user', COALESCE(b.usename, ''), 'xact_secs', EXTRACT(EPOCH FROM clock_timestamp() - b.xact_start),
       'root', b.blk IS NULL OR cardinality(b.blk) = 0,
       'truncated', COALESCE(octet_length(b.query) >= s.size - 4, false)) ORDER BY b.pid)
     FROM ` + agents.RTAQueryTag + ` b WHERE b.pid = ANY(w.blk))
FROM ` + agents.RTAQueryTag + ` w, (SELECT setting::int AS size FROM pg_settings WHERE name = 'track_activity_query_size') s
WHERE w.backend_type = 'client backend' AND w.state IS DISTINCT FROM 'idle' AND w.pid <> pg_backend_pid()`

const defaultCollectInterval = 2 * time.Second

// PostgreSQLRTA extracts Real-Time Analytics data (currently running queries) from PostgreSQL.
type PostgreSQLRTA struct {
	db                *sql.DB
	dbInstanceAddress string
	serviceID         string
	serviceName       string
	collectInterval   time.Duration
	l                 *logrus.Entry
	changes           chan agents.Change
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

	for rows.Next() {
		var (
			p                 rtav1.QueryPostgreSQLData
			q                 rtav1.QueryData
			xactStart, qStart sql.NullTime
			duration, waited  *float64
			blocked           bool
			blockers          []byte
		)

		err = rows.Scan(&p.Pid, &q.QueryRawJson, &p.DatabaseName, &p.Username, &p.ApplicationName,
			&p.State, &p.WaitEventType, &p.WaitEvent, &q.ClientAddress, &q.QueryText, &p.QueryId,
			&xactStart, &qStart, &duration, &p.QueryTextTruncated, &blocked, &waited, &blockers)
		if err != nil {
			return nil, err
		}

		p.BlockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_NOT_BLOCKED
		if blocked {
			p.BlockedStatus = rtav1.BlockedStatus_BLOCKED_STATUS_BLOCKED
			p.BlockedBy, err = toBlockers(blockers, waited)
			if err != nil {
				return nil, err
			}
		}
		if xactStart.Valid {
			p.TransactionStartTime = timestamppb.New(xactStart.Time)
		}
		if qStart.Valid {
			p.QueryStartTime = timestamppb.New(qStart.Time)
		}

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

	withBlockingChains(res)

	return res, nil
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

// withBlockingChains extends each waiter's direct blockers (pg_blocking_pids) with the blockers of those blockers,
// so that, as for MySQL, blocked_by lists every transaction ahead in the chain and root marks its head.
// The chain is read from the same snapshot; each entry keeps the waiter's own wait duration.
func withBlockingChains(queries []*rtav1.QueryData) {
	direct := make(map[int64][]*rtav1.BlockingTransaction)
	for _, q := range queries {
		p := q.GetPostgresqlPayload()
		if len(p.GetBlockedBy()) != 0 {
			direct[int64(p.Pid)] = p.BlockedBy
		}
	}

	for _, q := range queries {
		p := q.GetPostgresqlPayload()
		if len(p.GetBlockedBy()) == 0 {
			continue
		}

		waited := p.BlockedBy[0].WaitDuration
		seen := map[int64]bool{int64(p.Pid): true}
		var chain []*rtav1.BlockingTransaction
		for queue := p.BlockedBy; len(queue) != 0; queue = queue[1:] {
			b := queue[0]
			if seen[b.BlockingConnId] {
				continue
			}
			seen[b.BlockingConnId] = true

			b = proto.CloneOf(b)
			b.WaitDuration = waited
			chain = append(chain, b)
			queue = append(queue, direct[b.BlockingConnId]...)
		}

		slices.SortFunc(chain, func(a, b *rtav1.BlockingTransaction) int { return cmp.Compare(a.BlockingConnId, b.BlockingConnId) })
		p.BlockedBy = chain
	}
}

// toBlockers converts the blockers JSON built by activityQuery; waited is how long the waiting session has waited.
func toBlockers(raw []byte, waited *float64) ([]*rtav1.BlockingTransaction, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var blockers []struct {
		Pid       int64    `json:"pid"`
		Query     string   `json:"query"`
		State     string   `json:"state"`
		User      string   `json:"user"`
		XactSecs  *float64 `json:"xact_secs"`
		Root      bool     `json:"root"`
		Truncated bool     `json:"truncated"`
	}
	err := json.Unmarshal(raw, &blockers)
	if err != nil {
		return nil, fmt.Errorf("cannot parse blockers: %w", err)
	}

	res := make([]*rtav1.BlockingTransaction, 0, len(blockers))
	for _, b := range blockers {
		res = append(res, &rtav1.BlockingTransaction{
			BlockingConnId:             b.Pid,
			BlockingQuery:              b.Query,
			BlockingCommand:            b.State,
			BlockingUsername:           b.User,
			WaitDuration:               seconds(waited),
			BlockerTransactionDuration: seconds(b.XactSecs),
			Root:                       b.Root,
			BlockingQueryTruncated:     b.Truncated,
		})
	}

	return res, nil
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
