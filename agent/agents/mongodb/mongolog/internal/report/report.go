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

package report

import (
	"context"
	"sort"
	"time"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

type Report struct {
	StartTs time.Time                // Start time of interval, UTC
	EndTs   time.Time                // Stop time of interval, UTC
	Buckets []*agentv1.MetricsBucket // per-class metrics
}

func MakeReport(_ context.Context, startTime, endTime time.Time, result *Result) *Report {
	// Sort classes by Query_time_sum, descending.
	sort.Sort(ByQueryTime(result.Buckets))

	// Make qan.Report from Result and other metadata (e.g. Interval).
	report := &Report{
		StartTs: startTime,
		EndTs:   endTime,
		Buckets: result.Buckets,
	}

	return report
}

// mongodb-mongolog --> Result --> qan.Report --> data.Spooler

// Data for an interval from slow log or performance schema (pfs) parser,
// passed to MakeReport() which transforms into a qan.Report{}.
type Result struct {
	Buckets []*agentv1.MetricsBucket
}

type ByQueryTime []*agentv1.MetricsBucket

func (a ByQueryTime) Len() int      { return len(a) }
func (a ByQueryTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByQueryTime) Less(i, j int) bool {
	if a == nil || a[i] == nil || a[j] == nil {
		return false
	}
	// descending order
	return a[i].Common.MQueryTimeSum > a[j].Common.MQueryTimeSum
}
