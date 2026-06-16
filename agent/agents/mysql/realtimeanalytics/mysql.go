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
	"fmt"
	"strconv"
	"strings"
	"time"

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
	changesBufferSize = 10
	// picosecondsPerNanosecond is used to convert MySQL picosecond latencies into Go durations.
	picosecondsPerNanosecond = 1000
)

// currentQueriesSQL fetches currently running queries from the sys schema.
// sys.x$processlist is the machine-readable (raw) version of sys.processlist
// (https://dev.mysql.com/doc/refman/8.4/en/sys-processlist.html); it exposes
// the same columns but with unformatted numeric latencies.
// We exclude background threads, idle ("Sleep") connections, the RTA agent's
// own connection and rows without a current statement.
const currentQueriesSQL = `
SELECT
    conn_id,
    COALESCE(user, ''),
    COALESCE(db, ''),
    COALESCE(command, ''),
    COALESCE(state, ''),
    COALESCE(statement_latency, 0),
    COALESCE(current_statement, ''),
    COALESCE(rows_examined, 0),
    COALESCE(rows_sent, 0),
    COALESCE(full_scan, ''),
    COALESCE(program_name, '')
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

	defer func() {
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}

		close(m.changes)
	}()

	db, addr, err := createConnection(ctx, m.dsn, m.files, m.tlsSkipVerify)
	if err != nil {
		m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)

		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}

		return
	}

	defer func() {
		_ = db.Close()
	}()

	m.db = db
	m.dbInstanceAddress = addr

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
			go func(curCtx context.Context) {
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

// collectProcessList queries sys.x$processlist and parses the result into a slice of *QueryData.
func (m *MySQLRTA) collectProcessList(ctx context.Context) ([]*rtav1.QueryData, error) {
	queryCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	rows, err := m.db.QueryContext(queryCtx, currentQueriesSQL)
	if err != nil {
		return nil, fmt.Errorf("sys.x$processlist not available or permission denied: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	collectTime := timestamppb.New(time.Now())

	var results []*rtav1.QueryData
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var r processlistRow
		if err := rows.Scan(&r.connID, &r.user, &r.db, &r.command, &r.state, &r.latencyPicos,
			&r.currentStmt, &r.rowsExamined, &r.rowsSent, &r.fullScan, &r.programName); err != nil {
			m.l.Warnf("Failed to scan processlist row: %v", err)
			continue
		}

		queryData := m.buildQueryData(&r)
		queryData.QueryCollectTime = collectTime

		results = append(results, queryData)
	}

	if err := rows.Err(); err != nil {
		m.l.Warnf("Failed to iterate processlist rows: %v", err)
		return nil, err
	}

	return results, nil
}

// processlistRow holds a single row scanned from sys.x$processlist.
type processlistRow struct {
	connID       uint64
	user         string
	db           string
	command      string
	state        string
	latencyPicos float64
	currentStmt  string
	rowsExamined int64
	rowsSent     int64
	fullScan     string
	programName  string
}

// buildQueryData converts a single sys.x$processlist row into a *QueryData.
func (m *MySQLRTA) buildQueryData(r *processlistRow) *rtav1.QueryData {
	execDuration := durationpb.New(time.Duration(r.latencyPicos/picosecondsPerNanosecond) * time.Nanosecond)

	mysqlPayload := &rtav1.QueryMySQLData{
		DbInstanceAddress: m.dbInstanceAddress,
		ProgramName:       r.programName,
		DatabaseName:      r.db,
		Command:           r.command,
		State:             r.state,
		Username:          r.user,
		RowsExamined:      r.rowsExamined,
		RowsSent:          r.rowsSent,
		FullScan:          strings.EqualFold(r.fullScan, "YES"),
	}

	rawJSON, err := json.Marshal(map[string]any{
		"conn_id":           r.connID,
		"user":              r.user,
		"db":                r.db,
		"command":           r.command,
		"state":             r.state,
		"statement_latency": r.latencyPicos,
		"current_statement": r.currentStmt,
		"rows_examined":     r.rowsExamined,
		"rows_sent":         r.rowsSent,
		"full_scan":         r.fullScan,
		"program_name":      r.programName,
	})
	if err != nil {
		m.l.Warnf("Failed to marshal raw query data: %v", err)
	}

	return &rtav1.QueryData{
		ServiceId:              m.serviceID,
		ServiceName:            m.serviceName,
		QueryId:                strconv.FormatUint(r.connID, 10),
		QueryText:              r.currentStmt,
		QueryRawJson:           string(rawJSON),
		QueryExecutionDuration: execDuration,
		Payload: &rtav1.QueryData_MySqlPayload{
			MySqlPayload: mysqlPayload,
		},
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
