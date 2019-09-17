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
