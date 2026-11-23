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

func acquireLease(t *testing.T, q *reform.Querier, who string) int64 {
	t.Helper()
	var epoch int64
	require.NoError(t, q.QueryRow("SELECT om_acquire_leader_lease($1, '30 seconds'::interval)", who).Scan(&epoch))
	return epoch
}

func newDispatchStep(operationID string) *models.OmOperationStep {
	return &models.OmOperationStep{
		ID: uuid.New().String(), OperationID: operationID, TargetKind: models.OmStepTargetRun,
		TargetID: operationID, Phase: "run", Step: "dispatch", Attempt: 1,
		IdempotencyKey: uuid.New().String(), Status: models.OmStepDispatching,
	}
}

func TestAdvanceStatusCAS(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)

	moved, err := advanceStatus(db.Querier, op, models.OmOperationRunning, models.OmOperationSucceeded, 1)
	require.NoError(t, err)
	assert.True(t, moved)

	var status string
	var version int32
	require.NoError(t, db.QueryRow("SELECT status, version FROM om_operations WHERE id = $1", op).Scan(&status, &version))
	assert.Equal(t, string(models.OmOperationSucceeded), status)
	assert.Equal(t, int32(2), version)

	t.Run("a stale version moves nothing", func(t *testing.T) {
		op := lockOperation(t, db.Querier, clusterID)
		moved, err := advanceStatus(db.Querier, op, models.OmOperationRunning, models.OmOperationSucceeded, 99)
		require.NoError(t, err)
		assert.False(t, moved)
	})

	t.Run("an illegal transition is refused", func(t *testing.T) {
		op := lockOperation(t, db.Querier, clusterID)
		_, err := advanceStatus(db.Querier, op, models.OmOperationRunning, models.OmOperationChecking, 1)
		require.Error(t, err)
	})
}

func TestFencedDispatchLeaderEpoch(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)

	epochA := acquireLease(t, db.Querier, "driver-a")

	dispatched := false
	err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		ok, err := fencedDispatch(tx, op, 1, epochA, newDispatchStep(op))
		dispatched = ok
		return err
	})
	require.NoError(t, err)
	assert.True(t, dispatched)

	var steps int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_operation_steps WHERE operation_id = $1", op).Scan(&steps))
	assert.Equal(t, 1, steps)

	t.Run("a deposed leader is fenced by the advanced epoch", func(t *testing.T) {
		op := lockOperation(t, db.Querier, clusterID)
		stale := acquireLease(t, db.Querier, "driver-a")
		acquireLease(t, db.Querier, "driver-b")

		dispatched := true
		err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
			ok, err := fencedDispatch(tx, op, 1, stale, newDispatchStep(op))
			dispatched = ok
			return err
		})
		require.NoError(t, err)
		assert.False(t, dispatched)

		var steps int
		require.NoError(t, db.QueryRow("SELECT count(*) FROM om_operation_steps WHERE operation_id = $1", op).Scan(&steps))
		assert.Equal(t, 0, steps)
	})

	t.Run("a stale version is fenced", func(t *testing.T) {
		op := lockOperation(t, db.Querier, clusterID)
		epoch := acquireLease(t, db.Querier, "driver-b")
		dispatched := true
		err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
			ok, err := fencedDispatch(tx, op, 99, epoch, newDispatchStep(op))
			dispatched = ok
			return err
		})
		require.NoError(t, err)
		assert.False(t, dispatched)
	})
}
