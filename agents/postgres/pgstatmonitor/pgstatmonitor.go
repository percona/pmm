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
	"time"

	"github.com/AlekSi/pointer"
	ver "github.com/hashicorp/go-version"
	"github.com/lib/pq"
	_ "github.com/lib/pq" // register SQL driver.
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-agent/agents"
	"github.com/percona/pmm-agent/utils/version"
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
	TextFiles            *agentpb.TextFiles
	AgentID              string
}

type pgStatMonitorVersion int

const (
	pgStatMonitorVersion06 pgStatMonitorVersion = iota
	pgStatMonitorVersion08
	pgStatMonitorVersion09
	pgStatMonitorVersion10PG12
	pgStatMonitorVersion10PG13
	pgStatMonitorVersion10PG14
)

const (
	queryTag = "pmm-agent:pgstatmonitor"
	// There is a feature in the FE that shows "n/a" for empty responses for dimensions.
	commandTextNotAvailable = ""
	commandTypeSelect       = "SELECT"
	commandTypeUpdate       = "UPDATE"
	commandTypeInsert       = "INSERT"
	commandTypeDelete       = "DELETE"
	commandTypeUtiity       = "UTILITY"
)

var commandTypeToText = []string{
	commandTextNotAvailable,
	commandTypeSelect,
	commandTypeUpdate,
	commandTypeInsert,
	commandTypeDelete,
	commandTypeUtiity,
	commandTextNotAvailable,
}

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

func getPGVersion(q *reform.Querier) (pgVersion float64, err error) {
	var v string
	err = q.QueryRow(fmt.Sprintf("SELECT /* %s */ version()", queryTag)).Scan(&v)
	if err != nil {
		return
	}
	v = version.ParsePostgreSQLVersion(v)
	return strconv.ParseFloat(v, 64)
}

func getPGMonitorVersion(q *reform.Querier) (pgStatMonitorVersion, error) {
	var result string
	err := q.QueryRow(fmt.Sprintf("SELECT /* %s */ pg_stat_monitor_version()", queryTag)).Scan(&result)
	if err != nil {
		return pgStatMonitorVersion06, errors.Wrap(err, "failed to get pg_stat_monitor version from DB")
	}
	pgsmVersion, err := ver.NewVersion(result)
	if err != nil {
		return pgStatMonitorVersion06, errors.Wrap(err, "failed to parse pg_stat_monitor version")
	}

	pgVersion, err := getPGVersion(q)
	if err != nil {
		return pgStatMonitorVersion06, err
	}

	switch {
	case pgsmVersion.Core().GreaterThanOrEqual(v10):
		if pgVersion >= 14 {
			return pgStatMonitorVersion10PG14, nil
		}
		if pgVersion >= 13 {
			return pgStatMonitorVersion10PG13, nil
		}
		return pgStatMonitorVersion10PG12, nil
	case pgsmVersion.GreaterThanOrEqual(v09):
		return pgStatMonitorVersion09, nil
	case pgsmVersion.GreaterThanOrEqual(v08):
		return pgStatMonitorVersion08, nil
	default:
		return pgStatMonitorVersion06, nil
	}
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
		m.l.Error(errors.Wrap(err, "failed to get extended monitor status"))
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
				m.l.Error(errors.Wrap(err, "getNewBuckets failed"))
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
			if currentPSM.pgStatMonitor.CmdType >= 0 && currentPSM.pgStatMonitor.CmdType < int32(len(commandTypeToText)) {
				mb.Postgresql.CmdType = commandTypeToText[currentPSM.pgStatMonitor.CmdType]
			} else {
				mb.Postgresql.CmdType = commandTextNotAvailable
				m.l.Warnf("failed to translate command type '%d' into text", currentPSM.pgStatMonitor.CmdType)
			}

			mb.Postgresql.TopQueryid = pointer.GetString(currentPSM.TopQueryID)
			mb.Postgresql.TopQuery = pointer.GetString(currentPSM.TopQuery)
			mb.Postgresql.ApplicationName = pointer.GetString(currentPSM.ApplicationName)
			mb.Postgresql.Planid = pointer.GetString(currentPSM.PlanID)
			mb.Postgresql.QueryPlan = pointer.GetString(currentPSM.QueryPlan)

			histogram, err := parseHistogramFromRespCalls(currentPSM.RespCalls, prevPSM.RespCalls)
			if err != nil {
				m.l.Warnf(err.Error())
			} else {
				mb.Postgresql.HistogramItems = histogram
			}

			if (currentPSM.PlanTotalTime - prevPSM.PlanTotalTime) != 0 {
				mb.Postgresql.MPlanTimeSum = float32(currentPSM.PlanTotalTime-prevPSM.PlanTotalTime) / 1000
				mb.Postgresql.MPlanTimeMin = float32(currentPSM.PlanMinTime) / 1000
				mb.Postgresql.MPlanTimeMax = float32(currentPSM.PlanMaxTime) / 1000
				mb.Postgresql.MPlanTimeCnt = count
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

				{float32(currentPSM.PlansCalls - prevPSM.PlansCalls), &mb.Postgresql.MPlansCallsSum, &mb.Postgresql.MPlansCallsCnt},
				{float32(currentPSM.WalFpi - prevPSM.WalFpi), &mb.Postgresql.MWalFpiSum, &mb.Postgresql.MWalFpiCnt},
				{float32(currentPSM.WalRecords - prevPSM.WalRecords), &mb.Postgresql.MWalRecordsSum, &mb.Postgresql.MWalRecordsCnt},
				{float32(currentPSM.WalBytes - prevPSM.WalBytes), &mb.Postgresql.MWalBytesSum, &mb.Postgresql.MWalBytesCnt},

				// convert milliseconds to seconds
				{float32(currentPSM.TotalTime-prevPSM.TotalTime) / 1000, &mb.Common.MQueryTimeSum, &mb.Common.MQueryTimeCnt},
				{float32(currentPSM.BlkReadTime-prevPSM.BlkReadTime) / 1000, &mb.Postgresql.MBlkReadTimeSum, &mb.Postgresql.MBlkReadTimeCnt},
				{float32(currentPSM.BlkWriteTime-prevPSM.BlkWriteTime) / 1000, &mb.Postgresql.MBlkWriteTimeSum, &mb.Postgresql.MBlkWriteTimeCnt},

				// convert microseconds to seconds
				{float32(currentPSM.CPUSysTime-prevPSM.CPUSysTime) / 1000000, &mb.Postgresql.MCpuSysTimeSum, &mb.Postgresql.MCpuSysTimeCnt},
				{float32(currentPSM.CPUUserTime-prevPSM.CPUUserTime) / 1000000, &mb.Postgresql.MCpuUserTimeSum, &mb.Postgresql.MCpuUserTimeCnt},

				{float32(currentPSM.WalBytes - prevPSM.WalBytes), &mb.Postgresql.MWalBytesSum, &mb.Postgresql.MWalBytesCnt},
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

func parseHistogramFromRespCalls(respCalls pq.StringArray, prevRespCalls pq.StringArray) ([]*agentpb.HistogramItem, error) {
	histogram := getHistogramRangesArray()
	for k, v := range respCalls {
		val, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse histogram")
		}

		histogram[k].Frequency = uint32(val)
	}

	for k, v := range prevRespCalls {
		val, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse histogram")
		}

		histogram[k].Frequency -= uint32(val)
	}

	return histogram, nil
}

func getHistogramRangesArray() []*agentpb.HistogramItem {
	// For now we using static ranges.
	// In future we will compute range values from pg_stat_monitor_settings.
	// pgsm_histogram_min, pgsm_histogram_max, pgsm_histogram_buckets.
	return []*agentpb.HistogramItem{
		{Range: "(0 - 3)"},
		{Range: "(3 - 10)"},
		{Range: "(10 - 31)"},
		{Range: "(31 - 100)"},
		{Range: "(100 - 316)"},
		{Range: "(316 - 1000)"},
		{Range: "(1000 - 3162)"},
		{Range: "(3162 - 10000)"},
		{Range: "(10000 - 31622)"},
		{Range: "(31622 - 100000)"},
	}
}

// Changes returns channel that should be read until it is closed.
func (m *PGStatMonitorQAN) Changes() <-chan agents.Change {
	return m.changes
}
