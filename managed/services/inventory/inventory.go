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
	"time"

	"github.com/AlekSi/pointer"
	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	cancelTime                  = 3 * time.Second
	serviceEnabled      float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "inventory"
)

type Inventory struct {
	mAgentsDesc   *prom.Desc
	mNodesDesc    *prom.Desc
	mServicesDesc *prom.Desc

	db *reform.DB
	r  agentsRegistry
}

func NewInventory(db *reform.DB, r agentsRegistry) *Inventory {
	i := &Inventory{
		mAgentsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "agents"),
			"The current information about agent",
			[]string{"agent_type", "service_id", "node_id", "pmm_agent_id", "disabled", "version"},
			nil),
		mNodesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "nodes"),
			"The current information about node",
			[]string{"node_type", "node_name", "container_name"},
			nil),
		mServicesDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
			"The current information about service",
			[]string{"service_type", "node_id"},
			nil),

		db: db,
		r:  r,
	}
	return i
}

func (i *Inventory) Describe(ch chan<- *prom.Desc) {
	prom.DescribeByCollect(i, ch)
}

func (i *Inventory) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), cancelTime)
	defer cancelCtx()

	var resAgents []*models.Agent
	var resNodes []*models.Node
	var resServices []*models.Service
	err := i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbAgents, dbAgentsError := models.FindAgents(tx.Querier, models.AgentFilters{})
		dbNodes, dbNodesError := models.FindNodes(tx.Querier, models.NodeFilters{})
		dbServices, dbServicesError := models.FindServices(tx.Querier, models.ServiceFilters{})

		if dbAgentsError != nil {
			return dbAgentsError
		}

		if dbNodesError != nil {
			return dbNodesError
		}

		if dbServicesError != nil {
			return dbServicesError
		}

		resAgents = dbAgents
		resNodes = dbNodes
		resServices = dbServices

		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

	for _, agent := range resAgents {
		disabled := 0
		connected := float64(0)

		pmmAgentID := pointer.GetString(agent.PMMAgentID)

		if agent.Disabled {
			disabled = 1
		} else {
			disabled = 0
		}

		if i.r.IsConnected(pmmAgentID) {
			connected = 1
		} else {
			connected = 0
		}

		agentMetricLabels := []string{
			string(agent.AgentType),
			pointer.GetString(agent.ServiceID),
			pointer.GetString(agent.NodeID),
			pmmAgentID,
			strconv.Itoa(disabled),
			pointer.GetString(agent.Version),
		}
		ch <- prom.MustNewConstMetric(i.mAgentsDesc, prom.GaugeValue, connected, agentMetricLabels...)
	}

	for _, node := range resNodes {
		nodeMetricLabels := []string{
			string(node.NodeType),
			node.NodeName,
			pointer.GetString(node.ContainerName),
		}
		ch <- prom.MustNewConstMetric(i.mNodesDesc, prom.GaugeValue, serviceEnabled, nodeMetricLabels...)
	}

	for _, service := range resServices {
		serviceMetricLabels := []string{
			string(service.ServiceType),
			service.NodeID,
		}
		ch <- prom.MustNewConstMetric(i.mServicesDesc, prom.GaugeValue, serviceEnabled, serviceMetricLabels...)
	}
}

var _ prom.Collector = (*Inventory)(nil)
