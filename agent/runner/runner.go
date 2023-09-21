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
	"runtime/pprof"
	"sync"
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
	defaultCapacity      = 32
)

// Runner executes jobs and actions.
type Runner struct {
	l *logrus.Entry

	actions chan actions.Action
	jobs    chan jobs.Job

	actionsMessages chan agentpb.AgentRequestPayload
	jobsMessages    chan agentpb.AgentResponsePayload

	sem *semaphore.Weighted
	wg  sync.WaitGroup

	rw      sync.RWMutex
	rCancel map[string]context.CancelFunc
}

// New creates new runner. If capacity is 0 then default value is used.
func New(capacity uint16) *Runner {
	l := logrus.WithField("component", "runner")
	if capacity == 0 {
		capacity = defaultCapacity
	}

	l.Infof("Runner capacity set to %d.", capacity)

	return &Runner{
		l:               l,
		actions:         make(chan actions.Action, bufferSize),
		jobs:            make(chan jobs.Job, bufferSize),
		sem:             semaphore.NewWeighted(int64(capacity)),
		rCancel:         make(map[string]context.CancelFunc),
		jobsMessages:    make(chan agentpb.AgentResponsePayload),
		actionsMessages: make(chan agentpb.AgentRequestPayload),
	}
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
	r.rw.RLock()
	defer r.rw.RUnlock()

	// Job removes itself from rCancel map. So here we only invoke cancel.
	if cancel, ok := r.rCancel[id]; ok {
		cancel()
	}
}

// IsRunning returns true if Action or Job with given ID still running.
func (r *Runner) IsRunning(id string) bool {
	r.rw.RLock()
	defer r.rw.RUnlock()
	_, ok := r.rCancel[id]

	return ok
}

func (r *Runner) handleJob(ctx context.Context, job jobs.Job) {
	jobID, jobType := job.ID(), job.Type()
	l := r.l.WithFields(logrus.Fields{"id": jobID, "type": jobType})

	if err := r.sem.Acquire(ctx, 1); err != nil {
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

	var nCtx context.Context
	var cancel context.CancelFunc
	if timeout := job.Timeout(); timeout != 0 {
		nCtx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		nCtx, cancel = context.WithCancel(ctx)
	}
	r.addCancel(jobID, cancel)

	r.wg.Add(1)
	run := func(ctx context.Context) {
		l.Infof("Job started.")

		defer func(start time.Time) {
			l.WithField("duration", time.Since(start).String()).Info("Job finished.")
		}(time.Now())

		defer r.sem.Release(1)
		defer r.wg.Done()
		defer cancel()
		defer r.removeCancel(jobID)

		err := job.Run(ctx, r.sendJobsMessage)
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

	go pprof.Do(nCtx, pprof.Labels("jobID", jobID, "type", string(jobType)), run)
}

func (r *Runner) handleAction(ctx context.Context, action actions.Action) {
	actionID, actionType := action.ID(), action.Type()
	l := r.l.WithFields(logrus.Fields{"id": actionID, "type": actionType})

	if err := r.sem.Acquire(ctx, 1); err != nil {
		l.Errorf("Failed to acquire token for an action: %v", err)
		r.sendActionsMessage(&agentpb.ActionResultRequest{
			ActionId: actionID,
			Done:     true,
			Error:    err.Error(),
		})
		return
	}

	var timeout time.Duration
	if timeout = action.Timeout(); timeout == 0 {
		timeout = defaultActionTimeout
	}

	nCtx, cancel := context.WithTimeout(ctx, timeout)
	r.addCancel(actionID, cancel)

	r.wg.Add(1)
	run := func(_ context.Context) {
		l.Infof("Action started.")

		defer func(start time.Time) {
			l.WithField("duration", time.Since(start).String()).Info("Action finished.")
		}(time.Now())

		defer r.sem.Release(1)
		defer r.wg.Done()
		defer cancel()
		defer r.removeCancel(actionID)

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
	go pprof.Do(nCtx, pprof.Labels("actionID", actionID, "type", actionType), run)
}

func (r *Runner) sendJobsMessage(payload agentpb.AgentResponsePayload) {
	r.jobsMessages <- payload
}

func (r *Runner) sendActionsMessage(payload agentpb.AgentRequestPayload) {
	r.actionsMessages <- payload
}

func (r *Runner) addCancel(jobID string, cancel context.CancelFunc) {
	r.rw.Lock()
	defer r.rw.Unlock()
	r.rCancel[jobID] = cancel
}

func (r *Runner) removeCancel(jobID string) {
	r.rw.Lock()
	defer r.rw.Unlock()
	delete(r.rCancel, jobID)
}
