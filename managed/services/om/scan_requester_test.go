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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const scanMemberPort = uint16(27017)

func scanMember(t *testing.T, db *reform.DB, clusterID, serviceID string) string {
	t.Helper()
	nodeID := uuid.New().String()
	_, err := db.Exec(
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ($1, 'generic', $2, '', '', '', '', now(), now())",
		nodeID, "node-"+serviceID)
	require.NoError(t, err)
	_, err = db.Exec(
		"INSERT INTO services (service_id, service_type, service_name, node_id, address, port, environment, cluster, replication_set, external_group, created_at, updated_at) "+
			"VALUES ($1, 'mongodb', $2, $3, $4, $5, '', '', '', '', now(), now())",
		serviceID, "svc-"+serviceID, nodeID, "host-"+serviceID, scanMemberPort)
	require.NoError(t, err)
	replicaSet := &models.OmReplicaSet{
		ID: uuid.New().String(), ClusterID: clusterID, Name: "rs-" + serviceID, Role: models.OmReplicaSetRolePlain,
	}
	require.NoError(t, db.Insert(replicaSet))
	rsMemberID := int32(1)
	member := &models.OmClusterMember{
		ID: uuid.New().String(), ClusterID: clusterID, ReplicaSetRef: &replicaSet.ID, Process: models.OmMemberMongod,
		Host: "host-" + serviceID, RSMemberID: &rsMemberID, State: models.OmMemberAccepted,
		NodeID: &nodeID, ServiceID: &serviceID, AddedByOperation: lockOperation(t, db.Querier, clusterID),
	}
	require.NoError(t, db.Insert(member))
	return member.ID
}

// The requester fulfils both the driver's rescan seam and the scheduler's per-cluster seam.
var (
	_ ScanRequester        = (*inventoryScanRequester)(nil)
	_ clusterScanRequester = (*inventoryScanRequester)(nil)
)

func scanRequesterFor(t *testing.T, db *reform.DB, handler http.HandlerFunc) *inventoryScanRequester {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &extensionsClient{baseURL: server.URL, token: "test-token", http: server.Client()}
	return newInventoryScanRequester(db, logrus.WithField("test", t.Name()), client)
}

func TestRequestScanPostsOperationRequest(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody scanRequest
	requester := scanRequesterFor(t, nil, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	})

	err := requester.RequestScan("op-1", scanProfileImportCheck, "op-1:3")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/apps/om_inventory/scans", gotPath)
	assert.Equal(t, scanProfileImportCheck, gotBody.Profile)
	assert.Equal(t, "op-1:3", gotBody.IdempotencyKey)
	assert.Equal(t, "op-1", gotBody.OperationID)
	assert.Empty(t, gotBody.ClusterID)
	assert.Empty(t, gotBody.Subjects)
}

func TestRequestScanPostsInstallNodeSubjects(t *testing.T) {
	db := lockTestDB(t)
	params, err := json.Marshal(installParams{
		NodeIDs: []string{"node-a", "node-b"},
		Hosts:   map[string]string{"node-a": "exec-a", "node-b": "exec-b"},
	})
	require.NoError(t, err)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: installKind, KindVersion: 1,
		Status: models.OmOperationChecking, Version: 1, ParamsRevision: 1,
		Trigger: models.OmTriggerUser, Principal: []byte(`{}`), Params: params, ErrorFields: []byte("{}"),
	}
	require.NoError(t, db.Insert(op))

	var gotBody scanRequest
	requester := scanRequesterFor(t, db, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	})

	err = requester.RequestScan(op.ID, scanProfileInstallCheck, op.ID+":1")
	require.NoError(t, err)
	assert.Equal(t, scanProfileInstallCheck, gotBody.Profile)
	assert.Equal(t, op.ID, gotBody.OperationID)
	require.Len(t, gotBody.Subjects, 2)
	byKey := make(map[string]string, 2)
	for _, subject := range gotBody.Subjects {
		assert.Equal(t, subjectKindInstallNode, subject.SubjectKind)
		byKey[subject.SubjectKey] = subject.Host
	}
	assert.Equal(t, map[string]string{"node-a": "exec-a", "node-b": "exec-b"}, byKey)
}

func TestRequestClusterScanPostsMemberSubjects(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	memberA := scanMember(t, db, clusterID, uuid.New().String())
	memberB := scanMember(t, db, clusterID, uuid.New().String())

	var gotBody scanRequest
	requester := scanRequesterFor(t, db, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	})

	err := requester.RequestClusterScan(t.Context(), clusterID, scanProfileManaged, clusterID+":managed:42")
	require.NoError(t, err)
	assert.Equal(t, scanProfileManaged, gotBody.Profile)
	assert.Equal(t, clusterID, gotBody.ClusterID)
	assert.Empty(t, gotBody.OperationID)
	require.Len(t, gotBody.Subjects, 2)
	keys := []string{gotBody.Subjects[0].SubjectKey, gotBody.Subjects[1].SubjectKey}
	assert.ElementsMatch(t, []string{memberA, memberB}, keys)
	for _, subject := range gotBody.Subjects {
		assert.Equal(t, subjectKindMember, subject.SubjectKind)
		assert.NotEmpty(t, subject.Host)
		assert.NotEmpty(t, subject.ServiceID)
		require.NotNil(t, subject.Port)
		assert.Equal(t, scanMemberPort, *subject.Port)
	}
}

func TestRequestScanSurfacesAnExtensionsFailure(t *testing.T) {
	requester := scanRequesterFor(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	err := requester.RequestScan("op-1", scanProfileManaged, "op-1:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "managed")
}
