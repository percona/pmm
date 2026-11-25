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

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// memberScanFacts are the credential-free facts the on-host scan produces for a member.
func memberScanFacts() map[string]any {
	return map[string]any{
		"binary_version":         "8.0.38",
		"database.config_sha256": "abc123",
		"database.set_name":      "rs0",
	}
}

// memberMetricFacts are the facts the exporter metrics carry for a member's service.
func memberMetricFacts() map[string]any {
	return map[string]any{
		fieldFCV:             "8.0",
		fieldRSConfigVersion: float64(1),
		fieldRSConfigTerm:    float64(2),
	}
}

type fakeMetricsFactReader struct {
	facts map[string]map[string]any
	err   error
}

func (f *fakeMetricsFactReader) ServiceMetricsFacts(_ context.Context, serviceIDs []string) (map[string]map[string]any, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]map[string]any, len(serviceIDs))
	for _, id := range serviceIDs {
		if facts, ok := f.facts[id]; ok {
			out[id] = facts
		}
	}
	return out, nil
}

func TestBuildInstallAcceptedState(t *testing.T) {
	rsMemberID := int32(0)
	serviceID := "svc-m0"
	members := []*models.OmClusterMember{{ID: "m0", Host: "node00", RSMemberID: &rsMemberID, ServiceID: &serviceID}}
	reader := &fakeFactReader{facts: map[string]map[string]any{"m0": memberScanFacts()}}
	metrics := &fakeMetricsFactReader{facts: map[string]map[string]any{serviceID: memberMetricFacts()}}

	state, ready := buildInstallAcceptedState(t.Context(), reader, metrics, members)
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
		&fakeFactReader{facts: map[string]map[string]any{"m0": {"binary_version": "8.0.38"}}}, metrics, members)
	assert.False(t, ready, "an incomplete observation is not ready")

	_, ready = buildInstallAcceptedState(t.Context(), &fakeFactReader{}, metrics, members)
	assert.False(t, ready, "a member with no observation yet is not ready")

	_, ready = buildInstallAcceptedState(t.Context(), reader, &fakeMetricsFactReader{}, members)
	assert.False(t, ready, "a member whose metrics are not in yet is not ready")

	unlinked := []*models.OmClusterMember{{ID: "m0", Host: "node00", RSMemberID: &rsMemberID}}
	_, ready = buildInstallAcceptedState(t.Context(), reader, metrics, unlinked)
	assert.False(t, ready, "a member not yet linked to a service is not ready")
}

func TestReconcileWritesAcceptedStateOnce(t *testing.T) {
	db := lockTestDB(t)
	clusterID := uuid.New().String()
	opID := uuid.New().String()

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "node00"})
	require.NoError(t, err)
	port := uint16(27017)
	address := "127.0.0.1"
	service, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "node00-mongod", NodeID: node.NodeID, Address: &address, Port: &port,
	})
	require.NoError(t, err)

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
		Host: "node00", RSMemberID: &rsMemberID, ServiceID: &service.ServiceID, State: models.OmMemberAccepted, AddedByOperation: opID,
	}
	require.NoError(t, db.Insert(member))

	reader := &fakeFactReader{facts: map[string]map[string]any{member.ID: memberScanFacts()}}
	metrics := &fakeMetricsFactReader{facts: map[string]map[string]any{service.ServiceID: memberMetricFacts()}}
	scheduler := NewScanScheduler(db, logrus.WithField("test", t.Name()), &fakeClusterScanRequester{}).
		WithReader(reader).WithMetricsReader(metrics)

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
		WithReader(&fakeFactReader{}).WithMetricsReader(&fakeMetricsFactReader{})
	scheduler.Tick(t.Context())

	var version *int32
	require.NoError(t, db.QueryRow("SELECT state_version FROM om_clusters WHERE id = $1", clusterID).Scan(&version))
	assert.Nil(t, version, "no accepted state is written until the members' facts are in")
}
