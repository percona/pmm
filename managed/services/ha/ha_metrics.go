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
//   - pmm_ha_leader_status  – 1 if this node is the current Raft leader, 0
//     otherwise.  Summing this across all nodes in the cluster enables the
//     PMMHALeaderMissing (sum == 0) and PMMHASplitBrain (sum > 1) alerts.
//
//   - pmm_ha_raft_term  – The current Raft consensus term.  Rapid growth
//     (changes(pmm_ha_raft_term[10m]) > 5) triggers the PMMHALeaderFlapping
//     alert that indicates an unstable network or crashing leader.
//
//   - pmm_ha_up{role="voter|nonvoter"}  – Always 1 for a live node, labelled
//     with the node's Raft suffrage role.  count(pmm_ha_up{role="voter"}) < 3
//     triggers the PMMHAQuorumAtRisk alert for a three-node cluster.
type HAMetricsCollector struct { //nolint:revive
	haService *Service

	mLeaderStatus *prom.Desc
	mRaftTerm     *prom.Desc
	mUp           *prom.Desc
}

// NewHAMetricsCollector creates a new HAMetricsCollector backed by the
// provided HA service.
func NewHAMetricsCollector(haService *Service) *HAMetricsCollector {
	return &HAMetricsCollector{
		haService: haService,
		mLeaderStatus: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "leader_status"),
			"Reports whether this PMM node currently holds the Raft leader lease. "+
				"Value is 1 for the leader and 0 for followers. "+
				"Use sum(pmm_ha_leader_status) to detect split-brain (>1) or a missing leader (==0).",
			[]string{"node_id"},
			nil),
		mRaftTerm: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "raft_term"),
			"The current Raft consensus term number as seen by this node. "+
				"Rapid increases indicate leader instability or frequent elections (leader flapping). "+
				"Use changes(pmm_ha_raft_term[10m]) > 5 to fire the PMMHALeaderFlapping alert.",
			[]string{"node_id"},
			nil),
		mUp: prom.NewDesc(
			prom.BuildFQName(haPrometheusNamespace, haPrometheusSubsystem, "up"),
			"Reports that this PMM node is up and participating in the cluster. "+
				"The 'role' label indicates the node's Raft suffrage: 'voter' nodes participate "+
				"in elections, 'nonvoter' nodes only replicate logs. "+
				"Use count(pmm_ha_up{role=\"voter\"}) to evaluate quorum health.",
			[]string{"node_id", "role"},
			nil),
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
}

var _ prom.Collector = (*HAMetricsCollector)(nil)
