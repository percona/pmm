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
	"database/sql"
	"errors"
	"fmt"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// ConfirmGuard carries the reviewed report digest and plan revision a confirm's opening CAS checks
// alongside the version, so a confirm binds exactly the plan the operator reviewed.
type ConfirmGuard struct {
	ReportDigest   string
	ParamsRevision int32
}

// StartOutcome is how one start-gate attempt ended.
type StartOutcome string

// Start-gate and check-phase outcomes.
const (
	StartedRunning        StartOutcome = "running"
	StartRejected         StartOutcome = "rejected"
	StartStaleInput       StartOutcome = "stale_input"
	StartBlockedAttention StartOutcome = "blocked_attention"
	StartQueued           StartOutcome = "queued"
	CheckFailed           StartOutcome = "failed"
	AwaitingConfirmation  StartOutcome = "awaiting_confirmation"
	CheckRescanRequested  StartOutcome = "rescan_requested"
)

// Recheck runs under the held locks: it re-reads base state and re-runs the kind's blocking rules
// against the pre-transaction facts snapshot, returning whether the reviewed input is now stale.
type Recheck func(tx *reform.TX) (bool, error)

// startGate runs the single start gate for an edge into running: opening version CAS, acquire locks,
// attention gate, under-lock recheck, then set running; or reject / return-to-checking / queue.
func startGate(
	ctx context.Context, db *reform.DB, op *models.OmOperation, from models.OmOperationStatus,
	seenVersion int32, confirm *ConfirmGuard, kind OperationKind, lockSubjects []string, recheck Recheck,
) (StartOutcome, error) {
	if op.ClusterID == nil && !kind.ClusterLess {
		return "", errors.New("start gate needs a cluster-scoped operation")
	}
	outcome := StartRejected
	txErr := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		opened, err := openingCAS(tx, op.ID, from, seenVersion, confirm)
		if err != nil {
			return err
		}
		if !opened {
			outcome = StartRejected
			return nil
		}
		err = acquireLocks(tx.Querier, op.ID, kind.LockSet(op))
		if err != nil {
			return err
		}
		_, blocked, err := checkAttentionGate(tx.Querier, kind.Attention, clusterScope(op), lockSubjects)
		if err != nil {
			return err
		}
		if blocked {
			outcome = StartBlockedAttention
			return returnToChecking(tx, op.ID, false)
		}
		stale, err := recheck(tx)
		if err != nil {
			return err
		}
		if stale {
			outcome = StartStaleInput
			return returnToChecking(tx, op.ID, true)
		}
		outcome = StartedRunning
		return enterRunning(tx, op.ID)
	})
	if txErr != nil {
		_, isConflict := parseLockConflict(txErr)
		if isConflict {
			err := queueAfterConflict(ctx, db, op.ID, from, seenVersion)
			if err != nil {
				return "", err
			}
			return StartQueued, nil
		}
		return "", txErr
	}
	return outcome, nil
}

// openingCAS bumps the version to claim the attempt while keeping the status, so a second writer with
// the same seen version is fenced; it does not set the final status.
func openingCAS(tx *reform.TX, operationID string, from models.OmOperationStatus, seenVersion int32, confirm *ConfirmGuard) (bool, error) {
	var res sql.Result
	var err error
	if confirm == nil {
		res, err = tx.Exec(
			"UPDATE om_operations SET version = version + 1 WHERE id = $1 AND status = $2 AND version = $3",
			operationID, string(from), seenVersion)
	} else {
		res, err = tx.Exec(
			"UPDATE om_operations SET version = version + 1 "+
				"WHERE id = $1 AND status = $2 AND version = $3 AND report_digest = $4 AND params_revision = $5",
			operationID, string(from), seenVersion, confirm.ReportDigest, confirm.ParamsRevision)
	}
	if err != nil {
		return false, fmt.Errorf("opening CAS failed: %w", err)
	}
	opened, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return opened > 0, nil
}

// returnToChecking releases the locks and moves the operation back to checking; invalidate bumps the
// plan revision so a later retry of the reviewed confirm cannot re-apply a stale plan.
func returnToChecking(tx *reform.TX, operationID string, invalidate bool) error {
	err := releaseLocks(tx.Querier, operationID)
	if err != nil {
		return err
	}
	query := "UPDATE om_operations SET status = 'checking' WHERE id = $1"
	if invalidate {
		query = "UPDATE om_operations SET status = 'checking', params_revision = params_revision + 1 WHERE id = $1"
	}
	_, err = tx.Exec(query, operationID)
	if err != nil {
		return fmt.Errorf("failed to return operation to checking: %w", err)
	}
	return nil
}

// enterRunning sets the operation running and stamps its start time; the locks it took are kept.
func enterRunning(tx *reform.TX, operationID string) error {
	_, err := tx.Exec(
		"UPDATE om_operations SET status = 'running', started_at = now() WHERE id = $1",
		operationID)
	if err != nil {
		return fmt.Errorf("failed to enter running: %w", err)
	}
	return nil
}

// queueAfterConflict CASes an operation to queued after a lock conflict aborted its start-gate
// transaction, because the rollback alone only restores the pre-CAS status.
func queueAfterConflict(ctx context.Context, db *reform.DB, operationID string, from models.OmOperationStatus, seenVersion int32) error {
	return db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		_, err := tx.Exec(
			"UPDATE om_operations SET status = 'queued', version = version + 1 WHERE id = $1 AND status = $2 AND version = $3",
			operationID, string(from), seenVersion)
		if err != nil {
			return fmt.Errorf("failed to queue after conflict: %w", err)
		}
		return nil
	})
}
