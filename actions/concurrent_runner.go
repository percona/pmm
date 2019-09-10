// pmm-agent
// Copyright 2019 Percona LLC
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

package actions

import (
	"context"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultTimeout = time.Second * 10

// ActionResult represents an Action result.
type ActionResult struct {
	ID     string
	Output []byte
	Error  string
}

// ConcurrentRunner represents concurrent Action runner.
// Action runner is component that can run an Actions.
type ConcurrentRunner struct {
	ctx     context.Context
	timeout time.Duration
	l       *logrus.Entry
	results chan ActionResult

	runningActions sync.WaitGroup

	rw            sync.RWMutex
	actionsCancel map[string]context.CancelFunc
}

// NewConcurrentRunner returns new runner.
// With this component you can run actions concurrently and read action results when they will be finished.
// If timeout is 0 it sets to default = 10 seconds.
//
// ConcurrentRunner is stopped when context passed to NewConcurrentRunner is canceled.
// Results are reported via Results() channel which must be read until it is closed.
func NewConcurrentRunner(ctx context.Context, timeout time.Duration) *ConcurrentRunner {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	r := &ConcurrentRunner{
		ctx:           ctx,
		timeout:       timeout,
		l:             logrus.WithField("component", "actions-runner"),
		results:       make(chan ActionResult),
		actionsCancel: make(map[string]context.CancelFunc),
	}

	// let all actions finish and send their results before closing it
	go func() {
		<-ctx.Done()
		r.runningActions.Wait()
		r.l.Infof("Done.")
		close(r.results)
	}()

	return r
}

// Start starts an Action in a separate goroutine.
func (r *ConcurrentRunner) Start(a Action) {
	if err := r.ctx.Err(); err != nil {
		r.l.Errorf("Ignoring Start: %s.", err)
		return
	}

	// FIXME There is a data race. Add must not be called concurrently with Wait, but it can be:
	// 0. no actions are running, WaitGroup has 0
	// 1. Start is called
	// 2. ctx is canceled on this line
	// 3. Wait is called in the goroutine above
	// 4. Add is called below
	// 5. Add panics with "sync: WaitGroup misuse: Add called concurrently with Wait"
	// See skipped test (run it in a loop with race detector).
	// https://jira.percona.com/browse/PMM-4112
	r.runningActions.Add(1)
	actionID, actionType := a.ID(), a.Type()
	ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
	run := func(ctx context.Context) {
		defer r.runningActions.Done()
		defer cancel()

		r.rw.Lock()
		r.actionsCancel[actionID] = cancel
		r.rw.Unlock()

		l := r.l.WithFields(logrus.Fields{"id": actionID, "type": actionType})
		l.Infof("Starting...")

		b, err := a.Run(ctx)

		r.rw.Lock()
		delete(r.actionsCancel, actionID)
		r.rw.Unlock()

		if err == nil {
			l.Infof("Done without error.")
		} else {
			l.Warnf("Done with error: %s.", err)
		}

		var errorS string
		if err != nil {
			errorS = err.Error()
		}
		r.results <- ActionResult{
			ID:     actionID,
			Output: b,
			Error:  errorS,
		}
	}
	go pprof.Do(ctx, pprof.Labels("actionID", actionID, "type", actionType), run)
}

// Results returns channel with Actions results.
func (r *ConcurrentRunner) Results() <-chan ActionResult {
	return r.results
}

// Stop stops running Action.
func (r *ConcurrentRunner) Stop(id string) {
	r.rw.RLock()
	defer r.rw.RUnlock()
	if cancel, ok := r.actionsCancel[id]; ok {
		cancel()
	}
}
