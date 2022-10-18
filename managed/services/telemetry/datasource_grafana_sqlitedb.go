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
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3" //nolint:golint
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type dsGrafanaSelect struct {
	log      *logrus.Entry
	config   DSGrafanaSqliteDB
	db       *sql.DB
	tempFile string
}

// check interfaces.
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
		log:      l,
		config:   config,
		db:       nil,
		tempFile: "",
	}
}

func (d *dsGrafanaSelect) PreFetch(ctx context.Context, config Config) error {
	// validate source file db
	sourceFileStat, err := os.Stat(d.config.DBFile)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return errors.Wrapf(err, "%s is not a regular file", d.config.DBFile)
	}

	source, err := os.Open(d.config.DBFile)
	if err != nil {
		d.log.Error(err)
		return err
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "grafana")
	if err != nil {
		d.log.Error(err)
		return err
	}
	defer func() {
		if err := source.Close(); err != nil {
			d.log.Errorf("Error closing file. %s", err)
		}
	}()

	nBytes, err := io.Copy(tempFile, source)
	if err != nil || nBytes == 0 {
		d.log.Error(err)
		return errors.Wrapf(err, "cannot create copy of database file %s", d.config.DBFile)
	}

	db, err := sql.Open("sqlite3", tempFile.Name())
	if err != nil {
		d.log.Error(err)
		return err
	}

	d.tempFile = tempFile.Name()
	d.db = db

	return nil
}

func (d *dsGrafanaSelect) FetchMetrics(ctx context.Context, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	if d.db == nil {
		return nil, errors.Errorf("temporary grafana database is not initialized: %s", d.config.DBFile)
	}
	return fetchMetricsFromDB(ctx, d.log, d.config.Timeout, d.db, config)
}

func (d *dsGrafanaSelect) PostFetch(ctx context.Context, config Config) error {
	err := d.db.Close()
	if err != nil {
		return err
	}

	err = os.Remove(d.tempFile)
	if err != nil {
		d.log.Errorf("Error removing file. %s", err)
	}

	return nil
}
