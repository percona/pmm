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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

func heldLocks(t *testing.T, db *reform.DB, operationID string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_locks WHERE operation_id = $1", operationID).Scan(&n))
	return n
}

func TestExternalHoldIdempotentAndRelease(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	deadline := time.Now().Add(time.Minute)
	resources := []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockShared}}

	opID, err := AcquireExternalHold(db.Querier, "backup-job-1", "backup", clusterID, resources, deadline)
	require.NoError(t, err)
	assert.Equal(t, externalOpID("backup-job-1"), opID)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", opID).Scan(&status))
	assert.Equal(t, string(models.OmOperationExternal), status)
	assert.Equal(t, 1, heldLocks(t, db, opID))

	t.Run("re-acquire by the same job reuses the hold", func(t *testing.T) {
		again, err := AcquireExternalHold(db.Querier, "backup-job-1", "backup", clusterID, resources, deadline)
		require.NoError(t, err)
		assert.Equal(t, opID, again)
		assert.Equal(t, 1, heldLocks(t, db, opID))
	})

	t.Run("release drops the locks and finalises the hold", func(t *testing.T) {
		require.NoError(t, ReleaseExternalHold(db.Querier, "backup-job-1"))
		assert.Equal(t, 0, heldLocks(t, db, opID))
		var status string
		require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", opID).Scan(&status))
		assert.Equal(t, string(models.OmOperationCancelled), status)
	})
}

func TestExternalHoldConflict(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	driverOp := lockOperation(t, db.Querier, clusterID)
	require.NoError(t, acquireLocks(db.Querier, driverOp, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

	_, err := AcquireExternalHold(db.Querier, "backup-job-2", "backup", clusterID,
		[]LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}, time.Now().Add(time.Minute))
	require.Error(t, err)
	conflict, ok := parseLockConflict(err)
	require.True(t, ok)
	assert.Equal(t, driverOp, conflict.HolderOperationID)
}
