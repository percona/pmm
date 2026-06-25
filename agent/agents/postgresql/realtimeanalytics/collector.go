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
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pgversion "github.com/percona/pmm/agent/utils/version"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const queryTag = "agent='rta-postgresql'"

type collector struct {
	db            *sql.DB
	agentID       string
	l             *logrus.Entry
	majorVersion  int
	hasQueryID    bool
	hasLeaderPID  bool
	hasPrivileges bool
	privChecked   bool
}

type sessionRow struct {
	pid              int32
	username         sql.NullString
	applicationName  sql.NullString
	clientAddr       sql.NullString
	clientPort       sql.NullInt64
	databaseName     sql.NullString
	state            sql.NullString
	waitEventType    sql.NullString
	waitEvent        sql.NullString
	backendType      sql.NullString
	xactStart        sql.NullTime
	queryStart       sql.NullTime
	stateChange      sql.NullTime
	query            sql.NullString
	queryID          sql.NullInt64
	leaderPID        sql.NullInt32
}

func newCollector(db *sql.DB, agentID string, l *logrus.Entry) (*collector, error) {
	c := &collector{
		db:      db,
		agentID: agentID,
		l:       l,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var versionStr string
	if err := db.QueryRowContext(ctx, "SELECT /* "+queryTag+" */ version()").Scan(&versionStr); err != nil {
		return nil, fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	majorStr, _ := pgversion.ParsePostgreSQLVersion(versionStr)
	if majorStr == "" {
		return nil, fmt.Errorf("failed to parse PostgreSQL version from %q", versionStr)
	}

	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL major version %q: %w", majorStr, err)
	}

	c.majorVersion = major
	c.hasQueryID = major >= 14
	c.hasLeaderPID = major >= 13

	return c, nil
}

func (c *collector) collectSessions(ctx context.Context) ([]*rtav1.QueryData, error) {
	if err := c.ensurePrivileges(ctx); err != nil {
		return nil, err
	}

	sessions, err := c.fetchSessions(ctx)
	if err != nil {
		return nil, err
	}

	lockChains, err := c.fetchLockChains(ctx)
	if err != nil {
		c.l.Warnf("Failed to fetch lock chains: %v", err)
		lockChains = nil
	}

	now := time.Now()
	results := make([]*rtav1.QueryData, 0, len(sessions))

	for _, row := range sessions {
		if row.state.Valid && row.state.String == "idle" {
			continue
		}

		queryData := c.rowToQueryData(row, lockChains[row.pid], now)
		if queryData != nil {
			results = append(results, queryData)
		}
	}

	return results, nil
}

func (c *collector) ensurePrivileges(ctx context.Context) error {
	if c.privChecked {
		if !c.hasPrivileges {
			return ErrInsufficientPrivileges
		}
		return nil
	}

	c.privChecked = true

	var hasPrivilege bool
	err := c.db.QueryRowContext(ctx,
		"SELECT /* "+queryTag+" */ has_privilege(current_user, 'pg_read_all_stats', 'USAGE')").Scan(&hasPrivilege)
	if err != nil {
		// Fall back to heuristic: compare total backends vs visible non-self backends.
		var total, visible int
		_ = c.db.QueryRowContext(ctx, "SELECT /* "+queryTag+" */ count(*) FROM pg_stat_activity").Scan(&total)
		_ = c.db.QueryRowContext(ctx,
			"SELECT /* "+queryTag+" */ count(*) FROM pg_stat_activity WHERE pid != pg_backend_pid()").Scan(&visible)
		c.hasPrivileges = total <= 1 || visible > 0
	} else {
		c.hasPrivileges = hasPrivilege
	}

	if !c.hasPrivileges {
		return ErrInsufficientPrivileges
	}

	return nil
}

func (c *collector) fetchSessions(ctx context.Context) ([]sessionRow, error) {
	query := c.sessionsQuery()

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pg_stat_activity query failed: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var results []sessionRow

	for rows.Next() {
		var row sessionRow
		dest := []any{
			&row.pid,
			&row.username,
			&row.applicationName,
			&row.clientAddr,
			&row.clientPort,
			&row.databaseName,
			&row.state,
			&row.waitEventType,
			&row.waitEvent,
			&row.backendType,
			&row.xactStart,
			&row.queryStart,
			&row.stateChange,
			&row.query,
		}

		if c.hasQueryID {
			dest = append(dest, &row.queryID)
		}
		if c.hasLeaderPID {
			dest = append(dest, &row.leaderPID)
		}

		if scanErr := rows.Scan(dest...); scanErr != nil {
			return nil, fmt.Errorf("failed to scan pg_stat_activity row: %w", scanErr)
		}

		if row.applicationName.Valid && strings.HasPrefix(row.applicationName.String, "pmm-rta-postgresql") {
			continue
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (c *collector) sessionsQuery() string {
	columns := []string{
		"pid",
		"usename",
		"application_name",
		"client_addr::text",
		"client_port",
		"datname",
		"state",
		"wait_event_type",
		"wait_event",
		"backend_type",
		"xact_start",
		"query_start",
		"state_change",
		"query",
	}

	if c.hasQueryID {
		columns = append(columns, "query_id")
	} else {
		columns = append(columns, "NULL::bigint AS query_id")
	}

	if c.hasLeaderPID {
		columns = append(columns, "leader_pid")
	} else {
		columns = append(columns, "NULL::integer AS leader_pid")
	}

	return fmt.Sprintf(`SELECT /* %s */ %s
FROM pg_stat_activity
WHERE pid != pg_backend_pid()
  AND backend_type NOT IN ('walsender', 'walreceiver')
  AND state IS DISTINCT FROM 'idle'`,
		queryTag, strings.Join(columns, ", "))
}

type lockChainLink struct {
	blockerPID   int32
	blockedPID   int32
	lockMode     string
	lockType     string
	blockerQuery string
	duration     time.Duration
}

func (c *collector) fetchLockChains(ctx context.Context) (map[int32][]lockChainLink, error) {
	const lockQuery = `SELECT /* ` + queryTag + ` */
  blocked_activity.pid AS blocked_pid,
  blocking_activity.pid AS blocking_pid,
  blocked_locks.mode AS lock_mode,
  blocked_locks.locktype AS lock_type,
  blocking_activity.query AS blocking_query,
  GREATEST(0, EXTRACT(EPOCH FROM (now() - blocking_activity.state_change))) AS blocking_duration_sec
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks
  ON blocking_locks.locktype = blocked_locks.locktype
  AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
  AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
  AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
  AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
  AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
  AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
  AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
  AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
  AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
  AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted`

	rows, err := c.db.QueryContext(ctx, lockQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	chains := make(map[int32][]lockChainLink)

	for rows.Next() {
		var link lockChainLink
		var durationSec float64

		if scanErr := rows.Scan(&link.blockedPID, &link.blockerPID, &link.lockMode, &link.lockType, &link.blockerQuery, &durationSec); scanErr != nil {
			return nil, scanErr
		}

		link.duration = time.Duration(durationSec * float64(time.Second))
		chains[link.blockedPID] = append(chains[link.blockedPID], link)
	}

	return chains, rows.Err()
}

func (c *collector) rowToQueryData(row sessionRow, chain []lockChainLink, now time.Time) *rtav1.QueryData {
	queryText := ""
	queryTruncated := false
	if row.query.Valid {
		queryText = row.query.String
	}

	startTime := row.queryStart
	if row.state.Valid && strings.Contains(row.state.String, "idle in transaction") && row.xactStart.Valid {
		startTime = row.xactStart
	}

	var execDuration time.Duration
	if startTime.Valid {
		execDuration = now.Sub(startTime.Time)
	}

	queryID := fingerprintQuery(queryText)
	if c.hasQueryID && row.queryID.Valid && row.queryID.Int64 != 0 {
		queryID = strconv.FormatInt(row.queryID.Int64, 10)
	}

	clientAddress := ""
	if row.clientAddr.Valid && row.clientAddr.String != "" {
		port := ""
		if row.clientPort.Valid {
			port = strconv.FormatInt(row.clientPort.Int64, 10)
		}
		clientAddress = net.JoinHostPort(row.clientAddr.String, port)
	}

	payload := &rtav1.QueryPostgreSQLData{
		Pid:             row.pid,
		QueryTruncated:  queryTruncated,
		LockChain:       make([]*rtav1.LockChainEntry, 0, len(chain)),
	}

	if row.state.Valid {
		payload.State = row.state.String
	}
	if row.waitEventType.Valid {
		payload.WaitEventType = row.waitEventType.String
	}
	if row.waitEvent.Valid {
		payload.WaitEvent = row.waitEvent.String
	}
	if row.backendType.Valid {
		payload.BackendType = row.backendType.String
	}
	if row.xactStart.Valid {
		payload.TransactionStartTime = timestamppb.New(row.xactStart.Time)
	}
	if row.stateChange.Valid {
		payload.StateChangeTime = timestamppb.New(row.stateChange.Time)
	}
	if row.leaderPID.Valid {
		payload.LeaderPid = row.leaderPID.Int32
	}
	if row.databaseName.Valid {
		payload.DatabaseName = row.databaseName.String
	}
	if row.username.Valid {
		payload.Username = row.username.String
	}
	if row.applicationName.Valid {
		payload.ApplicationName = row.applicationName.String
	}

	for _, link := range chain {
		payload.LockChain = append(payload.LockChain, &rtav1.LockChainEntry{
			Pid:       link.blockerPID,
			LockMode:  link.lockMode,
			LockType:  link.lockType,
			Granted:   true,
			QueryText: link.blockerQuery,
			Duration:  durationpb.New(link.duration),
		})
	}

	return &rtav1.QueryData{
		QueryId:                  queryID,
		QueryText:                queryText,
		QueryExecutionDuration:   durationpb.New(execDuration),
		QueryCollectTime:         timestamppb.New(now),
		ClientAddress:            clientAddress,
		Payload:                  &rtav1.QueryData_PostgresqlPayload{PostgresqlPayload: payload},
	}
}
