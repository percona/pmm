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

func failedOp(t *testing.T, q *reform.Querier, clusterID string, kindVersion int32) *models.OmOperation {
	t.Helper()
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "test", KindVersion: kindVersion, ClusterID: &clusterID,
		Status: models.OmOperationFailed, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}
	require.NoError(t, q.Insert(op))
	return op
}

func operationFailedResolved(t *testing.T, db *reform.DB, clusterID string) bool {
	t.Helper()
	var resolved bool
	require.NoError(t, db.QueryRow(
		"SELECT resolved_at IS NOT NULL FROM om_attention WHERE cluster_id = $1 AND reason = 'operation_failed' AND subject_id = $1",
		clusterID).Scan(&resolved))
	return resolved
}

func TestAbandonClearsAttentionAndReleasesLocks(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := failedOp(t, db.Querier, clusterID, 1)
	require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))
	raiseAttention(t, db.Querier, clusterID, AttentionOperationFailed, "")

	abandoned, err := abandonOperation(t.Context(), db, op, 1)
	require.NoError(t, err)
	assert.True(t, abandoned)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationAbandoned), status)
	assert.Equal(t, 0, heldLocks(t, db, op.ID))
	assert.True(t, operationFailedResolved(t, db, clusterID))
}

func TestResumeReturnsToRunningAndClearsAttention(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := failedOp(t, db.Querier, clusterID, 1)
	raiseAttention(t, db.Querier, clusterID, AttentionOperationFailed, "")

	resumed, err := resumeOperation(t.Context(), db, op, 1, 1)
	require.NoError(t, err)
	assert.True(t, resumed)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationRunning), status)
	assert.True(t, operationFailedResolved(t, db, clusterID))
}

func TestResumeRefusesDifferentKindVersion(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := failedOp(t, db.Querier, clusterID, 1)

	resumed, err := resumeOperation(t.Context(), db, op, 1, 2)
	require.NoError(t, err)
	assert.False(t, resumed)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationFailed), status)
}
