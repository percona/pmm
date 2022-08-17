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

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

//go:generate $PMM_RELEASE_PATH/mockery -name=agentsRegistry -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=agentsStateUpdater -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=prometheusService -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=checksService -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=grafanaClient -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=jobsService -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=connectionChecker -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=defaultsFileParser -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=versionCache -case=snake -inpkg -testonly

// agentsRegistry is a subset of methods of agents.Registry used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsRegistry interface {
	IsConnected(pmmAgentID string) bool
	Kick(ctx context.Context, pmmAgentID string)
}

// agentsStateUpdater is subset of methods of agents.StateUpdater used by this package.
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

// checksService is a subset of methods of checks.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type checksService interface {
	StartChecks(checkNames []string) error
	GetSecurityCheckResults() ([]services.CheckResult, error)
	GetChecks() (map[string]check.Check, error)
	GetChecksResults(ctx context.Context, serviceID string) ([]services.CheckResult, error)
	GetDisabledChecks() ([]string, error)
	DisableChecks(checkNames []string) error
	EnableChecks(checkNames []string) error
	ChangeInterval(params map[string]check.Interval) error
	ToggleCheckAlert(ctx context.Context, alertID string, newStatus bool) error
}

// grafanaClient is a subset of methods of grafana.Client used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type grafanaClient interface {
	CreateAnnotation(context.Context, []string, time.Time, string, string) (string, error)
}

// jobsService is a subset of methods of agents.JobsService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type jobsService interface {
	StopJob(jobID string) error
}

// connectionChecker is a subset of methods of agents.ConnectionCheck.
// We use it instead of real type for testing and to avoid dependency cycle.
type connectionChecker interface {
	CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error
}

// versionCache is a subset of methods of versioncache.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type versionCache interface {
	RequestSoftwareVersionsUpdate()
}

// defaultsFileParser is a subset of methods of agents.ParseDefaultsFile.
// We use it instead of real type for testing and to avoid dependency cycle.
type defaultsFileParser interface {
	ParseDefaultsFile(ctx context.Context, pmmAgentID, filePath string, serviceType models.ServiceType) (*models.ParseDefaultsFileResult, error)
}
