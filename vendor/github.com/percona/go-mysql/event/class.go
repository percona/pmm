/*
	Copyright (c) 2014-2015, Percona LLC and/or its affiliates. All rights reserved.

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
	Id            string   // 32-character hex checksum of fingerprint
	Fingerprint   string   // canonical form of query: values replaced with "?"
	Metrics       *Metrics // statistics for each metric, e.g. max Query_time
	TotalQueries  uint     // total number of queries in class
	UniqueQueries uint     // unique number of queries in class
	Example       *Example `json:",omitempty"` // sample query with max Query_time
	// --
	outliers uint
	lastDb   string
	sample   bool
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
func NewClass(id, fingerprint string, sample bool) *Class {
	class := &Class{
		Id:           id,
		Fingerprint:  fingerprint,
		Metrics:      NewMetrics(),
		TotalQueries: 0,
		Example:      &Example{},
		sample:       sample,
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
			stats.Sum += newStats.Sum
			stats.Avg = Float64(stats.Sum / float64(c.TotalQueries))
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
			stats.Sum += newStats.Sum
			stats.Avg = Uint64(stats.Sum / uint64(c.TotalQueries))
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
	c.TotalQueries = (c.TotalQueries * rateLimit) + c.outliers
	c.Metrics.Finalize(rateLimit, c.TotalQueries)
	if c.Example.QueryTime == 0 {
		c.Example = nil
	}
}
