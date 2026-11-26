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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

func operationAPIService(t *testing.T, db *reform.DB) *Service {
	t.Helper()
	reg := NewOperationRegistry()
	require.NoError(t, reg.Register(OperationKind{
		Kind:                    "test",
		CancellableWhileRunning: true,
		LockSet: func(op *models.OmOperation) []LockResource {
			return []LockResource{{Kind: "cluster", ID: *op.ClusterID, Mode: models.OmLockExclusive}}
		},
	}))
	return &Service{db: db, l: logrus.WithField("test", t.Name()), operationRegistry: reg}
}

func TestListOperationsHandler(t *testing.T) {
	db := lockTestDB(t)
	svc := operationAPIService(t, db)
	clusterA := lockCluster(t, db.Querier)
	clusterB := lockCluster(t, db.Querier)
	lockOperation(t, db.Querier, clusterA)
	lockOperation(t, db.Querier, clusterA)
	lockOperation(t, db.Querier, clusterB)

	scoped, err := svc.ListOperations(t.Context(), &omv1.ListOperationsRequest{ClusterId: &clusterA})
	require.NoError(t, err)
	assert.Len(t, scoped.Operations, 2)
	assert.Equal(t, clusterA, scoped.Operations[0].GetClusterId())
	assert.Equal(t, string(models.OmOperationRunning), scoped.Operations[0].Status)

	all, err := svc.ListOperations(t.Context(), &omv1.ListOperationsRequest{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all.Operations), 3)
}

func TestGetOperationHandler(t *testing.T) {
	db := lockTestDB(t)
	svc := operationAPIService(t, db)
	clusterID := lockCluster(t, db.Querier)
	opID := lockOperation(t, db.Querier, clusterID)
	dispatchingStep(t, db.Querier, opID)

	got, err := svc.GetOperation(t.Context(), &omv1.GetOperationRequest{OperationId: opID})
	require.NoError(t, err)
	assert.Equal(t, opID, got.Operation.Id)
	require.Len(t, got.Steps, 1)
	assert.Equal(t, string(models.OmStepDispatching), got.Steps[0].Status)

	t.Run("a missing operation is NotFound", func(t *testing.T) {
		_, err := svc.GetOperation(t.Context(), &omv1.GetOperationRequest{OperationId: uuid.New().String()})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

func TestCancelOperationHandler(t *testing.T) {
	db := lockTestDB(t)
	svc := operationAPIService(t, db)
	clusterID := lockCluster(t, db.Querier)
	op := opWithStatus(t, db.Querier, clusterID, models.OmOperationQueued)
	require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

	resp, err := svc.CancelOperation(t.Context(), &omv1.CancelOperationRequest{OperationId: op.ID, Version: 1})
	require.NoError(t, err)
	assert.Equal(t, string(models.OmOperationCancelled), resp.Operation.Status)
	assert.Equal(t, operationPrincipal, resp.Operation.GetCancelledBy())
	assert.Equal(t, 0, heldLocks(t, db, op.ID))

	t.Run("a stale version is FailedPrecondition", func(t *testing.T) {
		other := opWithStatus(t, db.Querier, clusterID, models.OmOperationQueued)
		_, err := svc.CancelOperation(t.Context(), &omv1.CancelOperationRequest{OperationId: other.ID, Version: 99})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("an unregistered kind is FailedPrecondition", func(t *testing.T) {
		bare := &Service{db: db, l: logrus.WithField("test", t.Name()), operationRegistry: NewOperationRegistry()}
		other := opWithStatus(t, db.Querier, clusterID, models.OmOperationQueued)
		_, err := bare.CancelOperation(t.Context(), &omv1.CancelOperationRequest{OperationId: other.ID, Version: 1})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

func TestResumeAndAbandonHandlers(t *testing.T) {
	db := lockTestDB(t)
	svc := operationAPIService(t, db)
	clusterID := lockCluster(t, db.Querier)

	t.Run("resume returns a failed operation to running", func(t *testing.T) {
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationFailed)
		resp, err := svc.ResumeOperation(t.Context(), &omv1.ResumeOperationRequest{OperationId: op.ID, Version: 1})
		require.NoError(t, err)
		assert.Equal(t, string(models.OmOperationRunning), resp.Operation.Status)
	})

	t.Run("abandon releases a failed operation", func(t *testing.T) {
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationFailed)
		require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))
		resp, err := svc.AbandonOperation(t.Context(), &omv1.AbandonOperationRequest{OperationId: op.ID, Version: 1})
		require.NoError(t, err)
		assert.Equal(t, string(models.OmOperationAbandoned), resp.Operation.Status)
		assert.Equal(t, 0, heldLocks(t, db, op.ID))
	})

	t.Run("a stale version does not resume", func(t *testing.T) {
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationFailed)
		_, err := svc.ResumeOperation(t.Context(), &omv1.ResumeOperationRequest{OperationId: op.ID, Version: 99})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

func TestConfirmOperationHandler(t *testing.T) {
	db := lockTestDB(t)
	svc := operationAPIService(t, db)
	clusterID := lockCluster(t, db.Querier)
	digest := "sha256:reviewed"
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationAwaitingConfirmation, Trigger: models.OmTriggerUser,
		Principal: []byte(`{}`), ReportDigest: &digest,
	}
	require.NoError(t, db.Insert(op))

	resp, err := svc.ConfirmOperation(t.Context(), &omv1.ConfirmOperationRequest{OperationId: op.ID, Version: 1})
	require.NoError(t, err)
	assert.Equal(t, string(models.OmOperationRunning), resp.Operation.Status)
	assert.Equal(t, 1, heldLocks(t, db, op.ID))

	t.Run("a stale version does not confirm", func(t *testing.T) {
		other := &models.OmOperation{
			ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
			Status: models.OmOperationAwaitingConfirmation, Trigger: models.OmTriggerUser,
			Principal: []byte(`{}`), ReportDigest: &digest,
		}
		require.NoError(t, db.Insert(other))
		_, err := svc.ConfirmOperation(t.Context(), &omv1.ConfirmOperationRequest{OperationId: other.ID, Version: 99})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("a non-awaiting operation is FailedPrecondition", func(t *testing.T) {
		running := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
		_, err := svc.ConfirmOperation(t.Context(), &omv1.ConfirmOperationRequest{OperationId: running.ID, Version: 1})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}
