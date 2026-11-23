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
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// operationPrincipal is the recorded actor for an operation action until caller identity is wired in.
const operationPrincipal = "operator"

// ListOperations returns the managed-cluster operation history, newest first.
func (s *Service) ListOperations(_ context.Context, req *omv1.ListOperationsRequest) (*omv1.ListOperationsResponse, error) {
	ops, err := listOperations(s.db.Querier, req.ClusterId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*omv1.Operation, len(ops))
	for i := range ops {
		out[i] = operationToProto(ops[i])
	}
	return &omv1.ListOperationsResponse{Operations: out}, nil
}

// GetOperation returns one operation and its dispatched steps.
func (s *Service) GetOperation(_ context.Context, req *omv1.GetOperationRequest) (*omv1.GetOperationResponse, error) {
	op, steps, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	out := make([]*omv1.OperationStep, len(steps))
	for i := range steps {
		out[i] = operationStepToProto(steps[i])
	}
	return &omv1.GetOperationResponse{Operation: operationToProto(op), Steps: out}, nil
}

// ConfirmOperation releases an operation awaiting confirmation to run, fencing on the reviewed version.
func (s *Service) ConfirmOperation(ctx context.Context, req *omv1.ConfirmOperationRequest) (*omv1.ConfirmOperationResponse, error) {
	op, _, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	if op.Status != models.OmOperationAwaitingConfirmation {
		return nil, status.Errorf(codes.FailedPrecondition, "operation is not awaiting confirmation (status '%s')", op.Status)
	}
	if op.ReportDigest == nil {
		return nil, status.Error(codes.FailedPrecondition, "operation has no sealed report to confirm")
	}
	kind, err := s.kindFor(op)
	if err != nil {
		return nil, err
	}
	guard := &ConfirmGuard{ReportDigest: *op.ReportDigest, ParamsRevision: op.ParamsRevision}
	recheck := func(*reform.TX) (bool, error) { return false, nil }
	outcome, err := startGate(ctx, s.db, op, models.OmOperationAwaitingConfirmation, req.GetVersion(), guard, kind, lockSubjects(kind.LockSet(op)), recheck)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if outcome != StartedRunning {
		return nil, confirmOutcomeError(outcome)
	}
	refreshed, err := s.reloadOperation(op.ID)
	if err != nil {
		return nil, err
	}
	return &omv1.ConfirmOperationResponse{Operation: refreshed}, nil
}

// CancelOperation cancels an operation, or asks a running one of a cancellable kind to compensate.
func (s *Service) CancelOperation(ctx context.Context, req *omv1.CancelOperationRequest) (*omv1.CancelOperationResponse, error) {
	op, _, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	kind, err := s.kindFor(op)
	if err != nil {
		return nil, err
	}
	moved, err := cancelOperation(ctx, s.db, op, req.GetVersion(), operationPrincipal, kind)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !moved {
		return nil, operationNotMoved(op.Status)
	}
	refreshed, err := s.reloadOperation(op.ID)
	if err != nil {
		return nil, err
	}
	return &omv1.CancelOperationResponse{Operation: refreshed}, nil
}

// ResumeOperation returns a failed, resumable operation to running.
func (s *Service) ResumeOperation(ctx context.Context, req *omv1.ResumeOperationRequest) (*omv1.ResumeOperationResponse, error) {
	op, _, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	moved, err := resumeOperation(ctx, s.db, op, req.GetVersion(), op.KindVersion)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !moved {
		return nil, operationNotMoved(op.Status)
	}
	refreshed, err := s.reloadOperation(op.ID)
	if err != nil {
		return nil, err
	}
	return &omv1.ResumeOperationResponse{Operation: refreshed}, nil
}

// AbandonOperation resolves a failed operation, releasing its locks and clearing its attention.
func (s *Service) AbandonOperation(ctx context.Context, req *omv1.AbandonOperationRequest) (*omv1.AbandonOperationResponse, error) {
	op, _, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	moved, err := abandonOperation(ctx, s.db, op, req.GetVersion())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !moved {
		return nil, operationNotMoved(op.Status)
	}
	refreshed, err := s.reloadOperation(op.ID)
	if err != nil {
		return nil, err
	}
	return &omv1.AbandonOperationResponse{Operation: refreshed}, nil
}

// reloadOperation re-reads an operation after an action and maps it to its proto form.
func (s *Service) reloadOperation(operationID string) (*omv1.Operation, error) {
	op, _, err := getOperation(s.db.Querier, operationID)
	if err != nil {
		return nil, operationReadError(err)
	}
	return operationToProto(op), nil
}

// kindFor resolves an operation's registered kind, refusing one no feature has registered.
func (s *Service) kindFor(op *models.OmOperation) (OperationKind, error) {
	kind, ok := s.operationRegistry.Kind(op.Kind)
	if !ok {
		return OperationKind{}, status.Errorf(codes.FailedPrecondition, "operation kind '%s' is not registered", op.Kind)
	}
	return kind, nil
}

// operationReadError maps a not-found operation to NotFound and anything else to Internal.
func operationReadError(err error) error {
	if errors.Is(err, models.ErrNotFound) {
		return status.Error(codes.NotFound, "operation not found")
	}
	return status.Error(codes.Internal, err.Error())
}

// operationNotMoved reports that an action applied to nothing: a wrong state or a stale version.
func operationNotMoved(current models.OmOperationStatus) error {
	return status.Errorf(codes.FailedPrecondition,
		"operation changed concurrently or is not in a state this action applies to (status '%s')", current)
}

// confirmOutcomeError maps a non-running start-gate outcome to a client error.
func confirmOutcomeError(outcome StartOutcome) error {
	switch outcome {
	case StartQueued:
		return status.Error(codes.Aborted, "operation queued behind a conflicting lock")
	case StartBlockedAttention:
		return status.Error(codes.FailedPrecondition, "cluster needs attention; operation returned to checking")
	case StartStaleInput:
		return status.Error(codes.FailedPrecondition, "reviewed plan is stale; operation returned to checking")
	default:
		return status.Error(codes.FailedPrecondition, "confirmation did not match the reviewed plan; it may have changed")
	}
}

// operationToProto maps an operation row to its wire form; secret-bearing params and report are withheld.
func operationToProto(op *models.OmOperation) *omv1.Operation {
	return &omv1.Operation{
		Id:                op.ID,
		Kind:              op.Kind,
		KindVersion:       op.KindVersion,
		ClusterId:         op.ClusterID,
		ParentOperationId: op.ParentOperationID,
		Status:            string(op.Status),
		Version:           op.Version,
		ParamsRevision:    op.ParamsRevision,
		Trigger:           string(op.Trigger),
		ReportDigest:      op.ReportDigest,
		ErrorCode:         op.ErrorCode,
		ConfirmedBy:       op.ConfirmedBy,
		CancelledBy:       op.CancelledBy,
		CreatedAt:         timestamppb.New(op.CreatedAt),
		StartedAt:         optTimestamp(op.StartedAt),
		FinishedAt:        optTimestamp(op.FinishedAt),
		NotBefore:         optTimestamp(op.NotBefore),
		DeadlineAt:        optTimestamp(op.DeadlineAt),
	}
}

// operationStepToProto maps a step row to its wire form; the executor result payload is withheld.
func operationStepToProto(step *models.OmOperationStep) *omv1.OperationStep {
	return &omv1.OperationStep{
		Id:            step.ID,
		TargetKind:    string(step.TargetKind),
		TargetId:      step.TargetID,
		Phase:         step.Phase,
		Step:          step.Step,
		Attempt:       step.Attempt,
		Status:        string(step.Status),
		Epoch:         step.Epoch,
		TaskHistoryId: step.TaskHistoryID,
		DispatchedAt:  optTimestamp(step.DispatchedAt),
		FinishedAt:    optTimestamp(step.FinishedAt),
	}
}

// optTimestamp maps a nullable time to a proto timestamp, leaving an unset time nil.
func optTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
