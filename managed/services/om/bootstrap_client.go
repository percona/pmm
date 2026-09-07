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

package om

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// This file is the client half of the bootstrap proxy: the shapes SEP's om_bootstrap
// app serves, and PMM's stepper's own read/write access to them (stepper.go). It
// reuses inventory_client.go's generic sepApp.call/inventoryCall/sepStatusError --
// see sep_client.go's own doc comment on why a second SEP-backed source gets a
// client by calling app() again on the one sepClient the Service already holds,
// rather than by copying the transport.
//
// Unlike probeSource, bootstrapClient does not implement factSource: it plays no
// part in the topology-collection pipeline sources.go/facts.go drive. It exists
// purely for the HA-leader-only stepper (PMM-15347/plan.md §4 item 9) to discover
// and drive bootstrap runs.

// bootstrapAppModule is where SEP mounts the bootstrap app.
const bootstrapAppModule = "om_bootstrap"

// bootstrapRequestTimeout bounds one call to the bootstrap app.
//
// Every dispatch call here is fire-and-forget on SEP's side -- it returns as soon
// as the Tasks API accepts the Nomad dispatch, not once the step finishes -- so
// nothing behind this endpoint should take seconds either.
const bootstrapRequestTimeout = 15 * time.Second

// sepBootstrapStep is one step's progress -- a host's, a rollback's, or a run's own
// run-level step -- matching om_bootstrap's StepRecord exactly (one shape serves all
// three there too; see its own doc comment).
type sepBootstrapStep struct {
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	Detail        *string    `json:"detail"`
	TaskHistoryID *int64     `json:"task_history_id"`
	AttemptCount  int        `json:"attempt_count"`
}

// sepBootstrapHost is one host's progress within a run.
type sepBootstrapHost struct {
	Host          string             `json:"host"`
	Steps         []sepBootstrapStep `json:"steps"`
	RollbackSteps []sepBootstrapStep `json:"rollback_steps"`
	FinalizeSteps []sepBootstrapStep `json:"finalize_steps"`
}

// sepBootstrapRun is one row of GET /runs, and the full body of GET /runs/{id}
// and every dispatch/finish response.
type sepBootstrapRun struct {
	ID             string             `json:"id"`
	Status         string             `json:"status"`
	InstallMethod  string             `json:"install_method"`
	OS             string             `json:"os"`
	MongoDBVersion string             `json:"mongodb_version"`
	ReplicaSetName string             `json:"replica_set_name"`
	StartedAt      time.Time          `json:"started_at"`
	FinishedAt     *time.Time         `json:"finished_at"`
	Hosts          []sepBootstrapHost `json:"hosts"`
	RunSteps       []sepBootstrapStep `json:"run_steps"`
	Error          *string            `json:"error"`
}

// sepTriggerBootstrapRunRequest is the body POST /runs takes.
type sepTriggerBootstrapRunRequest struct {
	Hosts          []string `json:"hosts"`
	InstallMethod  string   `json:"install_method"`
	OS             string   `json:"os"`
	MongoDBVersion string   `json:"mongodb_version"`
	ReplicaSetName string   `json:"replica_set_name"`
}

// sepDispatchStepRequest is the optional body every :dispatch route takes -- the
// per-dispatch secrets a step needs but om_bootstrap never persists (a keyFile's
// content, the generated MongoDB user's credentials). See stepper.go's own
// generation/persistence of these values.
type sepDispatchStepRequest struct {
	Params map[string]string `json:"params,omitempty"`
}

// sepFinishBootstrapRunRequest is the body POST /runs/{id}:finish takes -- the
// stepper's own decision that a run is done, one way or the other.
type sepFinishBootstrapRunRequest struct {
	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`
}

// Bootstrap run statuses, matching om_bootstrap's BootstrapRunStatus values
// exactly -- SEP persists these by name, not by an integer, so the two sides
// agree on the literal strings rather than an ordinal.
const (
	bootstrapRunRunning    = "running"
	bootstrapRunSucceeded  = "succeeded"
	bootstrapRunFailed     = "failed"
	bootstrapRunRolledBack = "rolled_back"
)

// Step statuses, matching om_bootstrap's StepStatus values.
const (
	bootstrapStepPending   = "pending"
	bootstrapStepRunning   = "running"
	bootstrapStepSucceeded = "succeeded"
	bootstrapStepFailed    = "failed"
	bootstrapStepSkipped   = "skipped"
)

// bootstrapClient is PMM's typed handle onto SEP's om_bootstrap app.
type bootstrapClient struct {
	app sepApp
}

// listRuns returns every run in the given status, newest first. An empty status
// returns runs regardless of status.
func (c *bootstrapClient) listRuns(ctx context.Context, runStatus string) ([]sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	query := url.Values{}
	if runStatus != "" {
		query.Set("status", runStatus)
	}
	runs := []sepBootstrapRun{}
	call := inventoryCall{method: http.MethodGet, path: "runs", query: query}
	err := c.app.call(ctx, call, &runs)
	if err != nil {
		return nil, err
	}
	return runs, nil
}

// getRun returns one run, reconciled against its in-flight dispatches as of this
// call -- see om_bootstrap's own GET /runs/{id} doc comment.
func (c *bootstrapClient) getRun(ctx context.Context, runID string) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("runs", runID)}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// triggerRun creates a bootstrap run, planning every host's steps up front.
func (c *bootstrapClient) triggerRun(ctx context.Context, req sepTriggerBootstrapRunRequest) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{method: http.MethodPost, path: "runs", body: req}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// dispatchStep dispatches one host's named step, optionally carrying params.
func (c *bootstrapClient) dispatchStep(ctx context.Context, runID, host, stepName string, params map[string]string) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "hosts", host, "steps", stepName+":dispatch"),
		body:   sepDispatchStepRequest{Params: params},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// dispatchFinalizeStep dispatches one host's named finalize step, optionally
// carrying params -- the same shape as dispatchStep, just against
// om_bootstrap's finalize route rather than its forward-step one.
func (c *bootstrapClient) dispatchFinalizeStep(ctx context.Context, runID, host, stepName string, params map[string]string) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "hosts", host, "finalize", stepName+":dispatch"),
		body:   sepDispatchStepRequest{Params: params},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// dispatchRunStep dispatches one run-level step, targeting the run's seed host.
func (c *bootstrapClient) dispatchRunStep(ctx context.Context, runID, stepName string, params map[string]string) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "run-steps", stepName+":dispatch"),
		body:   sepDispatchStepRequest{Params: params},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// dispatchRollbackStep dispatches one host's named rollback step. Rollback steps
// take no params -- see om_bootstrap's own dispatch_rollback_step doc comment.
func (c *bootstrapClient) dispatchRollbackStep(ctx context.Context, runID, host, stepName string) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "hosts", host, "rollback", stepName+":dispatch"),
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// finishRun records the stepper's own decision that a run is done -- failed, with
// retries exhausted, or rolled back. runStatus must be bootstrapRunFailed or
// bootstrapRunRolledBack; SUCCEEDED is never requested here, since om_bootstrap
// infers it on its own the moment every step actually succeeds (see
// reconcile.py's module docstring).
func (c *bootstrapClient) finishRun(ctx context.Context, runID, runStatus string, detail *string) (*sepBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &sepBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID) + ":finish",
		body:   sepFinishBootstrapRunRequest{Status: runStatus, Error: detail},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}
