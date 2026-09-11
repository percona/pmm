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

// shortenIntervals makes the loop cycle fast enough to observe. Both values are only read.
func shortenIntervals(t *testing.T) {
	t.Helper()

	drop, retry := defaultDropOldPartitionInterval, retentionRetryInterval
	t.Cleanup(func() {
		defaultDropOldPartitionInterval, retentionRetryInterval = drop, retry
	})
	defaultDropOldPartitionInterval = time.Millisecond
	retentionRetryInterval = time.Millisecond
}

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

// dropRecorder counts calls and returns err each time.
type dropRecorder struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (d *dropRecorder) drop(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++

	return d.err
}

func (d *dropRecorder) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.calls
}

// runLoop starts the loop and returns a stop function that cancels it and waits for it to exit.
func runLoop(t *testing.T, drop func(context.Context) error) func() {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runRetentionLoop(ctx, drop)
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

// Retention is applied on every node, on a schedule, with no leadership to wait for: the period
// is fixed at start-up, so every replica agrees on which partitions are old.
func TestRetentionLoopDropsOnSchedule(t *testing.T) {
	shortenIntervals(t)
	captureLogs(t)
	resetPasses(t)

	var rec dropRecorder
	stop := runLoop(t, rec.drop)
	waitFor(t, func() bool { return rec.count() >= 2 })
	stop()

	assert.GreaterOrEqual(t, rec.count(), 2, "the loop must keep cycling")
	assert.Positive(t, passes(t, retentionApplied))
	assert.Zero(t, passes(t, retentionFailed))
}

// The drop is synchronous, so without a context it could hold up shutdown for as long as
// ClickHouse takes to answer. The loop must hand its own context down to it.
func TestRetentionLoopCancelsAnInFlightDrop(t *testing.T) {
	shortenIntervals(t)
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
		runRetentionLoop(ctx, drop)
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
	shortenIntervals(t)
	// Only the retry interval stays short: if a failed drop waited out the daily interval,
	// this test would time out rather than pass.
	defaultDropOldPartitionInterval = time.Hour
	hook := captureLogs(t)
	resetPasses(t)

	rec := dropRecorder{err: errors.New("clickhouse said no")}
	stop := runLoop(t, rec.drop)
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

// An alert on a condition that has never happened must read as zero rather than as no data,
// which only holds if every label value exists before the first pass.
func TestRetentionCountersAreSeeded(t *testing.T) {
	resetPasses(t)

	seedRetentionCounters()

	assert.Equal(t, 2, testutil.CollectAndCount(mRetentionPasses), "every outcome needs a zero series")
	assert.Zero(t, passes(t, retentionApplied))
	assert.Zero(t, passes(t, retentionFailed))
}
