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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type fakeExecutor struct {
	result StepResultKind
}

func (f *fakeExecutor) Run(context.Context, StepPlan, string) (StepResultKind, []byte, error) {
	return f.result, []byte(`{"kind":"` + string(f.result) + `"}`), nil
}

func runningKind(onSucceeded *bool) OperationKind {
	return OperationKind{
		Kind: "test",
		LockSet: func(op *models.OmOperation) []LockResource {
			return []LockResource{{Kind: "cluster", ID: *op.ClusterID, Mode: models.OmLockExclusive}}
		},
		Steps: func(op *models.OmOperation) []StepPlan {
			return []StepPlan{{Phase: "run", Step: "do", TargetKind: models.OmStepTargetRun, TargetID: op.ID}}
		},
		OnSucceeded: func(_ *reform.TX, _ *models.OmOperation) error {
			*onSucceeded = true
			return nil
		},
	}
}

func runningOp(t *testing.T, db *reform.DB, clusterID string) (string, int64) {
	t.Helper()
	op := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
	require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))
	return op.ID, acquireLease(t, db.Querier, "driver")
}

func TestDriveRunningDispatchesAndSucceeds(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	opID, epoch := runningOp(t, db, clusterID)
	ran := false
	kind := runningKind(&ran)

	outcome, err := driveRunning(t.Context(), db, opID, kind, &fakeExecutor{result: StepResultOk}, epoch)
	require.NoError(t, err)
	assert.Equal(t, RunStepSucceeded, outcome)

	outcome, err = driveRunning(t.Context(), db, opID, kind, &fakeExecutor{result: StepResultOk}, epoch)
	require.NoError(t, err)
	assert.Equal(t, RunOperationSucceeded, outcome)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", opID).Scan(&status))
	assert.Equal(t, string(models.OmOperationSucceeded), status)
	assert.True(t, ran, "onSucceeded ran")
	assert.Equal(t, 0, heldLocks(t, db, opID))
}

func TestDriveRunningStepFailureFailsOperation(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	opID, epoch := runningOp(t, db, clusterID)
	ran := false

	outcome, err := driveRunning(t.Context(), db, opID, runningKind(&ran), &fakeExecutor{result: StepResultFailed}, epoch)
	require.NoError(t, err)
	assert.Equal(t, RunStepFailed, outcome)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", opID).Scan(&status))
	assert.Equal(t, string(models.OmOperationFailed), status)

	var active int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM om_attention WHERE cluster_id = $1 AND reason = 'operation_failed' AND resolved_at IS NULL", clusterID).Scan(&active))
	assert.Equal(t, 1, active)
}

func TestDriveRunningFencedByStaleEpoch(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	opID, stale := runningOp(t, db, clusterID)
	acquireLease(t, db.Querier, "newer-leader")
	ran := false

	outcome, err := driveRunning(t.Context(), db, opID, runningKind(&ran), &fakeExecutor{result: StepResultOk}, stale)
	require.NoError(t, err)
	assert.Equal(t, RunFenced, outcome)

	var steps int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_operation_steps WHERE operation_id = $1", opID).Scan(&steps))
	assert.Equal(t, 0, steps)
}
