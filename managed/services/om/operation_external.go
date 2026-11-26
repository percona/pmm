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
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

// externalHoldNamespace derives a durable operation id from a writer's job id, so a retry reuses the
// same external hold rather than creating a second one.
var externalHoldNamespace = uuid.NewSHA1(uuid.NameSpaceOID, []byte("om-external-hold"))

// externalOpID is the deterministic om_operations id for a writer's job.
func externalOpID(jobID string) string {
	return uuid.NewSHA1(externalHoldNamespace, []byte(jobID)).String()
}

// AcquireExternalHold takes locks for a non-operation PMM backup writer through a durable
// external-status op keyed by the job id; re-acquire is idempotent (ON CONFLICT), a real conflict raises.
func AcquireExternalHold(q *reform.Querier, jobID, kind, clusterID string, resources []LockResource, deadline time.Time) (string, error) {
	operationID := externalOpID(jobID)
	principal, err := json.Marshal(map[string]string{"kind": "external", "job": jobID})
	if err != nil {
		return "", fmt.Errorf("failed to marshal external principal: %w", err)
	}
	_, err = q.Exec(
		"INSERT INTO om_operations "+
			"(id, kind, kind_version, cluster_id, status, version, params_revision, trigger, principal, params, error_fields, deadline_at, created_at) "+
			"VALUES ($1, $2, 1, $3, 'external', 1, 1, 'policy', $4, '{}', '{}', $5, now()) ON CONFLICT (id) DO NOTHING",
		operationID, kind, clusterID, principal, deadline)
	if err != nil {
		return "", fmt.Errorf("failed to record external hold: %w", err)
	}
	for _, r := range resources {
		_, err = q.Exec(
			"INSERT INTO om_locks (resource_kind, resource_id, operation_id, mode) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			r.Kind, r.ID, operationID, string(r.Mode))
		if err != nil {
			return "", fmt.Errorf("failed to take external lock on %s %s: %w", r.Kind, r.ID, err)
		}
	}
	return operationID, nil
}

// HeartbeatExternalHold extends a hold's lease, so the reaper does not reclaim a live writer's locks.
func HeartbeatExternalHold(q *reform.Querier, jobID string, deadline time.Time) error {
	_, err := q.Exec(
		"UPDATE om_operations SET deadline_at = $1 WHERE id = $2 AND status = 'external'",
		deadline, externalOpID(jobID))
	if err != nil {
		return fmt.Errorf("failed to heartbeat external hold: %w", err)
	}
	return nil
}

// ReleaseExternalHold drops a hold's locks and finalises its operation; it is a no-op once released.
func ReleaseExternalHold(q *reform.Querier, jobID string) error {
	operationID := externalOpID(jobID)
	err := releaseLocks(q, operationID)
	if err != nil {
		return err
	}
	_, err = q.Exec(
		"UPDATE om_operations SET status = 'cancelled', version = version + 1, finished_at = now() WHERE id = $1 AND status = 'external'",
		operationID)
	if err != nil {
		return fmt.Errorf("failed to finalise external hold: %w", err)
	}
	return nil
}
