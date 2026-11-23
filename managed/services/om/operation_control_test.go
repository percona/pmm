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

func opWithStatus(t *testing.T, q *reform.Querier, clusterID string, status models.OmOperationStatus) *models.OmOperation {
	t.Helper()
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
		Status: status, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}
	require.NoError(t, q.Insert(op))
	return op
}

func TestListAndGetOperations(t *testing.T) {
	db := lockTestDB(t)
	clusterA := lockCluster(t, db.Querier)
	clusterB := lockCluster(t, db.Querier)
	opA := lockOperation(t, db.Querier, clusterA)
	lockOperation(t, db.Querier, clusterA)
	lockOperation(t, db.Querier, clusterB)

	forA, err := listOperations(db.Querier, &clusterA)
	require.NoError(t, err)
	assert.Len(t, forA, 2)

	all, err := listOperations(db.Querier, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 3)

	t.Run("get returns the operation and its steps", func(t *testing.T) {
		dispatchingStep(t, db.Querier, opA)
		op, steps, err := getOperation(db.Querier, opA)
		require.NoError(t, err)
		assert.Equal(t, opA, op.ID)
		assert.Len(t, steps, 1)
	})

	t.Run("get a missing operation is ErrNotFound", func(t *testing.T) {
		_, _, err := getOperation(db.Querier, uuid.New().String())
		assert.ErrorIs(t, err, models.ErrNotFound)
	})
}

func TestCancelQueuedOperation(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := opWithStatus(t, db.Querier, clusterID, models.OmOperationQueued)
	require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

	moved, err := cancelOperation(t.Context(), db, op, 1, "admin", OperationKind{Kind: "test"})
	require.NoError(t, err)
	assert.True(t, moved)

	var status, cancelledBy string
	require.NoError(t, db.QueryRow("SELECT status, cancelled_by FROM om_operations WHERE id = $1", op.ID).Scan(&status, &cancelledBy))
	assert.Equal(t, string(models.OmOperationCancelled), status)
	assert.Equal(t, "admin", cancelledBy)
	assert.Equal(t, 0, heldLocks(t, db, op.ID))

	t.Run("a stale version does not cancel", func(t *testing.T) {
		other := opWithStatus(t, db.Querier, clusterID, models.OmOperationQueued)
		moved, err := cancelOperation(t.Context(), db, other, 99, "admin", OperationKind{Kind: "test"})
		require.NoError(t, err)
		assert.False(t, moved)
		var status string
		require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", other.ID).Scan(&status))
		assert.Equal(t, string(models.OmOperationQueued), status)
	})
}

func TestCancelRunningOperation(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)

	t.Run("a cancellable kind moves to cancelling", func(t *testing.T) {
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
		moved, err := cancelOperation(t.Context(), db, op, 1, "admin", OperationKind{Kind: "test", CancellableWhileRunning: true})
		require.NoError(t, err)
		assert.True(t, moved)
		var status string
		require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
		assert.Equal(t, string(models.OmOperationCancelling), status)
	})

	t.Run("a non-cancellable kind does not move", func(t *testing.T) {
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
		moved, err := cancelOperation(t.Context(), db, op, 1, "admin", OperationKind{Kind: "test"})
		require.NoError(t, err)
		assert.False(t, moved)
		var status string
		require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
		assert.Equal(t, string(models.OmOperationRunning), status)
	})
}

func TestCancelRunsOnTerminalHook(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := opWithStatus(t, db.Querier, clusterID, models.OmOperationQueued)

	ran := false
	kind := OperationKind{Kind: "test", OnTerminal: func(*reform.TX, *models.OmOperation) error {
		ran = true
		return nil
	}}
	moved, err := cancelOperation(t.Context(), db, op, 1, "operator", kind)
	require.NoError(t, err)
	require.True(t, moved)
	assert.True(t, ran, "the onTerminal hook runs on a terminal cancel")
	var cancelledBy string
	require.NoError(t, db.QueryRow("SELECT cancelled_by FROM om_operations WHERE id = $1", op.ID).Scan(&cancelledBy))
	assert.Equal(t, "operator", cancelledBy)
}
