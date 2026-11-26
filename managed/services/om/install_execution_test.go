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
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type fakeBootstrapRunner struct {
	triggered *extensionsTriggerBootstrapRunRequest
	run       *extensionsBootstrapRun
}

func (f *fakeBootstrapRunner) triggerRun(_ context.Context, req extensionsTriggerBootstrapRunRequest) (*extensionsBootstrapRun, error) {
	f.triggered = &req
	return &extensionsBootstrapRun{ID: "run-1", Status: bootstrapRunRunning}, nil
}

func (f *fakeBootstrapRunner) getRun(_ context.Context, _ string) (*extensionsBootstrapRun, error) {
	return f.run, nil
}

func installOpWithPlan(t *testing.T, db *reform.DB) *models.OmOperation {
	t.Helper()
	params, err := json.Marshal(installParams{
		NodeIDs:        []string{"node-a"},
		Hosts:          map[string]string{"node-a": "exec-a"},
		OS:             "ubuntu",
		ReplicaSetName: "rs0",
		MongoDBVersion: "7.0.8",
	})
	require.NoError(t, err)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: installKind, KindVersion: 1,
		Status: models.OmOperationRunning, Version: 1, ParamsRevision: 1,
		Trigger: models.OmTriggerUser, Principal: []byte(`{}`), Params: params, ErrorFields: []byte("{}"),
	}
	require.NoError(t, db.Insert(op))
	return op
}

func insertBootstrapStep(t *testing.T, db *reform.DB, op *models.OmOperation, key string, result []byte) {
	t.Helper()
	require.NoError(t, db.Insert(&models.OmOperationStep{
		ID: uuid.New().String(), OperationID: op.ID, TargetKind: models.OmStepTargetRun, TargetID: op.ID,
		Phase: installBootstrapPhase, Step: installBootstrapStep, Attempt: 1, IdempotencyKey: key,
		Status: models.OmStepUnknown, Result: result,
	}))
}

func TestInstallExecutorCreatesTheRunOnTheFirstProbe(t *testing.T) {
	db := lockTestDB(t)
	op := installOpWithPlan(t, db)
	fake := &fakeBootstrapRunner{}
	executor := &installStepExecutor{db: db, bootstrap: fake, op: op}

	result, resultJSON, err := executor.Run(t.Context(), StepPlan{}, "op/run/bootstrap/1")
	require.NoError(t, err)
	assert.Equal(t, StepResultTimedOut, result, "a freshly started run is inconclusive")
	require.NotNil(t, fake.triggered)
	assert.Equal(t, []string{"exec-a"}, fake.triggered.Hosts, "the run targets the node's executor host")
	assert.Equal(t, "ubuntu", fake.triggered.OS)
	assert.Equal(t, "7.0.8", fake.triggered.MongoDBVersion)

	ref := installRunRef{}
	require.NoError(t, json.Unmarshal(resultJSON, &ref))
	assert.Equal(t, "run-1", ref.RunID, "the step result references the run the probe created")
}

func TestInstallExecutorMapsRunStatus(t *testing.T) {
	for _, tc := range []struct {
		runStatus string
		result    StepResultKind
	}{
		{bootstrapRunRunning, StepResultTimedOut},
		{bootstrapRunSucceeded, StepResultOk},
		{bootstrapRunFailed, StepResultFailed},
		{bootstrapRunRolledBack, StepResultFailed},
	} {
		t.Run(tc.runStatus, func(t *testing.T) {
			db := lockTestDB(t)
			op := installOpWithPlan(t, db)
			key := op.ID + "/run/bootstrap/1"
			insertBootstrapStep(t, db, op, key, []byte(`{"run_id":"run-1"}`))
			fake := &fakeBootstrapRunner{run: &extensionsBootstrapRun{ID: "run-1", Status: tc.runStatus}}
			executor := &installStepExecutor{db: db, bootstrap: fake, op: op}

			result, _, err := executor.Run(t.Context(), StepPlan{}, key)
			require.NoError(t, err)
			assert.Equal(t, tc.result, result)
			assert.Nil(t, fake.triggered, "a run already exists, so no second one is created")
		})
	}
}

func TestInstallExecutorRecordsManifest(t *testing.T) {
	db := lockTestDB(t)
	op := installOpWithPlan(t, db)
	key := op.ID + "/run/bootstrap/1"
	insertBootstrapStep(t, db, op, key, []byte(`{"run_id":"run-1"}`))
	fake := &fakeBootstrapRunner{run: &extensionsBootstrapRun{
		ID: "run-1", Status: bootstrapRunSucceeded,
		Repo: "percona-release", Packages: []string{"percona-server-mongodb"}, ControlUnit: "mongod",
	}}
	executor := &installStepExecutor{db: db, bootstrap: fake, op: op, l: logrus.WithField("test", t.Name())}

	result, _, err := executor.Run(t.Context(), StepPlan{}, key)
	require.NoError(t, err)
	require.Equal(t, StepResultOk, result)

	var raw []byte
	require.NoError(t, db.QueryRow("SELECT params FROM om_operations WHERE id = $1", op.ID).Scan(&raw))
	manifest := installParams{}
	require.NoError(t, json.Unmarshal(raw, &manifest))
	assert.Equal(t, "percona-release", manifest.Repo)
	assert.Equal(t, []string{"percona-server-mongodb"}, manifest.Packages)
	assert.Equal(t, "mongod", manifest.ControlUnit)
}
