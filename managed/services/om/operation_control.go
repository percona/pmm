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
	"fmt"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// listOperations returns the operations for a cluster, or all of them when clusterID is nil, newest
// first; the caller's history API surfaces them.
func listOperations(q *reform.Querier, clusterID *string) ([]*models.OmOperation, error) {
	var rows []reform.Struct
	var err error
	if clusterID == nil {
		rows, err = q.SelectAllFrom(models.OmOperationTable, "ORDER BY created_at DESC")
	} else {
		rows, err = q.SelectAllFrom(models.OmOperationTable, "WHERE cluster_id = $1 ORDER BY created_at DESC", *clusterID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	operations := make([]*models.OmOperation, len(rows))
	for i, row := range rows {
		operations[i] = row.(*models.OmOperation) //nolint:forcetypeassert
	}
	return operations, nil
}

// getOperation returns one operation and its steps, newest step dispatch first, or ErrNotFound.
func getOperation(q *reform.Querier, operationID string) (*models.OmOperation, []*models.OmOperationStep, error) {
	record, err := q.FindByPrimaryKeyFrom(models.OmOperationTable, operationID)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil, models.ErrNotFound
		}
		return nil, nil, fmt.Errorf("failed to read operation: %w", err)
	}
	operation := record.(*models.OmOperation) //nolint:forcetypeassert
	rows, err := q.SelectAllFrom(models.OmOperationStepTable, "WHERE operation_id = $1 ORDER BY attempt, step", operationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read operation steps: %w", err)
	}
	steps := make([]*models.OmOperationStep, len(rows))
	for i, row := range rows {
		steps[i] = row.(*models.OmOperationStep) //nolint:forcetypeassert
	}
	return operation, steps, nil
}

// cancelOperation cancels an operation: a running one goes to cancelling (if its kind allows) for
// compensation; a pre-running one goes to cancelled, releasing locks and running the onTerminal hook.
func cancelOperation(ctx context.Context, db *reform.DB, op *models.OmOperation, seenVersion int32, cancelledBy string, kind OperationKind) (bool, error) {
	moved := false
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		if op.Status == models.OmOperationRunning {
			if !kind.CancellableWhileRunning {
				return nil
			}
			advanced, err := advanceStatus(tx.Querier, op.ID, models.OmOperationRunning, models.OmOperationCancelling, seenVersion)
			moved = advanced
			return err
		}
		advanced, err := advanceStatus(tx.Querier, op.ID, op.Status, models.OmOperationCancelled, seenVersion)
		if err != nil {
			return err
		}
		if !advanced {
			return nil
		}
		moved = true
		err = releaseLocks(tx.Querier, op.ID)
		if err != nil {
			return err
		}
		err = deleteAdministratorSecret(tx.Querier, op.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE om_operations SET cancelled_by = $1, finished_at = now() WHERE id = $2", cancelledBy, op.ID)
		if err != nil {
			return fmt.Errorf("failed to record cancellation: %w", err)
		}
		if kind.OnTerminal != nil {
			return kind.OnTerminal(tx, op)
		}
		return nil
	})
	return moved, err
}
