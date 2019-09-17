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
	"fmt"
	"time"

	"github.com/percona/go-mysql/log"
)

// A Result contains a global class and per-ID classes with finalized metric
// statistics. The classes are keyed on class ID.
type Result struct {
	Global    *Class            // all classes
	Class     map[string]*Class // keyed on class ID
	RateLimit uint
	Error     string
}

// An Aggregator groups events by class ID. When there are no more events,
// a call to Finalize computes all metric statistics and returns a Result.
type Aggregator struct {
	samples     bool
	utcOffset   time.Duration
	outlierTime float64
	// --
	global    *Class
	classes   map[string]*Class
	rateLimit uint
}

// NewAggregator returns a new Aggregator.
// outlierTime is https://www.percona.com/doc/percona-server/5.5/diagnostics/slow_extended_55.html#slow_query_log_always_write_time
func NewAggregator(samples bool, utcOffset time.Duration, outlierTime float64) *Aggregator {
	a := &Aggregator{
		samples:     samples,
		utcOffset:   utcOffset,
		outlierTime: outlierTime,
		// --
		global:  NewClass("", "", "", "", "", "", false),
		classes: make(map[string]*Class),
	}
	return a
}

// AddEvent adds the event to the aggregator, automatically creating new classes
// as needed.
func (a *Aggregator) AddEvent(event *log.Event, id, user, host, db, server, fingerprint string) {
	if a.rateLimit != event.RateLimit {
		a.rateLimit = event.RateLimit
	}

	outlier := false
	if a.outlierTime > 0 && event.TimeMetrics["Query_time"] > a.outlierTime {
		outlier = true
	}

	a.global.AddEvent(event, outlier)

	// Group events by all dimentions.
	ident := fmt.Sprintf("%s;%s;%s;%s;%s", id, user, host, db, server)
	class, ok := a.classes[ident]
	if !ok {
		class = NewClass(id, user, host, db, server, fingerprint, a.samples)
		a.classes[ident] = class
	}
	class.AddEvent(event, outlier)
}

// Finalize calculates all metric statistics and returns a Result.
// Call this function when done adding events to the aggregator.
func (a *Aggregator) Finalize() Result {
	a.global.Finalize(a.rateLimit)
	a.global.UniqueQueries = uint(len(a.classes))
	for _, class := range a.classes {
		class.Finalize(a.rateLimit)
		class.UniqueQueries = 1
		if class.Example != nil && class.Example.Ts != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", class.Example.Ts); err != nil {
				class.Example.Ts = ""
			} else {
				class.Example.Ts = t.Add(a.utcOffset).Format("2006-01-02 15:04:05")
			}
		}
	}
	return Result{
		Global:    a.global,
		Class:     a.classes,
		RateLimit: a.rateLimit,
	}
}
