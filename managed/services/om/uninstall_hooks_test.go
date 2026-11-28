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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

func TestDeregisterClusterServices(t *testing.T) {
	db := lockTestDB(t)
	clusterID := uuid.New().String()
	opID := uuid.New().String()

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "node00-dereg"})
	require.NoError(t, err)
	port := uint16(27017)
	address := "127.0.0.1"
	service, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "node00-mongod-dereg", NodeID: node.NodeID, Address: &address, Port: &port,
	})
	require.NoError(t, err)

	require.NoError(t, db.Insert(&models.OmCluster{
		ID: clusterID, Name: "rs0-dereg", Kind: models.OmClusterReplicaSet,
		Origin: models.OmClusterInstalled, Status: models.OmClusterManaged, CreatedBy: opID,
	}))
	require.NoError(t, db.Insert(&models.OmOperation{
		ID: opID, Kind: installKind, KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationSucceeded, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}))
	rs := &models.OmReplicaSet{ID: uuid.New().String(), ClusterID: clusterID, Name: "rs0-dereg", Role: models.OmReplicaSetRolePlain}
	require.NoError(t, db.Insert(rs))
	rsMemberID := int32(0)
	member := &models.OmClusterMember{
		ID: uuid.New().String(), ClusterID: clusterID, ReplicaSetRef: &rs.ID, Process: models.OmMemberMongod,
		Host: "node00-dereg", RSMemberID: &rsMemberID, ServiceID: &service.ServiceID,
		State: models.OmMemberAccepted, AddedByOperation: opID,
	}
	require.NoError(t, db.Insert(member))

	run := func() error {
		return db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
			return deregisterClusterServices(tx, clusterID)
		})
	}

	require.NoError(t, run())
	_, err = models.FindServiceByID(db.Querier, service.ServiceID)
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err), "the member's service was deregistered")

	require.NoError(t, run(), "a re-run skips the already-removed service")
}
