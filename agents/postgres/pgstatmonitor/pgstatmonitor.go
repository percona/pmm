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
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // register SQL driver.
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-agent/agents"
)

const defaultWaitTime = 60 * time.Second

// PGStatMonitorQAN QAN services connects to PostgreSQL and extracts stats.
type PGStatMonitorQAN struct {
	q            *reform.Querier
	dbCloser     io.Closer
	agentID      string
	l            *logrus.Entry
	changes      chan agents.Change
	monitorCache *statMonitorCache

	// By default, query shows the actual parameter instead of the placeholder.
	// It is quite useful when users want to use that query and try to run that
	// query to check the abnormalities. But in most cases users like the queries
	// with a placeholder. This parameter is used to toggle between the two said
	// options.
	pgsmNormalizedQuery  bool
	waitTime             time.Duration
	disableQueryExamples bool
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
	settings, err := q.SelectAllFrom(pgStatMonitorSettingsView, "")
	if err != nil {
		return nil, err
	}

	var normalizedQuery bool
	waitTime := defaultWaitTime
	for _, row := range settings {
		setting := row.(*pgStatMonitorSettings)
		switch setting.Name {
		case "pg_stat_monitor.pgsm_normalized_query":
			normalizedQuery = setting.Value == 1
		case "pg_stat_monitor.pgsm_bucket_time":
			if setting.Value < int64(defaultWaitTime.Seconds()) {
				waitTime = time.Duration(setting.Value) * time.Second
			}
		}
	}

	return &PGStatMonitorQAN{
		q:                    q,
		dbCloser:             dbCloser,
		agentID:              agentID,
		l:                    l,
		changes:              make(chan agents.Change, 10),
		monitorCache:         newStatMonitorCache(l),
		pgsmNormalizedQuery:  normalizedQuery,
		waitTime:             waitTime,
		disableQueryExamples: disableQueryExamples,
	}, nil
}

func getPGMonitorVersion(q *reform.Querier) (pgMonitorVersion float64, err error) {
	var v string
	err = q.QueryRow(fmt.Sprintf("SELECT /* %s */ pg_stat_monitor_version()", queryTag)).Scan(&v)
	if err != nil {
		return
	}
	split := strings.Split(v, ".")
	return strconv.ParseFloat(fmt.Sprintf("%s.%s%s", split[0], split[1], split[2]), 64)
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
	if current, _, err := m.monitorCache.getStatMonitorExtended(ctx, m.q, m.pgsmNormalizedQuery); err == nil {
		m.monitorCache.refresh(current)
		m.l.Debugf("Got %d initial stat monitor.", len(current))
		running = true
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_RUNNING}
	} else {
		m.l.Error(err)
		m.changes <- agents.Change{Status: inventorypb.AgentStatus_WAITING}
	}

	// query pg_stat_monitor every waitTime seconds
	start := time.Now()
	m.l.Debugf("Scheduling next collection in %s at %s.", m.waitTime, start.Add(m.waitTime).Format("15:04:05"))
	t := time.NewTimer(m.waitTime)
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

			lengthS := uint32(m.waitTime.Seconds())
			buckets, err := m.getNewBuckets(ctx, lengthS)

			start = time.Now()
			m.l.Debugf("Scheduling next collection in %s at %s.", m.waitTime, start.Add(m.waitTime).Format("15:04:05"))
			t.Reset(m.waitTime)

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

func (m *PGStatMonitorQAN) getNewBuckets(ctx context.Context, periodLengthSecs uint32) ([]*agentpb.MetricsBucket, error) {
	current, prev, err := m.monitorCache.getStatMonitorExtended(ctx, m.q, m.pgsmNormalizedQuery)
	if err != nil {
		return nil, err
	}

	buckets := m.makeBuckets(current, prev)
	m.l.Debugf("Made %d buckets out of %d stat monitor in %d interval.",
		len(buckets), len(current), periodLengthSecs)

	// merge prev and current in cache
	m.monitorCache.refresh(current)
	m.l.Debugf("statMonitorCache: %s", m.monitorCache.stats())

	// add agent_id and timestamps
	for i, b := range buckets {
		b.Common.AgentId = m.agentID
		b.Common.PeriodLengthSecs = periodLengthSecs

		buckets[i] = b
	}

	return buckets, nil
}

// makeBuckets uses current state of pg_stat_monitor table and accumulated previous state
// to make metrics buckets.
func (m *PGStatMonitorQAN) makeBuckets(current, cache map[time.Time]map[string]*pgStatMonitorExtended) []*agentpb.MetricsBucket {
	res := make([]*agentpb.MetricsBucket, 0, len(current))

	for bucketStartTime, bucket := range current {
		prev := cache[bucketStartTime]
		for queryID, currentPSM := range bucket {
			var prevPSM *pgStatMonitorExtended
			if prev != nil {
				prevPSM = prev[queryID]
			}
			if prevPSM == nil {
				prevPSM = new(pgStatMonitorExtended)
			}
			count := float32(currentPSM.Calls - prevPSM.Calls)
			switch {
			case count == 0:
				// Another way how this is possible is if pg_stat_monitor was truncated,
				// and then the same number of queries were made.
				// Currently, we can't differentiate between those situations.
				m.l.Debugf("Skipped due to the same number of queries: %s.", currentPSM)
				continue
			case count < 0:
				m.l.Debugf("Truncate detected (negative count). Treating as a new query: %s.", currentPSM)
				prevPSM = new(pgStatMonitorExtended)
				count = float32(currentPSM.Calls)
			case prevPSM.Calls == 0:
				m.l.Debugf("New query: %s.", currentPSM)
			default:
				m.l.Debugf("Normal query: %s.", currentPSM)
			}

			mb := &agentpb.MetricsBucket{
				Common: &agentpb.MetricsBucket_Common{
					IsTruncated:         currentPSM.IsQueryTruncated,
					Fingerprint:         currentPSM.Fingerprint,
					Database:            currentPSM.Database,
					Tables:              currentPSM.Relations,
					Username:            currentPSM.Username,
					Queryid:             currentPSM.QueryID,
					NumQueries:          count,
					ClientHost:          currentPSM.ClientIP,
					AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
					PeriodStartUnixSecs: uint32(currentPSM.BucketStartTime.Unix()),
				},
				Postgresql: new(agentpb.MetricsBucket_PostgreSQL),
			}

			if !m.disableQueryExamples && currentPSM.Example != "" {
				mb.Common.Example = currentPSM.Example
				mb.Common.ExampleFormat = agentpb.ExampleFormat_EXAMPLE // nolint:staticcheck
				mb.Common.ExampleType = agentpb.ExampleType_RANDOM
			}

			for _, p := range []struct {
				value float32  // result value: currentPSM.SumXXX-prevPSM.SumXXX
				sum   *float32 // MetricsBucket.XXXSum field to write value
				cnt   *float32 // MetricsBucket.XXXCnt field to write count
			}{
				// convert milliseconds to seconds
				{float32(currentPSM.TotalTime-prevPSM.TotalTime) / 1000, &mb.Common.MQueryTimeSum, &mb.Common.MQueryTimeCnt},
				{float32(currentPSM.Rows - prevPSM.Rows), &mb.Postgresql.MRowsSum, &mb.Postgresql.MRowsCnt},

				{float32(currentPSM.SharedBlksHit - prevPSM.SharedBlksHit), &mb.Postgresql.MSharedBlksHitSum, &mb.Postgresql.MSharedBlksHitCnt},
				{float32(currentPSM.SharedBlksRead - prevPSM.SharedBlksRead), &mb.Postgresql.MSharedBlksReadSum, &mb.Postgresql.MSharedBlksReadCnt},
				{float32(currentPSM.SharedBlksDirtied - prevPSM.SharedBlksDirtied), &mb.Postgresql.MSharedBlksDirtiedSum, &mb.Postgresql.MSharedBlksDirtiedCnt},
				{float32(currentPSM.SharedBlksWritten - prevPSM.SharedBlksWritten), &mb.Postgresql.MSharedBlksWrittenSum, &mb.Postgresql.MSharedBlksWrittenCnt},

				{float32(currentPSM.LocalBlksHit - prevPSM.LocalBlksHit), &mb.Postgresql.MLocalBlksHitSum, &mb.Postgresql.MLocalBlksHitCnt},
				{float32(currentPSM.LocalBlksRead - prevPSM.LocalBlksRead), &mb.Postgresql.MLocalBlksReadSum, &mb.Postgresql.MLocalBlksReadCnt},
				{float32(currentPSM.LocalBlksDirtied - prevPSM.LocalBlksDirtied), &mb.Postgresql.MLocalBlksDirtiedSum, &mb.Postgresql.MLocalBlksDirtiedCnt},
				{float32(currentPSM.LocalBlksWritten - prevPSM.LocalBlksWritten), &mb.Postgresql.MLocalBlksWrittenSum, &mb.Postgresql.MLocalBlksWrittenCnt},

				{float32(currentPSM.TempBlksRead - prevPSM.TempBlksRead), &mb.Postgresql.MTempBlksReadSum, &mb.Postgresql.MTempBlksReadCnt},
				{float32(currentPSM.TempBlksWritten - prevPSM.TempBlksWritten), &mb.Postgresql.MTempBlksWrittenSum, &mb.Postgresql.MTempBlksWrittenCnt},

				// convert milliseconds to seconds
				{float32(currentPSM.BlkReadTime-prevPSM.BlkReadTime) / 1000, &mb.Postgresql.MBlkReadTimeSum, &mb.Postgresql.MBlkReadTimeCnt},
				{float32(currentPSM.BlkWriteTime-prevPSM.BlkWriteTime) / 1000, &mb.Postgresql.MBlkWriteTimeSum, &mb.Postgresql.MBlkWriteTimeCnt},

				// convert microseconds to seconds
				{float32(currentPSM.CPUSysTime-prevPSM.CPUSysTime) / 1000000, &mb.Postgresql.MCpuSysTimeSum, &mb.Postgresql.MCpuSysTimeCnt},
				{float32(currentPSM.CPUUserTime-prevPSM.CPUUserTime) / 1000000, &mb.Postgresql.MCpuUserTimeSum, &mb.Postgresql.MCpuUserTimeCnt},
			} {
				if p.value != 0 {
					*p.sum = p.value
					*p.cnt = count
				}
			}

			res = append(res, mb)
		}
	}

	return res
}

// Changes returns channel that should be read until it is closed.
func (m *PGStatMonitorQAN) Changes() <-chan agents.Change {
	return m.changes
}
