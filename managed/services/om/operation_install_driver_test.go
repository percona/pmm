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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

func installRegistry(t *testing.T) *OperationRegistry {
	t.Helper()
	reg := NewOperationRegistry()
	require.NoError(t, reg.Register(InstallKind()))
	return reg
}

func checkingInstallOp(t *testing.T, db *reform.DB, nodeIDs ...string) *models.OmOperation {
	t.Helper()
	hosts := make(map[string]string, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		hosts[nodeID] = "exec-" + nodeID
	}
	params, err := json.Marshal(installParams{NodeIDs: nodeIDs, Hosts: hosts})
	require.NoError(t, err)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: installKind, KindVersion: 1,
		Status: models.OmOperationChecking, Version: 1, ParamsRevision: 1,
		Trigger: models.OmTriggerUser, Principal: []byte(`{}`), Params: params, ErrorFields: []byte("{}"),
	}
	require.NoError(t, db.Insert(op))
	return op
}

func installFactReader(perNode map[string]map[string]any) *fakeFactReader {
	facts := make(map[string]map[string]any, len(perNode))
	at := make(map[string]time.Time, len(perNode))
	now := time.Now()
	for node, f := range perNode {
		facts[node] = f
		at[node] = now
	}
	return &fakeFactReader{facts: facts, at: at}
}

func TestInstallDriverRunsWhenNodesAreReady(t *testing.T) {
	db := lockTestDB(t)
	reg := installRegistry(t)
	op := checkingInstallOp(t, db, "node-a", "node-b")
	reader := installFactReader(map[string]map[string]any{"node-a": nodeFacts(), "node-b": nodeFacts()})
	runner := newTestRunner(t, db, reg, newFactCheckProvider(reader), fakeExecutorProvider{result: StepResultOk})

	runner.Tick(t.Context())

	// Readiness passes, the gate takes the node locks, the enter-running hook creates the cluster, and
	// the bootstrap step succeeds, so the install drives to succeeded.
	state, _ := opStatus(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationSucceeded), state)
	assert.Equal(t, 0, heldLocks(t, db, op.ID))

	clusterID := installRecordID(op.ID, "cluster")
	cluster, err := db.FindByPrimaryKeyFrom(models.OmClusterTable, clusterID)
	require.NoError(t, err)
	record := cluster.(*models.OmCluster)
	assert.Equal(t, models.OmClusterManaged, record.Status, "a succeeded install leaves the cluster managed")
	assert.Equal(t, models.OmClusterInstalled, record.Origin)
	assert.Equal(t, op.ID, record.CreatedBy)

	var linked string
	require.NoError(t, db.QueryRow("SELECT cluster_id FROM om_operations WHERE id = $1", op.ID).Scan(&linked))
	assert.Equal(t, clusterID, linked, "the operation is linked to the cluster it created")

	var replicaSets, members int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_replica_sets WHERE cluster_id = $1", clusterID).Scan(&replicaSets))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_cluster_members WHERE cluster_id = $1 AND removed_at IS NULL", clusterID).Scan(&members))
	assert.Equal(t, 1, replicaSets)
	assert.Equal(t, 2, members, "a member is recorded for each installed node")

	var markers int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM om_owned_items WHERE cluster_id = $1 AND kind = 'marker' AND state = 'present'", clusterID).Scan(&markers))
	assert.Equal(t, 1, markers, "a succeeded install records a cluster install marker")
}

func TestInstallOnEnterRunningIsIdempotent(t *testing.T) {
	db := lockTestDB(t)
	op := checkingInstallOp(t, db, "node-a")

	for range 2 {
		err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
			return installOnEnterRunning(tx, op)
		})
		require.NoError(t, err)
	}

	clusterID := installRecordID(op.ID, "cluster")
	var clusters, replicaSets int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_clusters WHERE id = $1", clusterID).Scan(&clusters))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_replica_sets WHERE cluster_id = $1", clusterID).Scan(&replicaSets))
	assert.Equal(t, 1, clusters, "a re-run creates no second cluster")
	assert.Equal(t, 1, replicaSets, "a re-run creates no second replica set")
}

func TestInstallOnTerminalTombstonesRecords(t *testing.T) {
	db := lockTestDB(t)
	op := checkingInstallOp(t, db, "node-a", "node-b")
	require.NoError(t, db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		err := installOnEnterRunning(tx, op)
		if err != nil {
			return err
		}
		return installOnSucceeded(tx, op)
	}))

	require.NoError(t, db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		return installOnTerminal(tx, op)
	}))

	clusterID := installRecordID(op.ID, "cluster")
	var clusterStatus string
	var removedAt *time.Time
	require.NoError(t, db.QueryRow("SELECT status, removed_at FROM om_clusters WHERE id = $1", clusterID).Scan(&clusterStatus, &removedAt))
	assert.Equal(t, string(models.OmClusterRemoved), clusterStatus)
	assert.NotNil(t, removedAt)

	var liveMembers, liveReplicaSets int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_cluster_members WHERE cluster_id = $1 AND removed_at IS NULL", clusterID).Scan(&liveMembers))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_replica_sets WHERE cluster_id = $1 AND removed_at IS NULL", clusterID).Scan(&liveReplicaSets))
	assert.Equal(t, 0, liveMembers, "members are tombstoned")
	assert.Equal(t, 0, liveReplicaSets, "replica sets are tombstoned")
}

func TestInstallDriverBlocksWhenANodeIsNotReady(t *testing.T) {
	db := lockTestDB(t)
	reg := installRegistry(t)
	op := checkingInstallOp(t, db, "node-a", "node-b")
	unready := nodeFacts()
	unready[factNoMongod] = false
	reader := installFactReader(map[string]map[string]any{"node-a": nodeFacts(), "node-b": unready})
	runner := newTestRunner(t, db, reg, newFactCheckProvider(reader), pendingStepExecutorProvider{})

	runner.Tick(t.Context())

	state, _ := opStatus(t, db, op.ID)
	require.Equal(t, string(models.OmOperationFailed), state)
	var code string
	require.NoError(t, db.QueryRow("SELECT error_code FROM om_operations WHERE id = $1", op.ID).Scan(&code))
	assert.Contains(t, code, "already runs mongod")
	assert.Equal(t, 0, heldLocks(t, db, op.ID))
}

func TestInstallDriverRescansWhenANodeWasNotScanned(t *testing.T) {
	db := lockTestDB(t)
	reg := installRegistry(t)
	op := checkingInstallOp(t, db, "node-a", "node-b")
	// node-b has no observation yet, so its facts read as absent; readiness cannot be judged, so the
	// driver requests a scan and leaves the operation checking rather than failing it fail-closed.
	reader := installFactReader(map[string]map[string]any{"node-a": nodeFacts()})
	scans := &fakeScanRequester{}
	runner := NewRunner(db, logrus.WithField("test", t.Name()), reg, newFactCheckProvider(reader), pendingStepExecutorProvider{}, scans)

	runner.Tick(t.Context())

	state, _ := opStatus(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationChecking), state)
	assert.Len(t, scans.requests, 1)
	assert.Equal(t, 0, heldLocks(t, db, op.ID))
}
