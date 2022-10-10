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
	"github.com/pkg/errors"
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"
)

type dsGrafanaSelect struct {
	log    *logrus.Entry
	config DSGrafanaSqliteDB
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
func NewDataSourceGrafanaSqliteDB(config DSGrafanaSqliteDB, l *logrus.Entry) DataSource {
	return &dsGrafanaSelect{
		log:    l,
		config: config,
	}
}

func (d *dsGrafanaSelect) FetchMetrics(ctx context.Context, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	// check if datasource is enabled
	if !d.Enabled() {
		d.log.Info("Telemetry for grafana database is disabled.")
		return nil, nil
	}

	// validate source file db
	sourceFileStat, err := os.Stat(d.config.DBFile)
	if err != nil {
		return nil, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return nil, errors.Wrapf(err, "%s is not a regular file", d.config.DBFile)
	}

	source, err := os.Open(d.config.DBFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := source.Close(); err != nil {
			d.log.Error("Error closing file: %s\n", err)
		}
	}()

	tempFile, err := os.CreateTemp(os.TempDir(), "grafana")
	if err != nil {
		d.log.Fatal(err)
		return nil, err
	}
	defer os.Remove(tempFile.Name()) //nolint:errcheck

	nBytes, err := io.Copy(tempFile, source)
	if err != nil || nBytes == 0 {
		d.log.Error(err)
		return nil, errors.Wrapf(err, "cannot create copy of database file %s", d.config.DBFile)
	}

	db, err := sql.Open("sqlite3", tempFile.Name())
	if err != nil {
		d.log.Error(err)
		return nil, err
	}

	result, err := fetchMetricsFromDB(ctx, d.log, d.config.Timeout, db, config)
	if err != nil {
		d.log.Error(err)
		return nil, err
	}

	return result, nil
}
