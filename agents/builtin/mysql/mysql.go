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

// Package mysql runs built-in QAN Agent for MySQL.
package mysql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-agent/agents/backoff"
)

// MySQL QAN services connects to MySQL and extracts performance data.
type MySQL struct {
	params  *Params
	l       *logrus.Entry
	changes chan Change
	backoff *backoff.Backoff
}

// Params represent Agent parameters.
type Params struct {
	DSN string
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request qanpb.CollectRequest
}

// New creates new MySQL QAN service.
func New(params *Params, l *logrus.Entry) *MySQL {
	return &MySQL{
		params:  params,
		l:       l,
		changes: make(chan Change, 10),
		backoff: backoff.New(),
	}
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *MySQL) Run(ctx context.Context) {
	m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}
	defer func() {
		m.changes <- Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	sqlDB, err := sql.Open("mysql", m.params.DSN)
	if err != nil {
		m.l.Error(err)
		return
	}
	defer sqlDB.Close()
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)

	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(m.l.Tracef))
	t := time.NewTicker(time.Second)
	defer t.Stop()

	var running bool
	for {
		select {
		case <-ctx.Done():
			m.changes <- Change{Status: inventorypb.AgentStatus_STOPPING}
			m.l.Infof("Context canceled.")
			return

		case <-t.C:
			request, err := m.get(db.Querier)
			if err != nil {
				m.l.Error(err)
				running = false
				m.changes <- Change{Status: inventorypb.AgentStatus_WAITING}
				time.Sleep(time.Second)
				m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}
				continue
			}

			if !running {
				m.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}
				running = true
			}

			select {
			case <-ctx.Done():
				t.Stop()
				break
			case m.changes <- Change{Request: request}:
				// nothing
			}
		}
	}
}

func (m *MySQL) get(q *reform.Querier) (qanpb.CollectRequest, error) {
	var res qanpb.CollectRequest
	structs, err := q.SelectAllFrom(eventsStatementsSummaryByDigestView, "")
	if err != nil {
		return res, err
	}

	for _, str := range structs {
		ess := str.(*eventsStatementsSummaryByDigest)

		// skipping catch-all row
		if ess.Digest == nil || ess.DigestText == nil {
			m.l.Debugf("Skipping %s.", ess)
			continue
		}

		// From https://dev.mysql.com/doc/relnotes/mysql/8.0/en/news-8-0-11.html:
		// > The Performance Schema could produce DIGEST_TEXT values with a trailing space.
		// > This no longer occurs. (Bug #26908015)
		*ess.DigestText = strings.TrimSpace(*ess.DigestText)

		// TODO https://jira.percona.com/browse/PMM-3594
		/*
		   A ton of open questions. Should pmm-agent:
		   * check that performance schema is enabled?
		   * check that statement_digest consumer is enabled?
		   * check/report the value of performance_schema_digests_size?
		   * TRUNCATE events_statements_summary_by_digest before reading?
		   * read events_statements_summary_by_digest every second? every 10 seconds? minute? other interval?
		   * report rows with NULL digest?
		   * get query by digest from events_statements_history_long?
		   * check/report the value of performance_schema_events_statements_history_long_size?
		   * set conditions for FIRST_SEEN / LAST_SEEN? what conditions?
		   * group/aggregate results? how?
		   * should github.com/percona/go-mysql/event be used?
		*/

		res.MetricsBucket = append(res.MetricsBucket, &qanpb.MetricsBucket{
			Queryid:     *ess.Digest,
			Fingerprint: *ess.DigestText,
			DServer:     "TODO",
			DDatabase:   "TODO",
			DSchema:     "TODO",
		})
	}
	return res, nil
}

// Changes returns channel that should be read until it is closed.
func (m *MySQL) Changes() <-chan Change {
	return m.changes
}
