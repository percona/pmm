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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// factAgentElevation is the uninstall_check fact naming the host's measured elevation capability, which
// the host-step privilege gate reads fail-closed.
const factAgentElevation = "agent.elevation"

// elevationNone is the measured elevation sent when no fresh capability fact is present, so the route's
// privilege gate refuses the destructive step rather than assuming root.
const elevationNone = "none"

// extensionsHostStepParams is the sealed per-member plan om_bootstrap's host-step route composes the
// concrete removals from; the route derives the package, config, keyFile and marker paths itself.
type extensionsHostStepParams struct {
	OwnerRunID string `json:"owner_run_id"`
	OS         string `json:"os"`
	DataPath   string `json:"data_path"`
	// LogPath is omitted when the sealed plan has none, as om_bootstrap rejects an empty path; it then
	// removes only the packaged unit's own log files.
	LogPath string `json:"log_path,omitempty"`
}

// extensionsHostStepRequest dispatches one member's teardown step to om_bootstrap's host-step route; the
// idempotency key is the executor token, so a resume re-sends the same dispatch.
type extensionsHostStepRequest struct {
	RunID            string                   `json:"run_id"`
	Kind             string                   `json:"kind"`
	IdempotencyKey   string                   `json:"idempotency_key"`
	Epoch            int64                    `json:"epoch"`
	CatalogueVersion int32                    `json:"catalogue_version"`
	NodeID           string                   `json:"node_id"`
	StepKind         string                   `json:"step_kind"`
	Target           string                   `json:"target"`
	Params           extensionsHostStepParams `json:"params"`
	Measured         string                   `json:"measured"`
}

// extensionsHostStepResponse is om_bootstrap's 202 reply, with exactly one field set: a submission to
// observe, or a terminal refusal the privilege gate returned before submitting anything.
type extensionsHostStepResponse struct {
	SubmissionID *string          `json:"submission_id"`
	Refusal      *hostStepRefusal `json:"refusal"`
}

// hostStepRefusal is the terminal result the host-step route returns when its privilege gate refuses the
// step before submitting it.
type hostStepRefusal struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

// Host-step observe statuses om_bootstrap reports for a submission; in-flight re-probes and gone is a
// submission the executor no longer knows about.
const (
	hostStepObserveOk       = "ok"
	hostStepObserveFailed   = "failed"
	hostStepObserveBlocked  = "blocked"
	hostStepObserveInFlight = "in_flight"
	hostStepObserveGone     = "gone"
)

// extensionsHostStepObservation is om_bootstrap's reconciled outcome for a submitted host step, read by
// re-probing the submission the dispatch returned.
type extensionsHostStepObservation struct {
	Status        string  `json:"status"`
	Detail        *string `json:"detail"`
	TaskHistoryID *int64  `json:"task_history_id"`
}

// dispatchHostStep dispatches one operations host step to om_bootstrap and returns its reply; the
// dispatch is idempotent on the request's idempotency key.
func (c *bootstrapClient) dispatchHostStep(ctx context.Context, req extensionsHostStepRequest) (*extensionsHostStepResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	resp := &extensionsHostStepResponse{}
	call := inventoryCall{method: http.MethodPost, path: "host-steps", body: req}
	err := c.app.call(ctx, call, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// observeHostStep reads om_bootstrap's reconciled outcome for a submitted host step, so a re-probe reaches
// a terminal result instead of re-submitting.
func (c *bootstrapClient) observeHostStep(ctx context.Context, submissionID string) (*extensionsHostStepObservation, error) {
	ctx, cancel := context.WithTimeout(ctx, bootstrapRequestTimeout)
	defer cancel()

	obs := &extensionsHostStepObservation{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("host-steps", submissionID)}
	err := c.app.call(ctx, call, obs)
	if err != nil {
		return nil, err
	}
	return obs, nil
}

// hostStepRunner is the subset of the om_bootstrap client the uninstall steps use, so a test can fake it.
type hostStepRunner interface {
	dispatchHostStep(ctx context.Context, req extensionsHostStepRequest) (*extensionsHostStepResponse, error)
	observeHostStep(ctx context.Context, submissionID string) (*extensionsHostStepObservation, error)
}

// hostStepRef is stored as a host step's result so a re-probe reaches the same submission, carrying the
// last observed detail.
type hostStepRef struct {
	SubmissionID string `json:"submission_id"`
	Detail       string `json:"detail,omitempty"`
}

// serviceForgetter drops one service from om_inventory, so install no longer sees it on its host.
type serviceForgetter interface {
	forgetService(ctx context.Context, serviceID string) error
}

// inventoryForgetter forgets a service through om_inventory's services route.
type inventoryForgetter struct {
	app extensionsApp
}

// forgetService deletes the service row. A service om_inventory no longer holds is already forgotten, so a
// not-found reply is success; that keeps a resumed forget step idempotent.
func (f inventoryForgetter) forgetService(ctx context.Context, serviceID string) error {
	err := f.app.call(ctx, inventoryCall{method: http.MethodDelete, path: inventoryPath("services", serviceID)}, nil)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}

// uninstallStepExecutor dispatches a member's teardown host step to om_bootstrap's host-step route and
// maps its reply; the dispatch is idempotent on the step's key. The forget step runs no host step: it drops
// the member's service from om_inventory instead.
type uninstallStepExecutor struct {
	db        *reform.DB
	bootstrap hostStepRunner
	forgetter serviceForgetter
	reader    factObservationReader
	op        *models.OmOperation
	l         *logrus.Entry
}

// Run submits the member's host step on the first probe and observes its submission on each later one: the
// submit returns in-flight so the driver re-probes, and the observe reaches a terminal outcome.
func (e *uninstallStepExecutor) Run(ctx context.Context, plan StepPlan, idempotencyKey string) (StepResultKind, []byte, error) {
	ctx = asService(ctx)
	params := uninstallParamsOf(e.op)
	member, ok := uninstallMemberByID(params, plan.TargetID)
	if !ok {
		return "", nil, fmt.Errorf("uninstall %s: no sealed plan for member %s", e.op.ID, plan.TargetID)
	}
	if plan.Step == uninstallForgetStep {
		return e.forget(ctx, member.MemberID)
	}
	step, err := e.dispatchStep(idempotencyKey)
	if err != nil {
		return "", nil, err
	}
	submissionID, err := storedSubmissionID(step)
	if err != nil {
		return "", nil, err
	}
	if submissionID != "" {
		return e.observe(ctx, submissionID)
	}
	measured, err := e.measuredElevation(ctx, member.MemberID)
	if err != nil {
		return "", nil, err
	}
	resp, err := e.bootstrap.dispatchHostStep(ctx, extensionsHostStepRequest{
		RunID:            e.op.ID,
		Kind:             e.op.Kind,
		IdempotencyKey:   idempotencyKey,
		Epoch:            step.Epoch,
		CatalogueVersion: e.op.KindVersion,
		NodeID:           member.NodeID,
		StepKind:         plan.Step,
		Target:           member.Host,
		Params: extensionsHostStepParams{
			OwnerRunID: params.InstallRunID,
			OS:         params.OS,
			DataPath:   member.DataPath,
			LogPath:    member.LogPath,
		},
		Measured: measured,
	})
	if err != nil {
		return "", nil, err
	}
	return dispatchResult(resp)
}

// forget drops the member's service from om_inventory. The service link is still on the member row here:
// the success hook clears it only after every member's steps have run. A member with no service link has
// nothing to forget.
func (e *uninstallStepExecutor) forget(ctx context.Context, memberID string) (StepResultKind, []byte, error) {
	row, err := e.db.FindByPrimaryKeyFrom(models.OmClusterMemberTable, memberID)
	if err != nil {
		return "", nil, fmt.Errorf("uninstall %s: failed to read member %s: %w", e.op.ID, memberID, err)
	}
	serviceID := row.(*models.OmClusterMember).ServiceID //nolint:forcetypeassert
	if serviceID == nil || *serviceID == "" {
		return StepResultOk, nil, nil
	}
	err = e.forgetter.forgetService(ctx, *serviceID)
	if err != nil {
		return "", nil, fmt.Errorf("uninstall %s: failed to forget service %s: %w", e.op.ID, *serviceID, err)
	}
	return StepResultOk, nil, nil
}

// observe reads the submitted step's reconciled outcome and maps it: ok succeeds, failed or a gone
// submission fails, blocked blocks, and in-flight re-probes.
func (e *uninstallStepExecutor) observe(ctx context.Context, submissionID string) (StepResultKind, []byte, error) {
	obs, err := e.bootstrap.observeHostStep(ctx, submissionID)
	if err != nil {
		return "", nil, err
	}
	detail := ""
	if obs.Detail != nil {
		detail = *obs.Detail
	}
	ref, err := json.Marshal(hostStepRef{SubmissionID: submissionID, Detail: detail})
	if err != nil {
		return "", nil, fmt.Errorf("failed to encode the host step reference: %w", err)
	}
	switch obs.Status {
	case hostStepObserveOk:
		return StepResultOk, ref, nil
	case hostStepObserveBlocked:
		return StepResultBlocked, ref, nil
	case hostStepObserveInFlight:
		return StepResultTimedOut, ref, nil
	default:
		return StepResultFailed, ref, nil
	}
}

// dispatchStep reads this dispatch's step, holding the leader epoch om_bootstrap fences on and the
// submission a prior probe stored; the driver commits it before the executor runs, so it is present.
func (e *uninstallStepExecutor) dispatchStep(idempotencyKey string) (*models.OmOperationStep, error) {
	row, err := e.db.SelectOneFrom(models.OmOperationStepTable, "WHERE idempotency_key = $1", idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("uninstall %s: failed to read the dispatch step: %w", e.op.ID, err)
	}
	return row.(*models.OmOperationStep), nil //nolint:forcetypeassert
}

// storedSubmissionID returns the submission a prior probe recorded on the step, or an empty string when
// the step has not been submitted yet.
func storedSubmissionID(step *models.OmOperationStep) (string, error) {
	if len(step.Result) == 0 {
		return "", nil
	}
	ref := hostStepRef{}
	err := json.Unmarshal(step.Result, &ref)
	if err != nil {
		return "", fmt.Errorf("failed to read the host step reference: %w", err)
	}
	return ref.SubmissionID, nil
}

// measuredElevation reads the member's current uninstall_check elevation capability, returning none when
// no fresh fact is present so the route's privilege gate refuses rather than assumes root.
func (e *uninstallStepExecutor) measuredElevation(ctx context.Context, memberID string) (string, error) {
	facts, _, err := e.reader.Observation(ctx, subjectKindMember, memberID, e.op.Kind+"_check")
	if err != nil {
		return "", fmt.Errorf("uninstall %s: failed to read the elevation of member %s: %w", e.op.ID, memberID, err)
	}
	elevation, ok := facts[factAgentElevation].(string)
	if !ok || elevation == "" {
		return elevationNone, nil
	}
	return elevation, nil
}

// dispatchResult maps the submit reply: a refusal is terminal, and an accepted submission is kept as the
// step reference so the next probe observes it instead of re-submitting.
func dispatchResult(resp *extensionsHostStepResponse) (StepResultKind, []byte, error) {
	if resp.Refusal != nil {
		return refusalResult(resp.Refusal.Kind), nil, nil
	}
	submissionID := ""
	if resp.SubmissionID != nil {
		submissionID = *resp.SubmissionID
	}
	ref, err := json.Marshal(hostStepRef{SubmissionID: submissionID})
	if err != nil {
		return "", nil, fmt.Errorf("failed to encode the host step reference: %w", err)
	}
	return StepResultTimedOut, ref, nil
}

// refusalResult maps the route's refusal to a terminal step result: a blocked refusal blocks, any other
// is a failure, so a refusal never re-probes.
func refusalResult(kind string) StepResultKind {
	if StepResultKind(kind) == StepResultBlocked {
		return StepResultBlocked
	}
	return StepResultFailed
}

// uninstallMemberByID finds a sealed member's removal plan by its member id.
func uninstallMemberByID(params uninstallParams, memberID string) (uninstallMember, bool) {
	for _, member := range params.Members {
		if member.MemberID == memberID {
			return member, true
		}
	}
	return uninstallMember{}, false
}
