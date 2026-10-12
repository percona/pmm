// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// captureLogs collects what the loop logs, and keeps it off the test output. The hook and the
// output are global, so callers must not run in parallel.
func captureLogs(t *testing.T) *logrustest.Hook {
	t.Helper()

	out := logrus.StandardLogger().Out
	logrus.SetOutput(io.Discard)
	hook := logrustest.NewLocal(logrus.StandardLogger())
	t.Cleanup(func() {
		logrus.SetOutput(out)
		logrus.StandardLogger().ReplaceHooks(logrus.LevelHooks{})
	})

	return hook
}

// resetPasses clears the package-level counter so each test reads only its own passes.
func resetPasses(t *testing.T) {
	t.Helper()

	mRetentionPasses.Reset()
	t.Cleanup(mRetentionPasses.Reset)
}

// passes reports how many retention passes ended in result.
func passes(t *testing.T, result string) float64 {
	t.Helper()

	return testutil.ToFloat64(mRetentionPasses.WithLabelValues(result))
}

// dropRecorder records when each call began and returns err each time.
type dropRecorder struct {
	mu     sync.Mutex
	starts []time.Time
	// How long a call takes before it returns, for the cases where the length of the pass is
	// what is under test.
	delay time.Duration
	err   error
}

func (d *dropRecorder) drop(context.Context) error {
	d.mu.Lock()
	d.starts = append(d.starts, time.Now())
	delay, err := d.delay, d.err
	d.mu.Unlock()

	time.Sleep(delay)

	return err
}

func (d *dropRecorder) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return len(d.starts)
}

// gaps reports the wait between the start of each pass and the start of the next.
func (d *dropRecorder) gaps() []time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()

	gaps := make([]time.Duration, 0, len(d.starts))
	for i := 1; i < len(d.starts); i++ {
		gaps = append(gaps, d.starts[i].Sub(d.starts[i-1]))
	}

	return gaps
}

// runLoop starts the loop and returns a stop function that cancels it and waits for it to exit.
func runLoop(t *testing.T, interval, retry time.Duration, drop func(context.Context) error) func() {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runRetentionLoop(ctx, interval, retry, drop)
		close(done)
	}()

	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("runRetentionLoop did not return after the context was canceled")
			}
		})
	}
	t.Cleanup(stop)

	return stop
}

// waitFor blocks until the condition holds, failing the test if it never does.
func waitFor(t *testing.T, until func() bool) {
	t.Helper()

	deadline := time.After(5 * time.Second)
	for !until() {
		select {
		case <-deadline:
			t.Fatal("the loop did not reach the expected state in time")
		case <-time.After(time.Millisecond):
		}
	}
}

// Retention is applied on every node, on a schedule, with no leadership to wait for: once a
// rollout completes every replica runs with the same period, so they agree on which partitions
// are old.
func TestRetentionLoopDropsOnSchedule(t *testing.T) {
	captureLogs(t)
	resetPasses(t)

	var rec dropRecorder
	stop := runLoop(t, time.Millisecond, time.Millisecond, rec.drop)
	waitFor(t, func() bool { return rec.count() >= 2 })
	stop()

	assert.GreaterOrEqual(t, rec.count(), 2, "the loop must keep cycling")
	assert.Positive(t, passes(t, retentionApplied))
	assert.Zero(t, passes(t, retentionFailed))
}

// The drop is synchronous, so without a context it could hold up shutdown for as long as
// ClickHouse takes to answer. The loop must hand its own context down to it.
func TestRetentionLoopCancelsAnInFlightDrop(t *testing.T) {
	captureLogs(t)
	resetPasses(t)

	started := make(chan struct{})
	// Blocks until the context it was handed is canceled, which only happens if the loop
	// actually passes its own down rather than a background one.
	drop := func(ctx context.Context) error {
		close(started)
		<-ctx.Done()

		return ctx.Err()
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runRetentionLoop(ctx, time.Millisecond, time.Millisecond, drop)
		close(done)
	}()

	<-started
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a drop in flight kept the loop from returning, which would hold up shutdown")
	}

	assert.Zero(t, passes(t, retentionFailed), "our own shutdown is not a retention failure")
}

// A failed drop must not wait out the full day before trying again.
func TestRetentionLoopRetriesAfterFailedDrop(t *testing.T) {
	// Only the retry interval is short: if a failed drop waited out the normal interval, this
	// test would time out rather than pass.
	const (
		interval = time.Hour
		retry    = time.Millisecond
	)

	hook := captureLogs(t)
	resetPasses(t)

	rec := dropRecorder{err: errors.New("clickhouse said no")}
	stop := runLoop(t, interval, retry, rec.drop)
	waitFor(t, func() bool { return rec.count() >= 2 })
	stop()

	assert.GreaterOrEqual(t, rec.count(), 2, "a failed drop must be retried at the short interval")

	var errCount int
	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.ErrorLevel {
			errCount++
		}
	}
	assert.Positive(t, errCount, "a failed drop must be reported at error level")
	assert.Positive(t, passes(t, retentionFailed))
	assert.Zero(t, passes(t, retentionApplied))
}

// A drop that takes longer to fail than the retry interval must still back off for that interval
// before the next attempt. Anchored on the start of the pass, the deadline would already be in the
// past and the loop would re-enter an already-degraded ClickHouse with no delay at all.
func TestRetentionLoopBacksOffAfterASlowFailure(t *testing.T) {
	// Only the retry path is under test, so the normal interval must not be what schedules the
	// second call.
	const (
		interval     = time.Hour
		retry        = 50 * time.Millisecond
		dropDuration = 100 * time.Millisecond
	)

	captureLogs(t)
	resetPasses(t)

	rec := dropRecorder{delay: dropDuration, err: errors.New("clickhouse took its time saying no")}
	stop := runLoop(t, interval, retry, rec.drop)
	waitFor(t, func() bool { return rec.count() >= 2 })
	stop()

	assert.GreaterOrEqual(t, rec.gaps()[0], dropDuration+retry,
		"a slow failure must still be followed by the retry interval rather than retried at once")
}

// A partition that can never be dropped must not keep a once-a-day task running every few minutes
// for good, on every replica. Nothing here can tell a transient failure from a permanent one, so
// the retry delay doubles after each consecutive failure and stops growing at the normal interval.
func TestRetentionLoopBackoffGrowsAndIsCapped(t *testing.T) {
	const (
		interval = 50 * time.Millisecond
		retry    = 10 * time.Millisecond
		// Far enough in for unbounded doubling to be unmistakable: it would be waiting 320ms
		// by the seventh pass, against a 50ms ceiling.
		wantPasses = 7
	)

	captureLogs(t)
	resetPasses(t)

	rec := dropRecorder{err: errors.New("clickhouse said no, and means it")}
	stop := runLoop(t, interval, retry, rec.drop)
	waitFor(t, func() bool { return rec.count() >= wantPasses })
	stop()

	gaps := rec.gaps()

	// Two doublings: 10ms, then 20ms, then 40ms. A timer never fires early, so a lower bound
	// on a wait cannot flake.
	assert.GreaterOrEqual(t, gaps[2], 4*retry, "the retry delay must grow while the drop keeps failing")
	// And there it stops. The margin is deliberately wide, because an upper bound on a wait is
	// only as tight as the machine running it; unbounded doubling would be at 320ms by here.
	assert.Less(t, gaps[5], 4*interval, "the retry delay must stop growing at the normal interval")
}

// An alert on a condition that has never happened must read as zero rather than as no data,
// which only holds if every label value exists before the first pass.
func TestRetentionCountersAreSeeded(t *testing.T) {
	resetPasses(t)

	seedRetentionCounters()

	assert.Equal(t, 2, testutil.CollectAndCount(mRetentionPasses), "every outcome needs a zero series")
	assert.Zero(t, passes(t, retentionApplied))
	assert.Zero(t, passes(t, retentionFailed))
}
