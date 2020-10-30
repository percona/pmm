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

package server

import (
	"context"
	"time"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"

	"github.com/percona/pmm-managed/models"
)

//go:generate mockery -name=grafanaClient -case=snake -inpkg -testonly
//go:generate mockery -name=prometheusService -case=snake -inpkg -testonly
//go:generate mockery -name=alertmanagerService -case=snake -inpkg -testonly
//go:generate mockery -name=prometheusAlertingRules -case=snake -inpkg -testonly
//go:generate mockery -name=supervisordService -case=snake -inpkg -testonly
//go:generate mockery -name=telemetryService -case=snake -inpkg -testonly
//go:generate mockery -name=platformService -case=snake -inpkg -testonly

// healthChecker interface wraps all services that implements the IsReady method to report the
// service health for the Readiness check.
type healthChecker interface {
	IsReady(ctx context.Context) error
}

// grafanaClient is a subset of methods of grafana.Client used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type grafanaClient interface {
	healthChecker
}

// prometheusService is a subset of methods of prometheus.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type prometheusService interface {
	RequestConfigurationUpdate()
	healthChecker
}

// alertmanagerService is a subset of methods of alertmanager.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type alertmanagerService interface {
	healthChecker
}

// prometheusAlertingRules is a subset of methods of prometheus.AlertingRules used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type prometheusAlertingRules interface {
	ValidateRules(ctx context.Context, rules string) error
	ReadRules() (string, error)
	RemoveRulesFile() error
	WriteRules(rules string) error
}

// supervisordService is a subset of methods of supervisord.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type supervisordService interface {
	InstalledPMMVersion(ctx context.Context) *version.PackageInfo
	LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time)
	ForceCheckUpdates(ctx context.Context) error

	StartUpdate() (uint32, error)
	UpdateRunning() bool
	UpdateLog(offset uint32) ([]string, uint32, error)

	UpdateConfiguration(settings *models.Settings) error
}

// telemetryService is a subset of methods of telemetry.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type telemetryService interface {
	DistributionMethod() serverpb.DistributionMethod
}

// platformService is a subset of methods of platform.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type platformService interface {
	SignUp(ctx context.Context, email, password string) error
	SignIn(ctx context.Context, email, password string) error
	SignOut(ctx context.Context) error
}
