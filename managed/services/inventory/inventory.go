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
	"sync"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	cancelTime                  = 3 * time.Second
	serviceEnabled      float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "inventory"
)

type Metric struct {
	labels []string
	value  float64
}

type Inventory struct {
	mAgentsDesc   *prom.Desc
	mNodesDesc    *prom.Desc
	mServicesDesc *prom.Desc

	api   inventoryAPI
	mutex sync.Mutex
}

func NewInventory(api inventoryAPI) *Inventory {
	i := &Inventory{
		mAgentsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "agents"),
			"The current information about agent",
			[]string{"agent_id", "agent_type", "service_id", "node_id", "pmm_agent_id", "disabled", "version"},
			nil),
		mNodesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "nodes"),
			"The current information about node",
			[]string{"node_id", "node_type", "node_name", "container_name"},
			nil),
		mServicesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
			"The current information about service",
			[]string{"service_id", "service_type", "node_id"},
			nil),
		api: api,
	}

	return i
}

func (i *Inventory) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(i, ch)
}

func (i *Inventory) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), cancelTime)
	defer cancelCtx()

	i.mutex.Lock()
	defer i.mutex.Unlock()

	agentMetrics, agentError := i.api.GetAgentDataForMetrics(ctx)

	if agentError != nil {
		fmt.Println(agentError)
		return
	}

	for _, agentMetric := range agentMetrics {
		ch <- prom.MustNewConstMetric(i.mAgentsDesc, prom.GaugeValue, agentMetric.value, agentMetric.labels...)
	}

	nodeMetrics, nodeError := i.api.GetNodeDataForMetrics(ctx)

	if nodeError != nil {
		fmt.Println(nodeError)
		return
	}

	for _, nodeMetric := range nodeMetrics {
		ch <- prom.MustNewConstMetric(i.mNodesDesc, prom.GaugeValue, nodeMetric.value, nodeMetric.labels...)
	}

	serviceMetrics, serviceError := i.api.GetServiceDataForMetrics(ctx)

	if serviceError != nil {
		fmt.Println(serviceError)
		return
	}

	for _, serviceMetric := range serviceMetrics {
		ch <- prom.MustNewConstMetric(i.mServicesDesc, prom.GaugeValue, serviceMetric.value, serviceMetric.labels...)
	}
}

var _ prom.Collector = (*Inventory)(nil)
