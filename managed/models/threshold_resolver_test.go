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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDefault = 80.0

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
		ServicesByCluster: map[string][]string{
			"prod": {"svc-1", "svc-2"},
		},
	}
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

func tombstone(scope ThresholdScope, target string, value float64) *AlertRuleThresholdOverride {
	o := override(scope, target, value)
	cleared := time.Now()
	o.ClearedAt = &cleared

	return o
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
			name:      "cluster override fans out onto every service in the cluster",
			overrides: []*AlertRuleThresholdOverride{override(ThresholdScopeCluster, "prod", 70)},
			expected:  map[string]float64{"svc-1": 70, "svc-2": 70},
		},
		{
			name: "service beats cluster regardless of value",
			overrides: []*AlertRuleThresholdOverride{
				override(ThresholdScopeCluster, "prod", 99),
				override(ThresholdScopeService, "svc-id-1", 50),
			},
			// svc-1 takes the more specific 50 even though the cluster value is larger:
			// precedence is by scope, not by magnitude.
			expected: map[string]float64{"svc-1": 50, "svc-2": 99},
		},
		{
			name: "unresolvable target is skipped, never defaulted",
			overrides: []*AlertRuleThresholdOverride{
				override(ThresholdScopeNode, "deleted-node-id", 90),
			},
			expected: map[string]float64{},
		},
		{
			name: "unknown cluster expands to nothing",
			overrides: []*AlertRuleThresholdOverride{
				override(ThresholdScopeCluster, "staging", 90),
			},
			expected: map[string]float64{},
		},
		{
			name: "tombstone with no surviving override resolves to the default",
			overrides: []*AlertRuleThresholdOverride{
				tombstone(ThresholdScopeNode, "node-id-1", 90),
			},
			expected: map[string]float64{"node-1": testDefault},
		},
		{
			name: "tombstone falls through to a covering cluster override, not the default",
			overrides: []*AlertRuleThresholdOverride{
				override(ThresholdScopeCluster, "prod", 70),
				tombstone(ThresholdScopeService, "svc-id-1", 50),
			},
			expected: map[string]float64{"svc-1": 70, "svc-2": 70},
		},
		{
			name: "tombstoned cluster override clears every service it covered",
			overrides: []*AlertRuleThresholdOverride{
				tombstone(ThresholdScopeCluster, "prod", 70),
			},
			expected: map[string]float64{"svc-1": testDefault, "svc-2": testDefault},
		},
		{
			name: "a tombstone never contributes its stale value",
			overrides: []*AlertRuleThresholdOverride{
				tombstone(ThresholdScopeNode, "node-id-1", 12345),
			},
			expected: map[string]float64{"node-1": testDefault},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := ResolveThresholds(tc.overrides, testDefault, testInventory())
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestResolveThresholdsPrecedenceAcrossAllScopes pins the order that is derivable rather
// than conventional: a service runs on exactly one node and belongs to at most one
// cluster, so a service override is strictly narrower than either and must win.
func TestResolveThresholdsPrecedenceAcrossAllScopes(t *testing.T) {
	t.Parallel()

	inv := testInventory()
	// A node whose name collides with a service name is the only way node and service
	// scope can reach the same target, since they otherwise resolve into separate label
	// namespaces. Nothing in the schema prevents it: node_name and service_name are
	// unique within their own tables, not across them.
	inv.NodeNames["node-id-3"] = "svc-1"

	overrides := []*AlertRuleThresholdOverride{
		override(ThresholdScopeCluster, "prod", 10),
		override(ThresholdScopeService, "svc-id-1", 20),
		override(ThresholdScopeNode, "node-id-3", 30),
	}

	resolved := ResolveThresholds(overrides, testDefault, inv)
	assert.InDelta(t, 20.0, resolved["svc-1"], 0.0001, "service scope must win over node and cluster")
}

// TestResolveThresholdsNodeBeatsCluster pins the conventional half of the order. Node and
// cluster cross-cut rather than nest - a cluster spans several nodes, a node hosts
// services from several clusters - so this is a chosen tie-break, not a containment.
func TestResolveThresholdsNodeBeatsCluster(t *testing.T) {
	t.Parallel()

	inv := testInventory()
	inv.NodeNames["node-id-3"] = "svc-1"

	overrides := []*AlertRuleThresholdOverride{
		override(ThresholdScopeCluster, "prod", 10),
		override(ThresholdScopeNode, "node-id-3", 30),
	}

	resolved := ResolveThresholds(overrides, testDefault, inv)
	assert.InDelta(t, 30.0, resolved["svc-1"], 0.0001)
}

func TestResolveThresholdsIsOrderIndependent(t *testing.T) {
	t.Parallel()

	forward := []*AlertRuleThresholdOverride{
		override(ThresholdScopeCluster, "prod", 99),
		override(ThresholdScopeService, "svc-id-1", 50),
	}
	reversed := []*AlertRuleThresholdOverride{forward[1], forward[0]}

	inv := testInventory()
	assert.Equal(t,
		ResolveThresholds(forward, testDefault, inv),
		ResolveThresholds(reversed, testDefault, inv),
		"precedence must not depend on row order returned by the database")
}

// TestResolveThresholdsEmitsOneValuePerTarget guards the invariant that matters most
// operationally: two series with identical labels make the Prometheus gatherer fail the
// entire /metrics response, taking every other collector down with it.
func TestResolveThresholdsEmitsOneValuePerTarget(t *testing.T) {
	t.Parallel()

	inv := testInventory()
	inv.ServicesByCluster["prod"] = []string{"svc-1", "svc-1", "svc-2"}

	overrides := []*AlertRuleThresholdOverride{
		override(ThresholdScopeCluster, "prod", 70),
		override(ThresholdScopeNode, "node-id-1", 90),
		tombstone(ThresholdScopeNode, "node-id-2", 60),
	}

	resolved := ResolveThresholds(overrides, testDefault, inv)
	require.Len(t, resolved, 4)
	assert.InDelta(t, 70.0, resolved["svc-1"], 0.0001)
	assert.InDelta(t, 70.0, resolved["svc-2"], 0.0001)
	assert.InDelta(t, 90.0, resolved["node-1"], 0.0001)
	assert.InDelta(t, testDefault, resolved["node-2"], 0.0001)
}

func BenchmarkResolveThresholds(b *testing.B) {
	inv := ThresholdInventory{
		NodeNames:         make(map[string]string, 1000),
		ServiceNames:      map[string]string{},
		ServicesByCluster: make(map[string][]string, 50),
	}

	var overrides []*AlertRuleThresholdOverride
	for i := range 1000 {
		id := fmt.Sprintf("node-id-%d", i)
		inv.NodeNames[id] = fmt.Sprintf("node-%d", i)
		overrides = append(overrides, override(ThresholdScopeNode, id, float64(i%100)))
	}

	for i := range 50 {
		cluster := fmt.Sprintf("cluster-%d", i)
		services := make([]string, 0, 200)
		for j := range 200 {
			services = append(services, fmt.Sprintf("svc-%d-%d", i, j))
		}
		inv.ServicesByCluster[cluster] = services
		overrides = append(overrides, override(ThresholdScopeCluster, cluster, 55))
	}

	for b.Loop() {
		ResolveThresholds(overrides, testDefault, inv)
	}
}
