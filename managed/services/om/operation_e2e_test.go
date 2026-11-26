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

// e2eKind is the worked-example kind the extension guide points to: it confirms for a user trigger,
// takes a cluster-exclusive lock, and runs one step; the end-to-end tests drive it through the surface.
func e2eKind() OperationKind {
	return OperationKind{
		Kind:                    "test",
		ConfirmsFor:             func(models.OmOperationTrigger) bool { return true },
		Resumable:               true,
		CancellableWhileRunning: true,
		LockSet: func(op *models.OmOperation) []LockResource {
			return []LockResource{{Kind: "cluster", ID: *op.ClusterID, Mode: models.OmLockExclusive}}
		},
		Steps: func(op *models.OmOperation) []StepPlan {
			return []StepPlan{{Phase: "run", Step: "apply", TargetKind: models.OmStepTargetRun, TargetID: op.ID}}
		},
	}
}

func e2eService(t *testing.T, db *reform.DB, reg *OperationRegistry) *Service {
	t.Helper()
	return &Service{db: db, l: logrus.WithField("test", t.Name()), operationRegistry: reg}
}

func e2eRegistry(t *testing.T) *OperationRegistry {
	t.Helper()
	reg := NewOperationRegistry()
	require.NoError(t, reg.Register(e2eKind()))
	return reg
}

func opStatus(t *testing.T, db *reform.DB, operationID string) (string, int32) {
	t.Helper()
	var status string
	var version int32
	require.NoError(t, db.QueryRow("SELECT status, version FROM om_operations WHERE id = $1", operationID).Scan(&status, &version))
	return status, version
}

func awaitingConfirmationOp(t *testing.T, db *reform.DB, clusterID string) *models.OmOperation {
	t.Helper()
	digest := "sha256:reviewed"
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: "test", KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationAwaitingConfirmation, Trigger: models.OmTriggerUser,
		Principal: []byte(`{}`), ReportDigest: &digest,
	}
	require.NoError(t, db.Insert(op))
	return op
}

func TestE2EOperationLifecycle(t *testing.T) {
	db := lockTestDB(t)
	reg := e2eRegistry(t)
	svc := e2eService(t, db, reg)

	t.Run("confirm releases a reviewed operation and the driver runs it to success", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		op := checkingOp(t, db.Querier, clusterID)
		runner := newTestRunner(t, db, reg,
			fakeCheckProvider{set: passingCheckSet(), facts: readyFacts()},
			fakeExecutorProvider{result: StepResultOk})

		runner.Tick(t.Context())
		state, version := opStatus(t, db, op.ID)
		require.Equal(t, string(models.OmOperationAwaitingConfirmation), state)

		resp, err := svc.ConfirmOperation(t.Context(), &omv1.ConfirmOperationRequest{OperationId: op.ID, Version: version})
		require.NoError(t, err)
		require.Equal(t, string(models.OmOperationRunning), resp.Operation.Status)

		runner.Tick(t.Context())
		state, _ = opStatus(t, db, op.ID)
		assert.Equal(t, string(models.OmOperationSucceeded), state)
		assert.Equal(t, 0, heldLocks(t, db, op.ID))
	})

	t.Run("a stale confirm is rejected and changes nothing", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		op := awaitingConfirmationOp(t, db, clusterID)

		_, err := svc.ConfirmOperation(t.Context(), &omv1.ConfirmOperationRequest{OperationId: op.ID, Version: 99})
		require.Error(t, err)

		state, _ := opStatus(t, db, op.ID)
		assert.Equal(t, string(models.OmOperationAwaitingConfirmation), state)
		assert.Equal(t, 0, heldLocks(t, db, op.ID))
	})

	t.Run("a confirm whose cluster lock is held is queued", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		holder := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
		require.NoError(t, acquireLocks(db.Querier, holder.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))
		op := awaitingConfirmationOp(t, db, clusterID)

		_, err := svc.ConfirmOperation(t.Context(), &omv1.ConfirmOperationRequest{OperationId: op.ID, Version: 1})
		require.Equal(t, codes.Aborted, status.Code(err), "a queued confirm reports Aborted, not success")

		opState, _ := opStatus(t, db, op.ID)
		assert.Equal(t, string(models.OmOperationQueued), opState)
		assert.Equal(t, 0, heldLocks(t, db, op.ID))
	})

	t.Run("a deposed leader is fenced; the current leader has the only effect", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
		require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))
		kind, _ := reg.Kind("test")

		deposed := acquireLease(t, db.Querier, "driver-a")
		acquireLease(t, db.Querier, "driver-b")

		outcome, err := driveRunning(t.Context(), db, op.ID, kind, &fakeExecutor{result: StepResultOk}, deposed)
		require.NoError(t, err)
		require.Equal(t, RunFenced, outcome)
		assert.Equal(t, 0, stepCount(t, db, op.ID))

		current := acquireLease(t, db.Querier, "driver-b")
		outcome, err = driveRunning(t.Context(), db, op.ID, kind, &fakeExecutor{result: StepResultOk}, current)
		require.NoError(t, err)
		require.Equal(t, RunStepSucceeded, outcome)
		assert.Equal(t, 1, stepCount(t, db, op.ID))

		var key string
		require.NoError(t, db.QueryRow("SELECT idempotency_key FROM om_operation_steps WHERE operation_id = $1", op.ID).Scan(&key))
		plan := kind.Steps(op)[0]
		assert.Equal(t, plan.idempotencyKey(op.ID, 1), key, "the dispatch uses the deterministic key, so a re-send reuses it")

		outcome, err = driveRunning(t.Context(), db, op.ID, kind, &fakeExecutor{result: StepResultOk}, current)
		require.NoError(t, err)
		assert.Equal(t, RunOperationSucceeded, outcome)
		assert.Equal(t, 1, stepCount(t, db, op.ID), "no second attempt is opened for a succeeded step")
	})

	t.Run("resume keeps the locks a failed operation holds", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationFailed)
		require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

		resp, err := svc.ResumeOperation(t.Context(), &omv1.ResumeOperationRequest{OperationId: op.ID, Version: 1})
		require.NoError(t, err)
		assert.Equal(t, string(models.OmOperationRunning), resp.Operation.Status)
		assert.Equal(t, 1, heldLocks(t, db, op.ID), "resume re-acquires nothing and keeps the held lock")
	})

	t.Run("abandon releases the locks a failed operation holds", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		op := opWithStatus(t, db.Querier, clusterID, models.OmOperationFailed)
		require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))

		resp, err := svc.AbandonOperation(t.Context(), &omv1.AbandonOperationRequest{OperationId: op.ID, Version: 1})
		require.NoError(t, err)
		assert.Equal(t, string(models.OmOperationAbandoned), resp.Operation.Status)
		assert.Equal(t, 0, heldLocks(t, db, op.ID))
	})

	t.Run("a secret-typed check fact never seals into the report", func(t *testing.T) {
		clusterID := lockCluster(t, db.Querier)
		op := checkingOp(t, db.Querier, clusterID)
		secretCheck := CheckSet{Facts: []CheckFact{{Name: "admin_password", Secret: true}}}
		runner := newTestRunner(t, db, reg,
			fakeCheckProvider{set: secretCheck, facts: map[string]any{"admin_password": "s3cr3t"}},
			fakeExecutorProvider{result: StepResultOk})

		runner.Tick(t.Context())

		state, _ := opStatus(t, db, op.ID)
		assert.Equal(t, string(models.OmOperationChecking), state)
		var digest *string
		require.NoError(t, db.QueryRow("SELECT report_digest FROM om_operations WHERE id = $1", op.ID).Scan(&digest))
		assert.Nil(t, digest, "the secret fact is rejected, so no report digest is sealed")

		got, err := svc.GetOperation(t.Context(), &omv1.GetOperationRequest{OperationId: op.ID})
		require.NoError(t, err)
		assert.Equal(t, op.ID, got.Operation.Id)
	})
}
