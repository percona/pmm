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

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// agentsRegistry is a subset of methods of agents.Registry used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsRegistry interface {
	IsConnected(pmmAgentID string) bool
	Kick(ctx context.Context, pmmAgentID string)
}

// agentService is a subset of methods of agents.AgentService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentService interface {
	Logs(ctx context.Context, pmmAgentID, agentID string, limit uint32) ([]string, uint32, error)
}

// agentsRegistry is a subset of methods of agents.StateUpdater used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsStateUpdater interface {
	RequestStateUpdate(ctx context.Context, pmmAgentID string)
}

// prometheusService is a subset of methods of victoriametrics.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
//
// FIXME Rename to victoriaMetrics.Service, update tests.
type prometheusService interface {
	RequestConfigurationUpdate()
}

// connectionChecker is a subset of methods of agents.ConnectionCheck.
// We use it instead of real type for testing and to avoid dependency cycle.
type connectionChecker interface {
	CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error
}

// serviceInfoBroker is a subset of methods of serviceinfobroker.ServiceInfoBroker used by this package.
type serviceInfoBroker interface {
	GetInfoFromService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error
}

// versionCache is a subset of methods of versioncache.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type versionCache interface {
	RequestSoftwareVersionsUpdate()
}

type inventoryMetrics interface {
	GetAgentMetrics(ctx context.Context) (metrics []Metric, err error)
	GetNodeMetrics(ctx context.Context) (metrics []Metric, err error)
	GetServiceMetrics(ctx context.Context) (metrics []Metric, err error)
}
