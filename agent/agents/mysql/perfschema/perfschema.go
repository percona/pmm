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

// Package perfschema runs built-in QAN Agent for MySQL performance schema.
package perfschema

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/AlekSi/pointer" // register SQL driver
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	mysqlDialects "gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/agents/cache"
	"github.com/percona/pmm/agent/queryparser"
	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/agent/utils/truncate"
	"github.com/percona/pmm/agent/utils/version"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/utils/sqlmetrics"
)

type (
	historyMap map[string]*eventsStatementsHistory
	summaryMap map[string]*eventsStatementsSummaryByDigest
)

const (
	retainHistory    = 5 * time.Minute
	refreshHistory   = 5 * time.Second
	historyCacheSize = 10000 // history cache size rows limit

	retainSummaries    = 25 * time.Hour // make it work for daily queries
	querySummaries     = time.Minute
	summariesCacheSize = 10000 // summary cache size rows limit
)

// PerfSchema QAN services connects to MySQL and extracts performance data.
type PerfSchema struct {
	q                      *reform.Querier
	dbCloser               io.Closer
	agentID                string
	disableCommentsParsing bool
	maxQueryLength         int32
	disableQueryExamples   bool
	l                      *logrus.Entry
	changes                chan agents.Change
	historyCache           *historyCache
	summaryCache           *summaryCache
	useLong                *bool
}

// Params represent Agent parameters.
type Params struct {
	DSN                    string
	AgentID                string
	DisableCommentsParsing bool
	MaxQueryLength         int32
	DisableQueryExamples   bool
	TextFiles              *agentv1.TextFiles
	TLSSkipVerify          bool
}

// newPerfSchemaParams holds all required parameters to instantiate a new PerfSchema.
type newPerfSchemaParams struct {
	Querier                *reform.Querier
	DBCloser               io.Closer
	AgentID                string
	DisableCommentsParsing bool
	MaxQueryLength         int32
	DisableQueryExamples   bool
	LogEntry               *logrus.Entry
}

const queryTag = "agent='perfschema'"

// getPerfschemaSummarySize returns size of rows for perfschema summary cache.
func getPerfschemaSummarySize(q reform.Querier, l *logrus.Entry) uint {
	var name string
	var size uint

	query := fmt.Sprintf("SHOW VARIABLES /* %s */ LIKE 'performance_schema_digests_size'", queryTag)
	err := q.QueryRow(query).Scan(&name, &size)
	if err != nil {
		l.Debug(err)
		size = summariesCacheSize
	}

	l.Infof("performance_schema_digests_size=%d", size)

	return size
}

// getPerfschemaHistorySize returns size of rows for perfschema history cache.
func getPerfschemaHistorySize(q reform.Querier, l *logrus.Entry) uint {
	var name string
	var size uint
	query := fmt.Sprintf("SHOW VARIABLES /* %s */ LIKE 'performance_schema_events_statements_history_long_size'", queryTag)
	err := q.QueryRow(query).Scan(&name, &size)
	if err != nil {
		l.Debug(err)
		size = historyCacheSize
	}

	l.Infof("performance_schema_events_statements_history_long_size=%d", size)

	return size
}

// New creates new PerfSchema QAN service.
func New(params *Params, l *logrus.Entry) (*PerfSchema, error) {
	if params.TextFiles != nil {
		err := tlshelpers.RegisterMySQLCerts(params.TextFiles.Files, params.TLSSkipVerify)
		if err != nil {
			return nil, err
		}
	}

	sqlDB, err := sql.Open("mysql", params.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)
	reformL := sqlmetrics.NewReform("mysql", params.AgentID, l.Tracef)
	// TODO register reformL metrics https://jira.percona.com/browse/PMM-4087
	q := reform.NewDB(sqlDB, mysqlDialects.Dialect, reformL).WithTag(queryTag)

	newParams := &newPerfSchemaParams{
		Querier:                q,
		DBCloser:               sqlDB,
		AgentID:                params.AgentID,
		DisableCommentsParsing: params.DisableCommentsParsing,
		MaxQueryLength:         params.MaxQueryLength,
		DisableQueryExamples:   params.DisableQueryExamples,
		LogEntry:               l,
	}
	return newPerfSchema(newParams)
}

func newPerfSchema(params *newPerfSchemaParams) (*PerfSchema, error) {
	historyCache, err := newHistoryCache(historyMap{}, retainHistory, getPerfschemaHistorySize(*params.Querier, params.LogEntry), params.LogEntry)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create cache")
	}

	summaryCache, err := newSummaryCache(summaryMap{}, retainSummaries, getPerfschemaSummarySize(*params.Querier, params.LogEntry), params.LogEntry)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create cache")
	}

	return &PerfSchema{
		q:                      params.Querier,
		dbCloser:               params.DBCloser,
		agentID:                params.AgentID,
		disableCommentsParsing: params.DisableCommentsParsing,
		maxQueryLength:         params.MaxQueryLength,
		disableQueryExamples:   params.DisableQueryExamples,
		l:                      params.LogEntry,
		changes:                make(chan agents.Change, 10),
		historyCache:           historyCache,
		summaryCache:           summaryCache,
	}, nil
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *PerfSchema) Run(ctx context.Context) {
	defer func() {
		m.dbCloser.Close() //nolint:errcheck
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(m.changes)
	}()

	// add current summaries to cache so they are not send as new on first iteration with incorrect timestamps
	var running bool
	var err error
	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	if s, err := getSummaries(m.q); err == nil {
		if err = m.summaryCache.Set(s); err == nil {
			m.l.Debugf("Got %d initial summaries.", len(s))
			running = true
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
		}
	}

	if err != nil {
		m.l.Error(err)
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
	}

	go m.runHistoryCacheRefresher(ctx)

	// query events_statements_summary_by_digest every minute at 00 seconds
	start := time.Now()
	wait := start.Truncate(querySummaries).Add(querySummaries).Sub(start)
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
			buckets, err := m.getNewBuckets(start, lengthS)

			start = time.Now()
			wait = start.Truncate(querySummaries).Add(querySummaries).Sub(start)
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

func (m *PerfSchema) runHistoryCacheRefresher(ctx context.Context) {
	t := time.NewTicker(refreshHistory)
	defer t.Stop()

	for {
		if err := m.refreshHistoryCache(ctx); err != nil {
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

func (m *PerfSchema) refreshHistoryCache(ctx context.Context) error {
	if m.useLong == nil {
		sqlVersion, vendor, err := version.GetMySQLVersion(ctx, m.q)
		if err != nil {
			return errors.Wrap(err, "cannot get MySQL version")
		}
		m.useLong = pointer.ToBool(vendor == version.MariaDBVendor && sqlVersion.Float() >= 11)
	}
	current, err := getHistory(m.q, m.useLong)
	if err != nil {
		return err
	}

	if err = m.historyCache.Set(current); err != nil {
		return err
	}
	m.l.Debugf("historyCache: %s", m.historyCache.cache.Stats())
	return nil
}

func (m *PerfSchema) getNewBuckets(periodStart time.Time, periodLengthSecs uint32) ([]*agentv1.MetricsBucket, error) {
	current, err := getSummaries(m.q)
	if err != nil {
		return nil, err
	}
	prev := make(summaryMap)
	if err = m.summaryCache.Get(prev); err != nil {
		return nil, err
	}

	buckets := makeBuckets(current, prev, m.l, m.maxQueryLength)
	startS := uint32(periodStart.Unix())
	m.l.Debugf("Made %d buckets out of %d summaries in %s+%d interval.",
		len(buckets), len(current), periodStart.Format("15:04:05"), periodLengthSecs)

	// merge prev and current in cache
	if err = m.summaryCache.Set(current); err != nil {
		return nil, err
	}
	m.l.Debugf("summaryCache: %s", m.summaryCache.cache.Stats())

	// add agent_id, timestamps, and examples from history cache
	history := make(historyMap)
	if err = m.historyCache.Get(history); err != nil {
		return nil, err
	}
	for i, b := range buckets {
		b.Common.AgentId = m.agentID
		b.Common.PeriodStartUnixSecs = startS
		b.Common.PeriodLengthSecs = periodLengthSecs

		//nolint:nestif
		if esh := history[queryIDWithSchema(b.Common.Schema, b.Common.Queryid)]; esh != nil {
			// TODO test if we really need that
			// If we don't need it, we can avoid polling events_statements_history completely
			// if query examples are disabled.
			if b.Common.Schema == "" {
				b.Common.Schema = pointer.GetString(esh.CurrentSchema)
			}

			if esh.SQLText != nil && *esh.SQLText != "" {
				explainFingerprint, placeholdersCount := queryparser.GetMySQLFingerprintPlaceholders(*esh.SQLText, *esh.DigestText)
				explainFingerprint, truncated := truncate.Query(explainFingerprint, m.maxQueryLength, truncate.GetDefaultMaxQueryLength())
				if truncated {
					b.Common.IsTruncated = truncated
				}
				b.Common.ExplainFingerprint = explainFingerprint
				b.Common.PlaceholdersCount = placeholdersCount

				if !m.disableQueryExamples {
					example, truncated := truncate.Query(*esh.SQLText, m.maxQueryLength, truncate.GetDefaultMaxQueryLength())
					fmt.Println(example)
					if truncated {
						b.Common.IsTruncated = truncated
					}
					b.Common.Example = example
					b.Common.ExampleType = agentv1.ExampleType_EXAMPLE_TYPE_RANDOM
				}

				if !m.disableCommentsParsing {
					comments, err := queryparser.MySQLComments(*esh.SQLText)
					if err != nil {
						m.l.Infof("cannot parse comments from query: %s", *esh.SQLText)
					}
					b.Common.Comments = comments
				}
			}
		}
		buckets[i] = b
	}

	return buckets, nil
}

// inc returns increment from prev to current, or 0, if there was a wrap-around.
func inc(current, prev uint64) float32 {
	if current <= prev {
		return 0
	}
	return float32(current - prev)
}

// makeBuckets uses current state of events_statements_summary_by_digest table and accumulated previous state
// to make metrics buckets;
// makeBuckets is a pure function for easier testing.
func makeBuckets(current, prev summaryMap, l *logrus.Entry, maxQueryLength int32) []*agentv1.MetricsBucket {
	res := make([]*agentv1.MetricsBucket, 0, len(current))

	for digest, currentESS := range current {
		prevESS := prev[digest]
		if prevESS == nil {
			prevESS = &eventsStatementsSummaryByDigest{}
		}

		switch {
		case currentESS.CountStar == prevESS.CountStar:
			// Another way how this is possible is if events_statements_summary_by_digest was truncated,
			// and then the same number of queries were made.
			// Currently, we can't differentiate between those situations.
			// TODO We probably could by using first_seen/last_seen columns.
			l.Tracef("Skipped due to the same number of queries: %s.", currentESS)
			continue
		case currentESS.CountStar < prevESS.CountStar:
			l.Debugf("Truncate detected. Treating as a new query: %s.", currentESS)
			prevESS = &eventsStatementsSummaryByDigest{}
		case prevESS.CountStar == 0:
			l.Debugf("New query: %s.", currentESS)
		default:
			l.Debugf("Normal query: %s.", currentESS)
		}

		count := inc(currentESS.CountStar, prevESS.CountStar)
		fingerprint, isTruncated := truncate.Query(*currentESS.DigestText, maxQueryLength, truncate.GetDefaultMaxQueryLength())
		mb := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Schema:                 pointer.GetString(currentESS.SchemaName), // TODO can it be NULL?
				Queryid:                *currentESS.Digest,
				Fingerprint:            fingerprint,
				IsTruncated:            isTruncated,
				NumQueries:             count,
				NumQueriesWithErrors:   inc(currentESS.SumErrors, prevESS.SumErrors),
				NumQueriesWithWarnings: inc(currentESS.SumWarnings, prevESS.SumWarnings),
				AgentType:              inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{},
		}

		for _, p := range []struct {
			value float32  // result value: currentESS.SumXXX-prevESS.SumXXX
			sum   *float32 // MetricsBucket.XXXSum field to write value
			cnt   *float32 // MetricsBucket.XXXCnt field to write count
		}{
			// Ordered the same as events_statements_summary_by_digest columns

			// convert picoseconds to seconds
			{inc(currentESS.SumTimerWait, prevESS.SumTimerWait) / 1000000000000, &mb.Common.MQueryTimeSum, &mb.Common.MQueryTimeCnt},
			{inc(currentESS.SumLockTime, prevESS.SumLockTime) / 1000000000000, &mb.Mysql.MLockTimeSum, &mb.Mysql.MLockTimeCnt},

			{inc(currentESS.SumRowsAffected, prevESS.SumRowsAffected), &mb.Mysql.MRowsAffectedSum, &mb.Mysql.MRowsAffectedCnt},
			{inc(currentESS.SumRowsSent, prevESS.SumRowsSent), &mb.Mysql.MRowsSentSum, &mb.Mysql.MRowsSentCnt},
			{inc(currentESS.SumRowsExamined, prevESS.SumRowsExamined), &mb.Mysql.MRowsExaminedSum, &mb.Mysql.MRowsExaminedCnt},

			{inc(currentESS.SumCreatedTmpDiskTables, prevESS.SumCreatedTmpDiskTables), &mb.Mysql.MTmpDiskTablesSum, &mb.Mysql.MTmpDiskTablesCnt},
			{inc(currentESS.SumCreatedTmpTables, prevESS.SumCreatedTmpTables), &mb.Mysql.MTmpTablesSum, &mb.Mysql.MTmpTablesCnt},
			{inc(currentESS.SumSelectFullJoin, prevESS.SumSelectFullJoin), &mb.Mysql.MFullJoinSum, &mb.Mysql.MFullJoinCnt},
			{inc(currentESS.SumSelectFullRangeJoin, prevESS.SumSelectFullRangeJoin), &mb.Mysql.MSelectFullRangeJoinSum, &mb.Mysql.MSelectFullRangeJoinCnt},
			{inc(currentESS.SumSelectRange, prevESS.SumSelectRange), &mb.Mysql.MSelectRangeSum, &mb.Mysql.MSelectRangeCnt},
			{inc(currentESS.SumSelectRangeCheck, prevESS.SumSelectRangeCheck), &mb.Mysql.MSelectRangeCheckSum, &mb.Mysql.MSelectRangeCheckCnt},
			{inc(currentESS.SumSelectScan, prevESS.SumSelectScan), &mb.Mysql.MFullScanSum, &mb.Mysql.MFullScanCnt},

			{inc(currentESS.SumSortMergePasses, prevESS.SumSortMergePasses), &mb.Mysql.MMergePassesSum, &mb.Mysql.MMergePassesCnt},
			{inc(currentESS.SumSortRange, prevESS.SumSortRange), &mb.Mysql.MSortRangeSum, &mb.Mysql.MSortRangeCnt},
			{inc(currentESS.SumSortRows, prevESS.SumSortRows), &mb.Mysql.MSortRowsSum, &mb.Mysql.MSortRowsCnt},
			{inc(currentESS.SumSortScan, prevESS.SumSortScan), &mb.Mysql.MSortScanSum, &mb.Mysql.MSortScanCnt},

			{inc(currentESS.SumNoIndexUsed, prevESS.SumNoIndexUsed), &mb.Mysql.MNoIndexUsedSum, &mb.Mysql.MNoIndexUsedCnt},
			{inc(currentESS.SumNoGoodIndexUsed, prevESS.SumNoGoodIndexUsed), &mb.Mysql.MNoGoodIndexUsedSum, &mb.Mysql.MNoGoodIndexUsedCnt},
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
func (m *PerfSchema) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector.
func (m *PerfSchema) Describe(ch chan<- *prometheus.Desc) { //nolint:revive
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (m *PerfSchema) Collect(ch chan<- prometheus.Metric) {
	historyStats := m.historyCache.cache.Stats()
	summaryStats := m.summaryCache.cache.Stats()
	historyMetrics := cache.MetricsFromStats(historyStats, m.agentID, "history")
	summaryMetrics := cache.MetricsFromStats(summaryStats, m.agentID, "summary")

	for _, metric := range historyMetrics {
		ch <- metric
	}
	for _, metric := range summaryMetrics {
		ch <- metric
	}
}

// check interfaces.
var (
	_ prometheus.Collector = (*PerfSchema)(nil)
)
