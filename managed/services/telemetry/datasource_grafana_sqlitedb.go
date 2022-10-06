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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	"context"
	"database/sql"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"
)

type dsGrafanaSelect struct {
	l      *logrus.Entry
	config DSGrafanaSqliteDB
	db     *sql.DB
}

// check interfaces
var (
	_ DataSource = (*dsGrafanaSelect)(nil)
)

// Enabled flag that determines if data source is enabled.
func (d *dsGrafanaSelect) Enabled() bool {
	return d.config.Enabled
}

// NewDataSourceGrafanaSqliteDB makes new data source for grafana sqlite database metrics.
func NewDataSourceGrafanaSqliteDB(config DSGrafanaSqliteDB, l *logrus.Entry) (DataSource, error) { //nolint:ireturn
	if !config.Enabled {
		return &dsGrafanaSelect{
			l:      l,
			config: config,
			db:     nil,
		}, nil
	}

	return &dsGrafanaSelect{
		l:      l,
		config: config,
		db:     nil, // TODO: sqlite3 initialization client
	}, nil
}

func (d *dsGrafanaSelect) FetchMetrics(ctx context.Context, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	return fetchMetricsFromDB(ctx, d.l, d.config.Timeout, d.db, config)
}
