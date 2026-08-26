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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// stubLeaderCheck serves status on every request, standing in for the pmm-managed next door.
// The returned func reports how many checks it has answered, which is the only reliable proof
// the loop is still polling: the follower branch logs at debug level and so is filtered out.
func stubLeaderCheck(t *testing.T, status int) (string, func() int) {
	t.Helper()

	var mu sync.Mutex
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		w.WriteHeader(status)
		if status == http.StatusBadRequest {
			// What pmm-managed actually sends on a follower. A bodyless 400 is undetermined,
			// not a follower verdict, so the stub has to carry the gRPC code.
			_, _ = w.Write([]byte(`{"code": 9, "message": "this PMM Server isn't the leader"}`))
		}
	}))
	t.Cleanup(srv.Close)

	return srv.URL, func() int {
		mu.Lock()
		defer mu.Unlock()

		return hits
	}
}

// stubBody serves status with an arbitrary body, for the answers stubLeaderCheck cannot express.
func stubBody(t *testing.T, status int, body string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

// shortenIntervals makes the loop cycle fast enough to observe. Both values are only read.
func shortenIntervals(t *testing.T) {
	t.Helper()

	drop, recheck := defaultDropOldPartitionInterval, leaderRecheckInterval
	t.Cleanup(func() {
		defaultDropOldPartitionInterval, leaderRecheckInterval = drop, recheck
	})
	defaultDropOldPartitionInterval = time.Millisecond
	leaderRecheckInterval = time.Millisecond
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
func runLoop(t *testing.T, drop func(context.Context) error, url string) func() {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runRetentionLoop(ctx, drop, url)
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

// Deleting data on an answer we do not have is the failure this gate exists to prevent, so
// neither a follower nor an undetermined answer may reach the drop.
func TestRetentionLoopDropsOnlyAsLeader(t *testing.T) {
	t.Run("Leader", func(t *testing.T) {
		shortenIntervals(t)
		captureLogs(t)
		resetPasses(t)

		var rec dropRecorder
		url, _ := stubLeaderCheck(t, http.StatusOK)
		stop := runLoop(t, rec.drop, url)
		waitFor(t, func() bool { return rec.count() >= 2 })
		stop()

		assert.GreaterOrEqual(t, rec.count(), 2, "the leader must keep cycling")
		assert.Positive(t, passes(t, retentionApplied))
		assert.Zero(t, passes(t, retentionFailed))
	})

	for _, tc := range []struct {
		name   string
		status int
		result string
	}{
		{"Follower", http.StatusBadRequest, retentionFollower},
		{"Undetermined", http.StatusInternalServerError, retentionUndetermined},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shortenIntervals(t)
			captureLogs(t)
			resetPasses(t)

			var rec dropRecorder
			url, checks := stubLeaderCheck(t, tc.status)
			stop := runLoop(t, rec.drop, url)
			// There is nothing to wait for, so let the loop prove it keeps checking and
			// still never drops.
			waitFor(t, func() bool { return checks() >= 3 })
			stop()

			assert.Zero(t, rec.count(), "nothing may be dropped unless this node is the leader")
			assert.Positive(t, passes(t, tc.result))
			assert.Zero(t, passes(t, retentionApplied))
		})
	}
}

// A proxy on port 7772 answering 400 must not read as "follower" and quietly stop retention on
// every node. Such an answer is rejected in leader.go, and the loop must count it as
// undetermined.
func TestRetentionLoopCountsAForeign400AsUndetermined(t *testing.T) {
	shortenIntervals(t)
	captureLogs(t)
	resetPasses(t)

	var rec dropRecorder
	url := stubBody(t, http.StatusBadRequest, `{"code": 3, "message": "invalid argument"}`)
	stop := runLoop(t, rec.drop, url)
	waitFor(t, func() bool { return passes(t, retentionUndetermined) >= 2 })
	stop()

	assert.Zero(t, passes(t, retentionFollower), "a 400 without FailedPrecondition is not a follower verdict")
	assert.Zero(t, rec.count(), "nothing may be dropped on an answer we could not read")
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

	url, _ := stubLeaderCheck(t, http.StatusOK)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runRetentionLoop(ctx, drop, url)
		close(done)
	}()

	<-started
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a drop in flight kept the loop from returning, which would hold up shutdown")
	}
}

// The metric is what makes a standing failure noticeable: an operator picks the threshold in an
// alert rather than inheriting one from this source file.
func TestRetentionLoopCountsUndetermined(t *testing.T) {
	shortenIntervals(t)
	captureLogs(t)
	resetPasses(t)

	var rec dropRecorder
	url, _ := stubLeaderCheck(t, http.StatusInternalServerError)
	stop := runLoop(t, rec.drop, url)
	waitFor(t, func() bool { return passes(t, retentionUndetermined) >= 3 })
	stop()

	assert.GreaterOrEqual(t, passes(t, retentionUndetermined), float64(3))
	assert.Zero(t, rec.count(), "nothing may be dropped while leadership is undetermined")
	for _, result := range []string{retentionApplied, retentionFailed, retentionFollower} {
		assert.Zero(t, passes(t, result), "result %q", result)
	}
}

// A failed drop must not wait out the full day before trying again.
func TestRetentionLoopRetriesAfterFailedDrop(t *testing.T) {
	shortenIntervals(t)
	// Only the recheck interval stays short: if a failed drop waited out the daily interval,
	// this test would time out rather than pass.
	defaultDropOldPartitionInterval = time.Hour
	hook := captureLogs(t)
	resetPasses(t)

	rec := dropRecorder{err: errors.New("clickhouse said no")}
	url, _ := stubLeaderCheck(t, http.StatusOK)
	stop := runLoop(t, rec.drop, url)
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
