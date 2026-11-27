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
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

func uninstallService(t *testing.T, db *reform.DB) *Service {
	t.Helper()
	return &Service{db: db, l: logrus.WithField("test", t.Name())}
}

func insertUninstallMarker(t *testing.T, db *reform.DB, clusterID, runID string) {
	t.Helper()
	opID := lockOperation(t, db.Querier, clusterID)
	err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		return models.InsertInstallMarker(tx, uuid.New().String(), clusterID, opID, runID)
	})
	require.NoError(t, err)
}

func TestTriggerUninstallRejectsIneligibleClusters(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)

	insert := func(origin models.OmClusterOrigin, kind models.OmClusterKind, st models.OmClusterStatus) string {
		id := uuid.New().String()
		require.NoError(t, db.Insert(&models.OmCluster{ID: id, Name: "rs0", Kind: kind, Origin: origin, Status: st, CreatedBy: "t"}))
		return id
	}

	for _, tc := range []struct {
		name   string
		origin models.OmClusterOrigin
		kind   models.OmClusterKind
		status models.OmClusterStatus
	}{
		{"imported", models.OmClusterImported, models.OmClusterReplicaSet, models.OmClusterManaged},
		{"sharded", models.OmClusterInstalled, models.OmClusterSharded, models.OmClusterManaged},
		{"already removing", models.OmClusterInstalled, models.OmClusterReplicaSet, models.OmClusterRemoving},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := insert(tc.origin, tc.kind, tc.status)
			_, err := svc.TriggerUninstall(t.Context(), &omv1.TriggerUninstallRequest{ClusterId: id})
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		})
	}

	t.Run("missing cluster", func(t *testing.T) {
		_, err := svc.TriggerUninstall(t.Context(), &omv1.TriggerUninstallRequest{ClusterId: uuid.New().String()})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

func TestTriggerUninstallRefusesWithoutMarker(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	clusterID := lockCluster(t, db.Querier)

	_, err := svc.TriggerUninstall(t.Context(), &omv1.TriggerUninstallRequest{ClusterId: clusterID})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestTriggerUninstallIsIdempotentPerCluster(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	clusterID := lockCluster(t, db.Querier)

	existing := uuid.New().String()
	require.NoError(t, db.Insert(&models.OmOperation{
		ID: existing, Kind: uninstallKind, KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationChecking, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}))

	resp, err := svc.TriggerUninstall(t.Context(), &omv1.TriggerUninstallRequest{ClusterId: clusterID})
	require.NoError(t, err)
	assert.Equal(t, existing, resp.OperationId)

	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_operations WHERE kind = $1 AND cluster_id = $2", uninstallKind, clusterID).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestTriggerUninstallRefusesLockedCluster(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	clusterID := lockCluster(t, db.Querier)
	insertUninstallMarker(t, db, clusterID, "run-1")
	holder := lockOperation(t, db.Querier, clusterID)
	require.NoError(t, acquireLocks(db.Querier, holder, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

	_, err := svc.TriggerUninstall(t.Context(), &omv1.TriggerUninstallRequest{ClusterId: clusterID})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), holder)
}

func TestTriggerUninstallSealsRemovalPlan(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	clusterID := lockCluster(t, db.Querier)
	opID := lockOperation(t, db.Querier, clusterID)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "node00-seal"})
	require.NoError(t, err)
	installOpID := uuid.New().String()
	installParamsRaw, err := json.Marshal(installParams{OS: "ubuntu"})
	require.NoError(t, err)
	require.NoError(t, db.Insert(&models.OmOperation{
		ID: installOpID, Kind: installKind, KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationSucceeded, Trigger: models.OmTriggerUser, Principal: []byte(`{}`), Params: installParamsRaw,
	}))
	_, err = db.Exec("UPDATE om_clusters SET created_by = $1 WHERE id = $2", installOpID, clusterID)
	require.NoError(t, err)

	rs := &models.OmReplicaSet{ID: uuid.New().String(), ClusterID: clusterID, Name: "rs0", Role: models.OmReplicaSetRolePlain}
	require.NoError(t, db.Insert(rs))
	rsMemberID := int32(0)
	member := &models.OmClusterMember{
		ID: uuid.New().String(), ClusterID: clusterID, ReplicaSetRef: &rs.ID, Process: models.OmMemberMongod,
		Host: "node00", NodeID: &node.NodeID, RSMemberID: &rsMemberID, State: models.OmMemberAccepted, AddedByOperation: opID,
	}
	require.NoError(t, db.Insert(member))
	insertUninstallMarker(t, db, clusterID, "run-77")

	accepted := &models.AcceptedState{
		FCV:         "7.0",
		Auth:        models.AcceptedAuth{Mechanisms: []string{"SCRAM-SHA-256"}, Membership: "keyfile"},
		ReplicaSets: map[string]models.AcceptedReplicaSet{"rs0": {}},
		Members: map[string]models.AcceptedMember{
			member.ID: {
				Process:       "mongod",
				Host:          "node00",
				BinaryVersion: "7.0.14",
				Control:       models.AcceptedControl{Method: "systemd", Unit: "mongod"},
				Install:       models.AcceptedInstall{Method: "packages", Packages: []string{"percona-server-mongodb"}},
				Config:        models.AcceptedConfig{File: "/etc/mongod.conf"},
				ConfigSHA256:  "abc",
				RSMemberID:    &rsMemberID,
				DataPath:      "/var/lib/mongodb",
				KeyfilePath:   "/etc/mongod.keyfile",
				LogPath:       "/var/log/mongod.log",
			},
		},
	}
	doc, err := models.MarshalAcceptedState(accepted)
	require.NoError(t, err)
	err = db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		_, appendErr := models.AppendClusterState(tx, clusterID, models.AcceptedStateSchemaVersion, doc, opID)
		return appendErr
	})
	require.NoError(t, err)

	resp, err := svc.TriggerUninstall(t.Context(), &omv1.TriggerUninstallRequest{ClusterId: clusterID})
	require.NoError(t, err)
	require.NotEmpty(t, resp.OperationId)

	var gotStatus string
	var paramsRaw []byte
	require.NoError(t, db.QueryRow("SELECT status, params FROM om_operations WHERE id = $1", resp.OperationId).Scan(&gotStatus, &paramsRaw))
	assert.Equal(t, string(models.OmOperationChecking), gotStatus)

	var params uninstallParams
	require.NoError(t, json.Unmarshal(paramsRaw, &params))
	assert.Equal(t, clusterID, params.ClusterID)
	assert.Equal(t, "rs0", params.ClusterName)
	assert.Equal(t, "run-77", params.InstallRunID)
	assert.Equal(t, "ubuntu", params.OS)
	require.Len(t, params.Members, 1)

	sealed := params.Members[0]
	assert.Equal(t, member.ID, sealed.MemberID)
	assert.Equal(t, "node00", sealed.Host)
	assert.Equal(t, node.NodeID, sealed.NodeID)
	assert.Equal(t, []string{"percona-server-mongodb"}, sealed.Packages)
	assert.Equal(t, "/var/lib/mongodb", sealed.DataPath)
	assert.Equal(t, "/etc/mongod.conf", sealed.ConfigFile)
	assert.Equal(t, "/etc/mongod.keyfile", sealed.KeyfilePath)
	assert.Equal(t, "/var/log/mongod.log", sealed.LogPath)
	assert.Equal(t, "mongod", sealed.ControlUnit)
	assert.Equal(t, uninstallMarkerPath, sealed.MarkerPath)
}
