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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	activityQuery = `
SELECT pid, datname, usename, application_name,
       COALESCE(host(client_addr), '') AS client_host,
       client_port, state, query,
       query_id, backend_start, xact_start, query_start,
       wait_event_type, wait_event, leader_pid
FROM pg_stat_activity
WHERE pid <> pg_backend_pid()
  AND backend_type = 'client backend'
  AND state <> 'idle'`

	locksQuery = `
SELECT blocked_locks.pid AS blocked_pid,
       blocking_locks.pid AS blocker_pid,
       blocked_locks.mode AS lock_mode,
       COALESCE(blocked_relation.relname, '') AS relation_name,
       blocking_activity.query AS blocker_query,
       blocking_activity.query_start AS blocker_query_start
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
LEFT JOIN pg_catalog.pg_class blocked_relation ON blocked_relation.oid = blocked_locks.relation
WHERE NOT blocked_locks.granted`

	trackActivityQuerySizeSQL = `SELECT current_setting('track_activity_query_size')::int`
)

var errMissingPgReadAllStats = errors.New("monitoring user lacks pg_read_all_stats role; grant pg_read_all_stats or use a superuser account")

func collectSessions(ctx context.Context, db *sql.DB) ([]*rtav1.QueryData, error) {
	trackSize, err := readTrackActivityQuerySize(ctx, db)
	if err != nil {
		return nil, err
	}

	sessions, err := readActivityRows(ctx, db, trackSize)
	if err != nil {
		if isPermissionError(err) {
			return nil, errMissingPgReadAllStats
		}
		return nil, err
	}

	lockRows, err := readLockRows(ctx, db)
	if err != nil && !isPermissionError(err) {
		return nil, err
	}

	sessionMap := make(map[int32]*sessionRow, len(sessions))
	for i := range sessions {
		sessionMap[sessions[i].PID] = &sessions[i]
	}

	lockChains := buildLockChains(lockRows, sessionMap)
	now := time.Now()

	results := make([]*rtav1.QueryData, 0, len(sessions))
	for _, session := range sessions {
		queryID := sessionQueryID(session)
		duration := sessionDuration(session, now)

		payload := &rtav1.QueryPostgreSQLData{
			DatabaseName:            session.DatabaseName,
			Username:                session.Username,
			ApplicationName:         session.ApplicationName,
			SessionState:            session.State,
			WaitEventType:           session.WaitEventType,
			WaitEvent:               session.WaitEvent,
			BackendPid:              session.PID,
			LeaderPid:               session.LeaderPID,
			QueryTextTruncated:      session.QueryTextTruncated,
			TrackActivityQuerySize:  session.TrackActivitySize,
			LockChain:               lockChains[session.PID],
		}

		if !session.TransactionStart.IsZero() {
			payload.TransactionStartTime = timestamppb.New(session.TransactionStart)
		}
		if !session.QueryStart.IsZero() {
			payload.QueryStartTime = timestamppb.New(session.QueryStart)
		}
		if session.ClientAddress != "" {
			payload.DbInstanceAddress = session.ClientAddress
		}

		results = append(results, &rtav1.QueryData{
			QueryId:                 queryID,
			QueryText:               strings.TrimSpace(session.Query),
			QueryRawJson:            session.RawJSON,
			QueryExecutionDuration:  durationpb.New(duration),
			ClientAddress:           session.ClientAddress,
			Payload:                 &rtav1.QueryData_PostgresPayload{PostgresPayload: payload},
		})
	}

	return results, nil
}

func readTrackActivityQuerySize(ctx context.Context, db *sql.DB) (int32, error) {
	var size int32
	err := db.QueryRowContext(ctx, trackActivityQuerySizeSQL).Scan(&size)
	if err != nil {
		return 1024, nil //nolint:mnd
	}
	return size, nil
}

func readActivityRows(ctx context.Context, db *sql.DB, trackSize int32) ([]sessionRow, error) {
	rows, err := db.QueryContext(ctx, activityQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var sessions []sessionRow
	for rows.Next() {
		var (
			session       sessionRow
			clientHost    sql.NullString
			clientPort    sql.NullInt64
			queryID       sql.NullInt64
			backendStart  sql.NullTime
			xactStart     sql.NullTime
			queryStart    sql.NullTime
			waitEventType sql.NullString
			waitEvent     sql.NullString
			leaderPID     sql.NullInt64
		)

		err = rows.Scan(
			&session.PID,
			&session.DatabaseName,
			&session.Username,
			&session.ApplicationName,
			&clientHost,
			&clientPort,
			&session.State,
			&session.Query,
			&queryID,
			&backendStart,
			&xactStart,
			&queryStart,
			&waitEventType,
			&waitEvent,
			&leaderPID,
		)
		if err != nil {
			return nil, err
		}

		if clientHost.Valid {
			if clientPort.Valid && clientPort.Int64 > 0 {
				session.ClientAddress = fmt.Sprintf("%s:%d", clientHost.String, clientPort.Int64)
			} else {
				session.ClientAddress = clientHost.String
			}
		}

		if queryID.Valid {
			session.QueryID = queryID.Int64
			session.HasQueryID = true
		}
		if backendStart.Valid {
			session.BackendStart = backendStart.Time
		}
		if xactStart.Valid {
			session.TransactionStart = xactStart.Time
		}
		if queryStart.Valid {
			session.QueryStart = queryStart.Time
		}
		if waitEventType.Valid {
			session.WaitEventType = waitEventType.String
		}
		if waitEvent.Valid {
			session.WaitEvent = waitEvent.String
		}
		if leaderPID.Valid {
			session.LeaderPID = int32(leaderPID.Int64) //nolint:gosec
		}

		session.TrackActivitySize = trackSize
		if trackSize > 0 && len(session.Query) >= int(trackSize) {
			session.QueryTextTruncated = true
		}

		raw, _ := json.Marshal(map[string]any{
			"pid":              session.PID,
			"datname":          session.DatabaseName,
			"usename":          session.Username,
			"application_name": session.ApplicationName,
			"client_addr":      session.ClientAddress,
			"state":            session.State,
			"query":            session.Query,
			"query_id":         session.QueryID,
			"xact_start":       session.TransactionStart,
			"query_start":      session.QueryStart,
			"wait_event_type":  session.WaitEventType,
			"wait_event":       session.WaitEvent,
			"leader_pid":       session.LeaderPID,
		})
		session.RawJSON = string(raw)

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func readLockRows(ctx context.Context, db *sql.DB) ([]lockRow, error) {
	rows, err := db.QueryContext(ctx, locksQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []lockRow
	for rows.Next() {
		var row lockRow
		var blockerQueryStart sql.NullTime
		err = rows.Scan(
			&row.BlockedPID,
			&row.BlockerPID,
			&row.LockMode,
			&row.RelationName,
			&row.BlockerQuery,
			&blockerQueryStart,
		)
		if err != nil {
			return nil, err
		}
		if blockerQueryStart.Valid {
			row.BlockerQueryAt = blockerQueryStart.Time
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func sessionQueryID(session sessionRow) string {
	if session.HasQueryID {
		return strconv.FormatInt(session.QueryID, 10)
	}

	sum := sha256.Sum256([]byte(strings.TrimSpace(session.Query)))
	return hex.EncodeToString(sum[:8])
}

func sessionDuration(session sessionRow, now time.Time) time.Duration {
	if session.State == "idle in transaction" && !session.TransactionStart.IsZero() {
		return now.Sub(session.TransactionStart)
	}
	if !session.QueryStart.IsZero() {
		return now.Sub(session.QueryStart)
	}
	if !session.BackendStart.IsZero() {
		return now.Sub(session.BackendStart)
	}
	return 0
}

func isPermissionError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "42501" // insufficient_privilege
	}
	return strings.Contains(strings.ToLower(err.Error()), "permission denied")
}
