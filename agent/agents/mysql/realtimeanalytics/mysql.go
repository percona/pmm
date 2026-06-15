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
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // register MySQL driver
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/tlshelpers"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	changesBufferSize = 10
	// minStatementLatencyPicoseconds is a minimal statement latency (10ms in picoseconds)
	// to report a running query. It mirrors the 0.01s threshold used by the MongoDB RTA agent
	// to filter out short-lived queries and reduce the amount of collected data.
	minStatementLatencyPicoseconds = int64(10 * time.Millisecond / time.Nanosecond * 1000) //nolint:mnd

	// queryTag is added to RTA queries so they can be excluded from the collected data.
	queryTag = "/* agent='rta-mysql' */"
)

// collectQuery fetches currently running statements from the sys schema processlist view.
// See https://dev.mysql.com/doc/refman/8.4/en/sys-processlist.html
// The x$ companion view is used as it exposes raw, unformatted values (picoseconds latency, raw SQL text).
// The agent's own monitoring queries are excluded by connection id and by the query tag.
const collectQuery = `SELECT conn_id, user, db, command, state, current_statement, statement_latency ` + queryTag + `
FROM sys.x$processlist
WHERE current_statement IS NOT NULL
  AND conn_id IS NOT NULL
  AND conn_id <> CONNECTION_ID()
  AND command = 'Query'
  AND statement_latency >= ?
  AND current_statement NOT LIKE '%` + "x$processlist" + `%'
ORDER BY statement_latency DESC`

// MySQLRTA extracts Real-Time Analytics data (currently running DB queries) from MySQL.
type MySQLRTA struct {
	agentID     string
	serviceID   string
	serviceName string
	l           *logrus.Entry

	// Channel to obtain data from this agent.
	changes chan agents.Change

	// DSN to connect to MySQL.
	mysqlDSN string
	// collectInterval is how often to collect data from MySQL.
	collectInterval time.Duration

	// db is the connection to MySQL, kept as a field to avoid reconnecting on every collection cycle.
	db *sql.DB
	// dbInstanceAddress is the MySQL instance address (host:port), fetched once on start.
	dbInstanceAddress string
}

// Params represent Agent parameters.
type Params struct {
	AgentID         string
	DSN             string             // DSN to connect to MySQL.
	ServiceID       string             // ServiceID shall be set in RTA queries to link them to the service.
	ServiceName     string             // ServiceName shall be set in RTA queries to link them to the service.
	CollectInterval time.Duration      // CollectInterval is how often to collect data from MySQL.
	TextFiles       *agentv1.TextFiles // TextFiles holds optional TLS certificates.
	TLSSkipVerify   bool               // TLSSkipVerify disables TLS certificate verification.
}

// New creates new MySQLRTA service.
func New(params *Params, l *logrus.Entry) (*MySQLRTA, error) {
	if params.TextFiles != nil {
		err := tlshelpers.RegisterMySQLCerts(params.TextFiles.Files, params.TLSSkipVerify)
		if err != nil {
			return nil, err
		}
	}

	return &MySQLRTA{
		agentID:         params.AgentID,
		serviceID:       params.ServiceID,
		serviceName:     params.ServiceName,
		mysqlDSN:        params.DSN,
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

	db, err := sql.Open("mysql", m.mysqlDSN)
	if err != nil {
		m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)

		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}

		return
	}

	// A single connection is enough to read the processlist periodically.
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	defer func() {
		_ = db.Close()
	}()

	if err = db.PingContext(ctx); err != nil {
		m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)

		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}

		return
	}

	m.db = db
	m.dbInstanceAddress = m.fetchInstanceAddress(ctx)

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}

	// fetch RTA data periodically
	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.l.Info("Stopping MySQL RTA agent")

			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			// m.changes channel will be closed in defer, so we don't need to close it here, just exit the function
			return
		case <-ticker.C:
			// We run collection in a separate goroutine to avoid blocking the main loop
			// and allow timely execution of next ticks in case collection/parsing takes longer
			// than the collect interval.
			go func(curCtx context.Context) {
				rtaQueryBucket, err := m.collectProcesslist(curCtx)
				if err != nil {
					m.l.Warnf("Processlist collection failed: %v", err)
					return
				}

				select {
				case <-curCtx.Done():
					// If context is done, we don't send anything to the channel.
					return
				default:
					if len(rtaQueryBucket) != 0 {
						// If we have data, send it to the channel.
						// If not, send only status without data to avoid triggering
						// unnecessary processing in the receiver.
						m.changes <- agents.Change{RTAQueriesBucket: rtaQueryBucket}
					}
				}
			}(ctx)
		}
	}
}

// collectProcesslist queries the sys schema processlist view and parses the result into a slice of *QueryData.
func (m *MySQLRTA) collectProcesslist(ctx context.Context) ([]*rtav1.QueryData, error) {
	rows, err := m.db.QueryContext(ctx, collectQuery, minStatementLatencyPicoseconds)
	if err != nil {
		return nil, fmt.Errorf("sys.x$processlist not available or permission denied: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var results []*rtav1.QueryData //nolint:prealloc
	currTime := timestamppb.New(time.Now())

	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		queryData, err := parseProcessRow(rows)
		if err != nil {
			m.l.Warnf("Failed to parse processlist row: %v", err)
			continue
		}

		queryData.ServiceId = m.serviceID
		queryData.ServiceName = m.serviceName
		queryData.QueryCollectTime = currTime
		if p, ok := queryData.Payload.(*rtav1.QueryData_MySqlPayload); ok {
			p.MySqlPayload.DbInstanceAddress = m.dbInstanceAddress
		}

		results = append(results, queryData)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate processlist rows: %w", err)
	}

	return results, nil
}

// fetchInstanceAddress returns the MySQL instance address (hostname:port) used to label collected queries.
// On any error it returns an empty string as the address is informational only.
func (m *MySQLRTA) fetchInstanceAddress(ctx context.Context) string {
	var hostname string
	var port int

	err := m.db.QueryRowContext(ctx, "SELECT @@hostname, @@port "+queryTag).Scan(&hostname, &port)
	if err != nil {
		m.l.Debugf("Failed to fetch MySQL instance address: %v", err)
		return ""
	}

	return fmt.Sprintf("%s:%d", hostname, port)
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
