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

// Package pgstatstatements runs built-in QAN Agent for PostgreSQL pg stats statements.
package pgstatstatements

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	_ "github.com/lib/pq" // register SQL driver
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/agents/cache"
	"github.com/percona/pmm/agent/queryparser"
	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/utils/sqlmetrics"
)

const (
	retainStatStatements = 25 * time.Hour // make it work for daily queries
	defaultPgssCacheSize = 5000           // cache size rows limit
	queryStatStatements  = time.Minute
)

var (
	pgStatVer1_8  = semver.MustParse("1.8.0")
	pgStatVer1_11 = semver.MustParse("1.11.0")
)

type statementsMap map[int64]*pgStatStatementsExtended

// PGStatStatementsQAN QAN services connects to PostgreSQL and extracts stats.
type PGStatStatementsQAN struct { //nolint:revive
	q                      *reform.Querier
	dbCloser               io.Closer
	agentID                string
	maxQueryLength         int32
	disableCommentsParsing bool
	l                      *logrus.Entry
	changes                chan agents.Change
	statementsCache        *statementsCache
}

// Params represent Agent parameters.
type Params struct {
	DSN                    string
	AgentID                string
	MaxQueryLength         int32
	DisableCommentsParsing bool
	TextFiles              *agentv1.TextFiles
}

const (
	queryTag     = "agent='pgstatstatements'"
	pgssMaxQuery = "SELECT /* " + queryTag + " */ setting FROM pg_settings WHERE name = 'pg_stat_statements.max'"
)

// New creates new PGStatStatementsQAN QAN service.
func New(params *Params, l *logrus.Entry) (*PGStatStatementsQAN, error) {
	sqlDB, err := sql.Open("postgres", params.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)

	reformL := sqlmetrics.NewReform("postgres", params.AgentID, l.Tracef)
	// TODO register reformL metrics https://jira.percona.com/browse/PMM-4087
	q := reform.NewDB(sqlDB, postgresql.Dialect, reformL).WithTag(queryTag)
	return newPgStatStatementsQAN(q, sqlDB, params.AgentID, params.MaxQueryLength, params.DisableCommentsParsing, l)
}

func getPgStatStatementsCacheSize(q *reform.Querier, l *logrus.Entry) uint {
	var pgSSCacheSize uint
	err := q.QueryRow(pgssMaxQuery).Scan(&pgSSCacheSize)
	if err != nil {
		l.WithError(err).Error("failed to get pg_stat_statements.max")
		return defaultPgssCacheSize
	}

	return pgSSCacheSize
}

func newPgStatStatementsQAN(q *reform.Querier, dbCloser io.Closer, agentID string, maxQueryLength int32, disableCommentsParsing bool, l *logrus.Entry) (*PGStatStatementsQAN, error) { //nolint:lll
	cacheSize := getPgStatStatementsCacheSize(q, l)
	statementCache, err := newStatementsCache(statementsMap{}, retainStatStatements, cacheSize, l)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create cache")
	}

	return &PGStatStatementsQAN{
		q:                      q,
		dbCloser:               dbCloser,
		agentID:                agentID,
		maxQueryLength:         maxQueryLength,
		disableCommentsParsing: disableCommentsParsing,
		l:                      l,
		changes:                make(chan agents.Change, 10),
		statementsCache:        statementCache,
	}, nil
}

func getPgStatVersion(q *reform.Querier) (semver.Version, error) {
	var v string
	var pgVersion semver.Version
	err := q.QueryRow(fmt.Sprintf("SELECT /* %s */ extVersion FROM pg_extension WHERE pg_extension.extname = 'pg_stat_statements'", queryTag)).Scan(&v)
	if err != nil {
		return pgVersion, err
	}

	switch strings.Count(v, ".") {
	case 1:
		v += ".0"
	case 0:
		v += ".0.0"
	}

	return semver.Parse(v)
}

// Run extracts stats data and sends it to the channel until ctx is canceled.
func (m *PGStatStatementsQAN) Run(ctx context.Context) {
	defer func() {
		m.dbCloser.Close() //nolint:errcheck
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(m.changes)
	}()

	// add current stat statements to cache, so they are not send as new on first iteration with incorrect timestamps
	var running bool
	var err error
	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	if current, _, err := m.getStatStatementsExtended(ctx); err == nil {
		if err = m.statementsCache.Set(current); err == nil {
			m.l.Debugf("Got %d initial stat statements.", len(current))
			running = true
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
		}
	}

	if err != nil {
		m.l.Error(err)
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
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
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			m.l.Infof("Context canceled.")
			return

		case <-t.C:
			if !running {
				m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}
			}

			lengthS := uint32(math.Round(wait.Seconds())) // round 59.9s/60.1s to 60s
			buckets, err := m.getNewBuckets(ctx, start, lengthS)

			start = time.Now()
			wait = start.Truncate(queryStatStatements).Add(queryStatStatements).Sub(start)
			m.l.Debugf("Scheduling next collection in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
			t.Reset(wait)

			if err != nil {
				m.l.Error(err)
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

// getStatStatementsExtended returns the current state of pg_stat_statements table with extended information (database, username, tables)
// and the previous cashed state.
func (m *PGStatStatementsQAN) getStatStatementsExtended(
	ctx context.Context,
) (statementsMap, statementsMap, error) {
	var totalN, newN, newSharedN, oldN int
	var err error
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		m.l.Debugf("Selected %d rows from pg_stat_statements in %s: %d new (%d shared tables), %d old.", totalN, dur, newN, newSharedN, oldN)
	}()

	current := make(statementsMap, m.statementsCache.cache.Len())
	prev := make(statementsMap, m.statementsCache.cache.Len())
	if err := m.statementsCache.Get(prev); err != nil {
		return nil, nil, err
	}

	q := m.q

	// load all databases and usernames first as we can't use querier while iterating over rows below
	databases := queryDatabases(q)
	usernames := queryUsernames(q)

	pgStatVersion, err := getPgStatVersion(q)
	if err != nil {
		return nil, nil, err
	}

	row, view := newPgStatMonitorStructs(pgStatVersion)
	columns := strings.Join(q.QualifiedColumns(view), ", ")

	rows, err := q.Query(fmt.Sprintf("SELECT /* %s */ %s FROM %s %s", queryTag, columns, q.QualifiedView(view), "WHERE queryid IS NOT NULL AND query IS NOT NULL"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get rows from pg_stat_statements")
	}
	defer rows.Close() //nolint:errcheck

	for ctx.Err() == nil {
		if err = q.NextRow(row, rows); err != nil {
			if errors.Is(err, reform.ErrNoRows) {
				err = nil
			}
			break
		}
		totalN++
		c := &pgStatStatementsExtended{
			pgStatStatements: *row,
			Database:         databases[row.DBID],
			Username:         usernames[row.UserID],
			RealQuery:        row.Query,
		}

		if p := prev[c.QueryID]; p != nil {
			oldN++
			newSharedN++

			c.Tables = p.Tables
			c.Query, c.IsQueryTruncated = p.Query, p.IsQueryTruncated
		} else {
			newN++

			c.Query, c.IsQueryTruncated = truncate.Query(c.Query, m.maxQueryLength, truncate.GetDefaultMaxQueryLength())
		}

		current[c.QueryID] = c
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err != nil {
		err = errors.Wrap(err, "failed to fetch pg_stat_statements")
	}

	return current, prev, err
}

func (m *PGStatStatementsQAN) getNewBuckets(ctx context.Context, periodStart time.Time, periodLengthSecs uint32) ([]*agentv1.MetricsBucket, error) {
	current, prev, err := m.getStatStatementsExtended(ctx)
	if err != nil {
		return nil, err
	}

	buckets := m.makeBuckets(current, prev)
	startS := uint32(periodStart.Unix())
	m.l.Debugf("Made %d buckets out of %d stat statements in %s+%d interval.",
		len(buckets), len(current), periodStart.Format("15:04:05"), periodLengthSecs)

	// merge prev and current in cache
	if err = m.statementsCache.Set(current); err != nil {
		return nil, err
	}
	m.l.Debugf("statStatementsCache: %s", m.statementsCache.cache.Stats())

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
// to make metrics buckets. It's a pure function for easier testing.
func (m *PGStatStatementsQAN) makeBuckets(current, prev statementsMap) []*agentv1.MetricsBucket {
	res := make([]*agentv1.MetricsBucket, 0, len(current))
	l := m.l

	for queryID, currentPSS := range current {
		prevPSS := prev[queryID]
		if prevPSS == nil {
			prevPSS = &pgStatStatementsExtended{}
		}
		count := float32(currentPSS.Calls - prevPSS.Calls)
		switch {
		case count == 0:
			// Another way how this is possible is if pg_stat_statements was truncated,
			// and then the same number of queries were made.
			// Currently, we can't differentiate between those situations.
			l.Tracef("Skipped due to the same number of queries: %s.", currentPSS)
			continue
		case count < 0:
			l.Debugf("Truncate detected. Treating as a new query: %s.", currentPSS)
			prevPSS = &pgStatStatementsExtended{}
			count = float32(currentPSS.Calls)
		case prevPSS.Calls == 0:
			l.Debugf("New query: %s.", currentPSS)
		default:
			l.Debugf("Normal query: %s.", currentPSS)
		}

		if len(currentPSS.Tables) == 0 {
			currentPSS.Tables = extractTables(currentPSS.RealQuery, m.maxQueryLength, l)
		}

		if !m.disableCommentsParsing {
			comments, err := queryparser.PostgreSQLComments(currentPSS.Query)
			if err != nil {
				l.Errorf("failed to parse comments for query: %s", currentPSS.Query)
			}
			currentPSS.Comments = comments
		}

		mb := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Database:    currentPSS.Database,
				Tables:      currentPSS.Tables,
				Username:    currentPSS.Username,
				Queryid:     strconv.FormatInt(currentPSS.QueryID, 10),
				Comments:    currentPSS.Comments,
				Fingerprint: currentPSS.Query,
				NumQueries:  count,
				AgentType:   inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				IsTruncated: currentPSS.IsQueryTruncated,
			},
			Postgresql: &agentv1.MetricsBucket_PostgreSQL{},
		}

		for _, p := range []struct {
			value float32  // result value: currentPSS.SumXXX-prevPSS.SumXXX
			sum   *float32 // MetricsBucket.XXXSum field to write value
			cnt   *float32 // MetricsBucket.XXXCnt field to write count
		}{
			// convert milliseconds to seconds
			{float32(currentPSS.TotalExecTime-prevPSS.TotalExecTime) / 1000, &mb.Common.MQueryTimeSum, &mb.Common.MQueryTimeCnt},
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

			// convert milliseconds to seconds
			{float32(currentPSS.SharedBlkReadTime-prevPSS.SharedBlkReadTime) / 1000, &mb.Postgresql.MSharedBlkReadTimeSum, &mb.Postgresql.MSharedBlkReadTimeCnt},
			{float32(currentPSS.SharedBlkWriteTime-prevPSS.SharedBlkWriteTime) / 1000, &mb.Postgresql.MSharedBlkWriteTimeSum, &mb.Postgresql.MSharedBlkWriteTimeCnt},
			{float32(currentPSS.LocalBlkReadTime-prevPSS.LocalBlkReadTime) / 1000, &mb.Postgresql.MLocalBlkReadTimeSum, &mb.Postgresql.MLocalBlkReadTimeCnt},
			{float32(currentPSS.LocalBlkWriteTime-prevPSS.LocalBlkWriteTime) / 1000, &mb.Postgresql.MLocalBlkWriteTimeSum, &mb.Postgresql.MLocalBlkWriteTimeCnt},
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

// Describe implements prometheus.Collector.
func (m *PGStatStatementsQAN) Describe(ch chan<- *prometheus.Desc) { //nolint:revive
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (m *PGStatStatementsQAN) Collect(ch chan<- prometheus.Metric) {
	stats := m.statementsCache.cache.Stats()
	metrics := cache.MetricsFromStats(stats, m.agentID, "")
	for _, metric := range metrics {
		ch <- metric
	}
}

// check interfaces.
var (
	_ prometheus.Collector = (*PGStatStatementsQAN)(nil)
)
