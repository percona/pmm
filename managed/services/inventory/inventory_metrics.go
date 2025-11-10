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

package inventory

import (
	"context"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

const (
	requestTimeout              = 3 * time.Second
	serviceEnabled      float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "inventory"
)

// Metric represents a metric for inventory purposes.
type Metric struct {
	labels []string
	value  float64
}

// InventoryMetrics represents a collection of inventory metrics.
type InventoryMetrics struct { //nolint:revive
	db       *reform.DB
	registry agentsRegistry
}

// InventoryMetricsCollector collects inventory metrics.
type InventoryMetricsCollector struct { //nolint:revive
	mAgentsDesc   *prom.Desc
	mNodesDesc    *prom.Desc
	mServicesDesc *prom.Desc

	metrics inventoryMetrics
}

// NewInventoryMetrics creates a new instance of InventoryMetrics.
func NewInventoryMetrics(db *reform.DB, registry agentsRegistry) *InventoryMetrics {
	return &InventoryMetrics{
		db:       db,
		registry: registry,
	}
}

// Standard label names for each object type (sorted for consistency)
var (
	// Node standard labels (from Node.UnifiedLabels)
	nodeLabelNames = []string{"az", "container_id", "container_name", "machine_id", "node_id", "node_model", "node_name", "node_type", "region"}

	// Service standard labels (from Service.UnifiedLabels + node labels from MergeLabels)
	serviceLabelNames = []string{"az", "cluster", "container_id", "container_name", "environment", "external_group", "machine_id", "node_id", "node_model", "node_name", "node_type", "region", "replication_set", "service_id", "service_name", "service_type"}

	// Agent standard labels (from Agent.UnifiedLabels + service + node labels from MergeLabels)
	// Note: agent_id, agent_type are from agent, others are from node/service
	// Additional agent-specific fields: disabled, pmm_agent_id, version
	agentLabelNames = []string{"agent_id", "agent_type", "az", "cluster", "container_id", "container_name", "disabled", "environment", "external_group", "machine_id", "node_id", "node_model", "node_name", "node_type", "pmm_agent_id", "region", "replication_set", "service_id", "service_name", "service_type", "version"}
)

// NewInventoryMetricsCollector creates a new instance of InventoryMetricsCollector.
func NewInventoryMetricsCollector(metrics inventoryMetrics) *InventoryMetricsCollector {
	return &InventoryMetricsCollector{
		mAgentsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "agents"),
			"Inventory Agent",
			agentLabelNames,
			nil),
		mNodesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "nodes"),
			"Inventory Node",
			nodeLabelNames,
			nil),
		mServicesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
			"Inventory Service",
			serviceLabelNames,
			nil),

		metrics: metrics,
	}
}

// extractLabelValues extracts label values in the order specified by labelNames from the labels map.
func extractLabelValues(labels map[string]string, labelNames []string) []string {
	values := make([]string, len(labelNames))
	for i, name := range labelNames {
		values[i] = labels[name]
	}
	return values
}

func getRunsOnNodeIDByPMMAgentID(agents []*models.Agent, pmmAgentID string) string {
	for _, agent := range agents {
		if agent.AgentID == pmmAgentID {
			return pointer.GetString(agent.RunsOnNodeID)
		}
	}
	return ""
}

// GetAgentMetrics retrieves agent metrics from InventoryMetrics.
func (i *InventoryMetrics) GetAgentMetrics(ctx context.Context) ([]Metric, error) {
	metrics := []Metric{}

	errTx := i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		filters := models.AgentFilters{}
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		filters.IgnoreNomad = !settings.IsNomadEnabled()

		dbAgents, err := models.FindAgents(tx.Querier, filters)
		if err != nil {
			return err
		}

		dbNodes, err := models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		dbServices, err := models.FindServices(tx.Querier, models.ServiceFilters{})
		if err != nil {
			return err
		}

		nodeMap := make(map[string]*models.Node, len(dbNodes))
		for _, node := range dbNodes {
			nodeMap[node.NodeID] = node
		}

		serviceMap := make(map[string]*models.Service, len(dbServices))
		for _, service := range dbServices {
			serviceMap[service.ServiceID] = service
		}

		for _, agent := range dbAgents {
			metricValue := float64(0)
			var node *models.Node
			var service *models.Service

			// Determine which node this agent runs on
			runsOnNodeID := pointer.GetString(agent.RunsOnNodeID)
			if runsOnNodeID == "" {
				// For non-PMM agents, find the node via PMM agent
				pmmAgentID := pointer.GetString(agent.PMMAgentID)
				if pmmAgentID != "" {
					runsOnNodeID = getRunsOnNodeIDByPMMAgentID(dbAgents, pmmAgentID)
				}
			}

			if runsOnNodeID != "" {
				node = nodeMap[runsOnNodeID]
			}

			// Get service if agent is associated with one
			serviceID := pointer.GetString(agent.ServiceID)
			if serviceID != "" {
				service = serviceMap[serviceID]
			}

			// Determine metric value
			if agent.AgentType == models.PMMAgentType {
				if i.registry.IsConnected(agent.AgentID) {
					metricValue = 1
				}
			} else {
				metricValue = float64(inventoryv1.AgentStatus_value[agent.Status])
			}

			// Merge labels: node + service + agent (same as scrape configs)
			labels, err := models.MergeLabels(node, service, agent)
			if err != nil {
				return err
			}

			// Add agent-specific fields that aren't in UnifiedLabels
			disabled := "0"
			if agent.Disabled {
				disabled = "1"
			}
			labels["disabled"] = disabled
			labels["pmm_agent_id"] = pointer.GetString(agent.PMMAgentID)
			labels["version"] = pointer.GetString(agent.Version)

			// Extract values in the order defined by agentLabelNames
			labelValues := extractLabelValues(labels, agentLabelNames)

			metrics = append(metrics, Metric{labels: labelValues, value: metricValue})
		}
		return nil
	})

	if errTx != nil {
		return nil, errors.WithStack(errTx)
	}
	return metrics, nil
}

// GetNodeMetrics retrieves node metrics from InventoryMetrics.
func (i *InventoryMetrics) GetNodeMetrics(ctx context.Context) ([]Metric, error) {
	var metrics []Metric

	errTx := i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbNodes, err := models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		for _, node := range dbNodes {
			labels, err := node.UnifiedLabels()
			if err != nil {
				return err
			}

			// Extract values in the order defined by nodeLabelNames
			labelValues := extractLabelValues(labels, nodeLabelNames)

			metrics = append(metrics, Metric{labels: labelValues, value: serviceEnabled})
		}

		return nil
	})

	if errTx != nil {
		return nil, errors.WithStack(errTx)
	}
	return metrics, nil
}

// GetServiceMetrics retrieves service metrics from InventoryMetrics.
func (i *InventoryMetrics) GetServiceMetrics(ctx context.Context) ([]Metric, error) {
	var metrics []Metric

	errTx := i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbServices, err := models.FindServices(tx.Querier, models.ServiceFilters{})
		if err != nil {
			return err
		}

		dbNodes, err := models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		nodeMap := make(map[string]*models.Node, len(dbNodes))
		for _, node := range dbNodes {
			nodeMap[node.NodeID] = node
		}

		for _, service := range dbServices {
			// Get node for this service to include node labels
			node := nodeMap[service.NodeID]

			// Merge labels: node + service (same as scrape configs)
			labels, err := models.MergeLabels(node, service, nil)
			if err != nil {
				return err
			}

			// Extract values in the order defined by serviceLabelNames
			labelValues := extractLabelValues(labels, serviceLabelNames)

			metrics = append(metrics, Metric{labels: labelValues, value: serviceEnabled})
		}

		return nil
	})

	if errTx != nil {
		return nil, errors.WithStack(errTx)
	}
	return metrics, nil
}

// Describe describes the InventoryMetricsCollector for Prometheus.
func (i *InventoryMetricsCollector) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(i, ch)
}

// Collect collects metrics for the InventoryMetricsCollector.
func (i *InventoryMetricsCollector) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelCtx()

	ctx = logger.Set(ctx, "inventoryMetrics")
	l := logger.Get(ctx)

	agentMetrics, err := i.metrics.GetAgentMetrics(ctx)
	if err != nil {
		l.Error(err)
		return
	}

	for _, agentMetric := range agentMetrics {
		ch <- prom.MustNewConstMetric(i.mAgentsDesc, prom.GaugeValue, agentMetric.value, agentMetric.labels...)
	}

	nodeMetrics, nodeError := i.metrics.GetNodeMetrics(ctx)

	if nodeError != nil {
		l.Error(nodeError)
		return
	}

	for _, nodeMetric := range nodeMetrics {
		ch <- prom.MustNewConstMetric(i.mNodesDesc, prom.GaugeValue, nodeMetric.value, nodeMetric.labels...)
	}

	serviceMetrics, serviceError := i.metrics.GetServiceMetrics(ctx)

	if serviceError != nil {
		l.Error(serviceError)
		return
	}

	for _, serviceMetric := range serviceMetrics {
		ch <- prom.MustNewConstMetric(i.mServicesDesc, prom.GaugeValue, serviceMetric.value, serviceMetric.labels...)
	}
}

var _ prom.Collector = (*InventoryMetricsCollector)(nil)
