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

package telemetry

import (
	"context"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	reporter "github.com/percona-platform/saas/gen/telemetry/reporter"

	"github.com/percona/pmm/api/serverpb"
)

// distributionUtilService service to get info about OS on which pmm server is running.
type distributionUtilService interface {
	getDistributionMethodAndOS() (serverpb.DistributionMethod, pmmv1.DistributionMethod, string)
}

// sender is interface which defines method for client which sends report with metrics.
type sender interface {
	SendTelemetry(ctx context.Context, report *reporter.ReportRequest) error
}

// DataSourceLocator locates data source by name.
type DataSourceLocator interface {
	LocateTelemetryDataSource(name string) (DataSource, error)
}

// DataSource telemetry data source.
type DataSource interface {
	Init(ctx context.Context) error
	FetchMetrics(ctx context.Context, config Config) ([]*pmmv1.ServerMetric_Metric, error)
	Dispose(ctx context.Context) error
	Enabled() bool
}
