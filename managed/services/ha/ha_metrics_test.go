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

package ha

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// collectSamples gathers the collector's metrics and returns them keyed by
// `name{label="value",...}`, so assertions do not depend on the exact wording of
// the metric HELP strings.
func collectSamples(t *testing.T, c *HAMetricsCollector) map[string]float64 {
	t.Helper()

	reg := prom.NewPedanticRegistry()
	require.NoError(t, reg.Register(c))

	families, err := reg.Gather()
	require.NoError(t, err)

	samples := make(map[string]float64)
	for _, mf := range families {
		for _, m := range mf.GetMetric() {
			labels := make([]string, 0, len(m.GetLabel()))
			for _, l := range m.GetLabel() {
				labels = append(labels, fmt.Sprintf("%s=%q", l.GetName(), l.GetValue()))
			}
			sort.Strings(labels)

			key := mf.GetName()
			if len(labels) != 0 {
				key = fmt.Sprintf("%s{%s}", key, strings.Join(labels, ","))
			}
			samples[key] = m.GetGauge().GetValue()
		}
	}

	return samples
}

func TestHAMetricsCollector_Disabled(t *testing.T) {
	t.Parallel()

	c := NewHAMetricsCollector(&Service{
		params: &models.HAParams{
			Enabled: false,
			NodeID:  "node-1",
			Nodes:   []string{"node-1", "node-2", "node-3"},
		},
	})

	// Nothing at all must be emitted when HA is disabled: the built-in HA alert
	// templates rely on the absence of these series to stay silent on standalone PMM.
	assert.Empty(t, collectSamples(t, c))
}

func TestHAMetricsCollector_EnabledBeforeRaftInit(t *testing.T) {
	t.Parallel()

	// raftNode is nil, which is the state during early startup: HA is enabled but
	// Raft has not been initialised yet.
	c := NewHAMetricsCollector(&Service{
		params: &models.HAParams{
			Enabled: true,
			NodeID:  "node-1",
			Nodes:   []string{"node-1", "node-2", "node-3"},
		},
	})

	assert.Equal(t, map[string]float64{
		`pmm_ha_leader_status{node_id="node-1"}`:      0,
		`pmm_ha_raft_term{node_id="node-1"}`:          0,
		`pmm_ha_up{node_id="node-1",role="nonvoter"}`: 1,
		`pmm_ha_expected_nodes{node_id="node-1"}`:     3,
	}, collectSamples(t, c))
}

func TestHAMetricsCollector_ExpectedNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		nodes []string
		want  float64
	}{
		{
			name:  "no peers defaults to one",
			nodes: nil,
			want:  1,
		},
		{
			name:  "single node cluster",
			nodes: []string{"node-1"},
			want:  1,
		},
		{
			name:  "three node cluster",
			nodes: []string{"node-1", "node-2", "node-3"},
			want:  3,
		},
		{
			name:  "five node cluster",
			nodes: []string{"node-1", "node-2", "node-3", "node-4", "node-5"},
			want:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewHAMetricsCollector(&Service{
				params: &models.HAParams{
					Enabled: true,
					NodeID:  "node-1",
					Nodes:   tt.nodes,
				},
			})

			samples := collectSamples(t, c)
			assert.InDelta(t, tt.want, samples[`pmm_ha_expected_nodes{node_id="node-1"}`], 0.0001)
		})
	}
}

func TestHAMetricsCollector_Describe(t *testing.T) {
	t.Parallel()

	c := NewHAMetricsCollector(&Service{
		params: &models.HAParams{
			Enabled: true,
			NodeID:  "node-1",
			Nodes:   []string{"node-1"},
		},
	})

	ch := make(chan *prom.Desc, 10)
	c.Describe(ch)
	close(ch)

	descs := make([]string, 0, len(ch))
	for d := range ch {
		descs = append(descs, d.String())
	}

	// Guards against emitting a metric in Collect without a matching descriptor.
	assert.Len(t, descs, 4)
}
