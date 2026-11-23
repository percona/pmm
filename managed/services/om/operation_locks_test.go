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
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

func lockTestDB(t *testing.T) *reform.DB {
	t.Helper()
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "encryption.key"))
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
}

func lockCluster(t *testing.T, q *reform.Querier) string {
	t.Helper()
	c := &models.OmCluster{
		ID: uuid.New().String(), Name: "rs0", Kind: models.OmClusterReplicaSet,
		Origin: models.OmClusterInstalled, Status: models.OmClusterManaged, CreatedBy: "t",
	}
	require.NoError(t, q.Insert(c))
	return c.ID
}

func lockOperation(t *testing.T, q *reform.Querier, clusterID string) string {
	t.Helper()
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "sync", KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationRunning, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}
	require.NoError(t, q.Insert(op))
	return op.ID
}

func TestAcquireLocksConflictAndRelease(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op1 := lockOperation(t, db.Querier, clusterID)
	op2 := lockOperation(t, db.Querier, clusterID)

	err := acquireLocks(db.Querier, op1, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}})
	require.NoError(t, err)

	err = acquireLocks(db.Querier, op2, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}})
	require.Error(t, err)
	conflict, ok := parseLockConflict(err)
	require.True(t, ok)
	assert.Equal(t, op1, conflict.HolderOperationID)
	assert.Equal(t, "exclusive", conflict.HolderMode)
	assert.Equal(t, "exclusive", conflict.RequestedMode)

	require.NoError(t, releaseLocks(db.Querier, op1))
	err = acquireLocks(db.Querier, op2, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}})
	require.NoError(t, err)
}

func TestAcquireLocksSortedSet(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := lockOperation(t, db.Querier, clusterID)

	err := acquireLocks(db.Querier, op, []LockResource{
		{Kind: "node", ID: "node-1", Mode: models.OmLockExclusive},
		{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive},
	})
	require.NoError(t, err)

	var held int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_locks WHERE operation_id = $1", op).Scan(&held))
	assert.Equal(t, 2, held)
}

func TestParseLockConflictNonConflict(t *testing.T) {
	t.Parallel()

	_, ok := parseLockConflict(assert.AnError)
	assert.False(t, ok)
}
