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

package alerting

import (
	"context"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// thresholdMetricFQName is the fully-qualified name of the emitted metric. It is
// kept in sync with thresholdMetricName used by the rule builder to reference it.
const thresholdMetricFQName = thresholdMetricName

// AlertThresholdMetricsCollector is a Prometheus collector that exposes the
// effective per-node threshold for every overridable parameter of every alert
// rule created from a template.
//
// For each registered rule it emits, for every node in the inventory, a
// pmm_alert_threshold{rule_id, param, node_name} gauge whose value is the
// per-node override when one exists, or the rule's default otherwise. Alert
// rules built from overridable templates compare their observed query against
// this series (see rule_builder.go), so emitting a value for every node ensures
// unoverridden nodes still evaluate against the default.
type AlertThresholdMetricsCollector struct {
	db *reform.DB
	l  *logrus.Entry

	desc *prom.Desc
}

// NewAlertThresholdMetricsCollector creates a collector backed by the given DB.
func NewAlertThresholdMetricsCollector(db *reform.DB) *AlertThresholdMetricsCollector {
	return &AlertThresholdMetricsCollector{
		db: db,
		l:  logrus.WithField("component", "alerting/threshold-metrics"),
		desc: prom.NewDesc(
			thresholdMetricFQName,
			"Effective per-node threshold value for an overridable alert rule parameter. "+
				"Alert rules created from overridable templates compare their observed value "+
				"against this metric, so per-node overrides take effect without editing the rule.",
			[]string{"rule_id", "param", thresholdJoinLabel},
			nil,
		),
	}
}

// Describe implements prom.Collector.
func (c *AlertThresholdMetricsCollector) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(c, ch)
}

// Collect implements prom.Collector. Failures are logged and yield no metrics
// for the affected scrape rather than aborting the whole /metrics response.
func (c *AlertThresholdMetricsCollector) Collect(ch chan<- prom.Metric) {
	err := c.db.InTransactionContext(context.Background(), nil, func(tx *reform.TX) error {
		rules, err := models.FindAlertRules(tx.Querier)
		if err != nil {
			return err
		}
		if len(rules) == 0 {
			return nil
		}

		nodes, err := models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		for _, rule := range rules {
			if len(rule.DefaultParams) == 0 {
				continue
			}

			overrides, err := models.FindThresholdOverridesByRule(tx.Querier, rule.RuleID)
			if err != nil {
				return err
			}

			// param -> node_id -> override value
			byParamNode := make(map[string]map[string]float64, len(rule.DefaultParams))
			for _, o := range overrides {
				m, ok := byParamNode[o.ParamName]
				if !ok {
					m = make(map[string]float64)
					byParamNode[o.ParamName] = m
				}
				m[o.NodeID] = o.Value
			}

			for paramName, defaultValue := range rule.DefaultParams {
				for _, node := range nodes {
					value := defaultValue
					if nodeOverrides, ok := byParamNode[paramName]; ok {
						if ov, ok := nodeOverrides[node.NodeID]; ok {
							value = ov
						}
					}

					ch <- prom.MustNewConstMetric(c.desc, prom.GaugeValue, value, rule.RuleID, paramName, node.NodeName)
				}
			}
		}

		return nil
	})
	if err != nil {
		c.l.Errorf("failed to collect alert threshold metrics: %v", err)
	}
}

var _ prom.Collector = (*AlertThresholdMetricsCollector)(nil)
