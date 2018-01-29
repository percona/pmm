/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package qan

import (
	"time"

	"github.com/percona/go-mysql/event"
	"github.com/percona/pmm/proto/metrics"
	"github.com/percona/pmm/proto/query"
)

type Report struct {
	UUID    string         // UUID of MySQL instance
	StartTs time.Time      // Start time of interval, UTC
	EndTs   time.Time      // Stop time of interval, UTC
	RunTime float64        // Time parsing data, seconds
	Global  *event.Class   // Metrics for all data
	Class   []*event.Class // per-class metrics
	// slow log:
	SlowLogFile     string `json:",omitempty"` // not slow_query_log_file if rotated
	SlowLogFileSize int64  `json:",omitempty"`
	StartOffset     int64  `json:",omitempty"` // parsing starts
	EndOffset       int64  `json:",omitempty"` // parsing stops, but...
	StopOffset      int64  `json:",omitempty"` // ...parsing didn't complete if stop < end
	RateLimit       uint   `json:",omitempty"` // Percona Server rate limit
}

type Profile struct {
	InstanceId   string      // UUID of MySQL instance
	Begin        time.Time   // time range [Begin, End)
	End          time.Time   // time range [Being, End)
	TotalTime    uint        // total seconds in time range minus gaps (missing periods)
	TotalQueries uint        // total unique class queries in time range
	RankBy       RankBy      // criteria for ranking queries compared to global
	Query        []QueryRank // 0=global, 1..N=queries
}

type RankBy struct {
	Metric string // default: Query_time
	Stat   string // default: sum
	Limit  uint   // default: 10
}

// start_ts, query_count, Query_time_sum
type QueryLog struct {
	Point          uint
	Start_ts       time.Time
	Query_count    float32
	Query_load     float32
	Query_time_avg float32
}

type QueryRank struct {
	Rank        uint    // compared to global, same as Profile.Ranks index
	Percentage  float64 // of global value
	Id          string  // hex checksum
	Abstract    string  // e.g. SELECT tbl
	Fingerprint string  // e.g. SELECT tbl
	QPS         float64 // ResponseTime.Cnt / Profile.TotalTime
	Load        float64 // Query_time_sum / (Profile.End - Profile.Begin)
	Log         []QueryLog
	Stats       metrics.Stats // this query's Profile.Metric stats
}

type QueryReport struct {
	InstanceId string                   // UUID of MySQL instance
	Begin      time.Time                // time range [Begin, End)
	End        time.Time                // time range [Being, End)
	Query      query.Query              // id, abstract, fingerprint, etc.
	Metrics    map[string]metrics.Stats // keyed on metric name, e.g. Query_time
	Example    query.Example            // query example
	Sparks     []interface{}            `json:",omitempty"`
	Metrics2   interface{}              `json:",omitempty"`
	Sparks2    interface{}              `json:",omitempty"`
}

type Summary struct {
	InstanceId string                   // UUID of MySQL instance
	Begin      time.Time                // time range [Begin, End)
	End        time.Time                // time range [Being, End)
	Metrics    map[string]metrics.Stats // keyed on metric name, e.g. Query_time
	Sparks     []interface{}            `json:",omitempty"`
	Metrics2   interface{}              `json:",omitempty"`
	Sparks2    interface{}              `json:",omitempty"`
}
