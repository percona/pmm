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

import "time"

// queryLogRow is one observed query execution read from system.query_log.
// Only finished phases (QueryFinish / ExceptionWhileProcessing) are read, so
// each row represents a completed query with final metric values. Columns that
// do not exist on the running ClickHouse version are left at their zero value
// by the dynamically built SELECT (see querylog.go schema detection).
type queryLogRow struct {
	// EventTime is the 1-second-granular query completion time. It is the
	// coarse watermark on ClickHouse versions without event_time_microseconds.
	EventTime time.Time
	// EventTimeMicro is the microsecond-granular completion time. It is zero
	// when the event_time_microseconds column is absent; callers must fall
	// back to EventTime in that case.
	EventTimeMicro time.Time

	// QueryID is the server-assigned query identifier, used for cross-interval
	// deduplication within the watermark second.
	QueryID string
	// Query is the raw SQL text including literals.
	Query string
	// NormalizedQueryHash is ClickHouse's own cityHash64 of the normalized
	// query (column normalized_query_hash). Zero when the column is absent.
	NormalizedQueryHash uint64
	// QueryKind is "Select", "Insert", "Create", ... (column query_kind).
	QueryKind string

	// Type is the query_log row type: 2 = QueryFinish, 4 = ExceptionWhileProcessing.
	Type uint8
	// ExceptionCode is the ClickHouse error code, 0 when the query succeeded.
	ExceptionCode int32

	// Databases / Tables are the fully qualified objects the query touched.
	Databases []string
	Tables    []string
	User      string

	// QueryDurationMs is the wall-clock execution time in milliseconds.
	QueryDurationMs uint64
	ReadRows        uint64
	ReadBytes       uint64
	ResultRows      uint64
	ResultBytes     uint64
	MemoryUsage     uint64
	WrittenRows     uint64
	WrittenBytes    uint64
}

// failed reports whether the row is an ExceptionWhileProcessing event (type 4).
func (r *queryLogRow) failed() bool { return r.Type == queryLogTypeExceptionWhileProcessing }

// watermark returns the most precise completion time available for the row:
// event_time_microseconds when present, otherwise event_time.
func (r *queryLogRow) watermark() time.Time {
	if !r.EventTimeMicro.IsZero() {
		return r.EventTimeMicro
	}
	return r.EventTime
}

// queryClass accumulates every observed execution that shares one fingerprint
// hash during a single collection interval. It is filled by makeBuckets and is
// independent of any proto type so it can be unit tested in isolation.
type queryClass struct {
	queryID     string // fingerprint hash, hex-encoded — becomes Common.Queryid
	fingerprint string // normalized query text shown in the UI
	example     string // a raw query sample with literals
	isTruncated bool   // fingerprint or example was truncated
	queryKind   string // ClickHouse query kind of the class

	database string
	tables   map[string]struct{}
	username string

	numQueries           uint64
	numQueriesWithErrors uint64
	errors               map[uint64]uint64 // ClickHouse error code -> occurrences

	// metric holds every per-execution sample of a tracked metric so that
	// makeBuckets can derive cnt/sum/min/max/p99.
	queryTime    []float32 // seconds
	readRows     []float32
	readBytes    []float32
	resultRows   []float32
	resultBytes  []float32
	memoryUsage  []float32
	writtenRows  []float32
	writtenBytes []float32
}
