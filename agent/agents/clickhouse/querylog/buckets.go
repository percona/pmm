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

package querylog

import (
	"sort"

	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

const (
	// MillisPerSecond converts query_duration_ms into the seconds QAN expects.
	millisPerSecond = 1000
	// P99Quantile is the quantile used for the *_p99 bucket metrics.
	p99Quantile = 0.99
)

// makeBuckets groups query-log rows by fingerprint hash and builds one
// MetricsBucket per query class. It is a pure function — no I/O, no agent
// state — so it is fully unit testable. AgentId and period fields are left
// zero for the caller (collect) to fill in.
func makeBuckets(rows []queryLogRow, maxQueryLength int32) []*agentv1.MetricsBucket {
	classes := aggregate(rows, maxQueryLength)

	res := make([]*agentv1.MetricsBucket, 0, len(classes))
	for _, c := range classes {
		res = append(res, c.toBucket())
	}
	return res
}

// aggregate folds rows into query classes keyed by fingerprint hash.
func aggregate(rows []queryLogRow, maxQueryLength int32) []*queryClass {
	byHash := make(map[string]*queryClass)
	order := make([]string, 0)

	for i := range rows {
		row := &rows[i]
		fp := fingerprint(row.Query)
		if fp == "" {
			// Fingerprint produced nothing usable — fall back to the raw,
			// truncated query so the class is still hashed deterministically.
			fp, _ = truncate.Query(row.Query, maxQueryLength, truncate.GetDefaultMaxQueryLength())
		}
		hash := hashFingerprint(row.NormalizedQueryHash, fp)

		c := byHash[hash]
		if c == nil {
			displayFP, truncated := truncate.Query(fp, maxQueryLength, truncate.GetDefaultMaxQueryLength())
			example, exampleTruncated := truncate.Query(row.Query, maxQueryLength, truncate.GetDefaultMaxQueryLength())
			c = &queryClass{
				queryID:     hash,
				fingerprint: displayFP,
				example:     example,
				isTruncated: truncated || exampleTruncated,
				queryKind:   row.QueryKind,
				database:    firstOf(row.Databases),
				username:    row.User,
				tables:      make(map[string]struct{}),
				errors:      make(map[uint64]uint64),
			}
			byHash[hash] = c
			order = append(order, hash)
		}

		c.add(row)
	}

	classes := make([]*queryClass, 0, len(order))
	for _, h := range order {
		classes = append(classes, byHash[h])
	}
	return classes
}

// add folds a single observed execution into the query class.
func (c *queryClass) add(row *queryLogRow) {
	c.numQueries++
	for _, t := range row.Tables {
		c.tables[t] = struct{}{}
	}
	if row.failed() {
		c.numQueriesWithErrors++
		// ClickHouse error codes are non-negative; ignore a malformed negative
		// value rather than wrapping it into a huge uint64 key.
		if row.ExceptionCode > 0 {
			c.errors[uint64(row.ExceptionCode)]++
		}
	}

	// query_duration_ms is milliseconds; QAN expects query time in seconds.
	c.queryTime = append(c.queryTime, float32(row.QueryDurationMs)/millisPerSecond)
	c.readRows = append(c.readRows, float32(row.ReadRows))
	c.readBytes = append(c.readBytes, float32(row.ReadBytes))
	c.resultRows = append(c.resultRows, float32(row.ResultRows))
	c.resultBytes = append(c.resultBytes, float32(row.ResultBytes))
	c.memoryUsage = append(c.memoryUsage, float32(row.MemoryUsage))
	c.writtenRows = append(c.writtenRows, float32(row.WrittenRows))
	c.writtenBytes = append(c.writtenBytes, float32(row.WrittenBytes))
}

// toBucket renders the accumulated class as a MetricsBucket.
func (c *queryClass) toBucket() *agentv1.MetricsBucket {
	common := &agentv1.MetricsBucket_Common{
		Queryid:              c.queryID,
		Fingerprint:          c.fingerprint,
		Database:             c.database,
		Tables:               sortedKeys(c.tables),
		Username:             c.username,
		AgentType:            inventoryv1.AgentType_AGENT_TYPE_QAN_CLICKHOUSE_QUERYLOG_AGENT,
		Example:              c.example,
		ExampleType:          agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
		IsTruncated:          c.isTruncated,
		NumQueries:           float32(c.numQueries),
		NumQueriesWithErrors: float32(c.numQueriesWithErrors),
	}
	if len(c.errors) > 0 {
		common.Errors = c.errors
	}
	fillStats(c.queryTime, &common.MQueryTimeCnt, &common.MQueryTimeSum, &common.MQueryTimeMin, &common.MQueryTimeMax, &common.MQueryTimeP99)

	ch := &agentv1.MetricsBucket_ClickHouse{QueryKind: c.queryKind}
	fillStats(c.readRows, &ch.MReadRowsCnt, &ch.MReadRowsSum, &ch.MReadRowsMin, &ch.MReadRowsMax, &ch.MReadRowsP99)
	fillStats(c.readBytes, &ch.MReadBytesCnt, &ch.MReadBytesSum, &ch.MReadBytesMin, &ch.MReadBytesMax, &ch.MReadBytesP99)
	fillStats(c.resultRows, &ch.MResultRowsCnt, &ch.MResultRowsSum, &ch.MResultRowsMin, &ch.MResultRowsMax, &ch.MResultRowsP99)
	fillStats(c.resultBytes, &ch.MResultBytesCnt, &ch.MResultBytesSum, &ch.MResultBytesMin, &ch.MResultBytesMax, &ch.MResultBytesP99)
	fillStats(c.memoryUsage, &ch.MMemoryUsageCnt, &ch.MMemoryUsageSum, &ch.MMemoryUsageMin, &ch.MMemoryUsageMax, &ch.MMemoryUsageP99)
	fillStats(c.writtenRows, &ch.MWrittenRowsCnt, &ch.MWrittenRowsSum, &ch.MWrittenRowsMin, &ch.MWrittenRowsMax, &ch.MWrittenRowsP99)
	fillStats(c.writtenBytes, &ch.MWrittenBytesCnt, &ch.MWrittenBytesSum, &ch.MWrittenBytesMin, &ch.MWrittenBytesMax, &ch.MWrittenBytesP99)

	return &agentv1.MetricsBucket{Common: common, Clickhouse: ch}
}

// fillStats writes cnt/sum/min/max/p99 of the samples into the target fields.
// Cnt is always the sample count; the remaining four are left zero for an
// empty sample set, matching the other QAN agents' "omit unset" convention.
func fillStats(samples []float32, cnt, sum, minOut, maxOut, p99 *float32) {
	if len(samples) == 0 {
		return
	}
	*cnt = float32(len(samples))

	var total, lo, hi float32
	lo = samples[0]
	hi = samples[0]
	for _, v := range samples {
		total += v
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	*sum = total
	*minOut = lo
	*maxOut = hi
	*p99 = percentile(samples, p99Quantile)
}

// firstOf returns the first element of s, or "" when s is empty.
func firstOf(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

// sortedKeys returns the set's keys in deterministic order.
func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
