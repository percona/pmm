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

package models_test

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

func omTestDB(t *testing.T) *reform.DB {
	t.Helper()
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "encryption.key"))
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
}

func newCluster(t *testing.T, q *reform.Querier) string {
	t.Helper()
	c := &models.OmCluster{
		ID: uuid.New().String(), Name: "rs0", Kind: models.OmClusterReplicaSet,
		Origin: models.OmClusterInstalled, Status: models.OmClusterInstalling, CreatedBy: "tester",
	}
	require.NoError(t, q.Insert(c))
	return c.ID
}

func newOperation(t *testing.T, q *reform.Querier, clusterID *string) string {
	t.Helper()
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "install", KindVersion: 1, ClusterID: clusterID,
		Status: models.OmOperationRunning, Trigger: models.OmTriggerUser, Principal: []byte(`{"kind":"user"}`),
	}
	require.NoError(t, q.Insert(op))
	return op.ID
}

func newReplicaSet(t *testing.T, q *reform.Querier, clusterID, rsID string) string {
	t.Helper()
	rs := &models.OmReplicaSet{
		ID: uuid.New().String(), ClusterID: clusterID, Name: "rs0", ReplicaSetID: &rsID,
		Role: models.OmReplicaSetRolePlain,
	}
	require.NoError(t, q.Insert(rs))
	return rs.ID
}

func TestOmAppendClusterState(t *testing.T) {
	db := omTestDB(t)
	clusterID := newCluster(t, db.Querier)
	opID := newOperation(t, db.Querier, &clusterID)

	var v1 *models.OmClusterState
	err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		s, err := models.AppendClusterState(tx, clusterID, 1, []byte(`{"fcv":"7.0"}`), opID)
		v1 = s
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), v1.Version)

	cluster := &models.OmCluster{ID: clusterID}
	require.NoError(t, db.Reload(cluster))
	require.NotNil(t, cluster.StateVersion)
	assert.Equal(t, int32(1), *cluster.StateVersion)

	err = db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		s, err := models.AppendClusterState(tx, clusterID, 1, []byte(`{"fcv":"7.0"}`), opID)
		if err == nil {
			assert.Equal(t, int32(2), s.Version)
		}
		return err
	})
	require.NoError(t, err)

	t.Run("rejects update and delete", func(t *testing.T) {
		_, err := db.Exec("UPDATE om_cluster_states SET schema_version = 9 WHERE cluster_id = $1 AND version = 1", clusterID)
		require.Error(t, err)
		_, err = db.Exec("DELETE FROM om_cluster_states WHERE cluster_id = $1 AND version = 1", clusterID)
		require.Error(t, err)
	})
}

func TestOmSecretEncryptedAtRest(t *testing.T) {
	db := omTestDB(t)
	clusterID := newCluster(t, db.Querier)
	rsID := newReplicaSet(t, db.Querier, clusterID, "rs0-id")

	const sentinel = "s3ntinel-automation-password"
	secret := &models.OmSecret{
		Purpose: "automation", ClusterID: &clusterID, ReplicaSetRef: &rsID,
		Generation: models.OmSecretCurrent, Payload: sentinel,
	}
	require.NoError(t, models.CreateOmSecret(db.Querier, secret))
	require.NotEmpty(t, secret.ID)
	assert.Equal(t, sentinel, secret.Payload)

	var raw string
	row := db.QueryRow("SELECT payload FROM om_secrets WHERE id = $1", secret.ID)
	require.NoError(t, row.Scan(&raw))
	assert.NotEqual(t, sentinel, raw)
	assert.NotContains(t, raw, sentinel)

	back, err := models.FindOmSecretByID(db.Querier, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, sentinel, back.Payload)
}

func TestOmCrossClusterReferenceRejected(t *testing.T) {
	db := omTestDB(t)
	clusterA := newCluster(t, db.Querier)
	clusterB := newCluster(t, db.Querier)
	opB := newOperation(t, db.Querier, &clusterB)
	rsA := newReplicaSet(t, db.Querier, clusterA, "rsA-id")

	err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		m := &models.OmClusterMember{
			ID: uuid.New().String(), ClusterID: clusterB, ReplicaSetRef: &rsA,
			Process: models.OmMemberMongod, Host: "node-x:27017", RSMemberID: new(int32(0)),
			State: models.OmMemberAccepted, AddedByOperation: opB,
		}
		return tx.Insert(m)
	})
	require.Error(t, err)
}

func TestOmLiveUniqueness(t *testing.T) {
	db := omTestDB(t)

	t.Run("mongo_cluster_id is live-unique", func(t *testing.T) {
		id := "shard-cluster-1"
		c1 := &models.OmCluster{ID: uuid.New().String(), Name: "a", Kind: models.OmClusterSharded, Origin: models.OmClusterInstalled, Status: models.OmClusterManaged, CreatedBy: "t", MongoClusterID: &id}
		require.NoError(t, db.Insert(c1))
		c2 := &models.OmCluster{ID: uuid.New().String(), Name: "b", Kind: models.OmClusterSharded, Origin: models.OmClusterImported, Status: models.OmClusterManaged, CreatedBy: "t", MongoClusterID: &id}
		require.Error(t, db.Insert(c2))
	})

	t.Run("replica_set_id is live-unique", func(t *testing.T) {
		cl := newCluster(t, db.Querier)
		_ = newReplicaSet(t, db.Querier, cl, "dup-rs-id")
		rs2 := &models.OmReplicaSet{ID: uuid.New().String(), ClusterID: cl, Name: "rs1", ReplicaSetID: new("dup-rs-id"), Role: models.OmReplicaSetRoleShard}
		require.Error(t, db.Insert(rs2))
	})
}

func TestOmLockConflictCarriesDetail(t *testing.T) {
	db := omTestDB(t)
	clusterID := newCluster(t, db.Querier)
	op1 := newOperation(t, db.Querier, &clusterID)
	op2 := newOperation(t, db.Querier, &clusterID)

	_, err := db.Exec("SELECT om_acquire_locks($1::uuid, ARRAY[ARRAY['cluster', $2, 'exclusive']]::text[][])", op1, clusterID)
	require.NoError(t, err)

	_, err = db.Exec("SELECT om_acquire_locks($1::uuid, ARRAY[ARRAY['cluster', $2, 'exclusive']]::text[][])", op2, clusterID)
	require.Error(t, err)
	var pqErr *pq.Error
	require.ErrorAs(t, err, &pqErr)
	assert.Equal(t, "55P03", string(pqErr.Code))
	assert.Contains(t, pqErr.Detail, "holder_operation_id="+op1)
	assert.Contains(t, pqErr.Detail, "requested_mode=exclusive")
}

func TestOmLeaderLease(t *testing.T) {
	db := omTestDB(t)
	acquire := func(who string) int64 {
		var epoch int64
		row := db.QueryRow("SELECT om_acquire_leader_lease($1, '30 seconds'::interval)", who)
		require.NoError(t, row.Scan(&epoch))
		return epoch
	}
	takeover := acquire("node-a")
	assert.Equal(t, int64(2), takeover)
	assert.Equal(t, int64(2), acquire("node-a"))
	assert.Equal(t, int64(3), acquire("node-b"))
}
