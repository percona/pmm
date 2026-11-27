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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func passingUninstallFacts() map[string]any {
	return map[string]any{
		factMarkerPresent: true,
		factHostReachable: true,
		factCanElevate:    true,
		factConfigSHA256:  "sha",
		factBinaryVersion: "7.0.14",
	}
}

func uninstallOp(t *testing.T, params uninstallParams) *models.OmOperation {
	t.Helper()
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	return &models.OmOperation{ID: "0123456789abcdef", Params: raw}
}

func sampleUninstallParams() uninstallParams {
	return uninstallParams{
		ClusterID:    "cluster-1",
		ClusterName:  "rs-prod",
		InstallRunID: "run-7",
		Members: []uninstallMember{
			{MemberID: "member-a", Host: "host-a", DataPath: "/data/a"},
			{MemberID: "member-b", Host: "host-b", DataPath: "/data/b"},
		},
	}
}

func passingUninstallContext(t *testing.T) CheckContext {
	t.Helper()
	params := sampleUninstallParams()
	subjects := map[string]map[string]any{}
	for _, member := range params.Members {
		subjects[member.MemberID] = map[string]any{
			factMarkerRunID:   params.InstallRunID,
			factOwnedPackages: []any{"percona-server-mongodb"},
		}
	}
	return CheckContext{Facts: passingUninstallFacts(), SubjectFacts: subjects}
}

func readinessSet(t *testing.T) CheckSet {
	t.Helper()
	params := sampleUninstallParams()
	params.Members[0].Packages = []string{"percona-server-mongodb"}
	params.Members[1].Packages = []string{"percona-server-mongodb"}
	return uninstallReadinessCheckSet(uninstallOp(t, params))
}

func TestUninstallReadinessPasses(t *testing.T) {
	got := runBlockingRules(readinessSet(t), passingUninstallContext(t))
	assert.False(t, got.Blocked)
}

func TestUninstallReadinessBlocksEachFailedBoolFact(t *testing.T) {
	codes := map[string]string{
		factMarkerPresent: uninstallReasonMarkerAbsent,
		factHostReachable: uninstallReasonHostUnreachable,
		factCanElevate:    uninstallReasonCannotElevate,
	}
	set := readinessSet(t)
	for fact, code := range codes {
		t.Run(fact, func(t *testing.T) {
			ctx := passingUninstallContext(t)
			ctx.Facts[fact] = false
			got := runBlockingRules(set, ctx)
			assert.True(t, got.Blocked)
			assert.Equal(t, fact, got.Path)
			assert.Equal(t, code, got.Reason)
		})
	}
}

func TestUninstallReadinessBlocksMissingFacts(t *testing.T) {
	got := runBlockingRules(readinessSet(t), CheckContext{Facts: map[string]any{}})
	assert.True(t, got.Blocked)
}

func TestUninstallReadinessIgnoresConfigAndBinaryChange(t *testing.T) {
	ctx := passingUninstallContext(t)
	ctx.Facts[factConfigSHA256] = "drifted"
	ctx.Facts[factBinaryVersion] = "7.0.99"
	got := runBlockingRules(readinessSet(t), ctx)
	assert.False(t, got.Blocked)
}

func TestUninstallReadinessBlocksMarkerMismatch(t *testing.T) {
	ctx := passingUninstallContext(t)
	ctx.SubjectFacts["member-b"][factMarkerRunID] = "run-other"
	got := runBlockingRules(readinessSet(t), ctx)
	assert.True(t, got.Blocked)
	assert.Equal(t, uninstallReasonMarkerMismatch, got.Reason)
	assert.Equal(t, "member-b", got.Path)
}

func TestUninstallReadinessAbsentMarkerIsNotMismatch(t *testing.T) {
	ctx := passingUninstallContext(t)
	delete(ctx.SubjectFacts["member-a"], factMarkerRunID)
	got := runBlockingRules(readinessSet(t), ctx)
	assert.False(t, got.Blocked)
}

func TestUninstallReadinessBlocksPackageDrift(t *testing.T) {
	ctx := passingUninstallContext(t)
	ctx.SubjectFacts["member-a"][factOwnedPackages] = []any{"percona-server-mongodb", "percona-server-mongodb-tools"}
	got := runBlockingRules(readinessSet(t), ctx)
	assert.True(t, got.Blocked)
	assert.Equal(t, uninstallReasonDrift, got.Reason)
	assert.Equal(t, "member-a", got.Path)
}

func TestUninstallReadinessAbsentPackagesIsNotDrift(t *testing.T) {
	ctx := passingUninstallContext(t)
	delete(ctx.SubjectFacts["member-a"], factOwnedPackages)
	delete(ctx.SubjectFacts["member-b"], factOwnedPackages)
	got := runBlockingRules(readinessSet(t), ctx)
	assert.False(t, got.Blocked)
}

func TestStringListFact(t *testing.T) {
	list, ok := stringListFact([]any{"a", "b"})
	require.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, list)

	_, ok = stringListFact([]any{"a", 2})
	assert.False(t, ok)

	_, ok = stringListFact("a")
	assert.False(t, ok)

	_, ok = stringListFact(nil)
	assert.False(t, ok)
}

func TestSameStringSet(t *testing.T) {
	assert.True(t, sameStringSet([]string{"a", "b"}, []string{"b", "a"}))
	assert.True(t, sameStringSet(nil, []string{}))
	assert.False(t, sameStringSet([]string{"a"}, []string{"a", "b"}))
	assert.False(t, sameStringSet([]string{"a", "a"}, []string{"a", "b"}))
}

func TestUninstallKindLocksClusterAndMembers(t *testing.T) {
	locks := UninstallKind().LockSet(uninstallOp(t, sampleUninstallParams()))
	require.Len(t, locks, 3)
	assert.Equal(t, "cluster", locks[0].Kind)
	assert.Equal(t, "cluster-1", locks[0].ID)
	assert.Equal(t, "member", locks[1].Kind)
	assert.Equal(t, "member-a", locks[1].ID)
	assert.Equal(t, "member-b", locks[2].ID)
	for _, lock := range locks {
		assert.Equal(t, models.OmLockExclusive, lock.Mode)
	}
}

func TestUninstallKindStepsStopRemoveThenForgetPerMember(t *testing.T) {
	steps := UninstallKind().Steps(uninstallOp(t, sampleUninstallParams()))
	require.Len(t, steps, 6)
	assert.Equal(t, uninstallStopStep, steps[0].Step)
	assert.Equal(t, uninstallRemoveStep, steps[1].Step)
	assert.Equal(t, uninstallForgetStep, steps[2].Step)
	assert.Equal(t, "member-a", steps[0].TargetID)
	assert.Equal(t, "member-a", steps[2].TargetID)
	assert.Equal(t, uninstallStopStep, steps[3].Step)
	assert.Equal(t, "member-b", steps[3].TargetID)
	for _, step := range steps {
		assert.Equal(t, models.OmStepTargetMember, step.TargetKind)
		assert.Equal(t, uninstallHostPhase, step.Phase)
	}
}

func TestUninstallKindSubjectsAreMembersByID(t *testing.T) {
	subjects := UninstallKind().Subjects(uninstallOp(t, sampleUninstallParams()))
	require.Len(t, subjects, 2)
	assert.Equal(t, subjectKindMember, subjects[0].Kind)
	assert.Equal(t, "member-a", subjects[0].Key)
	assert.Equal(t, "member-b", subjects[1].Key)
}

func TestUninstallConfirmPhrase(t *testing.T) {
	phrase := uninstallConfirmPhrase(uninstallOp(t, sampleUninstallParams()))
	assert.Equal(t, "REMOVE rs-prod 012345", phrase)
}

func TestUninstallKindDeclaration(t *testing.T) {
	kind := UninstallKind()
	assert.Equal(t, "uninstall", kind.Kind)
	assert.True(t, kind.Resumable)
	assert.True(t, kind.Attention.ExemptAll)
	assert.Equal(t, []string{"admin"}, kind.AuthMethods)
	require.NotNil(t, kind.ConfirmPhrase)
	assert.True(t, kind.Confirms(models.OmTriggerUser))
}

func TestUninstallKindRegisters(t *testing.T) {
	registry := NewOperationRegistry()
	require.NoError(t, registry.Register(UninstallKind()))
	_, ok := registry.Kind("uninstall")
	assert.True(t, ok)
}
