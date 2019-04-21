// pmm-agent
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package slowlog runs built-in QAN Agent for MySQL slow log.
package slowlog

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"
)

const (
	queryInterval = time.Minute
)

// SlowLog extracts performance data from MySQL slow log.
type SlowLog struct {
	db      *reform.DB
	agentID string
	l       *logrus.Entry
	changes chan Change
}

// Params represent Agent parameters.
type Params struct {
	DSN     string
	AgentID string
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request *qanpb.CollectRequest
}

// New creates new MySQL QAN service.
func New(params *Params, l *logrus.Entry) (*SlowLog, error) {
	sqlDB, err := sql.Open("mysql", params.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(l.Tracef))

	return newMySQL(db, params.AgentID, l), nil
}

func newMySQL(db *reform.DB, agentID string, l *logrus.Entry) *SlowLog {
	return &SlowLog{
		db:      db,
		agentID: agentID,
		l:       l,
		changes: make(chan Change, 10),
	}
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *SlowLog) Run(ctx context.Context) {
	defer func() {
		m.db.DBInterface().(*sql.DB).Close() //nolint:errcheck
		m.changes <- Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	var running bool
	m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}

	start := time.Now().Truncate(0) // strip monotoning clock reading
	wait := start.Truncate(queryInterval).Add(queryInterval).Sub(start)
	m.l.Debugf("Scheduling next collection in %s at %s.", wait, start.Add(wait))
	t := time.NewTimer(wait)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			m.changes <- Change{Status: inventorypb.AgentStatus_STOPPING}
			m.l.Infof("Context canceled.")
			return

		case <-t.C:
			if !running {
				m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}
			}

			buckets, err := m.getNewBuckets()

			start = time.Now().Truncate(0) // strip monotoning clock reading
			wait = start.Truncate(queryInterval).Add(queryInterval).Sub(start)
			m.l.Debugf("Scheduling next collection in %s at %s.", wait, start.Add(wait))
			t.Reset(wait)

			if err != nil {
				m.l.Error(err)
				running = false
				m.changes <- Change{Status: inventorypb.AgentStatus_WAITING}
				continue
			}

			if !running {
				running = true
				m.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}
			}

			m.changes <- Change{Request: &qanpb.CollectRequest{MetricsBucket: buckets}}
		}
	}
}

func (m *SlowLog) getNewBuckets() ([]*qanpb.MetricsBucket, error) {
	// TODO add AgentUUID to buckets
	return makeBuckets()
}

// makeBuckets XXX.
//
// makeBuckets is a pure function for easier testing.
func makeBuckets() ([]*qanpb.MetricsBucket, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// Changes returns channel that should be read until it is closed.
func (m *SlowLog) Changes() <-chan Change {
	return m.changes
}
