// Copyright (C) 2017 Percona LLC
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
	"strconv"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	requestTimeout              = 3 * time.Second
	serviceEnabled      float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "inventory"
)

type Metric struct {
	labels []string
	value  float64
}

//goland:noinspection GoNameStartsWithPackageName
type InventoryMetrics struct {
	db       *reform.DB
	registry agentsRegistry

	mutex sync.Mutex
}

//goland:noinspection GoNameStartsWithPackageName
type InventoryMetricsCollector struct {
	mAgentsDesc   *prom.Desc
	mNodesDesc    *prom.Desc
	mServicesDesc *prom.Desc

	metrics inventoryMetrics

	mutex sync.Mutex
}

func NewInventoryMetrics(db *reform.DB, registry agentsRegistry) *InventoryMetrics {
	return &InventoryMetrics{
		db:       db,
		registry: registry,
	}
}

func NewInventoryMetricsCollector(metrics inventoryMetrics) *InventoryMetricsCollector {
	return &InventoryMetricsCollector{
		mAgentsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "agents"),
			"Inventory Agent",
			[]string{"agent_id", "agent_type", "service_id", "node_id", "pmm_agent_id", "disabled", "version"},
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

func (i *InventoryMetrics) GetAgentMetrics(ctx context.Context) (metrics []Metric, err error) {
	metrics = []Metric{}

	err = i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbAgents, dbAgentsError := models.FindAgents(tx.Querier, models.AgentFilters{})

		if dbAgentsError != nil {
			return dbAgentsError
		}

		for _, agent := range dbAgents {
			disabled := 0
			connected := float64(0)

			pmmAgentID := pointer.GetString(agent.PMMAgentID)

			if agent.Disabled {
				disabled = 1
			} else {
				disabled = 0
			}

			if i.registry.IsConnected(pmmAgentID) {
				connected = 1
			} else {
				connected = 0
			}

			agentMetricLabels := []string{
				agent.AgentID,
				string(agent.AgentType),
				pointer.GetString(agent.ServiceID),
				pointer.GetString(agent.NodeID),
				pmmAgentID,
				strconv.Itoa(disabled),
				pointer.GetString(agent.Version),
			}

			metrics = append(metrics, Metric{labels: agentMetricLabels, value: connected})
		}
		return nil
	})

	return metrics, err
}

func (i *InventoryMetrics) GetNodeMetrics(ctx context.Context) (metrics []Metric, err error) {
	metrics = []Metric{}

	err = i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbNodes, dbNodesError := models.FindNodes(tx.Querier, models.NodeFilters{})

		if dbNodesError != nil {
			return dbNodesError
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

	return metrics, err
}

func (i *InventoryMetrics) GetServiceMetrics(ctx context.Context) (metrics []Metric, err error) {
	metrics = []Metric{}

	err = i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbServices, dbServicesError := models.FindServices(tx.Querier, models.ServiceFilters{})

		if dbServicesError != nil {
			return dbServicesError
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

	return metrics, err
}

func (i *InventoryMetricsCollector) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(i, ch)
}

func (i *InventoryMetricsCollector) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelCtx()

	i.mutex.Lock()
	defer i.mutex.Unlock()

	agentMetrics, agentError := i.metrics.GetAgentMetrics(ctx)

	if agentError != nil {
		fmt.Println(agentError)
		return
	}

	for _, agentMetric := range agentMetrics {
		ch <- prom.MustNewConstMetric(i.mAgentsDesc, prom.GaugeValue, agentMetric.value, agentMetric.labels...)
	}

	nodeMetrics, nodeError := i.metrics.GetNodeMetrics(ctx)

	if nodeError != nil {
		fmt.Println(nodeError)
		return
	}

	for _, nodeMetric := range nodeMetrics {
		ch <- prom.MustNewConstMetric(i.mNodesDesc, prom.GaugeValue, nodeMetric.value, nodeMetric.labels...)
	}

	serviceMetrics, serviceError := i.metrics.GetServiceMetrics(ctx)

	if serviceError != nil {
		fmt.Println(serviceError)
		return
	}

	for _, serviceMetric := range serviceMetrics {
		ch <- prom.MustNewConstMetric(i.mServicesDesc, prom.GaugeValue, serviceMetric.value, serviceMetric.labels...)
	}
}

var _ prom.Collector = (*InventoryMetricsCollector)(nil)
