// Copyright (C) 2023 Percona LLC
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

// Package runner implements concurrent jobs.Job and actions.Action runner.
package runner

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/runner/actions"
	"github.com/percona/pmm/agent/runner/jobs"
	agenterrors "github.com/percona/pmm/agent/utils/errors"
	"github.com/percona/pmm/api/agentpb"
)

const (
	bufferSize           = 256
	defaultActionTimeout = 10 * time.Second // default timeout for compatibility with an older server
	defaultTotalCapacity = 32               // how many concurrent operations are allowed in total
	defaultTokenCapacity = 2                // how many concurrent operations on a single resource (usually DB instance) are allowed
)

// Runner executes jobs and actions.
type Runner struct {
	l *logrus.Entry

	actions chan actions.Action
	jobs    chan jobs.Job

	actionsMessages chan agentpb.AgentRequestPayload
	jobsMessages    chan agentpb.AgentResponsePayload

	wg sync.WaitGroup

	// cancels holds cancel functions for running actions and jobs.
	cancelsM sync.RWMutex
	cancels  map[string]context.CancelFunc

	// running holds IDs of running actions and jobs.
	runningM sync.RWMutex
	running  map[string]struct{}

	// gSem is a global semaphore to limit total number of concurrent operations performed by the runner.
	gSem *semaphore.Weighted

	// tokenCapacity is a limit of concurrent operations on a single resource, usually database instance.
	tokenCapacity uint16

	// lSems is a map of local semaphores to limit number of concurrent operations on a single database instance.
	// Key is a token which is typically is a hash of DSN(only host:port pair), value is a semaphore.
	lSemsM sync.Mutex
	lSems  map[string]*entry
}

// entry stores local semaphore and its counter.
type entry struct {
	count atomic.Int32
	sem   *semaphore.Weighted
}

// New creates new runner. If capacity is 0 then default value is used.
func New(totalCapacity, tokenCapacity uint16) *Runner {
	l := logrus.WithField("component", "runner")
	if totalCapacity == 0 {
		totalCapacity = defaultTotalCapacity
	}

	if tokenCapacity == 0 {
		tokenCapacity = defaultTokenCapacity
	}

	l.Infof("Runner capacity set to %d, token capacity set to %d", totalCapacity, tokenCapacity)

	return &Runner{
		l:               l,
		actions:         make(chan actions.Action, bufferSize),
		jobs:            make(chan jobs.Job, bufferSize),
		cancels:         make(map[string]context.CancelFunc),
		running:         make(map[string]struct{}),
		jobsMessages:    make(chan agentpb.AgentResponsePayload),
		actionsMessages: make(chan agentpb.AgentRequestPayload),
		tokenCapacity:   tokenCapacity,
		gSem:            semaphore.NewWeighted(int64(totalCapacity)),
		lSems:           make(map[string]*entry),
	}
}

// acquire acquires global and local semaphores.
func (r *Runner) acquire(ctx context.Context, token string) error {
	if err := r.acquireL(ctx, token); err != nil {
		return err
	}

	if err := r.gSem.Acquire(ctx, 1); err != nil {
		r.releaseL(token)
		return err
	}

	return nil
}

// release releases global and local semaphores.
func (r *Runner) release(token string) {
	r.gSem.Release(1)

	r.releaseL(token)
}

// acquireL acquires local semaphore for given token.
func (r *Runner) acquireL(ctx context.Context, token string) error {
	if token != "" {
		r.lSemsM.Lock()

		e, ok := r.lSems[token]
		if !ok {
			e = &entry{sem: semaphore.NewWeighted(int64(r.tokenCapacity))}
			r.lSems[token] = e
		}
		r.lSemsM.Unlock()

		if err := e.sem.Acquire(ctx, 1); err != nil {
			return err
		}
		e.count.Add(1)
	}

	return nil
}

// releaseL releases local semaphore for given token.
func (r *Runner) releaseL(token string) {
	if token != "" {
		r.lSemsM.Lock()

		if e, ok := r.lSems[token]; ok {
			e.sem.Release(1)
			if v := e.count.Add(-1); v == 0 {
				delete(r.lSems, token)
			}
		}
		r.lSemsM.Unlock()
	}
}

// lSemsLen returns number of local semaphores in use.
func (r *Runner) lSemsLen() int {
	r.lSemsM.Lock()
	defer r.lSemsM.Unlock()
	return len(r.lSems)
}

// Run starts jobs execution loop. It reads jobs from the channel and starts them in separate goroutines.
func (r *Runner) Run(ctx context.Context) {
	for {
		select {
		case action := <-r.actions:
			r.handleAction(ctx, action)
		case job := <-r.jobs:
			r.handleJob(ctx, job)
		case <-ctx.Done():
			r.wg.Wait() // wait for all actions and jobs termination
			close(r.actionsMessages)
			close(r.jobsMessages)
			return
		}
	}
}

// StartAction starts given actions.Action.
func (r *Runner) StartAction(action actions.Action) error {
	select {
	case r.actions <- action:
		return nil
	default:
		return agenterrors.ErrActionQueueOverflow
	}
}

// StartJob starts given jobs.Job.
func (r *Runner) StartJob(job jobs.Job) error {
	select {
	case r.jobs <- job:
		return nil
	default:
		return errors.New("jobs queue overflowed")
	}
}

// JobsMessages returns channel with Jobs messages.
func (r *Runner) JobsMessages() <-chan agentpb.AgentResponsePayload {
	return r.jobsMessages
}

// ActionsResults return chanel with Actions results payload.
func (r *Runner) ActionsResults() <-chan agentpb.AgentRequestPayload {
	return r.actionsMessages
}

// Stop stops running Action or Job.
func (r *Runner) Stop(id string) {
	r.cancelsM.RLock()
	defer r.cancelsM.RUnlock()

	// Job removes itself from cancels map. So here we only invoke cancel.
	if cancel, ok := r.cancels[id]; ok {
		cancel()
	}
}

// IsRunning returns true if Action or Job with given ID still running.
func (r *Runner) IsRunning(id string) bool {
	r.runningM.RLock()
	defer r.runningM.RUnlock()
	_, ok := r.running[id]

	return ok
}

// createTokenFromDSN returns unique database instance id (token) calculated as a hash from host:port part of the DSN.
func createTokenFromDSN(dsn string) (string, error) {
	if dsn == "" {
		return "", nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse DSN")
	}

	host := u.Host
	// If host is empty, use the whole DSN for hash calculation.
	// It can give worse granularity, but it's better than nothing.
	if host == "" {
		host = dsn
	}

	h := sha256.New()
	h.Write([]byte(host))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (r *Runner) handleJob(ctx context.Context, job jobs.Job) {
	jobID, jobType := job.ID(), job.Type()
	l := r.l.WithFields(logrus.Fields{"id": jobID, "type": jobType})

	token, err := createTokenFromDSN(job.DSN())
	if err != nil {
		r.l.Warnf("Failed to get token for job: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	r.addCancel(jobID, cancel)

	r.wg.Add(1)
	run := func(ctx context.Context) {
		defer func(start time.Time) {
			l.WithField("duration", time.Since(start).String()).Info("Job finished.")
		}(time.Now())

		defer r.wg.Done()
		defer cancel()
		defer r.removeCancel(jobID)

		l.Debug("Acquiring tokens for a job.")
		if err := r.acquire(ctx, token); err != nil {
			l.Errorf("Failed to acquire token for a job: %v", err)
			r.sendJobsMessage(&agentpb.JobResult{
				JobId:     job.ID(),
				Timestamp: timestamppb.Now(),
				Result: &agentpb.JobResult_Error_{
					Error: &agentpb.JobResult_Error{
						Message: err.Error(),
					},
				},
			})
			return
		}
		defer r.release(token)

		var nCtx context.Context
		var nCancel context.CancelFunc
		if timeout := job.Timeout(); timeout != 0 {
			nCtx, nCancel = context.WithTimeout(ctx, timeout)
			defer nCancel()
		} else {
			// If timeout is not provided then use parent context
			nCtx = ctx
		}

		// Mark job as running.
		r.addStarted(jobID)
		defer r.removeStarted(jobID)
		l.Info("Job started.")

		err := job.Run(nCtx, r.sendJobsMessage)
		if err != nil {
			r.sendJobsMessage(&agentpb.JobResult{
				JobId:     job.ID(),
				Timestamp: timestamppb.Now(),
				Result: &agentpb.JobResult_Error_{
					Error: &agentpb.JobResult_Error{
						Message: err.Error(),
					},
				},
			})
			l.Warnf("Job terminated with error: %+v", err)
		}
	}

	go pprof.Do(ctx, pprof.Labels("jobID", jobID, "type", string(jobType)), run)
}

func (r *Runner) handleAction(ctx context.Context, action actions.Action) {
	actionID, actionType := action.ID(), action.Type()
	l := r.l.WithFields(logrus.Fields{"id": actionID, "type": actionType})

	instanceID, err := createTokenFromDSN(action.DSN())
	if err != nil {
		r.l.Warnf("Failed to get instance ID for action: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	r.addCancel(actionID, cancel)

	r.wg.Add(1)
	run := func(ctx context.Context) {
		defer func(start time.Time) {
			l.WithField("duration", time.Since(start).String()).Info("Action finished.")
		}(time.Now())

		defer r.wg.Done()
		defer cancel()
		defer r.removeCancel(actionID)

		l.Debug("Acquiring tokens for an action.")
		if err := r.acquire(ctx, instanceID); err != nil {
			l.Errorf("Failed to acquire token for an action: %v", err)
			r.sendActionsMessage(&agentpb.ActionResultRequest{
				ActionId: actionID,
				Done:     true,
				Error:    err.Error(),
			})
			return
		}
		defer r.release(instanceID)

		var timeout time.Duration
		if timeout = action.Timeout(); timeout == 0 {
			timeout = defaultActionTimeout
		}

		nCtx, nCancel := context.WithTimeout(ctx, timeout)
		defer nCancel()

		// Mark action as running.
		r.addStarted(actionID)
		defer r.removeStarted(actionID)
		l.Infof("Action started.")

		output, err := action.Run(nCtx)
		var errMsg string
		if err != nil {
			errMsg = err.Error()
			l.Warnf("Action terminated with error: %+v", err)
			l.Debugf("Action produced output: %s", string(output))
		}
		r.sendActionsMessage(&agentpb.ActionResultRequest{
			ActionId: actionID,
			Done:     true,
			Output:   output,
			Error:    errMsg,
		})
	}
	go pprof.Do(ctx, pprof.Labels("actionID", actionID, "type", actionType), run)
}

func (r *Runner) sendJobsMessage(payload agentpb.AgentResponsePayload) {
	r.jobsMessages <- payload
}

func (r *Runner) sendActionsMessage(payload agentpb.AgentRequestPayload) {
	r.actionsMessages <- payload
}

func (r *Runner) addCancel(jobID string, cancel context.CancelFunc) {
	r.cancelsM.Lock()
	defer r.cancelsM.Unlock()
	r.cancels[jobID] = cancel
}

func (r *Runner) removeCancel(jobID string) {
	r.cancelsM.Lock()
	defer r.cancelsM.Unlock()
	delete(r.cancels, jobID)
}

func (r *Runner) addStarted(actionID string) {
	r.runningM.Lock()
	defer r.runningM.Unlock()
	r.running[actionID] = struct{}{}
}

func (r *Runner) removeStarted(actionID string) {
	r.runningM.Lock()
	defer r.runningM.Unlock()
	delete(r.running, actionID)
}
