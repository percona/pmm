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

package server

import (
	"context"
	"time"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

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

// prometheusService is a subset of methods of victoriametrics.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
//
// FIXME Rename to victoriaMetrics.Service, update tests.
type prometheusService interface {
	RequestConfigurationUpdate()
	healthChecker
}

// alertmanagerService is a subset of methods of alertmanager.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type alertmanagerService interface {
	RequestConfigurationUpdate()
	healthChecker
}

// checksService is a subset of methods of checks.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type checksService interface {
	StartChecks(checkNames []string) error
	CollectAdvisors(ctx context.Context)
	CleanupAlerts()
	UpdateIntervals(rare, standard, frequent time.Duration)
}

// vmAlertService is a subset of methods of vmalert.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
type vmAlertService interface {
	RequestConfigurationUpdate()
	healthChecker
}

// vmAlertExternalRules is a subset of methods of vmalert.ExternalRules used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type vmAlertExternalRules interface {
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

	UpdateConfiguration(settings *models.Settings, ssoDetails *models.PerconaSSODetails) error
}

// telemetryService is a subset of methods of telemetry.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type telemetryService interface {
	DistributionMethod() serverpb.DistributionMethod
	GetSummaries() []string
}

// agentsStateUpdater is subset of methods of agents.StateUpdater used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsStateUpdater interface {
	UpdateAgentsState(ctx context.Context) error
}

// rulesService is a subset of methods of ia.RulesService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type rulesService interface {
	WriteVMAlertRulesFiles()
	RemoveVMAlertRulesFiles() error
}

type emailer interface {
	Send(ctx context.Context, settings *models.EmailAlertingSettings, emailTo string) error
}

// templatesService is a subset of methods of ia.TemplatesService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type templatesService interface {
	CollectTemplates(ctx context.Context)
}

// haService is a subset of methods of ha.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type haService interface {
	IsLeader() bool
}
