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
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	// thresholdCollectTimeout bounds one scrape. The threshold collector shares
	// /debug/metrics, and its 0.9 * MR budget, with the inventory and HA collectors, so
	// overrunning here would take those down too.
	thresholdCollectTimeout = 3 * time.Second

	// thresholdCtxCheckInterval is how often the emission loop re-checks the deadline.
	// The queries are bounded by the context, but the loop that follows them is not,
	// so without this a large enough result set could run past the scrape budget with
	// nothing stopping it.
	thresholdCtxCheckInterval = 1000

	// thresholdMetricName is the gauge the injected threshold query reads. It is shared
	// with rule_builder.go on purpose: the metric name and its label set are a contract
	// between the collector and the generated PromQL, and a rule pointing at a metric
	// nobody emits fails silently - it simply never fires.
	thresholdMetricName = "pmm_alert_threshold_override"

	thresholdRuleIDLabel = "rule_id"
	thresholdParamLabel  = "param"
	// thresholdTargetLabel is generic rather than node_name/service_name because one
	// fixed descriptor has to serve every scope; the rule query maps it onto whichever
	// label it joins on with label_replace.
	thresholdTargetLabel = "target"
)

// AlertThresholdMetricsCollector exposes the effective threshold for every target that
// carries an override.
//
// Only overridden targets are emitted. Emitting a series for every target instead would
// scale with inventory rather than with what was actually tuned: measured at 1,000 nodes
// and 140 rule/parameter groups it consumed 77-82% of the 9s scrape budget, against 1-7%
// for this shape. Targets with no override get their threshold from the default clause
// of the rule query instead.
type AlertThresholdMetricsCollector struct {
	db *reform.DB
	l  *logrus.Entry

	desc *prom.Desc
}

// NewAlertThresholdMetricsCollector creates a new instance of AlertThresholdMetricsCollector.
func NewAlertThresholdMetricsCollector(db *reform.DB) *AlertThresholdMetricsCollector {
	return &AlertThresholdMetricsCollector{
		db: db,
		l:  logrus.WithField("component", "alerting/threshold-metrics"),
		desc: prom.NewDesc(
			thresholdMetricName,
			"Effective alert threshold for a rule parameter and target. Emitted only where an "+
				"override or a tombstone exists; targets without either fall back to the rule's "+
				"default, which the rule query materialises from its own observed expression.",
			[]string{thresholdRuleIDLabel, thresholdParamLabel, thresholdTargetLabel},
			nil,
		),
	}
}

// Describe sends the metric description to the provided channel.
//
// This deliberately does not use prom.DescribeByCollect, which would run a full Collect,
// and therefore a database query, merely to describe the collector.
func (c *AlertThresholdMetricsCollector) Describe(ch chan<- *prom.Desc) {
	ch <- c.desc
}

// thresholdGroup is the set of override rows sharing one rule and parameter, which is
// the granularity precedence is resolved at.
type thresholdGroup struct {
	ruleID    string
	paramName string
	overrides []*models.AlertRuleThresholdOverride
}

// Collect sends the collected metrics to the provided channel. A failure is logged and
// yields no threshold metrics for that scrape rather than failing the whole response.
func (c *AlertThresholdMetricsCollector) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), thresholdCollectTimeout)
	defer cancelCtx()

	var (
		groups []thresholdGroup
		rules  map[string]*models.AlertRule
		inv    models.ThresholdInventory
	)

	errTx := c.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		overrides, err := models.FindAllThresholdOverrides(tx.Querier)
		if err != nil {
			return err
		}

		// The common case is no overrides at all, and it costs nothing.
		if len(overrides) == 0 {
			return nil
		}

		groups = groupThresholdOverrides(overrides)

		allRules, err := models.FindAlertRules(tx.Querier)
		if err != nil {
			return err
		}

		rules = make(map[string]*models.AlertRule, len(allRules))
		for _, rule := range allRules {
			rules[rule.RuleID] = rule
		}

		inv, err = loadThresholdInventory(tx.Querier, overrides)

		return err
	})
	if errTx != nil {
		c.l.Warnf("Failed to collect alert thresholds: %v", errTx)

		return
	}

	emitted := 0
	for _, group := range groups {
		rule, ok := rules[group.ruleID]
		if !ok {
			continue
		}

		param, ok := rule.Params[group.paramName]
		if !ok {
			continue
		}

		for target, value := range models.ResolveThresholds(group.overrides, param.Default, inv) {
			if emitted%thresholdCtxCheckInterval == 0 && ctx.Err() != nil {
				c.l.Warnf("Alert threshold collection timed out after %d series", emitted)

				return
			}
			emitted++

			ch <- prom.MustNewConstMetric(c.desc, prom.GaugeValue, value, group.ruleID, group.paramName, target)
		}
	}
}

// groupThresholdOverrides partitions rows by rule and parameter, which is the unit
// precedence applies within: an override on one parameter says nothing about another.
func groupThresholdOverrides(overrides []*models.AlertRuleThresholdOverride) []thresholdGroup {
	type key struct {
		ruleID    string
		paramName string
	}

	index := make(map[key]int)

	var groups []thresholdGroup

	for _, override := range overrides {
		k := key{ruleID: override.RuleID, paramName: override.ParamName}

		i, ok := index[k]
		if !ok {
			groups = append(groups, thresholdGroup{ruleID: k.ruleID, paramName: k.paramName})
			i = len(groups) - 1
			index[k] = i
		}

		groups[i].overrides = append(groups[i].overrides, override)
	}

	return groups
}

// loadThresholdInventory resolves the IDs the overrides actually reference, rather than
// reading the whole inventory. That is what keeps this collector's cost a function of
// how many targets were tuned instead of how large the fleet is.
func loadThresholdInventory(q *reform.Querier, overrides []*models.AlertRuleThresholdOverride) (models.ThresholdInventory, error) {
	var nodeIDs, serviceIDs []string

	for _, override := range overrides {
		switch override.Scope {
		case models.ThresholdScopeNode:
			nodeIDs = append(nodeIDs, override.Target)
		case models.ThresholdScopeService:
			serviceIDs = append(serviceIDs, override.Target)
		case models.ThresholdScopeCluster:
			// Cluster scope needs services looked up by cluster label, which arrives
			// with the service/cluster increment. Until then such a row cannot be
			// created through the API, and one inserted directly resolves to nothing
			// and is simply inert.
		}

		// do not add `default:` to make exhaustive linter do its job
	}

	inv := models.ThresholdInventory{}

	if len(nodeIDs) != 0 {
		nodes, err := models.FindNodesByIDs(q, nodeIDs)
		if err != nil {
			return inv, err
		}

		inv.NodeNames = make(map[string]string, len(nodes))
		for _, node := range nodes {
			inv.NodeNames[node.NodeID] = node.NodeName
		}
	}

	if len(serviceIDs) != 0 {
		services, err := models.FindServicesByIDs(q, serviceIDs)
		if err != nil {
			return inv, err
		}

		inv.ServiceNames = make(map[string]string, len(services))
		for id, service := range services {
			inv.ServiceNames[id] = service.ServiceName
		}
	}

	return inv, nil
}

// check interfaces.
var _ prom.Collector = (*AlertThresholdMetricsCollector)(nil)
