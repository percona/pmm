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

type fakeHostStepRunner struct {
	dispatched *extensionsHostStepRequest
	resp       *extensionsHostStepResponse
	observed   string
	obs        *extensionsHostStepObservation
}

func (f *fakeHostStepRunner) dispatchHostStep(_ context.Context, req extensionsHostStepRequest) (*extensionsHostStepResponse, error) {
	f.dispatched = &req
	return f.resp, nil
}

func (f *fakeHostStepRunner) observeHostStep(_ context.Context, submissionID string) (*extensionsHostStepObservation, error) {
	f.observed = submissionID
	return f.obs, nil
}

func elevationReader(elevation string) *fakeFactReader {
	facts := map[string]map[string]any{}
	if elevation != "" {
		facts["m1"] = map[string]any{factAgentElevation: elevation}
	}
	return &fakeFactReader{facts: facts}
}

func uninstallOpWithPlan(t *testing.T, db *reform.DB) *models.OmOperation {
	t.Helper()
	params, err := json.Marshal(uninstallParams{
		ClusterID: "c1", ClusterName: "rs", InstallRunID: "run-9", OS: "ubuntu",
		Members: []uninstallMember{{
			MemberID: "m1", Host: "host-a", NodeID: "node-1",
			Packages: []string{"percona-server-mongodb"},
			DataPath: "/var/lib/mongo", ConfigFile: "/etc/mongod.conf", KeyfilePath: "/etc/keyfile",
			LogPath:    "/var/log/mongodb/mongod.log",
			MarkerPath: "/etc/mongod.om-bootstrap", ControlUnit: "mongod",
		}},
	})
	require.NoError(t, err)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: uninstallKind, KindVersion: 3,
		Status: models.OmOperationRunning, Version: 1, ParamsRevision: 1,
		Trigger: models.OmTriggerUser, Principal: []byte(`{}`), Params: params, ErrorFields: []byte("{}"),
	}
	require.NoError(t, db.Insert(op))
	return op
}

func insertUninstallStep(t *testing.T, db *reform.DB, op *models.OmOperation, key, step string, epoch int64, submissionID string) {
	t.Helper()
	var result []byte
	if submissionID != "" {
		var err error
		result, err = json.Marshal(hostStepRef{SubmissionID: submissionID})
		require.NoError(t, err)
	}
	require.NoError(t, db.Insert(&models.OmOperationStep{
		ID: uuid.New().String(), OperationID: op.ID, TargetKind: models.OmStepTargetRun, TargetID: op.ID,
		Phase: uninstallHostPhase, Step: step, Attempt: 1, IdempotencyKey: key, Epoch: epoch,
		Status: models.OmStepDispatching, Result: result,
	}))
}

func submittedResponse(id string) *extensionsHostStepResponse {
	return &extensionsHostStepResponse{SubmissionID: &id}
}

type fakeForgetter struct {
	deleted []string
}

func (f *fakeForgetter) forgetService(_ context.Context, serviceID string) error {
	f.deleted = append(f.deleted, serviceID)
	return nil
}

type fakeNudger struct {
	nudged [][]string
}

func (f *fakeNudger) nudgeInventory(_ context.Context, nodeIDs []string) {
	f.nudged = append(f.nudged, nodeIDs)
}

// uninstallForgetOp stores a running uninstall whose sealed plan names the given member, so Run resolves it
// before dispatching the forget step at it.
func uninstallForgetOp(t *testing.T, db *reform.DB, memberID string) *models.OmOperation {
	t.Helper()
	params, err := json.Marshal(uninstallParams{
		ClusterID: uuid.New().String(), ClusterName: "rs", InstallRunID: "run-9", OS: "ubuntu",
		Members: []uninstallMember{{MemberID: memberID, Host: "host-a"}},
	})
	require.NoError(t, err)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: uninstallKind, KindVersion: 3,
		Status: models.OmOperationRunning, Version: 1, ParamsRevision: 1,
		Trigger: models.OmTriggerUser, Principal: []byte(`{}`), Params: params, ErrorFields: []byte("{}"),
	}
	require.NoError(t, db.Insert(op))
	return op
}

// memberWithoutService inserts an accepted mongod member with no PMM service linked, which the forget step
// has nothing to forget on.
func memberWithoutService(t *testing.T, db *reform.DB, clusterID string) string {
	t.Helper()
	nodeID := uuid.New().String()
	_, err := db.Exec(
		"INSERT INTO nodes (node_id, node_type, node_name, distro, node_model, az, address, created_at, updated_at) "+
			"VALUES ($1, 'generic', 'node-nolink', '', '', '', '', now(), now())",
		nodeID)
	require.NoError(t, err)
	replicaSet := &models.OmReplicaSet{
		ID: uuid.New().String(), ClusterID: clusterID, Name: "rs-nolink", Role: models.OmReplicaSetRolePlain,
	}
	require.NoError(t, db.Insert(replicaSet))
	rsMemberID := int32(2)
	member := &models.OmClusterMember{
		ID: uuid.New().String(), ClusterID: clusterID, ReplicaSetRef: &replicaSet.ID, Process: models.OmMemberMongod,
		Host: "host-nolink", RSMemberID: &rsMemberID, State: models.OmMemberAccepted,
		NodeID: &nodeID, AddedByOperation: lockOperation(t, db.Querier, clusterID),
	}
	require.NoError(t, db.Insert(member))
	return member.ID
}

func TestUninstallForgetDeletesTheServiceAndNudges(t *testing.T) {
	db := lockTestDB(t)
	serviceID := uuid.New().String()
	memberID := scanMember(t, db, lockCluster(t, db.Querier), serviceID)
	op := uninstallForgetOp(t, db, memberID)
	forgetter := &fakeForgetter{}
	nudger := &fakeNudger{}
	e := &uninstallStepExecutor{db: db, forgetter: forgetter, nudger: nudger, op: op, l: logrus.NewEntry(logrus.New())}
	plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallForgetStep, TargetKind: models.OmStepTargetMember, TargetID: memberID}

	kind, _, err := e.Run(t.Context(), plan, "k")

	require.NoError(t, err)
	assert.Equal(t, StepResultOk, kind)
	assert.Equal(t, []string{serviceID}, forgetter.deleted)
	assert.Len(t, nudger.nudged, 1, "the forget step nudges a resync")
}

func TestUninstallForgetSkipsAMemberWithNoServiceLink(t *testing.T) {
	db := lockTestDB(t)
	memberID := memberWithoutService(t, db, lockCluster(t, db.Querier))
	op := uninstallForgetOp(t, db, memberID)
	forgetter := &fakeForgetter{}
	nudger := &fakeNudger{}
	e := &uninstallStepExecutor{db: db, forgetter: forgetter, nudger: nudger, op: op, l: logrus.NewEntry(logrus.New())}
	plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallForgetStep, TargetKind: models.OmStepTargetMember, TargetID: memberID}

	kind, _, err := e.Run(t.Context(), plan, "k")

	require.NoError(t, err)
	assert.Equal(t, StepResultOk, kind)
	assert.Empty(t, forgetter.deleted, "a member with no service link forgets nothing")
	assert.Empty(t, nudger.nudged)
}

func TestUninstallStepExecutorSubmitsOnFirstRun(t *testing.T) {
	db := lockTestDB(t)
	op := uninstallOpWithPlan(t, db)
	key := op.ID + "/member/m1/host/uninstall_remove/1"
	insertUninstallStep(t, db, op, key, uninstallRemoveStep, 7, "")
	runner := &fakeHostStepRunner{resp: submittedResponse("s1")}
	e := &uninstallStepExecutor{db: db, bootstrap: runner, reader: elevationReader("sudo_all"), op: op, l: logrus.NewEntry(logrus.New())}
	plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallRemoveStep, TargetKind: models.OmStepTargetMember, TargetID: "m1"}

	kind, ref, err := e.Run(t.Context(), plan, key)
	require.NoError(t, err)
	assert.Equal(t, StepResultTimedOut, kind)
	assert.Empty(t, runner.observed, "the first run submits, it does not observe")

	require.NotNil(t, runner.dispatched)
	assert.Equal(t, op.ID, runner.dispatched.RunID)
	assert.Equal(t, key, runner.dispatched.IdempotencyKey)
	assert.Equal(t, int64(7), runner.dispatched.Epoch)
	assert.Equal(t, int32(3), runner.dispatched.CatalogueVersion)
	assert.Equal(t, "node-1", runner.dispatched.NodeID)
	assert.Equal(t, uninstallRemoveStep, runner.dispatched.StepKind)
	assert.Equal(t, "host-a", runner.dispatched.Target)
	assert.Equal(t, "sudo_all", runner.dispatched.Measured)
	assert.Equal(t, "run-9", runner.dispatched.Params.OwnerRunID)
	assert.Equal(t, "ubuntu", runner.dispatched.Params.OS)
	assert.Equal(t, "/var/lib/mongo", runner.dispatched.Params.DataPath)
	assert.Equal(t, "/var/log/mongodb/mongod.log", runner.dispatched.Params.LogPath)

	gotRef := hostStepRef{}
	require.NoError(t, json.Unmarshal(ref, &gotRef))
	assert.Equal(t, "s1", gotRef.SubmissionID)
}

func TestUninstallStepExecutorRefusalIsTerminal(t *testing.T) {
	refusal := func(kind StepResultKind) *extensionsHostStepResponse {
		return &extensionsHostStepResponse{Refusal: &hostStepRefusal{Kind: string(kind)}}
	}
	cases := map[string]struct {
		resp *extensionsHostStepResponse
		want StepResultKind
	}{
		"blocked":        {refusal(StepResultBlocked), StepResultBlocked},
		"failed":         {refusal(StepResultFailed), StepResultFailed},
		"other terminal": {refusal(StepResultUnknown), StepResultFailed},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			db := lockTestDB(t)
			op := uninstallOpWithPlan(t, db)
			key := op.ID + "/member/m1/host/uninstall_stop/1"
			insertUninstallStep(t, db, op, key, uninstallStopStep, 1, "")
			runner := &fakeHostStepRunner{resp: tc.resp}
			e := &uninstallStepExecutor{db: db, bootstrap: runner, reader: elevationReader("root"), op: op, l: logrus.NewEntry(logrus.New())}
			plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallStopStep, TargetKind: models.OmStepTargetMember, TargetID: "m1"}

			kind, _, err := e.Run(t.Context(), plan, key)
			require.NoError(t, err)
			assert.Equal(t, tc.want, kind)
			assert.Empty(t, runner.observed, "a refusal is terminal, no observe")
		})
	}
}

func TestUninstallStepExecutorObservesSubmission(t *testing.T) {
	detail := "data directory not empty"
	cases := map[string]struct {
		obs  *extensionsHostStepObservation
		want StepResultKind
	}{
		"in_flight": {&extensionsHostStepObservation{Status: hostStepObserveInFlight}, StepResultTimedOut},
		"ok":        {&extensionsHostStepObservation{Status: hostStepObserveOk}, StepResultOk},
		"failed":    {&extensionsHostStepObservation{Status: hostStepObserveFailed, Detail: &detail}, StepResultFailed},
		"blocked":   {&extensionsHostStepObservation{Status: hostStepObserveBlocked}, StepResultBlocked},
		"gone":      {&extensionsHostStepObservation{Status: hostStepObserveGone}, StepResultFailed},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			db := lockTestDB(t)
			op := uninstallOpWithPlan(t, db)
			key := op.ID + "/member/m1/host/uninstall_remove/1"
			insertUninstallStep(t, db, op, key, uninstallRemoveStep, 7, "sub-1")
			runner := &fakeHostStepRunner{obs: tc.obs}
			e := &uninstallStepExecutor{db: db, bootstrap: runner, reader: elevationReader("root"), op: op, l: logrus.NewEntry(logrus.New())}
			plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallRemoveStep, TargetKind: models.OmStepTargetMember, TargetID: "m1"}

			kind, ref, err := e.Run(t.Context(), plan, key)
			require.NoError(t, err)
			assert.Equal(t, tc.want, kind)
			assert.Equal(t, "sub-1", runner.observed, "a re-probe observes the stored submission")
			assert.Nil(t, runner.dispatched, "a re-probe does not re-submit")

			gotRef := hostStepRef{}
			require.NoError(t, json.Unmarshal(ref, &gotRef))
			assert.Equal(t, "sub-1", gotRef.SubmissionID)
			if tc.obs.Detail != nil {
				assert.Equal(t, detail, gotRef.Detail)
			}
		})
	}
}

func TestUninstallStepExecutorDefaultsElevation(t *testing.T) {
	db := lockTestDB(t)
	op := uninstallOpWithPlan(t, db)
	key := op.ID + "/member/m1/host/uninstall_stop/1"
	insertUninstallStep(t, db, op, key, uninstallStopStep, 1, "")
	runner := &fakeHostStepRunner{resp: submittedResponse("s1")}
	e := &uninstallStepExecutor{db: db, bootstrap: runner, reader: elevationReader(""), op: op, l: logrus.NewEntry(logrus.New())}
	plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallStopStep, TargetKind: models.OmStepTargetMember, TargetID: "m1"}

	_, _, err := e.Run(t.Context(), plan, key)
	require.NoError(t, err)
	require.NotNil(t, runner.dispatched)
	assert.Equal(t, elevationNone, runner.dispatched.Measured)
}

func TestUninstallStepExecutorUnknownMember(t *testing.T) {
	db := lockTestDB(t)
	op := uninstallOpWithPlan(t, db)
	runner := &fakeHostStepRunner{resp: &extensionsHostStepResponse{}}
	e := &uninstallStepExecutor{db: db, bootstrap: runner, reader: elevationReader(""), op: op, l: logrus.NewEntry(logrus.New())}
	plan := StepPlan{Phase: uninstallHostPhase, Step: uninstallStopStep, TargetKind: models.OmStepTargetMember, TargetID: "unknown"}

	_, _, err := e.Run(t.Context(), plan, "k")
	require.Error(t, err)
}

func TestExtensionsHostStepParamsOmitAnEmptyLogPath(t *testing.T) {
	// om_bootstrap's route rejects an empty path, so a plan sealed without one must not send the field.
	body, err := json.Marshal(extensionsHostStepParams{OwnerRunID: "run-9", OS: "ubuntu", DataPath: "/var/lib/mongo"})
	require.NoError(t, err)
	assert.NotContains(t, string(body), "log_path")

	body, err = json.Marshal(extensionsHostStepParams{OwnerRunID: "run-9", OS: "ubuntu", DataPath: "/var/lib/mongo", LogPath: "/var/log/mongodb/mongod.log"})
	require.NoError(t, err)
	assert.Contains(t, string(body), `"log_path":"/var/log/mongodb/mongod.log"`)
}
