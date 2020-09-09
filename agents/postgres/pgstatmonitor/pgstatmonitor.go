// pmm-agent
// Copyright 2019 Percona LLC
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

// Package pgstatmonitor runs built-in QAN Agent for PostgreSQL pg stat monitor.
package pgstatmonitor

import (
	"context"
	"database/sql"
	"io"
	"math"
	"time"

	_ "github.com/lib/pq" // register SQL driver
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-agent/agents"
)

const (
	retainStatMonitor = 25 * time.Hour // make it work for daily queries
	queryStatMonitor  = time.Minute
)

// PGStatMonitorQAN QAN services connects to PostgreSQL and extracts stats.
type PGStatMonitorQAN struct {
	disableQueryExamples bool
	// By default, query shows the actual parameter instead of the placeholder.
	// It is quite useful when users want to use that query and try to run that
	// query to check the abnormalities. But in most cases users like the queries
	// with a placeholder. This parameter is used to toggle between the two said
	// options.
	pgsmNormalizedQuery bool
	q                   *reform.Querier
	dbCloser            io.Closer
	agentID             string
	l                   *logrus.Entry
	changes             chan agents.Change
	monitorCache        *statMonitorCache
}

// Params represent Agent parameters.
type Params struct {
	DSN                  string
	DisableQueryExamples bool
	AgentID              string
}

const queryTag = "pmm-agent:pgstatmonitor"

// New creates new PGStatMonitorQAN QAN service.
func New(params *Params, l *logrus.Entry) (*PGStatMonitorQAN, error) {
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

	return newPgStatMonitorQAN(q, sqlDB, params.AgentID, params.DisableQueryExamples, l)
}

func newPgStatMonitorQAN(q *reform.Querier, dbCloser io.Closer, agentID string, disableQueryExamples bool, l *logrus.Entry) (*PGStatMonitorQAN, error) {
	value, err := q.FindOneFrom(pgStatMonitorSettingsView, "name", "pg_stat_monitor.pgsm_normalized_query")
	if err != nil {
		return nil, err
	}

	return &PGStatMonitorQAN{
		q:                    q,
		pgsmNormalizedQuery:  value.(*pgStatMonitorSettings).Value == 1,
		dbCloser:             dbCloser,
		agentID:              agentID,
		l:                    l,
		changes:              make(chan agents.Change, 10),
		monitorCache:         newStatMonitorCache(retainStatMonitor, l),
		disableQueryExamples: disableQueryExamples,
	}, nil
}

// Run extracts stats data and sends it to the channel until ctx is canceled.
func (m *PGStatMonitorQAN) Run(ctx context.Context) {
	defer func() {
		m.dbCloser.Close() //nolint:errcheck
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	// add current stat monitor to cache so they are not send as new on first iteration with incorrect timestamps
	var running bool
	m.changes <- agents.Change{Status: inventorypb.AgentStatus_STARTING}
	if current, _, err := m.monitorCache.getStatMonitorExtended(ctx, m.q, m.pgsmNormalizedQuery, m.disableQueryExamples); err == nil {
		m.monitorCache.refresh(current)
		m.l.Debugf("Got %d initial stat monitor.", len(current))
		running = true
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_RUNNING}
	} else {
		m.l.Error(err)
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_WAITING}
	}

	// query pg_stat_monitor every minute at 00 seconds
	start := time.Now()
	wait := start.Truncate(queryStatMonitor).Add(queryStatMonitor).Sub(start)
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
			buckets, err := m.getNewBuckets(ctx, start, lengthS)

			start = time.Now()
			wait = start.Truncate(queryStatMonitor).Add(queryStatMonitor).Sub(start)
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

func (m *PGStatMonitorQAN) getNewBuckets(ctx context.Context, periodStart time.Time, periodLengthSecs uint32) ([]*agentpb.MetricsBucket, error) {
	current, prev, err := m.monitorCache.getStatMonitorExtended(ctx, m.q, m.pgsmNormalizedQuery, m.disableQueryExamples)
	if err != nil {
		return nil, err
	}

	buckets := m.makeBuckets(current, prev)
	startS := uint32(periodStart.Unix())
	m.l.Debugf("Made %d buckets out of %d stat monitor in %s+%d interval.",
		len(buckets), len(current), periodStart.Format("15:04:05"), periodLengthSecs)

	// merge prev and current in cache
	m.monitorCache.refresh(current)
	m.l.Debugf("statMonitorCache: %s", m.monitorCache.stats())

	// add agent_id and timestamps
	for i, b := range buckets {
		b.Common.AgentId = m.agentID
		b.Common.PeriodStartUnixSecs = startS
		b.Common.PeriodLengthSecs = periodLengthSecs

		buckets[i] = b
	}

	return buckets, nil
}

// makeBuckets uses current state of pg_stat_monitor table and accumulated previous state
// to make metrics buckets.
func (m *PGStatMonitorQAN) makeBuckets(current, prev map[string]*pgStatMonitorExtended) []*agentpb.MetricsBucket {
	res := make([]*agentpb.MetricsBucket, 0, len(current))
	now := time.Now()

	for queryID, currentPSS := range current {
		prevPSS := prev[queryID]
		if prevPSS == nil {
			prevPSS = new(pgStatMonitorExtended)
		}
		count := float32(currentPSS.Calls - prevPSS.Calls)
		switch {
		case count == 0:
			// Another way how this is possible is if pg_stat_monitor was truncated,
			// and then the same number of queries were made.
			// Currently, we can't differentiate between those situations.
			m.l.Tracef("Skipped due to the same number of queries: %s.", currentPSS)
			continue
		case count < 0 || (now.Sub(currentPSS.BucketStartTime) > queryStatMonitor):
			m.l.Debugf("Truncate detected. Treating as a new query: %s.", currentPSS)
			prevPSS = new(pgStatMonitorExtended)
			count = float32(currentPSS.Calls)
		case prevPSS.Calls == 0:
			m.l.Debugf("New query: %s.", currentPSS)
		default:
			m.l.Debugf("Normal query: %s.", currentPSS)
		}

		mb := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				IsTruncated: currentPSS.IsQueryTruncated,
				Fingerprint: currentPSS.Fingerprint,
				Database:    currentPSS.Database,
				Tables:      currentPSS.TablesNames,
				Username:    currentPSS.Username,
				Queryid:     currentPSS.QueryID,
				NumQueries:  count,
				AgentType:   inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
			},
			Postgresql: new(agentpb.MetricsBucket_PostgreSQL),
		}

		if !m.disableQueryExamples && currentPSS.Example != "" {
			mb.Common.Example = currentPSS.Example
			mb.Common.ExampleFormat = agentpb.ExampleFormat_EXAMPLE
			mb.Common.ExampleType = agentpb.ExampleType_RANDOM
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

			// convert milliseconds to seconds
			{float32(currentPSS.BlkReadTime-prevPSS.BlkReadTime) / 1000, &mb.Postgresql.MBlkReadTimeSum, &mb.Postgresql.MBlkReadTimeCnt},
			{float32(currentPSS.BlkWriteTime-prevPSS.BlkWriteTime) / 1000, &mb.Postgresql.MBlkWriteTimeSum, &mb.Postgresql.MBlkWriteTimeCnt},

			{float32(currentPSS.CPUSysTime-prevPSS.CPUSysTime) / 1000, &mb.Postgresql.MCpuSysTimeSum, &mb.Postgresql.MCpuSysTimeCnt},
			{float32(currentPSS.CPUUserTime-prevPSS.CPUUserTime) / 1000, &mb.Postgresql.MCpuUserTimeSum, &mb.Postgresql.MCpuUserTimeCnt},
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
func (m *PGStatMonitorQAN) Changes() <-chan agents.Change {
	return m.changes
}
