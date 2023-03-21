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
	"strconv"

	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type API struct {
	db *reform.DB
	r  agentsRegistry
}

func NewInventoryAPI(db *reform.DB, r agentsRegistry) API {
	i := API{
		db: db,
		r:  r,
	}
	return i
}

func (i API) GetAgentDataForMetrics(ctx context.Context) (metrics []Metric, err error) {
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

			if i.r.IsConnected(pmmAgentID) {
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

func (i API) GetNodeDataForMetrics(ctx context.Context) (metrics []Metric, err error) {
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

func (i API) GetServiceDataForMetrics(ctx context.Context) (metrics []Metric, err error) {
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
