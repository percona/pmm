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
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type fakeCheckProvider struct {
	set   CheckSet
	facts map[string]any
	ages  map[string]time.Duration
	err   error
}

func (f fakeCheckProvider) CheckInputs(context.Context, *models.OmOperation, OperationKind) (CheckSet, map[string]any, map[string]time.Duration, error) {
	return f.set, f.facts, f.ages, f.err
}

type fakeExecutorProvider struct {
	result StepResultKind
	err    error
}

func (f fakeExecutorProvider) Executor(*models.OmOperation, OperationKind) (StepExecutor, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeExecutor{result: f.result}, nil
}

func runnerKind() OperationKind {
	return OperationKind{
		Kind: "test",
		LockSet: func(op *models.OmOperation) []LockResource {
			return []LockResource{{Kind: "cluster", ID: *op.ClusterID, Mode: models.OmLockExclusive}}
		},
		Steps: func(op *models.OmOperation) []StepPlan {
			return []StepPlan{{Phase: "run", Step: "do", TargetKind: models.OmStepTargetRun, TargetID: op.ID}}
		},
	}
}

func runnerRegistry(t *testing.T) *OperationRegistry {
	t.Helper()
	reg := NewOperationRegistry()
	require.NoError(t, reg.Register(runnerKind()))
	return reg
}

func newTestRunner(t *testing.T, db *reform.DB, reg *OperationRegistry, checks CheckProvider, executors StepExecutorProvider) *Runner {
	t.Helper()
	return NewRunner(db, logrus.WithField("test", t.Name()), reg, checks, executors, &fakeScanRequester{})
}

func runningWithLock(t *testing.T, db *reform.DB, clusterID string) *models.OmOperation {
	t.Helper()
	op := opWithStatus(t, db.Querier, clusterID, models.OmOperationRunning)
	require.NoError(t, acquireLocks(db.Querier, op.ID, []LockResource{{Kind: "cluster", ID: clusterID, Mode: models.OmLockExclusive}}))
	return op
}

func stepCount(t *testing.T, db *reform.DB, operationID string) int {
	t.Helper()
	var steps int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM om_operation_steps WHERE operation_id = $1", operationID).Scan(&steps))
	return steps
}

func TestRunnerDrivesCheckingThroughToSuccess(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	r := newTestRunner(t, db, runnerRegistry(t),
		fakeCheckProvider{set: passingCheckSet(), facts: readyFacts()},
		fakeExecutorProvider{result: StepResultOk})
	r.Tick(t.Context())

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationSucceeded), status)
	assert.Equal(t, 0, heldLocks(t, db, op.ID))
}

func TestRunnerDrivesRunningToSuccess(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := runningWithLock(t, db, clusterID)

	r := newTestRunner(t, db, runnerRegistry(t),
		fakeCheckProvider{}, fakeExecutorProvider{result: StepResultOk})
	r.Tick(t.Context())

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationSucceeded), status)
	assert.Equal(t, 0, heldLocks(t, db, op.ID))
}

func TestRunnerSkipsUnregisteredKind(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	r := newTestRunner(t, db, NewOperationRegistry(),
		fakeCheckProvider{set: passingCheckSet(), facts: readyFacts()},
		fakeExecutorProvider{result: StepResultOk})
	r.Tick(t.Context())

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationChecking), status)
}

func TestRunnerNonLeaderDoesNothing(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := runningWithLock(t, db, clusterID)

	r := newTestRunner(t, db, runnerRegistry(t),
		fakeCheckProvider{}, fakeExecutorProvider{result: StepResultOk}).
		WithLeader(fakeHA{leader: false, leaderID: "node-b"}, "driver")
	r.Tick(t.Context())

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationRunning), status)
	assert.Equal(t, 0, stepCount(t, db, op.ID))
}

func TestRunnerExecutorErrorLeavesOperationRunning(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := runningWithLock(t, db, clusterID)

	r := newTestRunner(t, db, runnerRegistry(t),
		fakeCheckProvider{}, fakeExecutorProvider{err: errors.New("no executor")})
	r.Tick(t.Context())

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationRunning), status)
	assert.Equal(t, 0, stepCount(t, db, op.ID))
}

func TestRunnerCheckInputsErrorLeavesOperationChecking(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	r := newTestRunner(t, db, runnerRegistry(t),
		fakeCheckProvider{err: errors.New("no facts")},
		fakeExecutorProvider{result: StepResultOk})
	r.Tick(t.Context())

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationChecking), status)
}
