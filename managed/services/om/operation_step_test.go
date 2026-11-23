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
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

func dispatchingStep(t *testing.T, q *reform.Querier, operationID string) string {
	t.Helper()
	step := &models.OmOperationStep{
		ID: uuid.New().String(), OperationID: operationID, TargetKind: models.OmStepTargetRun,
		TargetID: operationID, Phase: "run", Step: uuid.New().String(), Attempt: 1,
		IdempotencyKey: uuid.New().String(), Epoch: 1, Status: models.OmStepDispatching,
	}
	require.NoError(t, q.Insert(step))
	return step.ID
}

func stepStatus(t *testing.T, db *reform.DB, stepID string) string {
	t.Helper()
	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operation_steps WHERE id = $1", stepID).Scan(&status))
	return status
}

func TestStepStateMachine(t *testing.T) {
	t.Parallel()
	assert.True(t, canStepTransition(models.OmStepPlanned, models.OmStepDispatching))
	assert.True(t, canStepTransition(models.OmStepDispatching, models.OmStepUnknown))
	assert.True(t, canStepTransition(models.OmStepUnknown, models.OmStepSucceeded))
	assert.False(t, canStepTransition(models.OmStepPlanned, models.OmStepSucceeded))
	assert.False(t, canStepTransition(models.OmStepSucceeded, models.OmStepFailed))
}

func TestMapStepResult(t *testing.T) {
	t.Parallel()
	assert.Equal(t, models.OmStepSucceeded, mapStepResult(StepResultOk))
	assert.Equal(t, models.OmStepFailed, mapStepResult(StepResultFailed))
	assert.Equal(t, models.OmStepFailed, mapStepResult(StepResultBlocked))
	assert.Equal(t, models.OmStepUnknown, mapStepResult(StepResultTimedOut))
	assert.Equal(t, models.OmStepUnknown, mapStepResult(StepResultUnknown))
}

func TestRecordStepResultIgnoresLateResult(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)
	step := dispatchingStep(t, db.Querier, op)

	moved, err := recordStepResult(db.Querier, step, models.OmStepDispatching, StepResultOk, []byte(`{"kind":"ok"}`))
	require.NoError(t, err)
	assert.True(t, moved)
	assert.Equal(t, string(models.OmStepSucceeded), stepStatus(t, db, step))

	moved, err = recordStepResult(db.Querier, step, models.OmStepDispatching, StepResultFailed, []byte(`{"kind":"failed"}`))
	require.NoError(t, err)
	assert.False(t, moved, "a late result for a resolved step is ignored")
	assert.Equal(t, string(models.OmStepSucceeded), stepStatus(t, db, step))
}

func TestRecordStepResultFromRunning(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)
	step := dispatchingStep(t, db.Querier, op)
	_, err := db.Exec("UPDATE om_operation_steps SET status = 'running' WHERE id = $1", step)
	require.NoError(t, err)

	moved, err := recordStepResult(db.Querier, step, models.OmStepRunning, StepResultOk, []byte(`{"kind":"ok"}`))
	require.NoError(t, err)
	assert.True(t, moved)
	assert.Equal(t, string(models.OmStepSucceeded), stepStatus(t, db, step))
}

func TestDeleteAdministratorSecret(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)
	expires := time.Now().Add(time.Hour)
	admin := &models.OmSecret{Purpose: "administrator", OperationID: &op, ExpiresAt: &expires, Generation: models.OmSecretCurrent, Payload: "pw"}
	require.NoError(t, models.CreateOmSecret(db.Querier, admin))

	require.NoError(t, deleteAdministratorSecret(db.Querier, op))
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_secrets WHERE operation_id = $1 AND purpose = 'administrator'", op).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestResolveUnknownStep(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)

	proven := dispatchingStep(t, db.Querier, op)
	_, err := recordStepResult(db.Querier, proven, models.OmStepDispatching, StepResultUnknown, nil)
	require.NoError(t, err)
	moved, err := resolveUnknownStep(db.Querier, proven, true)
	require.NoError(t, err)
	assert.True(t, moved)
	assert.Equal(t, string(models.OmStepSucceeded), stepStatus(t, db, proven))

	disproven := dispatchingStep(t, db.Querier, op)
	_, err = recordStepResult(db.Querier, disproven, models.OmStepDispatching, StepResultTimedOut, nil)
	require.NoError(t, err)
	moved, err = resolveUnknownStep(db.Querier, disproven, false)
	require.NoError(t, err)
	assert.True(t, moved)
	assert.Equal(t, string(models.OmStepFailed), stepStatus(t, db, disproven))
}

func TestPromotePendingSecret(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)
	rs := &models.OmReplicaSet{ID: uuid.New().String(), ClusterID: clusterID, Name: "rs0", Role: models.OmReplicaSetRolePlain}
	require.NoError(t, db.Insert(rs))

	current := &models.OmSecret{Purpose: "automation", ClusterID: &clusterID, ReplicaSetRef: &rs.ID, Generation: models.OmSecretCurrent, Payload: "old"}
	require.NoError(t, models.CreateOmSecret(db.Querier, current))
	pending := &models.OmSecret{Purpose: "automation", ClusterID: &clusterID, ReplicaSetRef: &rs.ID, OperationID: &op, Generation: models.OmSecretPending, Payload: "new"}
	require.NoError(t, models.CreateOmSecret(db.Querier, pending))

	require.NoError(t, promotePendingSecret(db.Querier, pending.ID))

	var currentCount int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM om_secrets WHERE purpose = 'automation' AND cluster_id = $1 AND generation = 'current'", clusterID).Scan(&currentCount))
	assert.Equal(t, 1, currentCount)

	var promotedGen, oldGen string
	require.NoError(t, db.QueryRow("SELECT generation FROM om_secrets WHERE id = $1", pending.ID).Scan(&promotedGen))
	require.NoError(t, db.QueryRow("SELECT generation FROM om_secrets WHERE id = $1", current.ID).Scan(&oldGen))
	assert.Equal(t, "current", promotedGen)
	assert.Equal(t, "retired", oldGen)
}
