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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

func checkingOp(t *testing.T, q *reform.Querier, clusterID string) *models.OmOperation {
	t.Helper()
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationChecking, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}
	require.NoError(t, q.Insert(op))
	return op
}

func clusterExclusiveKind() OperationKind {
	return OperationKind{
		Kind: "test",
		LockSet: func(op *models.OmOperation) []LockResource {
			return []LockResource{{Kind: "cluster", ID: *op.ClusterID, Mode: models.OmLockExclusive}}
		},
	}
}

func notStale(*reform.TX) (bool, error) { return false, nil }

func opRow(t *testing.T, db *reform.DB, id string) (string, int32, int32, int) {
	t.Helper()
	var status string
	var version, paramsRevision int32
	require.NoError(t, db.QueryRow("SELECT status, version, params_revision FROM om_operations WHERE id = $1", id).Scan(&status, &version, &paramsRevision))
	var locks int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_locks WHERE operation_id = $1", id).Scan(&locks))
	return status, version, paramsRevision, locks
}

func TestStartGateRuns(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	outcome, err := startGate(t.Context(), db, op, models.OmOperationChecking, 1, nil, clusterExclusiveKind(), nil, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartedRunning, outcome)

	status, _, _, locks := opRow(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationRunning), status)
	assert.Equal(t, 1, locks)
}

func TestStartGateRejectsStaleVersion(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	outcome, err := startGate(t.Context(), db, op, models.OmOperationChecking, 99, nil, clusterExclusiveKind(), nil, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartRejected, outcome)

	status, version, _, locks := opRow(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationChecking), status)
	assert.Equal(t, int32(1), version)
	assert.Equal(t, 0, locks)
}

func TestStartGateStaleInputReturnsCheckingAndInvalidates(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	stale := func(*reform.TX) (bool, error) { return true, nil }
	outcome, err := startGate(t.Context(), db, op, models.OmOperationChecking, 1, nil, clusterExclusiveKind(), nil, stale)
	require.NoError(t, err)
	assert.Equal(t, StartStaleInput, outcome)

	status, _, paramsRevision, locks := opRow(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationChecking), status)
	assert.Equal(t, int32(2), paramsRevision)
	assert.Equal(t, 0, locks)
}

func TestStartGateBlockedByAttention(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)
	raiseAttention(t, db.Querier, clusterID, AttentionDrift, "")

	outcome, err := startGate(t.Context(), db, op, models.OmOperationChecking, 1, nil, clusterExclusiveKind(), nil, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartBlockedAttention, outcome)

	status, _, _, locks := opRow(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationChecking), status)
	assert.Equal(t, 0, locks)
}

func TestStartGateFromQueuedWithSubjects(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationQueued, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}
	require.NoError(t, db.Insert(op))

	lockSubjects := []string{uuid.New().String()}
	outcome, err := startGate(t.Context(), db, op, models.OmOperationQueued, 1, nil, clusterExclusiveKind(), lockSubjects, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartedRunning, outcome)

	status, _, _, locks := opRow(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationRunning), status)
	assert.Equal(t, 1, locks)
}

func TestStartGateConfirmFencesPlan(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	digest := "abc123"
	newConfirmOp := func() *models.OmOperation {
		op := &models.OmOperation{
			ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
			Status: models.OmOperationAwaitingConfirmation, Trigger: models.OmTriggerUser,
			Principal: []byte(`{}`), ReportDigest: &digest,
		}
		require.NoError(t, db.Insert(op))
		return op
	}

	matching := newConfirmOp()
	outcome, err := startGate(t.Context(), db, matching, models.OmOperationAwaitingConfirmation, 1,
		&ConfirmGuard{ReportDigest: digest, ParamsRevision: 1}, clusterExclusiveKind(), nil, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartedRunning, outcome)

	stale := newConfirmOp()
	outcome, err = startGate(t.Context(), db, stale, models.OmOperationAwaitingConfirmation, 1,
		&ConfirmGuard{ReportDigest: "wrong", ParamsRevision: 1}, clusterExclusiveKind(), nil, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartRejected, outcome)
	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", stale.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationAwaitingConfirmation), status)
}

func TestStartGateLockConflictQueues(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	holder := lockOperation(t, db.Querier, clusterID)
	require.NoError(t, acquireLocks(db.Querier, holder, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

	op := checkingOp(t, db.Querier, clusterID)
	outcome, err := startGate(t.Context(), db, op, models.OmOperationChecking, 1, nil, clusterExclusiveKind(), nil, notStale)
	require.NoError(t, err)
	assert.Equal(t, StartQueued, outcome)

	status, _, _, locks := opRow(t, db, op.ID)
	assert.Equal(t, string(models.OmOperationQueued), status)
	assert.Equal(t, 0, locks)
}
