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
	"time"

	"github.com/AlekSi/pointer"
	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"
)

const (
	retainHistory  = 5 * time.Minute
	refreshHistory = 5 * time.Second

	retainSummaries = 25 * time.Hour // make it work for daily queries
	querySummaries  = time.Minute
)

// MySQL QAN services connects to MySQL and extracts performance data.
type MySQL struct {
	db           *reform.DB
	l            *logrus.Entry
	changes      chan Change
	historyCache *historyCache
	summaryCache *summaryCache
}

// Params represent Agent parameters.
type Params struct {
	DSN string
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request *qanpb.CollectRequest
}

// New creates new MySQL QAN service.
func New(params *Params, l *logrus.Entry) (*MySQL, error) {
	sqlDB, err := sql.Open("mysql", params.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(l.Tracef))

	return newMySQL(db, l), nil
}

func newMySQL(db *reform.DB, l *logrus.Entry) *MySQL {
	return &MySQL{
		db:           db,
		l:            l,
		changes:      make(chan Change, 10),
		historyCache: newHistoryCache(retainHistory),
		summaryCache: newSummaryCache(retainSummaries),
	}
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *MySQL) Run(ctx context.Context) {
	defer func() {
		m.db.DBInterface().(*sql.DB).Close() //nolint:errcheck
		m.changes <- Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	// add current summaries to cache so they are not send as new on first iteration with incorrect timestamps
	var running bool
	m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}
	if s, err := getSummaries(m.db.Querier); err == nil {
		m.summaryCache.refresh(s)
		m.l.Debugf("Got %d initial summaries.", len(s))
		running = true
		m.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}
	} else {
		m.l.Error(err)
		m.changes <- Change{Status: inventorypb.AgentStatus_WAITING}
	}

	go m.runHistoryCacheRefresher(ctx)

	// query events_statements_summary_by_digest every minute at 00 seconds
	start := time.Now().Truncate(0) // strip monotoning clock reading
	wait := start.Truncate(querySummaries).Add(querySummaries).Sub(start)
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

			buckets, err := m.getNewBuckets(start, wait)

			start = time.Now().Truncate(0) // strip monotoning clock reading
			wait = start.Truncate(querySummaries).Add(querySummaries).Sub(start)
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

func (m *MySQL) runHistoryCacheRefresher(ctx context.Context) {
	t := time.NewTicker(refreshHistory)
	defer t.Stop()

	for {
		if err := m.refreshHistoryCache(); err != nil {
			m.l.Error(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// nothing, continue loop
		}
	}
}

func (m *MySQL) refreshHistoryCache() error {
	current, err := getHistory(m.db.Querier)
	if err != nil {
		return err
	}
	m.historyCache.refresh(current)
	return nil
}

func (m *MySQL) getNewBuckets(periodStart time.Time, periodLength time.Duration) ([]*qanpb.MetricsBucket, error) {
	current, err := getSummaries(m.db.Querier)
	if err != nil {
		return nil, err
	}
	prev := m.summaryCache.get()

	buckets := makeBuckets(current, prev, m.l)
	m.l.Debugf("Made %d buckets out of %d summaries.", len(buckets), len(current))

	// merge prev and current in cache
	m.summaryCache.refresh(current)

	// add timestamps and examples from history cache
	startS := uint32(periodStart.Unix())
	lengthS := uint32(periodLength.Seconds())
	history := m.historyCache.get()
	for i, b := range buckets {
		b.PeriodStartUnixSecs = startS
		b.PeriodLengthSecs = lengthS

		if esh := history[b.Queryid]; esh != nil {
			// TODO test if we really need that
			if b.DSchema == "" {
				b.DSchema = pointer.GetString(esh.CurrentSchema)
			}

			if esh.SQLText != nil {
				b.Example = *esh.SQLText
				b.ExampleFormat = qanpb.ExampleFormat_EXAMPLE
				b.ExampleType = qanpb.ExampleType_RANDOM
			}
		}

		buckets[i] = b
	}

	return buckets, nil
}

// makeBuckets uses current state of events_statements_summary_by_digest table and accumulated previous state
// to make metrics buckets.
//
// makeBuckets is a pure function for easier testing.
func makeBuckets(current, prev map[string]*eventsStatementsSummaryByDigest, l *logrus.Entry) []*qanpb.MetricsBucket {
	res := make([]*qanpb.MetricsBucket, 0, len(current))

	for digest, currentESS := range current {
		prevESS := prev[digest]
		if prevESS == nil {
			prevESS = new(eventsStatementsSummaryByDigest)
		}
		count := float32(currentESS.CountStar - prevESS.CountStar)
		switch {
		case count == 0:
			// TODO
			// Another way how this is possible is if events_statements_summary_by_digest was truncated,
			// and then the same number of queries were made.
			// Currently, we can't differentiate between those situations.
			// We probably could by using first_seen/last_seen columns.
			l.Debugf("Skipped due to the same number of queries: %s.", currentESS)
			continue
		case count < 0:
			l.Debugf("Truncate detected. Treating as a new query: %s.", currentESS)
			prevESS = new(eventsStatementsSummaryByDigest)
			count = float32(currentESS.CountStar)
		case prevESS.CountStar == 0:
			l.Debugf("New query: %s.", currentESS)
		default:
			l.Debugf("Normal query: %s.", currentESS)
		}

		mb := &qanpb.MetricsBucket{
			DSchema:                pointer.GetString(currentESS.SchemaName), // TODO can it be NULL?
			Queryid:                *currentESS.Digest,
			Fingerprint:            *currentESS.DigestText,
			NumQueries:             count,
			NumQueriesWithErrors:   float32(currentESS.SumErrors - prevESS.SumErrors),
			NumQueriesWithWarnings: float32(currentESS.SumWarnings - prevESS.SumWarnings),
			MetricsSource:          qanpb.MetricsSource_MYSQL_PERFSCHEMA,
		}

		for _, p := range []struct {
			value float32  // result value: currentESS.SumXXX-prevESS.SumXXX
			sum   *float32 // MetricsBucket.XXXSum field to write value
			cnt   *float32 // MetricsBucket.XXXCnt field to write count
		}{
			// in order of events_statements_summary_by_digest columns

			// convert picoseconds to seconds
			{float32(currentESS.SumTimerWait-prevESS.SumTimerWait) / 1e12, &mb.MQueryTimeSum, &mb.MQueryTimeCnt},
			{float32(currentESS.SumLockTime-prevESS.SumLockTime) / 1e12, &mb.MLockTimeSum, &mb.MLockTimeCnt},

			{float32(currentESS.SumRowsAffected - prevESS.SumRowsAffected), &mb.MRowsAffectedSum, &mb.MRowsAffectedCnt},
			{float32(currentESS.SumRowsSent - prevESS.SumRowsSent), &mb.MRowsSentSum, &mb.MRowsSentCnt},
			{float32(currentESS.SumRowsExamined - prevESS.SumRowsExamined), &mb.MRowsExaminedSum, &mb.MRowsExaminedCnt},

			{float32(currentESS.SumCreatedTmpDiskTables - prevESS.SumCreatedTmpDiskTables), &mb.MTmpDiskTablesSum, &mb.MTmpDiskTablesCnt},
			{float32(currentESS.SumCreatedTmpTables - prevESS.SumCreatedTmpTables), &mb.MTmpTablesSum, &mb.MTmpTablesCnt},
			{float32(currentESS.SumSelectFullJoin - prevESS.SumSelectFullJoin), &mb.MFullJoinSum, &mb.MFullJoinCnt},
			{float32(currentESS.SumSelectFullRangeJoin - prevESS.SumSelectFullRangeJoin), &mb.MSelectFullRangeJoinSum, &mb.MSelectFullRangeJoinCnt},
			{float32(currentESS.SumSelectRange - prevESS.SumSelectRange), &mb.MSelectRangeSum, &mb.MSelectRangeCnt},
			{float32(currentESS.SumSelectRangeCheck - prevESS.SumSelectRangeCheck), &mb.MSelectRangeCheckSum, &mb.MSelectRangeCheckCnt},
			{float32(currentESS.SumSelectScan - prevESS.SumSelectScan), &mb.MFullScanSum, &mb.MFullScanCnt},

			{float32(currentESS.SumSortMergePasses - prevESS.SumSortMergePasses), &mb.MMergePassesSum, &mb.MMergePassesCnt},
			{float32(currentESS.SumSortRange - prevESS.SumSortRange), &mb.MSortRangeSum, &mb.MSortRangeCnt},
			{float32(currentESS.SumSortRows - prevESS.SumSortRows), &mb.MSortRowsSum, &mb.MSortRowsCnt},
			{float32(currentESS.SumSortScan - prevESS.SumSortScan), &mb.MSortScanSum, &mb.MSortScanCnt},

			{float32(currentESS.SumNoIndexUsed - prevESS.SumNoIndexUsed), &mb.MNoIndexUsedSum, &mb.MNoIndexUsedCnt},
			{float32(currentESS.SumNoGoodIndexUsed - prevESS.SumNoGoodIndexUsed), &mb.MNoGoodIndexUsedSum, &mb.MNoGoodIndexUsedCnt},
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
func (m *MySQL) Changes() <-chan Change {
	return m.changes
}
