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
	"context"
	"fmt"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/fingerprinter"
	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	mongostats "github.com/percona/percona-toolkit/src/go/mongolib/stats"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents/mongodb/internal/report"
	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

var DefaultInterval = time.Duration(time.Minute)

const (
	reportChanBuffer      = 1000
	millisecondsToSeconds = 1000
	microsecondsToSeconds = 1000000
)

// New returns configured *Aggregator
func New(timeStart time.Time, agentID string, logger *logrus.Entry, maxQueryLength int32) *Aggregator {
	aggregator := &Aggregator{
		agentID:        agentID,
		logger:         logger,
		maxQueryLength: maxQueryLength,
	}

	// create duration from interval
	aggregator.d = DefaultInterval

	// create mongolib stats
	fp := fingerprinter.NewFingerprinter(fingerprinter.DefaultKeyFilters())
	aggregator.mongostats = mongostats.New(fp)

	// create new interval
	aggregator.newInterval(timeStart)

	return aggregator
}

// Aggregator aggregates system.profile document
type Aggregator struct {
	agentID        string
	maxQueryLength int32
	logger         *logrus.Entry

	// provides
	reportChan chan *report.Report

	// interval
	mx         sync.RWMutex
	timeStart  time.Time
	timeEnd    time.Time
	d          time.Duration
	t          *time.Timer
	mongostats *mongostats.Stats

	// state
	m        sync.Mutex      // Lock() to protect internal consistency of the service
	running  bool            // Is this service running?
	doneChan chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg       *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Add aggregates new system.profile document
func (a *Aggregator) Add(ctx context.Context, doc proto.SystemProfile) error {
	a.m.Lock()
	defer a.m.Unlock()
	if !a.running {
		return fmt.Errorf("aggregator is not running")
	}

	ts := doc.Ts.UTC()

	// if new doc is outside of interval then finish old interval and flush it
	if !ts.Before(a.timeEnd) {
		a.flush(ctx, ts)
	}

	// we had some activity so reset timer
	a.t.Reset(a.d)

	return a.mongostats.Add(doc)
}

func (a *Aggregator) Start() <-chan *report.Report {
	a.m.Lock()
	defer a.m.Unlock()
	if a.running {
		a.logger.Debugln("aggregator already running.")
		return a.reportChan
	}

	// create new channels over which we will communicate to...
	// ... outside world by sending collected docs
	a.reportChan = make(chan *report.Report, reportChanBuffer)
	// ... inside goroutine to close it
	a.doneChan = make(chan struct{})

	// timeout after not receiving data for interval time
	a.t = time.NewTimer(a.d)

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	a.wg = &sync.WaitGroup{}
	a.wg.Add(1)

	ctx := context.Background()
	labels := pprof.Labels("component", "mongodb.aggregator")
	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(ctx, a.wg, a, a.doneChan)
	})

	a.running = true
	return a.reportChan
}

func (a *Aggregator) Stop() {
	a.m.Lock()
	defer a.m.Unlock()
	if !a.running {
		return
	}
	a.running = false

	// notify goroutine to close
	close(a.doneChan)

	// wait for goroutines to exit
	a.wg.Wait()

	// close reportChan
	close(a.reportChan)
}

func start(ctx context.Context, wg *sync.WaitGroup, aggregator *Aggregator, doneChan <-chan struct{}) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()
	for {
		select {
		case <-aggregator.t.C:
			// When Tail()ing system.profile collection you don't know if sample
			// is last sample in the collection until you get sample with higher timestamp than interval.
			// For this, in cases where we generate only few test queries,
			// but still expect them to show after interval expires, we need to implement timeout.
			// This introduces another issue, that in case something goes wrong, and we get metrics for old interval too late, they will be skipped.
			// A proper solution would be to allow fixing old samples, but API and qan-agent doesn't allow this, yet.
			aggregator.Flush(ctx)

			aggregator.m.Lock()
			aggregator.t.Reset(aggregator.d)
			aggregator.m.Unlock()
		case <-doneChan:
			// Check if we should shutdown.
			return
		}
	}
}

func (a *Aggregator) Flush(ctx context.Context) {
	a.m.Lock()
	defer a.m.Unlock()
	a.logger.Debugf("Flushing aggregator at: %s", time.Now())
	a.flush(ctx, time.Now())
}

func (a *Aggregator) flush(ctx context.Context, ts time.Time) {
	r := a.interval(ctx, ts)
	if r != nil {
		a.logger.Tracef("Sending report to reportChan:\n %v", r)
		a.reportChan <- r
	}
}

// interval sets interval if necessary and returns *qan.Report for old interval if not empty
func (a *Aggregator) interval(ctx context.Context, ts time.Time) *report.Report {
	// create new interval
	defer a.newInterval(ts)

	// let's check if we have anything to send for current interval
	if len(a.mongostats.Queries()) == 0 {
		// if there are no queries then we don't create report #PMM-927
		a.logger.Tracef("No information to send for interval: '%s - %s'", a.timeStart.Format(time.RFC3339), a.timeEnd.Format(time.RFC3339))
		return nil
	}

	// create result
	result := a.createResult(ctx)

	// translate result into report and return it
	return report.MakeReport(ctx, a.timeStart, a.timeEnd, result)
}

// TimeStart returns start time for current interval
func (a *Aggregator) TimeStart() time.Time {
	a.mx.RLock()
	defer a.mx.RUnlock()
	return a.timeStart
}

// TimeEnd returns end time for current interval
func (a *Aggregator) TimeEnd() time.Time {
	a.mx.RLock()
	defer a.mx.RUnlock()
	return a.timeEnd
}

func (a *Aggregator) newInterval(ts time.Time) {
	a.mx.Lock()
	defer a.mx.Unlock()
	// reset stats
	a.mongostats.Reset()

	// truncate to the duration e.g 12:15:35 with 1 minute duration it will be 12:15:00
	a.timeStart = ts.UTC().Truncate(a.d)
	// create ending time by adding interval
	a.timeEnd = a.timeStart.Add(a.d)
}

func (a *Aggregator) createResult(_ context.Context) *report.Result {
	queries := a.mongostats.Queries()
	queryStats := queries.CalcQueriesStats(int64(DefaultInterval))
	var buckets []*agentv1.MetricsBucket

	a.logger.Tracef("Queries: %#v", queries)
	a.logger.Tracef("Query Stats: %#v", queryStats)

	for _, v := range queryStats {
		db := ""
		collection := ""
		s := strings.SplitN(v.Namespace, ".", 2)
		if len(s) == 2 {
			db = s[0]
			collection = s[1]
		}

		fingerprint, _ := truncate.Query(v.Fingerprint, a.maxQueryLength, truncate.GetMongoDBDefaultMaxQueryLength())
		query, truncated := truncate.Query(v.Query, a.maxQueryLength, truncate.GetMongoDBDefaultMaxQueryLength())
		bucket := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid:             v.QueryHash,
				Fingerprint:         fingerprint,
				Database:            db,
				Tables:              []string{collection},
				Username:            v.User,
				ClientHost:          v.Client,
				AgentId:             a.agentID,
				AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT,
				PeriodStartUnixSecs: uint32(a.timeStart.Truncate(1 * time.Minute).Unix()),
				PeriodLengthSecs:    uint32(a.d.Seconds()),
				Example:             query,
				ExampleType:         agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
				NumQueries:          float32(v.Count),
				IsTruncated:         truncated,
				Comments:            nil, // PMM-11866
			},
			Mongodb: &agentv1.MetricsBucket_MongoDB{},
		}

		bucket.Common.MQueryTimeCnt = float32(v.Count) // PMM-13788
		bucket.Common.MQueryTimeMax = float32(v.QueryTime.Max) / millisecondsToSeconds
		bucket.Common.MQueryTimeMin = float32(v.QueryTime.Min) / millisecondsToSeconds
		bucket.Common.MQueryTimeP99 = float32(v.QueryTime.Pct99) / millisecondsToSeconds
		bucket.Common.MQueryTimeSum = float32(v.QueryTime.Total) / millisecondsToSeconds

		bucket.Mongodb.MDocsReturnedCnt = float32(v.Count) // PMM-13788
		bucket.Mongodb.MDocsReturnedMax = float32(v.Returned.Max)
		bucket.Mongodb.MDocsReturnedMin = float32(v.Returned.Min)
		bucket.Mongodb.MDocsReturnedP99 = float32(v.Returned.Pct99)
		bucket.Mongodb.MDocsReturnedSum = float32(v.Returned.Total)

		bucket.Mongodb.MResponseLengthCnt = float32(v.ResponseLengthCount)
		bucket.Mongodb.MResponseLengthMax = float32(v.ResponseLength.Max)
		bucket.Mongodb.MResponseLengthMin = float32(v.ResponseLength.Min)
		bucket.Mongodb.MResponseLengthP99 = float32(v.ResponseLength.Pct99)
		bucket.Mongodb.MResponseLengthSum = float32(v.ResponseLength.Total)

		bucket.Mongodb.MFullScanCnt = float32(v.CollScanCount)
		bucket.Mongodb.MFullScanSum = float32(v.CollScanCount) // Sum is same like count in this case
		bucket.Mongodb.PlanSummary = v.PlanSummary

		bucket.Mongodb.ApplicationName = v.AppName

		bucket.Mongodb.MDocsExaminedCnt = float32(v.DocsExaminedCount)
		bucket.Mongodb.MDocsExaminedMax = float32(v.DocsExamined.Max)
		bucket.Mongodb.MDocsExaminedMin = float32(v.DocsExamined.Min)
		bucket.Mongodb.MDocsExaminedP99 = float32(v.DocsExamined.Pct99)
		bucket.Mongodb.MDocsExaminedSum = float32(v.DocsExamined.Total)

		bucket.Mongodb.MKeysExaminedCnt = float32(v.KeysExaminedCount)
		bucket.Mongodb.MKeysExaminedMax = float32(v.KeysExamined.Max)
		bucket.Mongodb.MKeysExaminedMin = float32(v.KeysExamined.Min)
		bucket.Mongodb.MKeysExaminedP99 = float32(v.KeysExamined.Pct99)
		bucket.Mongodb.MKeysExaminedSum = float32(v.KeysExamined.Total)

		bucket.Mongodb.MLocksGlobalAcquireCountReadSharedCnt = float32(v.LocksGlobalAcquireCountReadShared)
		bucket.Mongodb.MLocksGlobalAcquireCountReadSharedSum = float32(v.LocksGlobalAcquireCountReadShared) // Sum is same like count in this case

		bucket.Mongodb.MLocksGlobalAcquireCountWriteSharedCnt = float32(v.LocksGlobalAcquireCountWriteShared)
		bucket.Mongodb.MLocksGlobalAcquireCountWriteSharedSum = float32(v.LocksGlobalAcquireCountWriteShared) // Sum is same like count in this case

		bucket.Mongodb.MLocksDatabaseAcquireCountReadSharedCnt = float32(v.LocksDatabaseAcquireCountReadShared)
		bucket.Mongodb.MLocksDatabaseAcquireCountReadSharedSum = float32(v.LocksDatabaseAcquireCountReadShared) // Sum is same like count in this case

		bucket.Mongodb.MLocksDatabaseAcquireWaitCountReadSharedCnt = float32(v.LocksDatabaseAcquireWaitCountReadShared)
		bucket.Mongodb.MLocksDatabaseAcquireWaitCountReadSharedSum = float32(v.LocksDatabaseAcquireWaitCountReadShared) // Sum is same like count in this case

		bucket.Mongodb.MLocksDatabaseTimeAcquiringMicrosReadSharedCnt = float32(v.LocksDatabaseTimeAcquiringMicrosReadSharedCount)
		bucket.Mongodb.MLocksDatabaseTimeAcquiringMicrosReadSharedMax = float32(v.LocksDatabaseTimeAcquiringMicrosReadShared.Max) / microsecondsToSeconds
		bucket.Mongodb.MLocksDatabaseTimeAcquiringMicrosReadSharedMin = float32(v.LocksDatabaseTimeAcquiringMicrosReadShared.Min) / microsecondsToSeconds
		bucket.Mongodb.MLocksDatabaseTimeAcquiringMicrosReadSharedP99 = float32(v.LocksDatabaseTimeAcquiringMicrosReadShared.Pct99) / microsecondsToSeconds
		bucket.Mongodb.MLocksDatabaseTimeAcquiringMicrosReadSharedSum = float32(v.LocksDatabaseTimeAcquiringMicrosReadShared.Total) / microsecondsToSeconds

		bucket.Mongodb.MLocksCollectionAcquireCountReadSharedCnt = float32(v.LocksCollectionAcquireCountReadShared)
		bucket.Mongodb.MLocksCollectionAcquireCountReadSharedSum = float32(v.LocksCollectionAcquireCountReadShared) // Sum is same like count in this case

		bucket.Mongodb.MStorageBytesReadCnt = float32(v.StorageBytesReadCount)
		bucket.Mongodb.MStorageBytesReadMax = float32(v.StorageBytesRead.Max)
		bucket.Mongodb.MStorageBytesReadMin = float32(v.StorageBytesRead.Min)
		bucket.Mongodb.MStorageBytesReadP99 = float32(v.StorageBytesRead.Pct99)
		bucket.Mongodb.MStorageBytesReadSum = float32(v.StorageBytesRead.Total)

		bucket.Mongodb.MStorageTimeReadingMicrosCnt = float32(v.StorageTimeReadingMicrosCount)
		bucket.Mongodb.MStorageTimeReadingMicrosMax = float32(v.StorageTimeReadingMicros.Max) / microsecondsToSeconds
		bucket.Mongodb.MStorageTimeReadingMicrosMin = float32(v.StorageTimeReadingMicros.Min) / microsecondsToSeconds
		bucket.Mongodb.MStorageTimeReadingMicrosP99 = float32(v.StorageTimeReadingMicros.Pct99) / microsecondsToSeconds
		bucket.Mongodb.MStorageTimeReadingMicrosSum = float32(v.StorageTimeReadingMicros.Total) / microsecondsToSeconds

		buckets = append(buckets, bucket)
	}

	return &report.Result{
		Buckets: buckets,
	}
}
