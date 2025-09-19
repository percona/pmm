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

package realtime

import (
	"context"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// TODO: Add mockery generation when needed
// go:generate ../../bin/mockery --name=agentsRegistry --case=snake --inpackage --testonly
// go:generate ../../bin/mockery --name=connectionChecker --case=snake --inpackage --testonly

// agentsRegistry is a subset of agents.Registry used by this package.
type agentsRegistry interface {
	IsConnected(agentID string) bool
}

// connectionChecker is a subset of agents.ConnectionChecker used by this package.
type connectionChecker interface {
	CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error
}

// stateUpdater is a subset of agents.StateUpdater used by this package.
type stateUpdater interface {
	RequestStateUpdate(ctx context.Context, pmmAgentID string)
}
