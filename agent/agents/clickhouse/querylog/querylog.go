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

// Package querylog runs the built-in QAN agent for ClickHouse. It reads
// completed query executions from system.query_log on minute boundaries and
// emits one MetricsBucket per fingerprinted query class.
//
// Unlike the pg_stat_statements agent, system.query_log is an append-only
// event table (one row per query phase) — like the MySQL slow log — so the
// agent tracks an event_time watermark and reads only new rows each interval
// rather than diffing cumulative counters.
package querylog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // database/sql driver "clickhouse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

const (
	// CollectInterval is the QAN aggregation period; collection is scheduled.
	// On minute boundaries to align buckets with the other QAN agents.
	collectInterval = time.Minute
	// QueryTimeout bounds a single system.query_log read or preflight check.
	queryTimeout = 30 * time.Second

	// Row type values from system.query_log.
	queryLogTypeQueryFinish              = 2
	queryLogTypeExceptionWhileProcessing = 4

	// ChangesBufferSize is the capacity of the changes channel; it absorbs a
	// few buckets before a slow consumer back-pressures the collection loop.
	changesBufferSize = 10
)

// requiredColumns are the system.query_log columns that must exist on every
// supported ClickHouse version; preflight rejects the table when any is
// missing. Optional columns (added in later releases) are detected via
// DESCRIBE TABLE and substituted with a typed zero default in buildSelectList,
// so the same agent works across ClickHouse versions.
var requiredColumns = []string{
	"type", "event_time", "query_id", "query",
	"query_duration_ms", "read_rows", "read_bytes",
	"memory_usage", "exception_code", "databases", "tables", "user",
}

// Params holds ClickHouseQueryLog construction parameters.
type Params struct {
	DSN            string
	AgentID        string
	MaxQueryLength int32
}

// ClickHouseQueryLog is the built-in QAN agent for ClickHouse system.query_log.
// It implements agents.BuiltinAgent.
type ClickHouseQueryLog struct {
	db             *sql.DB
	agentID        string
	maxQueryLength int32
	l              *logrus.Entry
	changes        chan agents.Change

	// watermark is the completion time of the newest row already processed;
	// only rows strictly newer than it are read on the next interval. It
	// starts at agent start time, so there is no historical back-fill.
	watermark time.Time
	// seenQueryIDs deduplicates rows whose event_time falls inside the
	// watermark second on ClickHouse versions without microsecond precision.
	seenQueryIDs map[string]struct{}
}

// New creates a ClickHouseQueryLog agent. The connection pool is opened lazily
// and validated on Run, so construction never blocks on an unreachable server.
func New(params Params, l *logrus.Entry) (*ClickHouseQueryLog, error) {
	db, err := sql.Open("clickhouse", params.DSN)
	if err != nil {
		return nil, fmt.Errorf("cannot open ClickHouse connection: %w", err)
	}
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	return &ClickHouseQueryLog{
		db:             db,
		agentID:        params.AgentID,
		maxQueryLength: params.MaxQueryLength,
		l:              l,
		changes:        make(chan agents.Change, changesBufferSize),
		watermark:      time.Now(),
		seenQueryIDs:   make(map[string]struct{}),
	}, nil
}

// Run reads system.query_log on minute boundaries and sends buckets to the
// changes channel until ctx is canceled.
func (m *ClickHouseQueryLog) Run(ctx context.Context) {
	defer func() {
		err := m.db.Close()
		if err != nil {
			m.l.Warnf("Failed to close ClickHouse connection: %s.", err)
		}
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(m.changes)
	}()

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	// running tracks whether the previous preflight/collection succeeded so
	// that status transitions (WAITING <-> RUNNING) are emitted exactly once.
	running := false
	columns, err := m.preflight(ctx)
	if err != nil {
		m.l.Warnf("Preflight failed, entering WAITING state: %s.", err)
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
	} else {
		running = true
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
	}

	start := time.Now()
	wait := start.Truncate(collectInterval).Add(collectInterval).Sub(start)
	m.l.Debugf("Scheduling next collection in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
	t := time.NewTimer(wait)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			m.l.Infof("Context canceled.")
			return

		case <-t.C:
			if !running {
				m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}
			}

			// Re-run preflight while not RUNNING: the table is created
			// lazily and log_queries may be toggled at runtime, so the
			// agent must auto-recover without a restart.
			if !running {
				columns, err = m.preflight(ctx)
				if err != nil {
					m.l.Warnf("Still not ready: %s.", err)
					m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
					m.reschedule(t, &start, &wait)
					continue
				}
			}

			lengthS := uint32(math.Round(wait.Seconds())) // round 59.9s/60.1s to 60s
			buckets, err := m.collect(ctx, columns, start, lengthS)
			m.reschedule(t, &start, &wait)

			if err != nil {
				m.l.Errorf("Collection failed: %s.", err)
				running = false
				m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
				continue
			}

			if !running {
				running = true
				m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
			}
			m.changes <- agents.Change{MetricsBucket: buckets}
		}
	}
}

// reschedule arms the timer for the next minute boundary and updates the
// interval start/wait bookkeeping in place.
func (m *ClickHouseQueryLog) reschedule(t *time.Timer, start *time.Time, wait *time.Duration) {
	*start = time.Now()
	*wait = start.Truncate(collectInterval).Add(collectInterval).Sub(*start)
	m.l.Debugf("Scheduling next collection in %s at %s.", *wait, start.Add(*wait).Format("15:04:05"))
	t.Reset(*wait)
}

// preflight verifies the server is reachable, system.query_log exists and
// query logging is enabled, then returns the set of usable column names. It
// returns an error (never panics) whenever the agent is not ready to collect.
func (m *ClickHouseQueryLog) preflight(ctx context.Context) (map[string]struct{}, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := m.db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot reach ClickHouse server: %w", err)
	}

	columns, err := m.describeQueryLog(ctx)
	if err != nil {
		return nil, fmt.Errorf("system.query_log is not available: %w", err)
	}
	for _, c := range requiredColumns {
		if _, ok := columns[c]; !ok {
			return nil, fmt.Errorf("system.query_log lacks required column %q", c)
		}
	}

	var logQueries uint8
	row := m.db.QueryRowContext(ctx,
		"SELECT value FROM system.settings WHERE name = 'log_queries'")
	err = row.Scan(&logQueries)
	if err != nil {
		return nil, fmt.Errorf("cannot read the log_queries setting: %w", err)
	}
	if logQueries == 0 {
		return nil, errors.New("log_queries is disabled; the agent cannot change server settings")
	}

	m.l.Infof("Preflight OK: system.query_log has %d columns, log_queries enabled.", len(columns))
	return columns, nil
}

// describeQueryLog returns the set of column names exposed by the running
// ClickHouse version's system.query_log table.
func (m *ClickHouseQueryLog) describeQueryLog(ctx context.Context) (map[string]struct{}, error) {
	rows, err := m.db.QueryContext(ctx, "DESCRIBE TABLE system.query_log")
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	columns := make(map[string]struct{})
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	// DESCRIBE returns name, type, default_type, default_expression, ... ;
	// scan only the first column and discard the rest.
	dest := make([]any, len(cols))
	var name string
	dest[0] = &name
	for i := 1; i < len(dest); i++ {
		dest[i] = new(sql.RawBytes)
	}
	for rows.Next() {
		err = rows.Scan(dest...)
		if err != nil {
			return nil, err
		}
		columns[name] = struct{}{}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return columns, nil
}

// collect reads new system.query_log rows, advances the watermark and builds
// metrics buckets for the interval.
func (m *ClickHouseQueryLog) collect(ctx context.Context, columns map[string]struct{}, periodStart time.Time, periodLengthSecs uint32) ([]*agentv1.MetricsBucket, error) { //nolint:lll
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := m.readRows(ctx, columns)
	if err != nil {
		return nil, err
	}

	buckets := makeBuckets(rows, m.maxQueryLength)
	// periodStart is a recent wall-clock time; clamp to the uint32 range so a
	// pathological clock cannot overflow the proto field.
	var startS uint32
	if unix := periodStart.Unix(); unix > 0 && unix <= int64(math.MaxUint32) {
		startS = uint32(unix)
	}
	for _, b := range buckets {
		b.Common.AgentId = m.agentID
		b.Common.PeriodStartUnixSecs = startS
		b.Common.PeriodLengthSecs = periodLengthSecs
	}

	m.l.Debugf("Made %d buckets out of %d query_log rows in %s+%d interval.",
		len(buckets), len(rows), periodStart.Format("15:04:05"), periodLengthSecs)
	return buckets, nil
}

// readRows selects every completed query newer than the watermark, advances
// the watermark, and refreshes the per-second dedup set.
func (m *ClickHouseQueryLog) readRows(ctx context.Context, columns map[string]struct{}) ([]queryLogRow, error) {
	hasMicro := has(columns, "event_time_microseconds")
	selectList := buildSelectList(columns)

	// event_time is selected with second granularity; >= keeps boundary-second
	// rows and the seenQueryIDs set removes the ones already counted.
	query := fmt.Sprintf( //nolint:gosec // column list is built from a fixed allow-list
		"SELECT %s FROM system.query_log "+
			"WHERE event_time >= ? AND type IN (%d, %d) "+
			"ORDER BY event_time",
		selectList, queryLogTypeQueryFinish, queryLogTypeExceptionWhileProcessing)

	sqlRows, err := m.db.QueryContext(ctx, query, m.watermark)
	if err != nil {
		return nil, fmt.Errorf("cannot read system.query_log: %w", err)
	}
	defer sqlRows.Close() //nolint:errcheck

	var result []queryLogRow
	newWatermark := m.watermark
	nextSeen := make(map[string]struct{})

	for sqlRows.Next() {
		row, err := scanRow(sqlRows, columns)
		if err != nil {
			return nil, fmt.Errorf("cannot scan system.query_log row: %w", err)
		}

		wm := row.watermark()
		// Drop rows already counted in a previous interval: anything strictly
		// before the watermark, or — on second-granular servers — the same
		// second already in the dedup set.
		if wm.Before(m.watermark) {
			continue
		}
		if !hasMicro && !wm.After(m.watermark) {
			if _, dup := m.seenQueryIDs[row.QueryID]; dup {
				continue
			}
		}

		result = append(result, row)

		switch {
		case wm.After(newWatermark):
			newWatermark = wm
			nextSeen = map[string]struct{}{row.QueryID: {}}
		case wm.Equal(newWatermark):
			nextSeen[row.QueryID] = struct{}{}
		}
	}
	err = sqlRows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating system.query_log rows: %w", err)
	}

	m.watermark = newWatermark
	m.seenQueryIDs = nextSeen
	return result, nil
}

// has reports whether a column is present in the detected schema.
func has(columns map[string]struct{}, name string) bool {
	_, ok := columns[name]
	return ok
}

// buildSelectList builds the SELECT column list, substituting a typed zero
// literal for every optional column the running ClickHouse version lacks so a
// uniform row layout is scanned regardless of server version.
func buildSelectList(columns map[string]struct{}) string {
	// Fixed order — scanRow relies on it.
	defaults := map[string]string{
		"event_time_microseconds": "toDateTime64(0, 6)",
		"normalized_query_hash":   "toUInt64(0)",
		"query_kind":              "''",
		"result_rows":             "toUInt64(0)",
		"result_bytes":            "toUInt64(0)",
		"written_rows":            "toUInt64(0)",
		"written_bytes":           "toUInt64(0)",
	}
	order := []string{
		"type", "event_time", "event_time_microseconds", "query_id", "query",
		"normalized_query_hash", "query_kind", "query_duration_ms",
		"read_rows", "read_bytes", "result_rows", "result_bytes",
		"memory_usage", "written_rows", "written_bytes",
		"exception_code", "databases", "tables", "user",
	}
	parts := make([]string, 0, len(order))
	for _, c := range order {
		if has(columns, c) {
			parts = append(parts, c)
			continue
		}
		if def, ok := defaults[c]; ok {
			parts = append(parts, fmt.Sprintf("%s AS %s", def, c))
			continue
		}
		// A required column is missing — preflight already rejected this, but
		// stay defensive and select a NULL placeholder rather than fail SQL.
		parts = append(parts, "NULL AS "+c)
	}
	return strings.Join(parts, ", ")
}

// scanRow scans one system.query_log row into a queryLogRow. The column order
// is the fixed order produced by buildSelectList.
func scanRow(rows *sql.Rows, _ map[string]struct{}) (queryLogRow, error) {
	var (
		r          queryLogRow
		typeVal    uint8
		databases  []string
		tables     []string
		queryKind  sql.NullString
		eventMicro time.Time
	)
	err := rows.Scan(
		&typeVal,
		&r.EventTime,
		&eventMicro,
		&r.QueryID,
		&r.Query,
		&r.NormalizedQueryHash,
		&queryKind,
		&r.QueryDurationMs,
		&r.ReadRows,
		&r.ReadBytes,
		&r.ResultRows,
		&r.ResultBytes,
		&r.MemoryUsage,
		&r.WrittenRows,
		&r.WrittenBytes,
		&r.ExceptionCode,
		&databases,
		&tables,
		&r.User,
	)
	if err != nil {
		return r, err
	}
	r.Type = typeVal
	r.QueryKind = queryKind.String
	r.Databases = databases
	r.Tables = tables
	// A zero DateTime64 means the column was absent; keep EventTimeMicro zero
	// so watermark() falls back to event_time.
	if !eventMicro.IsZero() && eventMicro.Unix() > 0 {
		r.EventTimeMicro = eventMicro
	}
	return r, nil
}

// percentile returns the p-th percentile (p in [0,1]) of values using the
// nearest-rank method. It is defined for any slice length: an empty slice
// yields 0, a single element yields that element, and an all-equal slice
// yields the shared value. The input is not mutated.
func percentile(values []float32, p float64) float32 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float32, len(values))
	copy(sorted, values)
	slices.Sort(sorted)

	rank := max(int(math.Ceil(p*float64(len(sorted))))-1, 0)
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

// Changes returns the channel that should be read until it is closed.
func (m *ClickHouseQueryLog) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector. The agent exposes no internal
// metrics, so it emits no descriptors.
func (m *ClickHouseQueryLog) Describe(chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector. The agent exposes no internal
// metrics.
func (m *ClickHouseQueryLog) Collect(chan<- prometheus.Metric) {}

// check interfaces.
var (
	_ agents.BuiltinAgent  = (*ClickHouseQueryLog)(nil)
	_ prometheus.Collector = (*ClickHouseQueryLog)(nil)
)
