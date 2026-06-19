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
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

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
	// picosecondsPerNanosecond is used to convert MySQL picosecond latencies into Go durations.
	picosecondsPerNanosecond = 1000
)

// currentQueriesSQL fetches currently running queries from the sys schema.
// sys.x$processlist is the machine-readable (raw) version of sys.processlist
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
func New(params *Params, l *logrus.Entry) (*MySQLRTA, error) {
	var files map[string]string
	if params.TextFiles != nil {
		files = params.TextFiles.Files
	}

	return &MySQLRTA{
		agentID:         params.AgentID,
		serviceID:       params.ServiceID,
		serviceName:     params.ServiceName,
		dsn:             params.DSN,
		files:           files,
		tlsSkipVerify:   params.TLSSkipVerify,
		collectInterval: params.CollectInterval,
		l:               l,
		changes:         make(chan agents.Change, changesBufferSize),
	}, nil
}

// Run extracts currently running DB queries from MySQL
// and sends it to the channel until ctx is canceled.
func (m *MySQLRTA) Run(ctx context.Context) {
	m.l.Info("Starting MySQL RTA agent")

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	// collectors tracks in-flight collection goroutines so we can wait for them
	// before closing m.changes, avoiding a "send on closed channel" race on shutdown.
	var collectors sync.WaitGroup

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
	if err := m.checkPrerequisites(ctx); err != nil {
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
			// Run collection in a separate goroutine to avoid blocking the main loop
			// and allow timely execution of next ticks in case collection takes longer
			// than the collect interval.
			collectors.Add(1)
			go func(curCtx context.Context) {
				defer collectors.Done()

				rtaQueryBucket, err := m.collectProcessList(curCtx)
				if err != nil {
					m.l.Warnf("processlist collection failed: %v", err)
					return
				}

				select {
				case <-curCtx.Done():
					return
				default:
					if len(rtaQueryBucket) != 0 {
						m.changes <- agents.Change{RTAQueriesBucket: rtaQueryBucket}
					}
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
	if err := m.db.QueryRowContext(checkCtx, "SELECT @@performance_schema").Scan(&performanceSchema); err != nil {
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
	_ = rows.Close()

	return nil
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

	var results []*rtav1.QueryData
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

		queryData := m.buildQueryData(row)
		queryData.QueryCollectTime = collectTime

		results = append(results, queryData)
	}

	if err := rows.Err(); err != nil {
		m.l.Warnf("Failed to iterate processlist rows: %v", err)
		return nil, err
	}

	return results, nil
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

	if err := rows.Scan(scanArgs...); err != nil {
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
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}

// buildQueryData converts a single sys.x$processlist row into a *QueryData.
// The complete row is preserved in QueryRawJson; a curated subset is exposed
// via the MySQL payload for the details view.
func (m *MySQLRTA) buildQueryData(row map[string]any) *rtav1.QueryData {
	execDuration := durationpb.New(time.Duration(mapFloat(row, "statement_latency")/picosecondsPerNanosecond) * time.Nanosecond)

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
	}

	rawJSON, err := json.MarshalIndent(row, "", "    ")
	if err != nil {
		m.l.Warnf("Failed to marshal raw query data: %v", err)
	}

	return &rtav1.QueryData{
		ServiceId:              m.serviceID,
		ServiceName:            m.serviceName,
		QueryId:                mapString(row, "conn_id"),
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
