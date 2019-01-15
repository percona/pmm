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

// Package log provides an interface and data structures for MySQL log parsers.
// Log parsing yields events that are aggregated to calculate metric statistics
// like max Query_time. See also percona.com/go-mysql/event/.
package log

import (
	"time"
)

// An event is a query like "SELECT col FROM t WHERE id = 1", some metrics like
// Query_time (slow log) or SUM_TIMER_WAIT (Performance Schema), and other
// metadata like default database, timestamp, etc. Metrics and metadata are not
// guaranteed to be defined--and frequently they are not--but at minimum an
// event is expected to define the query and Query_time metric. Other metrics
// and metadata vary according to MySQL version, distro, and configuration.
type Event struct {
	Offset        uint64    // byte offset in file at which event starts
	OffsetEnd     uint64    // byte offset in file at which event ends
	Ts            time.Time // timestamp of event
	Admin         bool      // true if Query is admin command
	Query         string    // SQL query or admin command
	User          string
	Host          string
	Db            string
	TimeMetrics   map[string]float64 // *_time and *_wait metrics
	NumberMetrics map[string]uint64  // most metrics
	BoolMetrics   map[string]bool    // yes/no metrics
	RateType      string             // Percona Server rate limit type
	RateLimit     uint               // Percona Server rate limit value
}

// NewEvent returns a new Event with initialized metric maps.
func NewEvent() *Event {
	event := new(Event)
	event.TimeMetrics = make(map[string]float64)
	event.NumberMetrics = make(map[string]uint64)
	event.BoolMetrics = make(map[string]bool)
	return event
}

// Options encapsulate common options for making a new LogParser.
type Options struct {
	StartOffset        uint64          // byte offset in file at which to start parsing
	FilterAdminCommand map[string]bool // admin commands to ignore
	Debug              bool            // print trace info to STDOUT
	DefaultLocation    *time.Location  // DefaultLocation to assume for logs in MySQL < 5.7 format.
}

// A LogParser sends events to a channel.
type LogParser interface {
	Start() error
	Stop()
	EventChan() <-chan *Event
}
