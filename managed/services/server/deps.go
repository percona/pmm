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
	"net/url"
	"time"

	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/managed/models"
)

// healthChecker interface wraps all services that implements the IsReady method to report the
// service health for the Readiness check.
type healthChecker interface { //nolint:iface
	IsReady(ctx context.Context) error
}

// grafanaClient is a subset of methods of grafana.Client used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type grafanaClient interface { //nolint:iface
	healthChecker
}

// prometheusService is a subset of methods of victoriametrics.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
//
// FIXME Rename to victoriaMetrics.Service, update tests.
type prometheusService interface { //nolint:iface
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
type vmAlertService interface { //nolint:iface
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
	UpdateConfiguration(settings *models.Settings, ssoDetails *models.PerconaSSODetails) error
}

// telemetryService is a subset of methods of telemetry.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type telemetryService interface {
	DistributionMethod() serverv1.DistributionMethod
	GetSummaries() []string
}

// agentsStateUpdater is subset of methods of agents.StateUpdater used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsStateUpdater interface {
	UpdateAgentsState(ctx context.Context) error
}

// templatesService is a subset of methods of alerting.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type templatesService interface {
	CollectTemplates(ctx context.Context)
}

// haService is a subset of methods of ha.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type haService interface {
	IsLeader() bool
}

// victoriaMetricsParams is a subset of methods of models.VMParams used by this package.
// We use it instead of real type to avoid dependency cycle.
type victoriaMetricsParams interface {
	ExternalVM() bool
	URLFor(path string) (*url.URL, error)
}

// nomadService represents an interface for managing and updating Nomad-related configurations in a given context.
type nomadService interface {
	UpdateConfiguration(settings *models.Settings) error
}
