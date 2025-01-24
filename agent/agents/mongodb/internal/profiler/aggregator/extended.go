package aggregator

import (
	"crypto/md5"
	"fmt"
	"sort"
	"sync"

	mathStats "github.com/montanaflynn/stats"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/percona/percona-toolkit/src/go/mongolib/stats"
	"github.com/percona/pmm/agent/agents/mongodb/internal/profiler/collector"
)

const planSummaryCollScan = "COLLSCAN"

type ExtendedStats struct {
	// dependencies
	fingerprinter stats.Fingerprinter

	// internal
	queryInfoAndCounters map[stats.GroupKey]*ExtendedQueryInfoAndCounters
	sync.RWMutex
}

type ExtendedQueryInfoAndCounters struct {
	stats.QueryInfoAndCounters
	PlanSummary   string
	CollScanCount int
	CollScanSum   int64
}

type ExtendedQueryStats struct {
	stats.QueryStats
	PlanSummary   string
	CollScanCount int
	CollScanSum   int64
}

type totalCounters struct {
	Count     int
	Scanned   float64
	Returned  float64
	QueryTime float64
	Bytes     float64
}

type ExtendedQueries []ExtendedQueryInfoAndCounters

func NewExtendedStats(fingerprinter stats.Fingerprinter) *ExtendedStats {
	s := &ExtendedStats{
		fingerprinter: fingerprinter,
	}

	s.Reset()
	return s
}

// Reset clears the collection of statistics
func (s *ExtendedStats) Reset() {
	s.Lock()
	defer s.Unlock()

	s.queryInfoAndCounters = make(map[stats.GroupKey]*ExtendedQueryInfoAndCounters)
}

// Queries returns all collected statistics
func (s *ExtendedStats) Queries() ExtendedQueries {
	s.Lock()
	defer s.Unlock()

	keys := stats.GroupKeys{}
	for key := range s.queryInfoAndCounters {
		keys = append(keys, key)
	}
	sort.Sort(keys)

	queries := []ExtendedQueryInfoAndCounters{}
	for _, key := range keys {
		queries = append(queries, *s.queryInfoAndCounters[key])
	}
	return queries
}

// Add adds collector.ExtendedSystemProfile to the collection of statistics
func (s *ExtendedStats) Add(doc collector.ExtendedSystemProfile) error {
	fp, err := s.fingerprinter.Fingerprint(doc.SystemProfile)
	if err != nil {
		return err // TODO &stats.StatsFingerprintError{err}
	}
	var qiac *ExtendedQueryInfoAndCounters
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
		qiac = &ExtendedQueryInfoAndCounters{
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

func (s *ExtendedStats) getQueryInfoAndCounters(key stats.GroupKey) (*ExtendedQueryInfoAndCounters, bool) {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.queryInfoAndCounters[key]
	return v, ok
}

func (s *ExtendedStats) setQueryInfoAndCounters(key stats.GroupKey, value *ExtendedQueryInfoAndCounters) {
	s.Lock()
	defer s.Unlock()

	s.queryInfoAndCounters[key] = value
}

// CalcQueriesStats calculates QueryStats for given uptime
func (q ExtendedQueries) CalcQueriesStats(uptime int64) []ExtendedQueryStats {
	qs := []ExtendedQueryStats{}
	tc := calcTotalCounters(q)

	for _, query := range q {
		queryStats := countersToStats(query, uptime, tc)
		qs = append(qs, queryStats)
	}

	return qs
}

func countersToStats(query ExtendedQueryInfoAndCounters, uptime int64, tc totalCounters) ExtendedQueryStats {
	queryStats := ExtendedQueryStats{
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

func calcTotalCounters(queries []ExtendedQueryInfoAndCounters) totalCounters {
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
