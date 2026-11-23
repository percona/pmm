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
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
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
			[]string{"agent_id", "agent_type", "service_id", "service_name", "node_id", "node_name", "environment", "cluster", "pmm_agent_id", "disabled", "version"},
			nil,
		),
		mNodesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "nodes"),
			"Inventory Node",
			[]string{"node_id", "node_type", "node_name", "container_name"},
			nil,
		),
		mServicesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
			"Inventory Service",
			[]string{"service_id", "service_type", "node_id"},
			nil,
		),

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

// scrapedEnvironmentAndCluster returns the environment and cluster labels that the metrics of
// the service are scraped with: the labels of its Node, the Service and the Agent merged, so a
// custom label on the Node or the Agent can set or override them. Reporting the same values lets
// the inventory metric be matched with the service's own metrics.
//
// Labels that can't be merged must not take the inventory metric away for every agent, since the
// service down alerts read it. The service's own values are returned then, along with the error.
func scrapedEnvironmentAndCluster(node *models.Node, service *models.Service, agent *models.Agent) (string, string, error) {
	labels, err := models.MergeLabels(node, service, agent)
	if err != nil {
		return service.Environment, service.Cluster, err
	}

	return labels["environment"], labels["cluster"], nil
}

// agentMetric returns the pmm_managed_inventory_agents series of one agent. The agents are needed
// to find the pmm-agent that runs it, and the nodes and services are keyed by their IDs.
//
// Only an agent of a service gets an environment and a cluster: nothing reads them for an agent
// of a node, such as node_exporter. Its own metrics do carry the node's labels, so the inventory
// metric can't be matched with them by environment or cluster.
func (i *InventoryMetrics) agentMetric(
	ctx context.Context,
	agent *models.Agent,
	agents []*models.Agent,
	nodes map[string]*models.Node,
	services map[string]*models.Service,
) Metric {
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
		runsOnNodeID = getRunsOnNodeIDByPMMAgentID(agents, pmmAgentID)
	}

	nodeName := ""
	if node, ok := nodes[runsOnNodeID]; ok {
		nodeName = node.NodeName
	}

	serviceID := pointer.GetString(agent.ServiceID)
	serviceName := ""
	environment := ""
	cluster := ""

	if service, ok := services[serviceID]; ok {
		serviceName = service.ServiceName

		// The Node is the one the Service belongs to, not the one the Agent runs on, which
		// differs for remote monitoring.
		var err error
		environment, cluster, err = scrapedEnvironmentAndCluster(nodes[service.NodeID], service, agent)
		if err != nil {
			logger.Get(ctx).Warnf("Failed to merge the labels of agent %s, using the environment and cluster of its service: %s.", agent.AgentID, err)
		}
	}

	return Metric{
		labels: []string{
			agent.AgentID,
			string(agent.AgentType),
			serviceID,
			serviceName,
			runsOnNodeID,
			nodeName,
			environment,
			cluster,
			pmmAgentID,
			disabled,
			pointer.GetString(agent.Version),
		},
		value: metricValue,
	}
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

		nodes := make(map[string]*models.Node, len(dbNodes))
		for _, node := range dbNodes {
			nodes[node.NodeID] = node
		}

		services := make(map[string]*models.Service, len(dbServices))
		for _, service := range dbServices {
			services[service.ServiceID] = service
		}

		for _, agent := range dbAgents {
			metrics = append(metrics, i.agentMetric(ctx, agent, dbAgents, nodes, services))
		}
		return nil
	})

	if errTx != nil {
		return nil, fmt.Errorf("failed to get agent metrics: %w", errTx)
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
		return nil, fmt.Errorf("failed to get node metrics: %w", errTx)
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
		return nil, fmt.Errorf("failed to get service metrics: %w", errTx)
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
