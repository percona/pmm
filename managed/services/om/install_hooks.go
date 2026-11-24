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

	"github.com/google/uuid"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// installRecordNamespace derives the stable ids of the records an install operation owns, so the
// enter-running hook re-runs idempotently after a recovery without creating a second cluster.
var installRecordNamespace = uuid.NewSHA1(uuid.NameSpaceOID, []byte("om-install-records"))

// installRecordID is the deterministic id of one record an install operation owns.
func installRecordID(operationID, part string) string {
	return uuid.NewSHA1(installRecordNamespace, []byte(operationID+":"+part)).String()
}

// installOnEnterRunning creates the installing cluster and its replica set and links the operation to the
// cluster, in the enter-running transaction; it is idempotent on the operation's derived cluster id.
func installOnEnterRunning(tx *reform.TX, op *models.OmOperation) error {
	clusterID := installRecordID(op.ID, "cluster")
	_, err := tx.FindByPrimaryKeyFrom(models.OmClusterTable, clusterID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, reform.ErrNoRows) {
		return err
	}
	params := installParamsOf(op)
	cluster := &models.OmCluster{
		ID: clusterID, Name: params.ReplicaSetName, Kind: models.OmClusterReplicaSet,
		Origin: models.OmClusterInstalled, Status: models.OmClusterInstalling, CreatedBy: op.ID,
	}
	err = tx.Insert(cluster)
	if err != nil {
		return err
	}
	replicaSet := &models.OmReplicaSet{
		ID: installRecordID(op.ID, "replica_set"), ClusterID: clusterID,
		Name: params.ReplicaSetName, Role: models.OmReplicaSetRolePlain,
	}
	err = tx.Insert(replicaSet)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE om_operations SET cluster_id = $1 WHERE id = $2", clusterID, op.ID)
	return err
}

// installOnSucceeded records the installed members and marks the cluster managed in the success
// transaction, idempotent on the members it already added; the node and service links come separately.
func installOnSucceeded(tx *reform.TX, op *models.OmOperation) error {
	clusterID := installRecordID(op.ID, "cluster")
	added := 0
	err := tx.QueryRow(
		"SELECT count(*) FROM om_cluster_members WHERE cluster_id = $1 AND added_by_operation = $2",
		clusterID, op.ID).Scan(&added)
	if err != nil {
		return err
	}
	if added == 0 {
		err = insertInstalledMembers(tx, op, clusterID)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(
		"UPDATE om_clusters SET status = 'managed', updated_at = now() WHERE id = $1 AND status = 'installing'",
		clusterID)
	return err
}

// insertInstalledMembers adds one accepted mongod member per target node, keyed by host and numbered in
// node order as rs.initiate numbers a fresh replica set.
func insertInstalledMembers(tx *reform.TX, op *models.OmOperation, clusterID string) error {
	replicaSetRef := installRecordID(op.ID, "replica_set")
	params := installParamsOf(op)
	for i, nodeID := range params.NodeIDs {
		rsMemberID := int32(i)
		member := &models.OmClusterMember{
			ID:               installRecordID(op.ID, "member:"+nodeID),
			ClusterID:        clusterID,
			ReplicaSetRef:    &replicaSetRef,
			Process:          models.OmMemberMongod,
			Host:             params.Hosts[nodeID],
			RSMemberID:       &rsMemberID,
			State:            models.OmMemberAccepted,
			AddedByOperation: op.ID,
		}
		err := tx.Insert(member)
		if err != nil {
			return err
		}
	}
	return nil
}
