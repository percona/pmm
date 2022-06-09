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
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

var DefaultInterval = time.Duration(time.Minute)

const reportChanBuffer = 1000

// New returns configured *Aggregator
func New(timeStart time.Time, agentID string, logger *logrus.Entry) *Aggregator {
	aggregator := &Aggregator{
		agentID: agentID,
		logger:  logger,
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
	agentID string
	logger  *logrus.Entry

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

func (a *Aggregator) createResult(ctx context.Context) *report.Result {
	queries := a.mongostats.Queries()
	queryStats := queries.CalcQueriesStats(int64(DefaultInterval))
	var buckets []*agentpb.MetricsBucket

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

		fingerprint, _ := truncate.Query(v.Fingerprint)
		query, truncated := truncate.Query(v.Query)
		bucket := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Queryid:             v.ID,
				Fingerprint:         fingerprint,
				Database:            db,
				Tables:              []string{collection},
				Username:            "",
				ClientHost:          "",
				AgentId:             a.agentID,
				AgentType:           inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT,
				PeriodStartUnixSecs: uint32(a.timeStart.Truncate(1 * time.Minute).Unix()),
				PeriodLengthSecs:    uint32(a.d.Seconds()),
				Example:             query,
				ExampleFormat:       agentpb.ExampleFormat_EXAMPLE,
				ExampleType:         agentpb.ExampleType_RANDOM,
				NumQueries:          float32(v.Count),
				IsTruncated:         truncated,
			},
			Mongodb: &agentpb.MetricsBucket_MongoDB{},
		}

		bucket.Common.MQueryTimeCnt = float32(v.Count) // TODO: Check is it right value
		bucket.Common.MQueryTimeMax = float32(v.QueryTime.Max)
		bucket.Common.MQueryTimeMin = float32(v.QueryTime.Min)
		bucket.Common.MQueryTimeP99 = float32(v.QueryTime.Pct99)
		bucket.Common.MQueryTimeSum = float32(v.QueryTime.Total)

		bucket.Mongodb.MDocsReturnedCnt = float32(v.Count) // TODO: Check is it right value
		bucket.Mongodb.MDocsReturnedMax = float32(v.Returned.Max)
		bucket.Mongodb.MDocsReturnedMin = float32(v.Returned.Min)
		bucket.Mongodb.MDocsReturnedP99 = float32(v.Returned.Pct99)
		bucket.Mongodb.MDocsReturnedSum = float32(v.Returned.Total)

		bucket.Mongodb.MDocsScannedCnt = float32(v.Count) // TODO: Check is it right value
		bucket.Mongodb.MDocsScannedMax = float32(v.Scanned.Max)
		bucket.Mongodb.MDocsScannedMin = float32(v.Scanned.Min)
		bucket.Mongodb.MDocsScannedP99 = float32(v.Scanned.Pct99)
		bucket.Mongodb.MDocsScannedSum = float32(v.Scanned.Total)

		bucket.Mongodb.MResponseLengthCnt = float32(v.Count) // TODO: Check is it right value
		bucket.Mongodb.MResponseLengthMax = float32(v.ResponseLength.Max)
		bucket.Mongodb.MResponseLengthMin = float32(v.ResponseLength.Min)
		bucket.Mongodb.MResponseLengthP99 = float32(v.ResponseLength.Pct99)
		bucket.Mongodb.MResponseLengthSum = float32(v.ResponseLength.Total)

		buckets = append(buckets, bucket)
	}

	return &report.Result{
		Buckets: buckets,
	}
}
