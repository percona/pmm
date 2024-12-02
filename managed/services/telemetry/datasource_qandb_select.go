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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	"context"
	"database/sql"

	telemetryv1 "github.com/percona/saas/gen/telemetry/generic"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type dsQanDBSelect struct {
	l      *logrus.Entry
	config DSConfigQAN
	db     *sql.DB
}

// check interfaces.
var (
	_ DataSource = (*dsQanDBSelect)(nil)
)

// Enabled flag that determines if data source is enabled.
func (d *dsQanDBSelect) Enabled() bool {
	return d.config.Enabled
}

// NewDsQanDBSelect make new QAN DB Select data source.
func NewDsQanDBSelect(config DSConfigQAN, l *logrus.Entry) (DataSource, error) {
	db, err := openQANDBConnection(config.DSN, config.Enabled, l)
	if err != nil {
		return nil, err
	}
	return &dsQanDBSelect{
		l:      l,
		config: config,
		db:     db,
	}, nil
}

func openQANDBConnection(dsn string, enabled bool, l *logrus.Entry) (*sql.DB, error) {
	if !enabled {
		return nil, nil //nolint:nilnil
	}

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open connection to QAN DB")
	}
	if err := db.Ping(); err != nil {
		l.Warnf("ClickHouse DB is not reachable [%s]: %s", dsn, err)
	}
	return db, nil
}

func (d *dsQanDBSelect) FetchMetrics(ctx context.Context, config Config) ([]*telemetryv1.GenericReport_Metric, error) {
	return fetchMetricsFromDB(ctx, d.l, d.config.Timeout, d.db, config)
}

func (d *dsQanDBSelect) Init(ctx context.Context) error { //nolint:revive
	return nil
}

func (d *dsQanDBSelect) Dispose(ctx context.Context) error { //nolint:revive
	return nil
}
