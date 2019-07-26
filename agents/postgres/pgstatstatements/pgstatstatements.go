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

// Package pgstatstatements runs built-in QAN Agent for PostgreSQL pg stats statements.
package pgstatstatements

import (
	"context"
	"database/sql"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/AlekSi/pointer"
	_ "github.com/lfittl/pg_query_go" // just to test build
	_ "github.com/lib/pq"             // register SQL driver
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-agent/agents"
)

const (
	retainStatStatements = 25 * time.Hour // make it work for daily queries
	queryStatStatements  = time.Minute
)

// PGStatStatementsQAN QAN services connects to PostgreSQL and extracts stats.
type PGStatStatementsQAN struct {
	q              *reform.Querier
	dbCloser       io.Closer
	agentID        string
	l              *logrus.Entry
	changes        chan agents.Change
	statementCache *statStatementCache
}

// Params represent Agent parameters.
type Params struct {
	DSN     string
	AgentID string
}

const queryTag = "pmm-agent:pgstatstatements"

// New creates new PGStatStatementsQAN QAN service.
func New(params *Params, l *logrus.Entry) (*PGStatStatementsQAN, error) {
	sqlDB, err := sql.Open("postgres", params.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)
	q := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(l.Tracef)).WithTag(queryTag)
	return newPgStatStatementsQAN(q, sqlDB, params.AgentID, l), nil
}

func newPgStatStatementsQAN(q *reform.Querier, dbCloser io.Closer, agentID string, l *logrus.Entry) *PGStatStatementsQAN {
	return &PGStatStatementsQAN{
		q:              q,
		dbCloser:       dbCloser,
		agentID:        agentID,
		l:              l,
		changes:        make(chan agents.Change, 10),
		statementCache: newStatStatementCache(retainStatStatements),
	}
}

// Run extracts stats data and sends it to the channel until ctx is canceled.
func (m *PGStatStatementsQAN) Run(ctx context.Context) {
	defer func() {
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	// add current stat statements to cache so they are not send as new on first iteration with incorrect timestamps
	var running bool
	m.changes <- agents.Change{Status: inventorypb.AgentStatus_STARTING}
	if s, err := getStatStatements(m.q); err == nil {
		m.statementCache.refresh(s)
		m.l.Debugf("Got %d initial stat statements.", len(s))
		running = true
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_RUNNING}
	} else {
		m.l.Error(err)
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_WAITING}
	}

	// query pg_stat_statements every minute at 00 seconds
	start := time.Now()
	wait := start.Truncate(queryStatStatements).Add(queryStatStatements).Sub(start)
	m.l.Debugf("Scheduling next collection in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
	t := time.NewTimer(wait)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			m.changes <- agents.Change{Status: inventorypb.AgentStatus_STOPPING}
			m.l.Infof("Context canceled.")
			return

		case <-t.C:
			if !running {
				m.changes <- agents.Change{Status: inventorypb.AgentStatus_STARTING}
			}

			lengthS := uint32(math.Round(wait.Seconds())) // round 59.9s/60.1s to 60s
			buckets, err := m.getNewBuckets(start, lengthS)

			start = time.Now()
			wait = start.Truncate(queryStatStatements).Add(queryStatStatements).Sub(start)
			m.l.Debugf("Scheduling next collection in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
			t.Reset(wait)

			if err != nil {
				m.l.Error(err)
				running = false
				m.changes <- agents.Change{Status: inventorypb.AgentStatus_WAITING}
				continue
			}

			if !running {
				running = true
				m.changes <- agents.Change{Status: inventorypb.AgentStatus_RUNNING}
			}

			m.changes <- agents.Change{MetricsBucket: buckets}
		}
	}
}

func (m *PGStatStatementsQAN) getNewBuckets(periodStart time.Time, periodLengthSecs uint32) ([]*agentpb.MetricsBucket, error) {
	current, err := getStatStatements(m.q)
	if err != nil {
		return nil, err
	}
	prev := m.statementCache.get()

	buckets := makeBuckets(m.q, current, prev, m.l)
	startS := uint32(periodStart.Unix())
	m.l.Debugf("Made %d buckets out of %d stat statements in %s+%d interval.",
		len(buckets), len(current), periodStart.Format("15:04:05"), periodLengthSecs)

	// merge prev and current in cache
	m.statementCache.refresh(current)

	// add agent_id and timestamps
	for i, b := range buckets {
		b.Common.AgentId = m.agentID
		b.Common.PeriodStartUnixSecs = startS
		b.Common.PeriodLengthSecs = periodLengthSecs

		buckets[i] = b
	}

	return buckets, nil
}

// makeBuckets uses current state of pg_stat_statements table and accumulated previous state
// to make metrics buckets.
//
// makeBuckets is a pure function for easier testing.
func makeBuckets(q *reform.Querier, current, prev map[int64]*pgStatStatements, l *logrus.Entry) []*agentpb.MetricsBucket {
	res := make([]*agentpb.MetricsBucket, 0, len(current))

	for queryID, currentPSS := range current {
		prevPSS := prev[queryID]
		if prevPSS == nil {
			prevPSS = new(pgStatStatements)
		}
		count := float32(currentPSS.Calls - prevPSS.Calls)
		switch {
		case count == 0:
			// TODO
			// Another way how this is possible is if pg_stat_statements was truncated,
			// and then the same number of queries were made.
			// Currently, we can't differentiate between those situations.
			l.Debugf("Skipped due to the same number of queries: %s.", currentPSS)
			continue
		case count < 0:
			l.Debugf("Truncate detected. Treating as a new query: %s.", currentPSS)
			prevPSS = new(pgStatStatements)
			count = float32(currentPSS.Calls)
		case prevPSS.Calls == 0:
			l.Debugf("New query: %s.", currentPSS)
		default:
			l.Debugf("Normal query: %s.", currentPSS)
		}
		pgStatDatabase := &pgStatDatabase{DatID: currentPSS.DBID}
		err := q.FindOneTo(pgStatDatabase, "datid", currentPSS.DBID)
		if err != nil {
			l.Debugf("Can't get db name for db: %d. %s", currentPSS.DBID, err)
		}
		pgUser := &pgUser{UserID: currentPSS.UserID}
		err = q.FindOneTo(pgUser, "usesysid", currentPSS.UserID)
		if err != nil {
			l.Debugf("Can't get username name for user: %d. %s", currentPSS.DBID, err)
		}

		mb := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Schema:      pointer.GetString(pgStatDatabase.DatName),
				Username:    pointer.GetString(pgUser.UserName),
				Queryid:     strconv.FormatInt(*currentPSS.QueryID, 10),
				Fingerprint: *currentPSS.Query,
				NumQueries:  count,
				//NumQueriesWithErrors:   float32(currentPSS.SumErrors - prevPSS.SumErrors),
				//NumQueriesWithWarnings: float32(currentPSS.SumWarnings - prevPSS.SumWarnings),
				AgentType: inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{},
		}

		for _, p := range []struct {
			value float32  // result value: currentPSS.SumXXX-prevPSS.SumXXX
			sum   *float32 // MetricsBucket.XXXSum field to write value
			cnt   *float32 // MetricsBucket.XXXCnt field to write count
		}{
			// convert milliseconds to seconds
			{float32(currentPSS.TotalTime-prevPSS.TotalTime) / 1000, &mb.Common.MQueryTimeSum, &mb.Common.MQueryTimeCnt},
			{float32(currentPSS.Rows - prevPSS.Rows), &mb.Postgresql.MRowsSum, &mb.Postgresql.MRowsCnt},

			{float32(currentPSS.SharedBlksHit - prevPSS.SharedBlksHit), &mb.Postgresql.MSharedBlksHitSum, &mb.Postgresql.MSharedBlksHitCnt},
			{float32(currentPSS.SharedBlksRead - prevPSS.SharedBlksRead), &mb.Postgresql.MSharedBlksReadSum, &mb.Postgresql.MSharedBlksReadCnt},
			{float32(currentPSS.SharedBlksDirtied - prevPSS.SharedBlksDirtied), &mb.Postgresql.MSharedBlksDirtiedSum, &mb.Postgresql.MSharedBlksDirtiedCnt},
			{float32(currentPSS.SharedBlksWritten - prevPSS.SharedBlksWritten), &mb.Postgresql.MSharedBlksWrittenSum, &mb.Postgresql.MSharedBlksWrittenCnt},

			{float32(currentPSS.LocalBlksHit - prevPSS.LocalBlksHit), &mb.Postgresql.MLocalBlksHitSum, &mb.Postgresql.MLocalBlksHitCnt},
			{float32(currentPSS.LocalBlksRead - prevPSS.LocalBlksRead), &mb.Postgresql.MLocalBlksReadSum, &mb.Postgresql.MLocalBlksReadCnt},
			{float32(currentPSS.LocalBlksDirtied - prevPSS.LocalBlksDirtied), &mb.Postgresql.MLocalBlksDirtiedSum, &mb.Postgresql.MLocalBlksDirtiedCnt},
			{float32(currentPSS.LocalBlksWritten - prevPSS.LocalBlksWritten), &mb.Postgresql.MLocalBlksWrittenSum, &mb.Postgresql.MLocalBlksWrittenCnt},

			{float32(currentPSS.TempBlksRead - prevPSS.TempBlksRead), &mb.Postgresql.MTempBlksReadSum, &mb.Postgresql.MTempBlksReadCnt},
			{float32(currentPSS.TempBlksWritten - prevPSS.TempBlksWritten), &mb.Postgresql.MTempBlksWrittenSum, &mb.Postgresql.MTempBlksWrittenCnt},

			{float32(currentPSS.BlkReadTime - prevPSS.BlkReadTime), &mb.Postgresql.MBlkReadTimeSum, &mb.Postgresql.MBlkReadTimeCnt},
			{float32(currentPSS.BlkWriteTime - prevPSS.BlkWriteTime), &mb.Postgresql.MBlkWriteTimeSum, &mb.Postgresql.MBlkWriteTimeCnt},
		} {
			if p.value != 0 {
				*p.sum = p.value
				*p.cnt = count
			}
		}

		res = append(res, mb)
	}

	return res
}

// Changes returns channel that should be read until it is closed.
func (m *PGStatStatementsQAN) Changes() <-chan agents.Change {
	return m.changes
}
