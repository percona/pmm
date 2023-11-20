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
	"time"

	"github.com/AlekSi/pointer"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// DataSourceName data source name.
type DataSourceName string

type dataSourceRegistry struct {
	l           *logrus.Entry
	dataSources map[DataSourceName]DataSource
}

// NewDataSourceRegistry makes new data source registry.
func NewDataSourceRegistry(config ServiceConfig, l *logrus.Entry) (DataSourceLocator, error) {
	pmmDB, err := NewDsPmmDBSelect(*config.DataSources.PmmDBSelect, l)
	if err != nil {
		return nil, err
	}

	qanDB, err := NewDsQanDBSelect(*config.DataSources.QanDBSelect, l)
	if err != nil {
		return nil, err
	}

	vmDB, err := NewDataSourceVictoriaMetrics(*config.DataSources.VM, l)
	if err != nil {
		return nil, err
	}

	grafanaDB := NewDsGrafanaDBSelect(*config.DataSources.GrafanaDBSelect, l)

	envVars := NewDataSourceEnvVars(*config.DataSources.EnvVars, l)

	return &dataSourceRegistry{
		l: l,
		dataSources: map[DataSourceName]DataSource{
			dsVM:              vmDB,
			dsPMMDBSelect:     pmmDB,
			dsQANDBSelect:     qanDB,
			dsGRAFANADBSelect: grafanaDB,
			dsEnvVars:         envVars,
		},
	}, nil
}

// LocateTelemetryDataSource returns data source by name.
func (r *dataSourceRegistry) LocateTelemetryDataSource(name string) (DataSource, error) {
	ds, ok := r.dataSources[DataSourceName(name)]
	if !ok {
		return nil, errors.Errorf("data source [%s] is not supported", name)
	}
	return ds, nil
}

func fetchMetricsFromDB(ctx context.Context, l *logrus.Entry, timeout time.Duration, db *sql.DB, config Config) ([]*pmmv1.ServerMetric_Metric, error) {
	localCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tx, err := db.BeginTx(localCtx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	// to minimize risk of modifying DB
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.Query("SELECT " + config.Query) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	strs := make([]*string, len(columns))
	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = &strs[i]
	}
	cfgColumns := config.mapByColumn()

	var metrics []*pmmv1.ServerMetric_Metric
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			l.Error(err)
			continue
		}

		for idx, column := range columns {
			value := pointer.GetString(strs[idx])

			if cols, ok := cfgColumns[column]; ok {
				for _, col := range cols {
					metrics = append(metrics, &pmmv1.ServerMetric_Metric{
						Key:   col.MetricName,
						Value: value,
					})
				}
			}
		}
	}

	return metrics, nil
}
