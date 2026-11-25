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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func memberFacts() map[string]any {
	return map[string]any{
		"binary_version":             "8.0.38",
		"database.fcv":               "8.0",
		"database.config_sha256":     "abc123",
		"database.set_name":          "rs0",
		"database.rs_config_version": float64(1),
		"database.rs_config_term":    float64(2),
	}
}

func TestBuildInstallAcceptedState(t *testing.T) {
	rsMemberID := int32(0)
	members := []*models.OmClusterMember{{ID: "m0", Host: "node00", RSMemberID: &rsMemberID}}
	reader := &fakeFactReader{facts: map[string]map[string]any{"m0": memberFacts()}}

	state, ready := buildInstallAcceptedState(t.Context(), reader, members)
	require.True(t, ready)
	assert.Equal(t, "8.0", state.FCV)
	assert.Equal(t, []string{installAuthMechanism}, state.Auth.Mechanisms)
	require.Contains(t, state.ReplicaSets, "rs0")
	assert.Equal(t, int32(1), state.ReplicaSets["rs0"].ObservedFrom.ConfigVersion)
	assert.Equal(t, int64(2), state.ReplicaSets["rs0"].ObservedFrom.Term)
	member := state.Members["m0"]
	assert.Equal(t, "8.0.38", member.BinaryVersion)
	assert.Equal(t, "abc123", member.ConfigSHA256)
	assert.Equal(t, installMethod, member.Install.Method)
	assert.Equal(t, "node00", member.Host)

	_, err := models.MarshalAcceptedState(state)
	require.NoError(t, err, "the built state is a valid document")

	_, ready = buildInstallAcceptedState(t.Context(),
		&fakeFactReader{facts: map[string]map[string]any{"m0": {"binary_version": "8.0.38"}}}, members)
	assert.False(t, ready, "an incomplete observation is not ready")

	_, ready = buildInstallAcceptedState(t.Context(), &fakeFactReader{}, members)
	assert.False(t, ready, "a member with no observation yet is not ready")
}

func TestReconcileWritesAcceptedStateOnce(t *testing.T) {
	db := lockTestDB(t)
	clusterID := uuid.New().String()
	opID := uuid.New().String()

	require.NoError(t, db.Insert(&models.OmCluster{
		ID: clusterID, Name: "rs0", Kind: models.OmClusterReplicaSet,
		Origin: models.OmClusterInstalled, Status: models.OmClusterManaged, CreatedBy: opID,
	}))
	require.NoError(t, db.Insert(&models.OmOperation{
		ID: opID, Kind: installKind, KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationSucceeded, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}))
	rs := &models.OmReplicaSet{ID: uuid.New().String(), ClusterID: clusterID, Name: "rs0", Role: models.OmReplicaSetRolePlain}
	require.NoError(t, db.Insert(rs))
	rsMemberID := int32(0)
	member := &models.OmClusterMember{
		ID: uuid.New().String(), ClusterID: clusterID, ReplicaSetRef: &rs.ID, Process: models.OmMemberMongod,
		Host: "node00", RSMemberID: &rsMemberID, State: models.OmMemberAccepted, AddedByOperation: opID,
	}
	require.NoError(t, db.Insert(member))

	reader := &fakeFactReader{facts: map[string]map[string]any{member.ID: memberFacts()}}
	scheduler := NewScanScheduler(db, logrus.WithField("test", t.Name()), &fakeClusterScanRequester{}).WithReader(reader)

	scheduler.Tick(t.Context())

	var version *int32
	require.NoError(t, db.QueryRow("SELECT state_version FROM om_clusters WHERE id = $1", clusterID).Scan(&version))
	require.NotNil(t, version, "the reconcile wrote an accepted state")
	assert.Equal(t, int32(1), *version)

	var document []byte
	require.NoError(t, db.QueryRow("SELECT document FROM om_cluster_states WHERE cluster_id = $1 AND version = 1", clusterID).Scan(&document))
	state, err := models.UnmarshalAcceptedState(document)
	require.NoError(t, err)
	assert.Equal(t, "8.0", state.FCV)

	scheduler.Tick(t.Context())
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_cluster_states WHERE cluster_id = $1", clusterID).Scan(&count))
	assert.Equal(t, 1, count, "a cluster that already has an accepted state is not written again")
}

func TestReconcileWaitsForFacts(t *testing.T) {
	db := lockTestDB(t)
	clusterID := clusterWithStatus(t, db.Querier, models.OmClusterManaged)

	scheduler := NewScanScheduler(db, logrus.WithField("test", t.Name()), &fakeClusterScanRequester{}).
		WithReader(&fakeFactReader{})
	scheduler.Tick(t.Context())

	var version *int32
	require.NoError(t, db.QueryRow("SELECT state_version FROM om_clusters WHERE id = $1", clusterID).Scan(&version))
	assert.Nil(t, version, "no accepted state is written until the members' facts are in")
}
