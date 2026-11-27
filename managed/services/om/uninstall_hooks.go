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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// uninstallOnEnterRunning moves the cluster to removing in the enter-running transaction; it is
// idempotent, moving only a managed or monitored cluster.
func uninstallOnEnterRunning(tx *reform.TX, op *models.OmOperation) error {
	params := uninstallParamsOf(op)
	_, err := tx.Exec(
		"UPDATE om_clusters SET status = $1, updated_at = now() WHERE id = $2 AND status IN ($3, $4)",
		models.OmClusterRemoving, params.ClusterID, models.OmClusterManaged, models.OmClusterMonitored)
	return err
}

// uninstallOnSucceeded settles the cluster's owned items (clearing before_secret_id to lift the
// RESTRICT) then deletes its own secrets and tombstones the cluster, replica sets and members.
func uninstallOnSucceeded(tx *reform.TX, op *models.OmOperation) error {
	params := uninstallParamsOf(op)
	err := settleClusterRecords(tx, op, params.ClusterID)
	if err != nil {
		return err
	}
	err = deregisterClusterServices(tx, params.ClusterID)
	if err != nil {
		return err
	}
	return tombstoneCluster(tx, op, params.ClusterID)
}

// uninstallOnTerminal abandons a teardown that began but did not succeed, recording each member host's
// residual state, marking the cluster cleanup_abandoned, then settling and tombstoning; a terminal
// reached before the teardown started leaves the cluster untouched, since no host was changed.
func uninstallOnTerminal(tx *reform.TX, op *models.OmOperation) error {
	if op.StartedAt == nil {
		return nil
	}
	params := uninstallParamsOf(op)
	residual, err := uninstallResidualHosts(tx, op, params)
	if err != nil {
		return err
	}
	err = models.SetOmOperationResidualHosts(tx.Querier, op.ID, residual)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"UPDATE om_clusters SET cleanup_abandoned = true, updated_at = now() WHERE id = $1",
		params.ClusterID)
	if err != nil {
		return err
	}
	err = settleClusterRecords(tx, op, params.ClusterID)
	if err != nil {
		return err
	}
	err = deregisterClusterServices(tx, params.ClusterID)
	if err != nil {
		return err
	}
	return tombstoneCluster(tx, op, params.ClusterID)
}

// deregisterClusterServices removes the inventory services and exporters install registered for the
// cluster's live members, so a removed cluster leaves no monitoring behind; it is idempotent.
func deregisterClusterServices(tx *reform.TX, clusterID string) error {
	rows, err := tx.SelectAllFrom(models.OmClusterMemberTable,
		"WHERE cluster_id = $1 AND removed_at IS NULL AND service_id IS NOT NULL", clusterID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		serviceID := row.(*models.OmClusterMember).ServiceID //nolint:forcetypeassert
		if serviceID == nil {
			continue
		}
		err = models.RemoveService(tx.Querier, *serviceID, models.RemoveCascade)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return fmt.Errorf("failed to deregister service %s: %w", *serviceID, err)
		}
	}
	return nil
}

// settleClusterRecords settles the cluster's owned items (clearing before_secret_id to lift the
// RESTRICT) and deletes its own non-owner-scoped secrets; owner_kind secrets survive.
func settleClusterRecords(tx *reform.TX, op *models.OmOperation, clusterID string) error {
	_, err := tx.Exec(
		"UPDATE om_owned_items SET state = $1, before_secret_id = NULL, removed_by_operation = $2, updated_at = now() "+
			"WHERE cluster_id = $3 AND state <> $1",
		models.OmOwnedRemoved, op.ID, clusterID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"DELETE FROM om_secrets WHERE cluster_id = $1 AND owner_kind IS NULL",
		clusterID)
	return err
}

// tombstoneCluster tombstones the cluster, its replica sets and its members; the live-unique indexes
// are WHERE removed_at IS NULL, so a later install may reuse the name and hosts.
func tombstoneCluster(tx *reform.TX, op *models.OmOperation, clusterID string) error {
	_, err := tx.Exec(
		"UPDATE om_cluster_members SET removed_at = now(), removed_by_operation = $2 WHERE cluster_id = $1 AND removed_at IS NULL",
		clusterID, op.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"UPDATE om_replica_sets SET removed_at = now() WHERE cluster_id = $1 AND removed_at IS NULL",
		clusterID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"UPDATE om_clusters SET status = 'removed', removed_at = now(), updated_at = now() WHERE id = $1 AND removed_at IS NULL",
		clusterID)
	return err
}

// uninstallResidualHosts classifies each sealed member host for the abandon record: removed when its
// remove step succeeded, still_running when even its stop step did not, otherwise not_removed.
func uninstallResidualHosts(tx *reform.TX, op *models.OmOperation, params uninstallParams) ([]models.OmResidualHost, error) {
	hosts := make([]models.OmResidualHost, 0, len(params.Members))
	for _, member := range params.Members {
		stopped, err := memberStepSucceeded(tx.Querier, op.ID, member.MemberID, uninstallStopStep)
		if err != nil {
			return nil, err
		}
		removed, err := memberStepSucceeded(tx.Querier, op.ID, member.MemberID, uninstallRemoveStep)
		if err != nil {
			return nil, err
		}
		status := models.OmResidualHostNotRemoved
		switch {
		case removed:
			status = models.OmResidualHostRemoved
		case !stopped:
			status = models.OmResidualHostStillRunning
		}
		hosts = append(hosts, models.OmResidualHost{Host: member.Host, Status: status})
	}
	return hosts, nil
}

// memberStepSucceeded reports whether a member's latest attempt of a step succeeded.
func memberStepSucceeded(q *reform.Querier, operationID, memberID, step string) (bool, error) {
	var status string
	err := q.QueryRow(
		"SELECT status FROM om_operation_steps WHERE operation_id = $1 AND target_kind = $2 AND target_id = $3 AND step = $4 "+
			"ORDER BY attempt DESC LIMIT 1",
		operationID, string(models.OmStepTargetMember), memberID, step).Scan(&status)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return models.OmStepStatus(status) == models.OmStepSucceeded, nil
}
