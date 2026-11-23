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

// advanceStatus is the version CAS that fences every transition: it moves the row to `to` only
// while it is still at `from` with the seen version, and reports whether it moved (invariant C6).
func advanceStatus(q *reform.Querier, operationID string, from, to models.OmOperationStatus, seenVersion int32) (bool, error) {
	if !CanTransition(from, to) {
		return false, fmt.Errorf("illegal operation transition %s -> %s", from, to)
	}
	res, err := q.Exec(
		"UPDATE om_operations SET status = $1, version = version + 1 WHERE id = $2 AND status = $3 AND version = $4",
		string(to), operationID, string(from), seenVersion)
	if err != nil {
		return false, fmt.Errorf("failed to advance operation status: %w", err)
	}
	moved, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return moved > 0, nil
}

// fencedDispatch records a dispatching step under the leader-epoch and version fence (C11): it holds
// the lease FOR SHARE and advances only while the epoch and version are current, else writes nothing.
func fencedDispatch(tx *reform.TX, operationID string, seenVersion int32, cachedEpoch int64, step *models.OmOperationStep) (bool, error) {
	var epoch int64
	err := tx.QueryRow("SELECT epoch FROM om_leader_lease WHERE singleton FOR SHARE").Scan(&epoch)
	if err != nil {
		return false, fmt.Errorf("failed to read leader lease: %w", err)
	}
	res, err := tx.Exec(
		"UPDATE om_operations SET version = version + 1 "+
			"WHERE id = $1 AND status = $2 AND version = $3 AND EXISTS (SELECT 1 FROM om_leader_lease WHERE epoch = $4)",
		operationID, string(models.OmOperationRunning), seenVersion, cachedEpoch)
	if err != nil {
		return false, fmt.Errorf("failed to fence dispatch: %w", err)
	}
	fenced, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if fenced == 0 {
		return false, nil
	}
	step.Epoch = cachedEpoch
	err = tx.Insert(step)
	if err != nil {
		return false, fmt.Errorf("failed to record dispatching step: %w", err)
	}
	return true, nil
}
