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
	"strconv"
	"time"
)

const bootstrapAppModule = "om_bootstrap"

// bootstrapRequestTimeout is short because dispatch calls return once the Nomad dispatch is accepted.
const bootstrapRequestTimeout = 15 * time.Second

type extensionsBootstrapStep struct {
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	Detail        *string    `json:"detail"`
	TaskHistoryID *int64     `json:"task_history_id"`
	AttemptCount  int        `json:"attempt_count"`
	// A nil Retryable means om_bootstrap did not say, which keeps the retry.
	Retryable *bool `json:"retryable"`
}

type extensionsBootstrapHost struct {
	Host          string                    `json:"host"`
	Steps         []extensionsBootstrapStep `json:"steps"`
	RollbackSteps []extensionsBootstrapStep `json:"rollback_steps"`
	FinalizeSteps []extensionsBootstrapStep `json:"finalize_steps"`
}

type extensionsBootstrapRun struct {
	ID              string                            `json:"id"`
	Status          string                            `json:"status"`
	InstallMethod   string                            `json:"install_method"`
	OS              string                            `json:"os"`
	MongoDBVersion  string                            `json:"mongodb_version"`
	ReplicaSetName  string                            `json:"replica_set_name"`
	DataPath        string                            `json:"data_path"`
	LogPath         string                            `json:"log_path"`
	Port            uint32                            `json:"port"`
	BindIP          string                            `json:"bind_ip"`
	MemberConfigs   map[string]extensionsMemberConfig `json:"member_configs"`
	StartedAt       time.Time                         `json:"started_at"`
	FinishedAt      *time.Time                        `json:"finished_at"`
	Hosts           []extensionsBootstrapHost         `json:"hosts"`
	RunSteps        []extensionsBootstrapStep         `json:"run_steps"`
	Error           *string                           `json:"error"`
	CancelRequested bool                              `json:"cancel_requested"`
}

type extensionsTriggerBootstrapRunRequest struct {
	Hosts           []string                          `json:"hosts"`
	InstallMethod   string                            `json:"install_method"`
	OS              string                            `json:"os"`
	MongoDBVersion  string                            `json:"mongodb_version"`
	ReplicaSetName  string                            `json:"replica_set_name"`
	DataPath        string                            `json:"data_path"`
	LogPath         string                            `json:"log_path"`
	Port            uint32                            `json:"port"`
	BindIP          string                            `json:"bind_ip"`
	MemberConfigs   map[string]extensionsMemberConfig `json:"member_configs,omitempty"`
	CreateAdminUser bool                              `json:"create_admin_user,omitempty"`
}

// extensionsMemberConfig uses pointers so unset fields are omitted and om_bootstrap's defaults apply;
// zero Priority and Votes would ask for a member that can neither vote nor be elected.
type extensionsMemberConfig struct {
	Priority  *uint32 `json:"priority,omitempty"`
	Votes     *bool   `json:"votes,omitempty"`
	Hidden    bool    `json:"hidden"`
	DelaySecs uint32  `json:"delay_secs"`
	BindIP    *string `json:"bind_ip,omitempty"`
}

// extensionsDispatchStepRequest carries per-dispatch secrets that om_bootstrap never persists.
type extensionsDispatchStepRequest struct {
	Params map[string]string `json:"params,omitempty"`
}

type extensionsFinishBootstrapRunRequest struct {
	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`
}

// Bootstrap run statuses, matching om_bootstrap's BootstrapRunStatus values.
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

type bootstrapClient struct {
	app extensionsApp
}

// listRuns returns runs newest first. A zero limit keeps PMM Extensions' default.
func (c *bootstrapClient) listRuns(ctx context.Context, runStatus string, limit int) ([]extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	query := url.Values{}
	if runStatus != "" {
		query.Set("status", runStatus)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	runs := []extensionsBootstrapRun{}
	call := inventoryCall{method: http.MethodGet, path: "runs", query: query}
	err := c.app.call(ctx, call, &runs)
	if err != nil {
		return nil, err
	}
	return runs, nil
}

// getRun returns the run as om_bootstrap reconciles it against its in-flight dispatches.
func (c *bootstrapClient) getRun(ctx context.Context, runID string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("runs", runID)}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (c *bootstrapClient) triggerRun(ctx context.Context, req extensionsTriggerBootstrapRunRequest) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{method: http.MethodPost, path: "runs", body: req}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (c *bootstrapClient) dispatchStep(ctx context.Context, runID, host, stepName string, params map[string]string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "hosts", host, "steps", stepName+":dispatch"),
		body:   extensionsDispatchStepRequest{Params: params},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (c *bootstrapClient) dispatchFinalizeStep(ctx context.Context, runID, host, stepName string, params map[string]string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "hosts", host, "finalize", stepName+":dispatch"),
		body:   extensionsDispatchStepRequest{Params: params},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// dispatchRunStep dispatches a run-level step, which targets the run's seed host.
func (c *bootstrapClient) dispatchRunStep(ctx context.Context, runID, stepName string, params map[string]string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID, "run-steps", stepName+":dispatch"),
		body:   extensionsDispatchStepRequest{Params: params},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (c *bootstrapClient) dispatchRollbackStep(ctx context.Context, runID, host, stepName string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
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

// cancelRun is idempotent; PMM Extensions stops the currently dispatching step on a best-effort basis.
func (c *bootstrapClient) cancelRun(ctx context.Context, runID string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID) + ":cancel",
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// finishRun takes bootstrapRunFailed or bootstrapRunRolledBack; om_bootstrap infers success on its own.
func (c *bootstrapClient) finishRun(ctx context.Context, runID, runStatus string, detail *string) (*extensionsBootstrapRun, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	run := &extensionsBootstrapRun{}
	call := inventoryCall{
		method: http.MethodPost,
		path:   inventoryPath("runs", runID) + ":finish",
		body:   extensionsFinishBootstrapRunRequest{Status: runStatus, Error: detail},
	}
	err := c.app.call(ctx, call, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}
