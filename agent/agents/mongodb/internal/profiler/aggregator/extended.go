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

package aggregator

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"sync"

	mathStats "github.com/montanaflynn/stats"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/percona/percona-toolkit/src/go/mongolib/stats"
	"github.com/percona/pmm/agent/agents/mongodb/internal/profiler/collector"
)

// Code below contains necessary code to extend stats from Percona Toolkit.
const (
	planSummaryCollScan = "COLLSCAN"
	planSummaryIXScan   = "IXSCAN"
)

type extendedStats struct {
	// dependencies
	fingerprinter stats.Fingerprinter

	// internal
	queryInfoAndCounters map[stats.GroupKey]*extendedQueryInfoAndCounters
	sync.RWMutex
}

type extendedQueryInfoAndCounters struct {
	stats.QueryInfoAndCounters
	PlanSummary   string
	CollScanCount int
	CollScanSum   int64
}

type extendedQueryStats struct {
	stats.QueryStats
	PlanSummary   string
	CollScanCount int
	CollScanSum   int64
}

type extendedQueries []extendedQueryInfoAndCounters

type totalCounters struct {
	Count     int
	Scanned   float64
	Returned  float64
	QueryTime float64
	Bytes     float64
}

func NewExtendedStats(fingerprinter stats.Fingerprinter) *extendedStats {
	s := &extendedStats{
		fingerprinter: fingerprinter,
	}

	s.Reset()
	return s
}

// Reset clears the collection of statistics
func (s *extendedStats) Reset() {
	s.Lock()
	defer s.Unlock()

	s.queryInfoAndCounters = make(map[stats.GroupKey]*extendedQueryInfoAndCounters)
}

// Queries returns all collected statistics
func (s *extendedStats) Queries() extendedQueries {
	s.Lock()
	defer s.Unlock()

	keys := stats.GroupKeys{}
	for key := range s.queryInfoAndCounters {
		keys = append(keys, key)
	}
	sort.Sort(keys)

	queries := []extendedQueryInfoAndCounters{}
	for _, key := range keys {
		queries = append(queries, *s.queryInfoAndCounters[key])
	}
	return queries
}

// Add adds collector.ExtendedSystemProfile to the collection of statistics
func (s *extendedStats) Add(doc collector.ExtendedSystemProfile) error {
	fp, err := s.fingerprinter.Fingerprint(doc.SystemProfile)
	if err != nil {
		return err // TODO &stats.StatsFingerprintError{err}
	}
	var qiac *extendedQueryInfoAndCounters
	var ok bool

	key := stats.GroupKey{
		Operation:   fp.Operation,
		Fingerprint: fp.Fingerprint,
		Namespace:   fp.Namespace,
	}
	if qiac, ok = s.getQueryInfoAndCounters(key); !ok {
		query := proto.NewExampleQuery(doc.SystemProfile)
		queryBson, err := bson.MarshalExtJSON(query, true, true)
		if err != nil {
			return err
		}
		qiac = &extendedQueryInfoAndCounters{
			QueryInfoAndCounters: stats.QueryInfoAndCounters{
				ID:          fmt.Sprintf("%x", md5.Sum([]byte(key.String()))),
				Operation:   fp.Operation,
				Fingerprint: fp.Fingerprint,
				Namespace:   fp.Namespace,
				TableScan:   false,
				Query:       string(queryBson),
			},
			PlanSummary: doc.PlanSummary,
		}
		s.setQueryInfoAndCounters(key, qiac)
	}
	qiac.Count++
	qiac.PlanSummary = doc.PlanSummary
	if qiac.PlanSummary == planSummaryCollScan {
		qiac.CollScanCount++
		qiac.CollScanSum += int64(doc.Millis)
	}
	if strings.HasPrefix(qiac.PlanSummary, planSummaryIXScan) {
		qiac.PlanSummary = planSummaryIXScan
	}
	// docsExamined is renamed from nscannedObjects in 3.2.0.
	// https://docs.mongodb.com/manual/reference/database-profiler/#system.profile.docsExamined
	s.Lock()
	if doc.NscannedObjects > 0 {
		qiac.NScanned = append(qiac.NScanned, float64(doc.NscannedObjects))
	} else {
		qiac.NScanned = append(qiac.NScanned, float64(doc.DocsExamined))
	}
	qiac.NReturned = append(qiac.NReturned, float64(doc.Nreturned))
	qiac.QueryTime = append(qiac.QueryTime, float64(doc.Millis))
	qiac.ResponseLength = append(qiac.ResponseLength, float64(doc.ResponseLength))
	if qiac.FirstSeen.IsZero() || qiac.FirstSeen.After(doc.Ts) {
		qiac.FirstSeen = doc.Ts
	}
	if qiac.LastSeen.IsZero() || qiac.LastSeen.Before(doc.Ts) {
		qiac.LastSeen = doc.Ts
	}
	s.Unlock()

	return nil
}

func (s *extendedStats) getQueryInfoAndCounters(key stats.GroupKey) (*extendedQueryInfoAndCounters, bool) {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.queryInfoAndCounters[key]
	return v, ok
}

func (s *extendedStats) setQueryInfoAndCounters(key stats.GroupKey, value *extendedQueryInfoAndCounters) {
	s.Lock()
	defer s.Unlock()

	s.queryInfoAndCounters[key] = value
}

// CalcQueriesStats calculates QueryStats for given uptime
func (q extendedQueries) CalcQueriesStats(uptime int64) []extendedQueryStats {
	qs := []extendedQueryStats{}
	tc := calcTotalCounters(q)

	for _, query := range q {
		queryStats := countersToStats(query, uptime, tc)
		qs = append(qs, queryStats)
	}

	return qs
}

func countersToStats(query extendedQueryInfoAndCounters, uptime int64, tc totalCounters) extendedQueryStats {
	queryStats := extendedQueryStats{
		QueryStats: stats.QueryStats{
			Count:          query.Count,
			ID:             query.ID,
			Operation:      query.Operation,
			Query:          query.Query,
			Fingerprint:    query.Fingerprint,
			Scanned:        calcStats(query.NScanned),
			Returned:       calcStats(query.NReturned),
			QueryTime:      calcStats(query.QueryTime),
			ResponseLength: calcStats(query.ResponseLength),
			FirstSeen:      query.FirstSeen,
			LastSeen:       query.LastSeen,
			Namespace:      query.Namespace,
			QPS:            float64(query.Count) / float64(uptime),
		},
		PlanSummary:   query.PlanSummary,
		CollScanCount: query.CollScanCount,
		CollScanSum:   query.CollScanSum,
	}
	if tc.Scanned > 0 {
		queryStats.Scanned.Pct = queryStats.Scanned.Total * 100 / tc.Scanned
	}
	if tc.Returned > 0 {
		queryStats.Returned.Pct = queryStats.Returned.Total * 100 / tc.Returned
	}
	if tc.QueryTime > 0 {
		queryStats.QueryTime.Pct = queryStats.QueryTime.Total * 100 / tc.QueryTime
	}
	if tc.Bytes > 0 {
		queryStats.ResponseLength.Pct = queryStats.ResponseLength.Total * 100 / tc.Bytes
	}
	if queryStats.Returned.Total > 0 {
		queryStats.Ratio = queryStats.Scanned.Total / queryStats.Returned.Total
	}

	return queryStats
}

func calcTotalCounters(queries []extendedQueryInfoAndCounters) totalCounters {
	tc := totalCounters{}

	for _, query := range queries {
		tc.Count += query.Count

		scanned, _ := mathStats.Sum(query.NScanned)
		tc.Scanned += scanned

		returned, _ := mathStats.Sum(query.NReturned)
		tc.Returned += returned

		queryTime, _ := mathStats.Sum(query.QueryTime)
		tc.QueryTime += queryTime

		bytes, _ := mathStats.Sum(query.ResponseLength)
		tc.Bytes += bytes
	}
	return tc
}

func calcStats(samples []float64) stats.Statistics {
	var s stats.Statistics
	s.Total, _ = mathStats.Sum(samples)
	s.Min, _ = mathStats.Min(samples)
	s.Max, _ = mathStats.Max(samples)
	s.Avg, _ = mathStats.Mean(samples)
	s.Pct95, _ = mathStats.PercentileNearestRank(samples, 95)
	s.Pct99, _ = mathStats.PercentileNearestRank(samples, 99)
	s.StdDev, _ = mathStats.StandardDeviation(samples)
	s.Median, _ = mathStats.Median(samples)
	return s
}
