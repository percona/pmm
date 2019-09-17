/*
Copyright (c) 2019, Percona LLC.
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package event

import (
	"github.com/percona/go-mysql/log"
)

const (
	// MaxExampleBytes defines to how many bytes truncate a query.
	MaxExampleBytes = 2 * 1024 * 10

	// TruncatedExampleSuffix is added to truncated query.
	TruncatedExampleSuffix = "..."
)

// A Class represents all events with the same fingerprint and class ID.
// This is only enforced by convention, so be careful not to mix events from
// different classes.
type Class struct {
	Id                   string // 32-character hex checksum of fingerprint
	User                 string
	Host                 string
	Db                   string
	Server               string
	LabelsKey            []string
	LabelsValue          []string
	Fingerprint          string   // canonical form of query: values replaced with "?"
	Metrics              *Metrics // statistics for each metric, e.g. max Query_time
	TotalQueries         uint     // total number of queries in class
	UniqueQueries        uint     // unique number of queries in class
	Example              *Example `json:",omitempty"` // sample query with max Query_time
	NumQueriesWithErrors float32
	ErrorsCode           []uint64
	ErrorsCount          []uint64
	// --
	outliers  uint
	lastDb    string
	errorsMap map[uint64]uint64 // ErrorsCode: ErrorsCount
	sample    bool
}

// A Example is a real query and its database, timestamp, and Query_time.
// If the query is larger than MaxExampleBytes, it is truncated and TruncatedExampleSuffix
// is appended.
type Example struct {
	QueryTime float64 // Query_time
	Db        string  // Schema: <db> or USE <db>
	Query     string  // truncated to MaxExampleBytes
	Size      int     `json:",omitempty"` // Original size of query.
	Ts        string  `json:",omitempty"` // in MySQL time zone
}

// NewClass returns a new Class for the class ID and fingerprint.
// If sample is true, the query with the greatest Query_time is saved.
func NewClass(id, user, host, db, server, fingerprint string, sample bool) *Class {
	class := &Class{
		Id:           id,
		User:         user,
		Host:         host,
		Db:           db,
		Server:       server,
		LabelsKey:    []string{},
		LabelsValue:  []string{},
		Fingerprint:  fingerprint,
		Metrics:      NewMetrics(),
		TotalQueries: 0,
		Example:      &Example{},
		sample:       sample,
		errorsMap:    map[uint64]uint64{},
	}
	return class
}

// AddEvent adds an event to the query class.
func (c *Class) AddEvent(e *log.Event, outlier bool) {
	if outlier {
		c.outliers++
	} else {
		c.TotalQueries++
	}

	c.Metrics.AddEvent(e, outlier)

	// Add labels
	c.LabelsKey = append(c.LabelsKey, e.LabelsKey...)
	c.LabelsValue = append(c.LabelsValue, e.LabelsValue...)

	// Add Errors
	if lastErrno, ok := e.NumberMetrics["Last_errno"]; ok && lastErrno > 0 {
		c.errorsMap[uint64(lastErrno)]++
		c.NumQueriesWithErrors++
	}

	// Save last db seen for this query. This helps ensure the sample query
	// has a db.
	if e.Db != "" {
		c.lastDb = e.Db
	}
	if c.sample {
		if n, ok := e.TimeMetrics["Query_time"]; ok {
			if float64(n) > c.Example.QueryTime {
				c.Example.QueryTime = float64(n)
				c.Example.Size = len(e.Query)
				if e.Db != "" {
					c.Example.Db = e.Db
				} else {
					c.Example.Db = c.lastDb
				}
				if len(e.Query) > MaxExampleBytes {
					c.Example.Query = e.Query[0:MaxExampleBytes-len(TruncatedExampleSuffix)] + TruncatedExampleSuffix
				} else {
					c.Example.Query = e.Query
				}
				if !e.Ts.IsZero() {
					// todo use time.RFC3339Nano instead
					c.Example.Ts = e.Ts.UTC().Format("2006-01-02 15:04:05")
				}
			}
		}
	}
}

// AddClass adds a Class to the current class. This is used with Performance
// Schema which returns pre-aggregated classes instead of events.
func (c *Class) AddClass(newClass *Class) {
	c.UniqueQueries++
	c.TotalQueries += newClass.TotalQueries
	c.Example = nil

	for newMetric, newStats := range newClass.Metrics.TimeMetrics {
		stats, ok := c.Metrics.TimeMetrics[newMetric]
		if !ok {
			m := *newStats
			c.Metrics.TimeMetrics[newMetric] = &m
		} else {
			stats.Cnt++
			stats.Sum += newStats.Sum
			if Float64Value(newStats.Min) < Float64Value(stats.Min) || stats.Min == nil {
				stats.Min = newStats.Min
			}
			if Float64Value(newStats.Max) > Float64Value(stats.Max) || stats.Max == nil {
				stats.Max = newStats.Max
			}
		}
	}

	for newMetric, newStats := range newClass.Metrics.NumberMetrics {
		stats, ok := c.Metrics.NumberMetrics[newMetric]
		if !ok {
			m := *newStats
			c.Metrics.NumberMetrics[newMetric] = &m
		} else {
			stats.Cnt++
			stats.Sum += newStats.Sum
			if Uint64Value(newStats.Min) < Uint64Value(stats.Min) || stats.Min == nil {
				stats.Min = newStats.Min
			}
			if Uint64Value(newStats.Max) > Uint64Value(stats.Max) || stats.Max == nil {
				stats.Max = newStats.Max
			}
		}
	}

	for newMetric, newStats := range newClass.Metrics.BoolMetrics {
		stats, ok := c.Metrics.BoolMetrics[newMetric]
		if !ok {
			m := *newStats
			c.Metrics.BoolMetrics[newMetric] = &m
		} else {
			stats.Cnt++
			stats.Sum += newStats.Sum
		}
	}
}

// Finalize calculates all metric statistics. Call this function when done
// adding events to the class.
func (c *Class) Finalize(rateLimit uint) {
	if rateLimit == 0 {
		rateLimit = 1
	}

	for code, count := range c.errorsMap {
		c.ErrorsCode = append(c.ErrorsCode, code)
		c.ErrorsCount = append(c.ErrorsCount, count)
	}

	c.TotalQueries = (c.TotalQueries * rateLimit) + c.outliers
	c.Metrics.Finalize(rateLimit, c.TotalQueries)
	if c.Example.QueryTime == 0 {
		c.Example = nil
	}
}
