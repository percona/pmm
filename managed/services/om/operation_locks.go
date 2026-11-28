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
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// lockNotAvailable is the SQLSTATE om_locks_check raises on a conflict.
const lockNotAvailable = "55P03"

// LockResource is one resource an operation locks, with the mode it takes.
type LockResource struct {
	Kind string
	ID   string
	Mode models.OmLockMode
}

// LockConflict is the holder an om_acquire_locks conflict names, parsed from the structured DETAIL.
type LockConflict struct {
	HolderOperationID string
	HolderMode        string
	RequestedMode     string
}

// acquireLocks takes an operation's whole lock set in one sorted pass through om_acquire_locks, so
// the trigger's conflict check runs and two operations never wait in opposite orders.
func acquireLocks(q *reform.Querier, operationID string, resources []LockResource) error {
	if len(resources) == 0 {
		return nil
	}
	args := []any{operationID}
	arrays := make([]string, 0, len(resources))
	for _, r := range resources {
		cols := []string{r.Kind, r.ID, string(r.Mode)}
		placeholders := make([]string, len(cols))
		for j, value := range cols {
			args = append(args, value)
			placeholders[j] = fmt.Sprintf("$%d", len(args))
		}
		arrays = append(arrays, "ARRAY["+strings.Join(placeholders, ",")+"]")
	}
	query := fmt.Sprintf("SELECT om_acquire_locks($1::uuid, ARRAY[%s]::text[][])", strings.Join(arrays, ","))
	_, err := q.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to acquire locks: %w", err)
	}
	return nil
}

// releaseLocks drops an operation's locks; the release matrix decides when the driver calls it.
func releaseLocks(q *reform.Querier, operationID string) error {
	_, err := q.Exec("DELETE FROM om_locks WHERE operation_id = $1", operationID)
	if err != nil {
		return fmt.Errorf("failed to release locks: %w", err)
	}
	return nil
}

// ReleasesLocks reports whether reaching a terminal status releases an operation's locks; a
// resumable failed keeps them for Resume.
func ReleasesLocks(status models.OmOperationStatus, resumable bool) bool {
	switch status {
	case models.OmOperationSucceeded, models.OmOperationCancelled,
		models.OmOperationRolledBack, models.OmOperationAbandoned:
		return true
	case models.OmOperationFailed:
		return !resumable
	default:
		return false
	}
}

// parseLockConflict returns the holder an om_acquire_locks conflict names, or false if the error is
// not a lock conflict.
func parseLockConflict(err error) (LockConflict, bool) {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || string(pqErr.Code) != lockNotAvailable {
		return LockConflict{}, false
	}
	conflict := LockConflict{}
	for part := range strings.SplitSeq(pqErr.Detail, ";") {
		key, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		switch key {
		case "holder_operation_id":
			conflict.HolderOperationID = value
		case "holder_mode":
			conflict.HolderMode = value
		case "requested_mode":
			conflict.RequestedMode = value
		}
	}
	return conflict, true
}
