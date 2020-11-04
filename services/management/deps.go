// pmm-managed
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

package management

import (
	"context"
	"time"

	"github.com/percona-platform/saas/pkg/check"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

//go:generate mockery -name=agentsRegistry -case=snake -inpkg -testonly
//go:generate mockery -name=prometheusService -case=snake -inpkg -testonly
//go:generate mockery -name=checksService -case=snake -inpkg -testonly
//go:generate mockery -name=grafanaClient -case=snake -inpkg -testonly

// agentsRegistry is a subset of methods of agents.Registry used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsRegistry interface {
	IsConnected(pmmAgentID string) bool
	Kick(ctx context.Context, pmmAgentID string)
	SendSetStateRequest(ctx context.Context, pmmAgentID string)
	CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) (err error)
}

// prometheusService is a subset of methods of victoriametrics.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
//
// FIXME Rename to victoriaMetrics.Service, update tests.
type prometheusService interface {
	RequestConfigurationUpdate()
}

// checksService is a subset of methods of checks.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type checksService interface {
	StartChecks(ctx context.Context) error
	GetSecurityCheckResults() ([]check.Result, error)
	GetAllChecks() []check.Check
	GetDisabledChecks() ([]string, error)
	DisableChecks(checkNames []string) error
	EnableChecks(checkNames []string) error
}

// grafanaClient is a subset of methods of grafana.Client used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type grafanaClient interface {
	CreateAnnotation(context.Context, []string, time.Time, string, string) (string, error)
}
