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
	"fmt"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// StepResultKind is the typed result E0-T3 returns; it carries no status, so the driver maps it onto
// the step status.
type StepResultKind string

// Typed step results from the Extensions run contract (E0-T3).
const (
	StepResultOk       StepResultKind = "ok"
	StepResultBlocked  StepResultKind = "blocked"
	StepResultFailed   StepResultKind = "failed"
	StepResultTimedOut StepResultKind = "timed_out"
	StepResultUnknown  StepResultKind = "unknown"
)

// legalStepTransitions is the step state machine; dispatching or running goes to unknown on a
// vanished task, and the proof probe resolves unknown.
var legalStepTransitions = map[models.OmStepStatus]map[models.OmStepStatus]bool{
	models.OmStepPlanned:     set(models.OmStepDispatching),
	models.OmStepDispatching: set(models.OmStepRunning, models.OmStepSucceeded, models.OmStepFailed, models.OmStepUnknown),
	models.OmStepRunning:     set(models.OmStepSucceeded, models.OmStepFailed, models.OmStepUnknown),
	models.OmStepUnknown:     set(models.OmStepSucceeded, models.OmStepFailed),
}

// canStepTransition reports whether the step state machine allows a move.
func canStepTransition(from, to models.OmStepStatus) bool {
	return legalStepTransitions[from][to]
}

// mapStepResult maps a typed E0-T3 result onto a step status: Ok succeeds, Failed and Blocked fail,
// and the inconclusive results become unknown for the proof probe to resolve.
func mapStepResult(result StepResultKind) models.OmStepStatus {
	switch result {
	case StepResultOk:
		return models.OmStepSucceeded
	case StepResultFailed, StepResultBlocked:
		return models.OmStepFailed
	default:
		return models.OmStepUnknown
	}
}

// recordStepResult moves a dispatched step to its mapped status, guarded on the expected current
// status so a late result for a step that already moved is ignored (reports moved=false).
func recordStepResult(q *reform.Querier, stepID string, from models.OmStepStatus, result StepResultKind, resultJSON []byte) (bool, error) {
	to := mapStepResult(result)
	if !canStepTransition(from, to) {
		return false, fmt.Errorf("illegal step transition %s -> %s", from, to)
	}
	res, err := q.Exec(
		"UPDATE om_operation_steps SET status = $1, result = $2, finished_at = now() WHERE id = $3 AND status = $4",
		string(to), resultJSON, stepID, string(from))
	if err != nil {
		return false, fmt.Errorf("failed to record step result: %w", err)
	}
	moved, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return moved > 0, nil
}

// resolveUnknownStep resolves an unknown step by its proof probe; a true proof succeeds it and a
// false proof fails it, after which attempt+1 is allowed.
func resolveUnknownStep(q *reform.Querier, stepID string, trueProof bool) (bool, error) {
	to := models.OmStepFailed
	if trueProof {
		to = models.OmStepSucceeded
	}
	res, err := q.Exec(
		"UPDATE om_operation_steps SET status = $1, finished_at = now() WHERE id = $2 AND status = $3",
		string(to), stepID, string(models.OmStepUnknown))
	if err != nil {
		return false, fmt.Errorf("failed to resolve unknown step: %w", err)
	}
	moved, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return moved > 0, nil
}

// promotePendingSecret retires the current secret for a pending one's scope and promotes the pending
// to current in one call, so the proving step's success rotates the credential atomically.
func promotePendingSecret(q *reform.Querier, pendingSecretID string) error {
	pending, err := models.FindOmSecretByID(q, pendingSecretID)
	if err != nil {
		return err
	}
	_, err = q.Exec(
		"UPDATE om_secrets SET generation = 'retired', rotated_at = now() "+
			"WHERE generation = 'current' AND purpose = $1 AND cluster_id IS NOT DISTINCT FROM $2 "+
			"AND replica_set_ref IS NOT DISTINCT FROM $3 AND discriminator IS NOT DISTINCT FROM $4",
		pending.Purpose, pending.ClusterID, pending.ReplicaSetRef, pending.Discriminator)
	if err != nil {
		return fmt.Errorf("failed to retire current secret: %w", err)
	}
	_, err = q.Exec(
		"UPDATE om_secrets SET generation = 'current', operation_id = NULL, rotated_at = now() WHERE id = $1",
		pendingSecretID)
	if err != nil {
		return fmt.Errorf("failed to promote pending secret: %w", err)
	}
	return nil
}

// deleteAdministratorSecret removes an operation's administrator secret on a terminal outcome; it is
// kept across a resumable failed, which is not terminal.
func deleteAdministratorSecret(q *reform.Querier, operationID string) error {
	_, err := q.Exec(
		"DELETE FROM om_secrets WHERE operation_id = $1 AND purpose = 'administrator'",
		operationID)
	if err != nil {
		return fmt.Errorf("failed to delete administrator secret: %w", err)
	}
	return nil
}
