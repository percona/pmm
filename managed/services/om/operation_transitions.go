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

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// clearOperationFailed resolves the cluster's operation_failed attention when an operation leaves
// failed; the row is kept as a watermark (E0-T1), so a cleared row never blocks the gate again.
func clearOperationFailed(q *reform.Querier, clusterID string) error {
	_, err := q.Exec(
		"SELECT om_attention_clear($1::uuid, 'operation_failed', 'cluster', $1::uuid, '', now())",
		clusterID)
	if err != nil {
		return fmt.Errorf("failed to clear operation_failed attention: %w", err)
	}
	return nil
}

// resumeOperation moves a resumable failed operation back to running (keeping its locks) and clears
// the operation_failed it raised; a different kind_version is refused and the row stays failed.
func resumeOperation(ctx context.Context, db *reform.DB, op *models.OmOperation, seenVersion, kindVersion int32) (bool, error) {
	if op.KindVersion != kindVersion {
		return false, nil
	}
	resumed := false
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		moved, err := advanceStatus(tx.Querier, op.ID, models.OmOperationFailed, models.OmOperationRunning, seenVersion)
		if err != nil {
			return err
		}
		if !moved {
			return nil
		}
		resumed = true
		return clearOperationFailed(tx.Querier, *op.ClusterID)
	})
	return resumed, err
}

// abandonOperation moves a failed operation to abandoned on an admin's explicit resolution, releasing
// its locks and clearing the operation_failed it raised. It reports whether it abandoned.
func abandonOperation(ctx context.Context, db *reform.DB, op *models.OmOperation, seenVersion int32) (bool, error) {
	abandoned := false
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		moved, err := advanceStatus(tx.Querier, op.ID, models.OmOperationFailed, models.OmOperationAbandoned, seenVersion)
		if err != nil {
			return err
		}
		if !moved {
			return nil
		}
		abandoned = true
		err = releaseLocks(tx.Querier, op.ID)
		if err != nil {
			return err
		}
		return clearOperationFailed(tx.Querier, *op.ClusterID)
	})
	return abandoned, err
}
