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
	"io"
	"io/ioutil"
	"os"

	_ "github.com/mattn/go-sqlite3" //nolint:golint
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
		db:     nil,
	}, nil
}

func (d *dsGrafanaSelect) FetchMetrics(ctx context.Context, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	// validate source file db
	sourceFileStat, err := os.Stat(d.config.DbFile)
	if err != nil {
		return nil, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", d.config.DbFile)
	}

	source, err := os.Open(d.config.DbFile)
	if err != nil {
		return nil, err
	}
	defer source.Close() //nolint:errcheck

	tempFile, err := ioutil.TempFile(os.TempDir(), "grafana")
	if err != nil {
		d.l.Fatal(err)
		return nil, err
	}
	defer os.Remove(tempFile.Name()) //nolint:errcheck

	nBytes, err := io.Copy(tempFile, source)
	if err != nil || nBytes == 0 {
		d.l.Error(err)
		return nil, fmt.Errorf("cannot copy file %s", d.config.DbFile)
	}

	db, err := sql.Open("sqlite3", tempFile.Name())
	if err != nil {
		d.l.Error(err)
		return nil, err
	}

	result, err := fetchMetricsFromDB(ctx, d.l, d.config.Timeout, db, config)
	if err != nil {
		d.l.Error(err)
		return nil, err
	}

	return result, nil
}
