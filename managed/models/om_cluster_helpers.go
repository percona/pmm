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
	"encoding/json"
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

// LinkInstalledMemberByHost connects a live, not-yet-linked installed member to the PMM node, service
// and exporter registered for its host; a run that recorded no such member links nothing.
func LinkInstalledMemberByHost(q *reform.Querier, host, nodeID, serviceID, exporterAgentID string) error {
	_, err := q.Exec(
		"UPDATE om_cluster_members SET node_id = $1, service_id = NULLIF($2, ''), exporter_agent_id = NULLIF($3, '') "+
			"WHERE host = $4 AND removed_at IS NULL AND node_id IS NULL",
		nodeID, serviceID, exporterAgentID, host)
	if err != nil {
		return fmt.Errorf("failed to link the installed member on host %s: %w", host, err)
	}
	return nil
}

// InsertInstallMarker records the cluster-scoped kind=marker owned item (on_release=keep) carrying the
// om_bootstrap run id, idempotent on the marker's stable id so a recovered success hook writes it once.
func InsertInstallMarker(tx *reform.TX, markerID, clusterID, operationID, runID string) error {
	spec, err := json.Marshal(map[string]string{"run_id": runID})
	if err != nil {
		return fmt.Errorf("failed to encode the install marker spec: %w", err)
	}
	_, err = tx.Exec(
		"INSERT INTO om_owned_items "+
			"(id, cluster_id, kind, natural_key, action, purpose, state, spec, on_release, created_by_operation, created_at, updated_at) "+
			"VALUES ($1, $2, 'marker', 'install', 'created', 'install', 'present', $3, 'keep', $4, now(), now()) "+
			"ON CONFLICT (id) DO NOTHING",
		markerID, clusterID, spec, operationID)
	if err != nil {
		return fmt.Errorf("failed to insert the install marker: %w", err)
	}
	return nil
}

// ManagedMemberServiceIDs returns the PMM service ids of live members of a managed cluster, so the
// observed topology can mark which of its clusters Operations manages.
func ManagedMemberServiceIDs(q *reform.Querier) (map[string]bool, error) {
	rows, err := q.SelectAllFrom(OmClusterMemberTable,
		"WHERE removed_at IS NULL AND service_id IS NOT NULL "+
			"AND cluster_id IN (SELECT id FROM om_clusters WHERE status = 'managed' AND removed_at IS NULL)")
	if err != nil {
		return nil, fmt.Errorf("failed to select managed member services: %w", err)
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		member := row.(*OmClusterMember) //nolint:forcetypeassert
		if member.ServiceID != nil {
			out[*member.ServiceID] = true
		}
	}
	return out, nil
}
