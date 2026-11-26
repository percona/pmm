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

func passingInstallFacts() map[string]any {
	return map[string]any{
		factRepoReachable:         true,
		factPackageManagerPresent: true,
		factNoMongod:              true,
		factFreeSpaceBytes:        installMinFreeSpaceBytes,
	}
}

func TestInstallReadinessPasses(t *testing.T) {
	got := runBlockingRules(installReadinessCheckSet(), CheckContext{Facts: passingInstallFacts()})
	assert.False(t, got.Blocked)
}

func TestInstallReadinessBlocksEachFailedBoolFact(t *testing.T) {
	facts := []string{factRepoReachable, factPackageManagerPresent, factNoMongod}
	set := installReadinessCheckSet()
	for _, fact := range facts {
		t.Run(fact, func(t *testing.T) {
			values := passingInstallFacts()
			values[fact] = false
			got := runBlockingRules(set, CheckContext{Facts: values})
			assert.True(t, got.Blocked)
			assert.Equal(t, fact, got.Path)
		})
	}
}

func TestInstallReadinessBlocksLowFreeSpace(t *testing.T) {
	values := passingInstallFacts()
	values[factFreeSpaceBytes] = installMinFreeSpaceBytes - 1
	got := runBlockingRules(installReadinessCheckSet(), CheckContext{Facts: values})
	assert.True(t, got.Blocked)
	assert.Equal(t, factFreeSpaceBytes, got.Path)
}

func TestInstallReadinessBlocksMissingFacts(t *testing.T) {
	got := runBlockingRules(installReadinessCheckSet(), CheckContext{Facts: map[string]any{}})
	assert.True(t, got.Blocked)
}

func TestInstallKindLocksTargetNodesExclusively(t *testing.T) {
	params, err := json.Marshal(installParams{NodeIDs: []string{"node-a", "node-b"}})
	require.NoError(t, err)
	locks := InstallKind().LockSet(&models.OmOperation{Params: params})
	require.Len(t, locks, 2)
	for _, lock := range locks {
		assert.Equal(t, "node", lock.Kind)
		assert.Equal(t, models.OmLockExclusive, lock.Mode)
	}
	assert.Equal(t, "node-a", locks[0].ID)
	assert.Equal(t, "node-b", locks[1].ID)
}
