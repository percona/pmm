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

import prom "github.com/prometheus/client_golang/prometheus"

const (
	haPrometheusNamespace = "pmm"
	haPrometheusSubsystem = "ha"
)

// HAMetricsCollector is a Prometheus collector that exposes HA Raft metrics.
//
// The following metrics are exposed (only when HA mode is enabled):
//
//   - pmm_ha_leader_status - 1 if this node is the current Raft leader, 0
//     otherwise. Consumed by the pmm_ha_no_leader and pmm_ha_split_brain
//     alert templates, which sum it across the cluster.
//
//   - pmm_ha_raft_term - The current Raft consensus term. Rapid growth
//     indicates an unstable network or a crashing leader. Consumed by the
//     pmm_ha_leader_flapping alert template.
//
//   - pmm_ha_up{role="voter|nonvoter"} - Always 1 for a live node. The role
//     label reports whether the node is in the current Raft configuration. A
//     node that goes down stops emitting the series rather than reporting 0.
//
//   - pmm_ha_expected_nodes - The number of nodes configured for this cluster,
//     used as the denominator for node-down and quorum alerting.
type HAMetricsCollector struct { //nolint:revive
	haService *Service

	mLeaderStatus  *prom.Desc
	mRaftTerm      *prom.Desc
	mUp            *prom.Desc
	mExpectedNodes *prom.Desc
}

// NewHAMetricsCollector creates a new HAMetricsCollector backed by the
// provided HA service.
func NewHAMetricsCollector(haService *Service) *HAMetricsCollector {
	return &HAMetricsCollector{
		haService: haService,
		mLeaderStatus: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "leader_status"),
			"Reports whether this PMM node currently holds the Raft leader lease. "+
				"Value is 1 for the leader and 0 for followers.",
			[]string{"node_id"},
			nil,
		),
		mRaftTerm: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "raft_term"),
			"The current Raft consensus term number as seen by this node. "+
				"Rapid increases indicate leader instability or frequent elections (leader flapping).",
			[]string{"node_id"},
			nil,
		),
		mUp: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "up"),
			"Reports that this PMM node is up and participating in the cluster. "+
				"The 'role' label is 'voter' when the node is in the current Raft configuration "+
				"and votes in elections, and 'nonvoter' when it is absent from that configuration.",
			[]string{"node_id", "role"},
			nil,
		),
		mExpectedNodes: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "expected_nodes"),
			"The number of PMM nodes configured for this HA cluster.",
			[]string{"node_id"},
			nil,
		),
	}
}

// Describe implements prom.Collector.
func (c *HAMetricsCollector) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(c, ch)
}

// Collect implements prom.Collector.
//
// No metrics are emitted when HA mode is disabled so that standalone PMM
// deployments do not trigger HA-specific alerting rules.
func (c *HAMetricsCollector) Collect(ch chan<- prom.Metric) {
	m := c.haService.GetMetrics()
	if !m.Enabled {
		return
	}

	nodeID := c.haService.Params().NodeID

	leaderValue := float64(0)
	if m.IsLeader {
		leaderValue = 1
	}
	ch <- prom.MustNewConstMetric(c.mLeaderStatus, prom.GaugeValue, leaderValue, nodeID)
	ch <- prom.MustNewConstMetric(c.mRaftTerm, prom.GaugeValue, float64(m.RaftTerm), nodeID)

	role := "nonvoter"
	if m.IsVoter {
		role = "voter"
	}
	ch <- prom.MustNewConstMetric(c.mUp, prom.GaugeValue, 1, nodeID, role)
	ch <- prom.MustNewConstMetric(c.mExpectedNodes, prom.GaugeValue, float64(m.ExpectedNodes), nodeID)
}

var _ prom.Collector = (*HAMetricsCollector)(nil)
