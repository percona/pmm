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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

type fakeFactReader struct {
	facts map[string]map[string]any
	at    map[string]time.Time
}

func (f *fakeFactReader) Observation(_ context.Context, _, subjectKey, _ string) (map[string]any, time.Time, error) {
	return f.facts[subjectKey], f.at[subjectKey], nil
}

func installOp(t *testing.T, nodeIDs ...string) *models.OmOperation {
	t.Helper()
	params, err := json.Marshal(installParams{NodeIDs: nodeIDs})
	require.NoError(t, err)
	return &models.OmOperation{Kind: installKind, Params: params}
}

func nodeFacts() map[string]any {
	return passingInstallFacts()
}

func TestCheckInputsAggregatesAndPasses(t *testing.T) {
	now := time.Now()
	a := nodeFacts()
	a[factFreeSpaceBytes] = float64(10 << 30)
	b := nodeFacts()
	b[factFreeSpaceBytes] = float64(6 << 30)
	reader := &fakeFactReader{
		facts: map[string]map[string]any{"node-a": a, "node-b": b},
		at:    map[string]time.Time{"node-a": now.Add(-time.Minute), "node-b": now.Add(-2 * time.Minute)},
	}
	provider := &factCheckProvider{reader: reader, now: func() time.Time { return now }}

	set, facts, ages, err := provider.CheckInputs(t.Context(), installOp(t, "node-a", "node-b"), InstallKind())
	require.NoError(t, err)
	assert.Equal(t, true, facts[factNodesReachable])
	assert.InDelta(t, float64(6<<30), facts[factFreeSpaceBytes], 0, "the least free space wins")
	assert.Equal(t, 2*time.Minute, ages[factFreeSpaceBytes], "the oldest age wins")
	assert.False(t, runBlockingRules(set, CheckContext{Facts: facts}).Blocked)
}

func TestCheckInputsBlocksWhenOneNodeFails(t *testing.T) {
	now := time.Now()
	a := nodeFacts()
	b := nodeFacts()
	b[factPortFree] = false
	reader := &fakeFactReader{
		facts: map[string]map[string]any{"node-a": a, "node-b": b},
		at:    map[string]time.Time{"node-a": now, "node-b": now},
	}
	provider := &factCheckProvider{reader: reader, now: func() time.Time { return now }}

	set, facts, _, err := provider.CheckInputs(t.Context(), installOp(t, "node-a", "node-b"), InstallKind())
	require.NoError(t, err)
	assert.Equal(t, false, facts[factPortFree])
	block := runBlockingRules(set, CheckContext{Facts: facts})
	assert.True(t, block.Blocked)
	assert.Equal(t, factPortFree, block.Path)
}

func TestCheckInputsLeavesOutAMissingFact(t *testing.T) {
	now := time.Now()
	a := nodeFacts()
	b := nodeFacts()
	delete(b, factOSSupported)
	reader := &fakeFactReader{
		facts: map[string]map[string]any{"node-a": a, "node-b": b},
		at:    map[string]time.Time{"node-a": now, "node-b": now},
	}
	provider := &factCheckProvider{reader: reader, now: func() time.Time { return now }}

	set, facts, _, err := provider.CheckInputs(t.Context(), installOp(t, "node-a", "node-b"), InstallKind())
	require.NoError(t, err)
	_, present := facts[factOSSupported]
	assert.False(t, present, "a fact missing from a node is left out so its rule blocks")
	block := runBlockingRules(set, CheckContext{Facts: facts})
	assert.True(t, block.Blocked)
	assert.Equal(t, factOSSupported, block.Path)
}
