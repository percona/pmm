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

package models

import (
	"errors"
	"fmt"

	"gopkg.in/reform.v1"
)

// AppendClusterState appends the next accepted-state version and moves the cluster's current
// pointer in one caller-provided transaction, locking the cluster row so concurrent appends serialise.
func AppendClusterState(tx *reform.TX, clusterID string, schemaVersion int32, document []byte, operationID string) (*OmClusterState, error) {
	var current *int32
	err := tx.QueryRow("SELECT state_version FROM om_clusters WHERE id = $1 FOR UPDATE", clusterID).Scan(&current)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to lock cluster for state append: %w", err)
	}

	next := int32(1)
	if current != nil {
		next = *current + 1
	}

	state := &OmClusterState{
		ClusterID:     clusterID,
		Version:       next,
		SchemaVersion: schemaVersion,
		Document:      document,
		OperationID:   operationID,
	}
	err = tx.Insert(state)
	if err != nil {
		return nil, fmt.Errorf("failed to append cluster state: %w", err)
	}

	_, err = tx.Exec("UPDATE om_clusters SET state_version = $1, updated_at = $2 WHERE id = $3", next, Now(), clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to move cluster state pointer: %w", err)
	}

	return state, nil
}
