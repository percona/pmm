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

package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInventory() ThresholdInventory {
	return ThresholdInventory{
		NodeNames: map[string]string{
			"node-id-1": "node-1",
			"node-id-2": "node-2",
		},
		ServiceNames: map[string]string{
			"svc-id-1": "svc-1",
			"svc-id-2": "svc-2",
		},
	}
}

// collidingInventory adds a node named like a service. That is the only way node and service
// scope can reach the same target, since they otherwise resolve into separate label
// namespaces; node_name and service_name are unique within their own tables, not across them.
func collidingInventory() ThresholdInventory {
	inv := testInventory()
	inv.NodeNames["node-id-3"] = "svc-1"

	return inv
}

func override(scope ThresholdScope, target string, value float64) *AlertRuleThresholdOverride {
	return &AlertRuleThresholdOverride{
		ID:        fmt.Sprintf("%s-%s", scope, target),
		RuleID:    "rule-1",
		ParamName: "threshold",
		Scope:     scope,
		Target:    target,
		Value:     value,
	}
}

func TestResolveThresholds(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		overrides []*AlertRuleThresholdOverride
		expected  map[string]float64
	}{
		{
			name:      "no overrides emits nothing",
			overrides: nil,
			expected:  map[string]float64{},
		},
		{
			name:      "node override resolves to the node name",
			overrides: []*AlertRuleThresholdOverride{override(ThresholdScopeNode, "node-id-1", 90)},
			expected:  map[string]float64{"node-1": 90},
		},
		{
			name:      "service override resolves to the service name",
			overrides: []*AlertRuleThresholdOverride{override(ThresholdScopeService, "svc-id-1", 91)},
			expected:  map[string]float64{"svc-1": 91},
		},
		{
			name: "unresolvable target is skipped, never defaulted",
			overrides: []*AlertRuleThresholdOverride{
				override(ThresholdScopeNode, "deleted-node-id", 90),
			},
			expected: map[string]float64{},
		},
		{
			name: "cluster override resolves to nothing",
			overrides: []*AlertRuleThresholdOverride{
				override(ThresholdScopeCluster, "prod", 90),
			},
			expected: map[string]float64{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := make(map[string]float64)
			for name, resolved := range ResolveThresholds(tc.overrides, testInventory()) {
				actual[name] = resolved.Value
			}

			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestResolveThresholdsServiceBeatsNode pins the derivable half of the order: a service
// runs on exactly one node, so a service override is strictly narrower and must win.
func TestResolveThresholdsServiceBeatsNode(t *testing.T) {
	t.Parallel()

	overrides := []*AlertRuleThresholdOverride{
		override(ThresholdScopeService, "svc-id-1", 20),
		override(ThresholdScopeNode, "node-id-3", 30),
		override(ThresholdScopeNode, "node-id-1", 90),
	}

	resolved := ResolveThresholds(overrides, collidingInventory())
	require.Len(t, resolved, 2, "one value per target: duplicate series fail the whole /metrics response")
	assert.InDelta(t, 20.0, resolved["svc-1"].Value, 0.0001, "service scope must win over node")
	assert.InDelta(t, 90.0, resolved["node-1"].Value, 0.0001)
}

func TestResolveThresholdsIsOrderIndependent(t *testing.T) {
	t.Parallel()

	forward := []*AlertRuleThresholdOverride{
		override(ThresholdScopeNode, "node-id-3", 99),
		override(ThresholdScopeService, "svc-id-1", 50),
	}
	reversed := []*AlertRuleThresholdOverride{forward[1], forward[0]}

	inv := collidingInventory()
	assert.Equal(t,
		ResolveThresholds(forward, inv),
		ResolveThresholds(reversed, inv),
		"precedence must not depend on row order returned by the database")
}

func BenchmarkResolveThresholds(b *testing.B) {
	inv := ThresholdInventory{NodeNames: make(map[string]string, 1000)}

	overrides := make([]*AlertRuleThresholdOverride, 0, 1000)
	for i := range 1000 {
		id := fmt.Sprintf("node-id-%d", i)
		inv.NodeNames[id] = fmt.Sprintf("node-%d", i)
		overrides = append(overrides, override(ThresholdScopeNode, id, float64(i%100)))
	}

	for b.Loop() {
		ResolveThresholds(overrides, inv)
	}
}

func TestResolvedThresholdIsOverridden(t *testing.T) {
	t.Parallel()

	// The API reports this as `is_overridden`, and the UI uses it to decide whether the
	// reset control is live, so a default that claims to be an override would offer a
	// reset that does nothing.
	assert.False(t, ResolvedThreshold{Value: 80}.IsOverridden())
	assert.True(t, ResolvedThreshold{Value: 90, Source: &AlertRuleThresholdOverride{}}.IsOverridden())
}

func TestTargetNamesSkipsTargetsOutsideInventory(t *testing.T) {
	t.Parallel()

	// The override table is polymorphic and carries no foreign key, so a row can outlive
	// its target. Such a row must resolve to nothing rather than to an empty join-label
	// value, which would match every series the rule produces.
	inv := ThresholdInventory{
		NodeNames:    map[string]string{"node-id-1": "node-1"},
		ServiceNames: map[string]string{"service-id-1": "service-1"},
	}

	assert.Nil(t, inv.targetNames(&AlertRuleThresholdOverride{
		Scope: ThresholdScopeService, Target: "service-id-gone",
	}))
	assert.Nil(t, inv.targetNames(&AlertRuleThresholdOverride{
		Scope: ThresholdScopeNode, Target: "node-id-gone",
	}))
	assert.Nil(t, inv.targetNames(&AlertRuleThresholdOverride{
		Scope: ThresholdScope("nonsense"), Target: "prod",
	}))

	assert.Equal(t, []string{"service-1"}, inv.targetNames(&AlertRuleThresholdOverride{
		Scope: ThresholdScopeService, Target: "service-id-1",
	}))
}
