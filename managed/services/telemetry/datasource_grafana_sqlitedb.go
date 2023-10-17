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
	"io"
	"os"

	// Events, errors and driver for grafana sqlite database.
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

type dsGrafanaSelect struct {
	l        *logrus.Entry
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
		l:        l,
		config:   config,
		db:       nil,
		tempFile: "",
	}
}

func (d *dsGrafanaSelect) Init(ctx context.Context) error {
	// validate source file db
	sourceFileStat, err := os.Stat(d.config.DBFile)
	if err != nil {
		return err
	}

	if sourceFileStat.Size() == 0 {
		return errors.Errorf("Sourcefile %s is empty.", d.config.DBFile)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return errors.Wrapf(err, "%s is not a regular file", d.config.DBFile)
	}

	source, err := os.Open(d.config.DBFile)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "grafana")
	if err != nil {
		return err
	}

	defer func() {
		if err := source.Close(); err != nil {
			d.l.Errorf("Error closing file. %s", err)
		}
	}()

	nBytes, err := io.Copy(tempFile, source)
	d.l.Debugf("grafana sqlitedb copied with total bytes: %d", nBytes)
	if err != nil || nBytes == 0 {
		return errors.Wrapf(err, "cannot create copy of database file %s", d.config.DBFile)
	}

	db, err := sql.Open("sqlite", tempFile.Name())
	if err != nil {
		return err
	}

	d.tempFile = tempFile.Name()
	d.db = db

	return nil
}

func (d *dsGrafanaSelect) FetchMetrics(ctx context.Context, config Config) ([]*pmmv1.ServerMetric_Metric, error) {
	if d.db == nil {
		return nil, errors.Errorf("temporary grafana database is not initialized: %s", d.config.DBFile)
	}
	return fetchMetricsFromDB(ctx, d.l, d.config.Timeout, d.db, config)
}

func (d *dsGrafanaSelect) Dispose(ctx context.Context) error {
	err := d.db.Close()
	if err != nil {
		return err
	}

	err = os.Remove(d.tempFile)
	if err != nil {
		return errors.Wrapf(err, "failed to remove sqlite database file")
	}

	return nil
}
