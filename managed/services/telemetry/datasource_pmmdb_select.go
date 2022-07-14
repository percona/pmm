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
	"fmt"
	"net/url"
	"time"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type dsPmmDBSelect struct {
	l      *logrus.Entry
	config DSConfigPMMDB
	db     *sql.DB
}

// check interfaces
var (
	_ DataSource = (*dsPmmDBSelect)(nil)
)

// Enabled flag that determines if data source is enabled.
func (d *dsPmmDBSelect) Enabled() bool {
	return d.config.Enabled
}

// NewDsPmmDBSelect make new PMM DB Select data source.
func NewDsPmmDBSelect(config DSConfigPMMDB, l *logrus.Entry) (DataSource, error) { //nolint:ireturn
	db, err := openPMMDBConnection(config)
	if err != nil {
		return nil, err
	}

	return &dsPmmDBSelect{
		l:      l,
		config: config,
		db:     db,
	}, nil
}

func openPMMDBConnection(config DSConfigPMMDB) (*sql.DB, error) {
	if !config.Enabled {
		return nil, nil //nolint:nilnil
	}

	var user *url.Userinfo
	if config.UseSeparateCredentials {
		user = url.UserPassword(config.SeparateCredentials.Username, config.SeparateCredentials.Password)
	} else {
		user = url.UserPassword(config.Credentials.Username, config.Credentials.Password)
	}
	uri := url.URL{
		Scheme:   config.DSN.Scheme,
		User:     user,
		Host:     config.DSN.Host,
		Path:     config.DSN.DB,
		RawQuery: config.DSN.Params,
	}
	dsn := uri.String()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a connection pool to PostgreSQL")
	}

	db.SetConnMaxIdleTime(time.Second * 30)
	db.SetConnMaxLifetime(time.Second * 180)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connection to PMM DB failed: %s", err)
	}

	return db, nil
}

func (d *dsPmmDBSelect) FetchMetrics(ctx context.Context, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	return fetchMetricsFromDB(ctx, d.l, d.config.Timeout, d.db, config)
}
