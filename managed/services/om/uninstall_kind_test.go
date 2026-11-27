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

func TestUninstallReadinessPasses(t *testing.T) {
	got := runBlockingRules(uninstallReadinessCheckSet(), CheckContext{Facts: passingUninstallFacts()})
	assert.False(t, got.Blocked)
}

func TestUninstallReadinessBlocksEachFailedBoolFact(t *testing.T) {
	facts := []string{factMarkerPresent, factHostReachable, factCanElevate}
	set := uninstallReadinessCheckSet()
	for _, fact := range facts {
		t.Run(fact, func(t *testing.T) {
			values := passingUninstallFacts()
			values[fact] = false
			got := runBlockingRules(set, CheckContext{Facts: values})
			assert.True(t, got.Blocked)
			assert.Equal(t, fact, got.Path)
		})
	}
}

func TestUninstallReadinessBlocksMissingFacts(t *testing.T) {
	got := runBlockingRules(uninstallReadinessCheckSet(), CheckContext{Facts: map[string]any{}})
	assert.True(t, got.Blocked)
}

func TestUninstallReadinessIgnoresConfigAndBinaryChange(t *testing.T) {
	values := passingUninstallFacts()
	values[factConfigSHA256] = "drifted"
	values[factBinaryVersion] = "7.0.99"
	got := runBlockingRules(uninstallReadinessCheckSet(), CheckContext{Facts: values})
	assert.False(t, got.Blocked)
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

func TestUninstallKindStepsStopThenRemovePerMember(t *testing.T) {
	steps := UninstallKind().Steps(uninstallOp(t, sampleUninstallParams()))
	require.Len(t, steps, 4)
	assert.Equal(t, uninstallStopStep, steps[0].Step)
	assert.Equal(t, uninstallRemoveStep, steps[1].Step)
	assert.Equal(t, "member-a", steps[0].TargetID)
	assert.Equal(t, "member-b", steps[2].TargetID)
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
