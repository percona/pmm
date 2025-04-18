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

// NewInventoryMetricsCollector creates a new instance of InventoryMetricsCollector.
func NewInventoryMetricsCollector(metrics inventoryMetrics) *InventoryMetricsCollector {
	return &InventoryMetricsCollector{
		mAgentsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "agents"),
			"Inventory Agent",
			[]string{"agent_id", "agent_type", "service_id", "node_id", "node_name", "pmm_agent_id", "disabled", "version"},
			nil),
		mNodesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "nodes"),
			"Inventory Node",
			[]string{"node_id", "node_type", "node_name", "container_name"},
			nil),
		mServicesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
			"Inventory Service",
			[]string{"service_id", "service_type", "node_id"},
			nil),

		metrics: metrics,
	}
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

		nodeMap := make(map[string]string, len(dbNodes))
		for _, node := range dbNodes {
			nodeMap[node.NodeID] = node.NodeName
		}

		for _, agent := range dbAgents {
			runsOnNodeID := ""
			disabled := "0"
			metricValue := float64(0)

			pmmAgentID := pointer.GetString(agent.PMMAgentID)

			if agent.Disabled {
				disabled = "1"
			}

			if agent.AgentType == models.PMMAgentType {
				if i.registry.IsConnected(agent.AgentID) {
					metricValue = 1
				}
				runsOnNodeID = pointer.GetString(agent.RunsOnNodeID)
			} else {
				metricValue = float64(inventoryv1.AgentStatus_value[agent.Status])
				runsOnNodeID = getRunsOnNodeIDByPMMAgentID(dbAgents, pmmAgentID)
			}

			nodeName := nodeMap[runsOnNodeID]
			agentMetricLabels := []string{
				agent.AgentID,
				string(agent.AgentType),
				pointer.GetString(agent.ServiceID),
				runsOnNodeID,
				nodeName,
				pmmAgentID,
				disabled,
				pointer.GetString(agent.Version),
			}

			metrics = append(metrics, Metric{labels: agentMetricLabels, value: metricValue})
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
			nodeMetricLabels := []string{
				node.NodeID,
				string(node.NodeType),
				node.NodeName,
				pointer.GetString(node.ContainerName),
			}

			metrics = append(metrics, Metric{labels: nodeMetricLabels, value: serviceEnabled})
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

		for _, service := range dbServices {
			serviceMetricLabels := []string{
				service.ServiceID,
				string(service.ServiceType),
				service.NodeID,
			}

			metrics = append(metrics, Metric{labels: serviceMetricLabels, value: serviceEnabled})
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
