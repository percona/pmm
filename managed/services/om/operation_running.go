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
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// StepPlan is one planned dispatch in a kind's catalogue; the real catalogue and on-host execution
// are E0-T3, and a fake executor stands in for tests.
type StepPlan struct {
	Phase      string
	Step       string
	TargetKind models.OmStepTargetKind
	TargetID   string
}

// idempotencyKey is the canonical encoding of (operation, target_kind, target_id, phase, step,
// attempt) that keys a dispatch; a recovery re-sends the same key rather than opening attempt+1.
func (p StepPlan) idempotencyKey(operationID string, attempt int32) string {
	return strings.Join([]string{operationID, string(p.TargetKind), p.TargetID, p.Phase, p.Step, strconv.Itoa(int(attempt))}, "/")
}

// StepExecutor runs a planned step and returns its typed result; E0-T3 provides the real one.
type StepExecutor interface {
	Run(ctx context.Context, plan StepPlan, idempotencyKey string) (StepResultKind, []byte, error)
}

// RunOutcome is how one running-phase tick ended.
type RunOutcome string

// Running-phase outcomes.
const (
	RunStepSucceeded      RunOutcome = "step_succeeded"
	RunStepPending        RunOutcome = "step_pending"
	RunStepFailed         RunOutcome = "step_failed"
	RunFenced             RunOutcome = "fenced"
	RunOperationSucceeded RunOutcome = "operation_succeeded"
)

// driveRunning advances a running operation by one tick: it dispatches the next not-yet-succeeded
// step through the fence and records its result, or finalises the operation when every step is done.
func driveRunning(ctx context.Context, db *reform.DB, operationID string, kind OperationKind, executor StepExecutor, cachedEpoch int64) (RunOutcome, error) {
	op, steps, err := getOperation(db.Querier, operationID)
	if err != nil {
		return "", err
	}
	done := succeededSteps(steps)
	pending := nonTerminalStepByPlan(steps)
	for _, plan := range kind.Steps(op) {
		key := plan.Phase + "/" + plan.Step
		if done[key] {
			continue
		}
		if step := pending[key]; step != nil {
			return reprobeStep(ctx, db, op, plan, step, executor)
		}
		return driveStep(ctx, db, op, plan, executor, cachedEpoch)
	}
	err = finalizeSucceeded(ctx, db, op, kind)
	if err != nil {
		return "", err
	}
	return RunOperationSucceeded, nil
}

// succeededSteps indexes the already-succeeded steps by phase and step name.
func succeededSteps(steps []*models.OmOperationStep) map[string]bool {
	done := make(map[string]bool, len(steps))
	for _, s := range steps {
		if s.Status == models.OmStepSucceeded {
			done[s.Phase+"/"+s.Step] = true
		}
	}
	return done
}

// nonTerminalStepByPlan indexes the in-flight steps (dispatched, running or unknown) by phase and step
// name, so the driver re-probes an already-dispatched step rather than opening a second one.
func nonTerminalStepByPlan(steps []*models.OmOperationStep) map[string]*models.OmOperationStep {
	pending := make(map[string]*models.OmOperationStep)
	for _, s := range steps {
		if s.Status == models.OmStepSucceeded || s.Status == models.OmStepFailed {
			continue
		}
		pending[s.Phase+"/"+s.Step] = s
	}
	return pending
}

// reprobeStep re-runs an in-flight step's executor under its own idempotency key and resolves it: an
// inconclusive result leaves it pending for the next tick, Ok succeeds it, and a failure fails the run.
func reprobeStep(ctx context.Context, db *reform.DB, op *models.OmOperation, plan StepPlan, step *models.OmOperationStep, executor StepExecutor) (RunOutcome, error) {
	result, resultJSON, err := executor.Run(ctx, plan, step.IdempotencyKey)
	if err != nil {
		return "", err
	}
	if mapStepResult(result) == models.OmStepUnknown {
		// Persist the run reference the executor just recorded so the next probe reuses the run
		// instead of opening a second; an already-unknown step already carries it.
		if step.Status != models.OmStepUnknown {
			err = db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
				_, e := recordStepResult(tx.Querier, step.ID, step.Status, result, resultJSON)
				return e
			})
			if err != nil {
				return "", err
			}
		}
		return RunStepPending, nil
	}
	moved := false
	err = db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var resolveErr error
		if step.Status == models.OmStepUnknown {
			moved, resolveErr = resolveUnknownStep(tx.Querier, step.ID, mapStepResult(result) == models.OmStepSucceeded)
			return resolveErr
		}
		moved, resolveErr = recordStepResult(tx.Querier, step.ID, step.Status, result, resultJSON)
		return resolveErr
	})
	if err != nil {
		return "", err
	}
	if !moved {
		return RunStepPending, nil
	}
	if mapStepResult(result) == models.OmStepSucceeded {
		return RunStepSucceeded, nil
	}
	return failRunningOperation(ctx, db, op.ID, *op.ClusterID, string(result))
}

// driveStep dispatches one step under the fence, runs it through the executor outside the transaction
// (a PL/pgSQL commit cannot self-commit before the executor call), and records the mapped result.
func driveStep(ctx context.Context, db *reform.DB, op *models.OmOperation, plan StepPlan, executor StepExecutor, cachedEpoch int64) (RunOutcome, error) {
	key := plan.idempotencyKey(op.ID, 1)
	step := &models.OmOperationStep{
		ID: uuid.New().String(), OperationID: op.ID, TargetKind: plan.TargetKind, TargetID: plan.TargetID,
		Phase: plan.Phase, Step: plan.Step, Attempt: 1, IdempotencyKey: key, Status: models.OmStepDispatching,
	}
	dispatched := false
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		ok, err := fencedDispatch(tx, op.ID, op.Version, cachedEpoch, step)
		dispatched = ok
		return err
	})
	if err != nil {
		return "", err
	}
	if !dispatched {
		return RunFenced, nil
	}
	result, resultJSON, err := executor.Run(ctx, plan, key)
	if err != nil {
		return "", err
	}
	_, err = recordStepResult(db.Querier, step.ID, models.OmStepDispatching, result, resultJSON)
	if err != nil {
		return "", err
	}
	switch mapStepResult(result) {
	case models.OmStepSucceeded:
		return RunStepSucceeded, nil
	case models.OmStepUnknown:
		return RunStepPending, nil
	default:
		return failRunningOperation(ctx, db, op.ID, *op.ClusterID, string(result))
	}
}

// finalizeSucceeded moves a running operation to succeeded, runs its onSucceeded hook, deletes the
// administrator secret and releases the locks, all in one transaction.
func finalizeSucceeded(ctx context.Context, db *reform.DB, op *models.OmOperation, kind OperationKind) error {
	return db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		moved, err := advanceStatus(tx.Querier, op.ID, models.OmOperationRunning, models.OmOperationSucceeded, op.Version)
		if err != nil {
			return err
		}
		if !moved {
			return nil
		}
		if kind.OnSucceeded != nil {
			err = kind.OnSucceeded(tx, op)
			if err != nil {
				return err
			}
		}
		err = deleteAdministratorSecret(tx.Querier, op.ID)
		if err != nil {
			return err
		}
		err = releaseLocks(tx.Querier, op.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE om_operations SET finished_at = now() WHERE id = $1", op.ID)
		return err
	})
}

// failRunningOperation moves a running operation to failed and raises operation_failed on its cluster,
// so the step's failure both stops the run and surfaces to the shared attention gate.
func failRunningOperation(ctx context.Context, db *reform.DB, operationID, clusterID, reason string) (RunOutcome, error) {
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		res, err := tx.Exec(
			"UPDATE om_operations SET status = 'failed', error_code = $1, version = version + 1 WHERE id = $2 AND status = 'running'",
			reason, operationID)
		if err != nil {
			return err
		}
		moved, err := res.RowsAffected()
		if err != nil || moved == 0 {
			return err
		}
		var raised bool
		return tx.QueryRow(
			"SELECT om_attention_upsert($1::uuid, 'operation_failed', 'cluster', $1::uuid, '', '{}'::jsonb, '{}'::jsonb, now())",
			clusterID).Scan(&raised)
	})
	if err != nil {
		return "", fmt.Errorf("failed to fail running operation: %w", err)
	}
	return RunStepFailed, nil
}
