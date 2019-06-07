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

// Package event aggregates MySQL log and Perfomance Schema events into query
// classes and calculates basic statistics for class metrics like max Query_time.
// Event aggregation into query classes is the foundation of MySQL log file and
// Performance Schema analysis.
//
// An event is a query like "SELECT col FROM t WHERE id = 1", some metrics like
// Query_time (slow log) or SUM_TIMER_WAIT (Performance Schema), and other
// metadata like default database, timestamp, etc. Events are grouped into query
// classes by fingerprinting the query (see percona.com/go-mysql/query/), then
// checksumming the fingerprint which yields a 16-character hexadecimal value
// called the class ID. As events are added to a class, metric values are saved.
// When there are no more events, the class is finalized to compute the statistics
// for all metrics.
//
// There are two types of classes: global and per-query. A global class contains
// all events added to it, regardless of fingerprint or class ID. This is used,
// for example, to aggregate all events in a log file. A per-query class contains
// events with the same fingerprint and class ID. This is only enforced by
// convention, so be careful not to mix events from different classes. This is
// used, for example, to aggregate unique queries in a log file, then sort the
// classes by some metric, like max Query_time, to find the slowest query
// relative to a global class for the same set of events.
package event
