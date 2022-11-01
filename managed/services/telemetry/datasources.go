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
	"encoding/json"
	"time"

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

// NewDataSourceRegistry makes new data source registry
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

	return &dataSourceRegistry{
		l: l,
		dataSources: map[DataSourceName]DataSource{
			"VM":           vmDB,
			"PMMDB_SELECT": pmmDB,
			"QANDB_SELECT": qanDB,
		},
	}, nil
}

// LocateTelemetryDataSource returns data source by name.
func (r *dataSourceRegistry) LocateTelemetryDataSource(name string) (DataSource, error) { //nolint:ireturn
	ds, ok := r.dataSources[DataSourceName(name)]
	if !ok {
		return nil, errors.Errorf("data source [%s] is not supported", name)
	}
	return ds, nil
}

func fetchMetricsFromDB(ctx context.Context, l *logrus.Entry, timeout time.Duration, db *sql.DB, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	localCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tx, err := db.BeginTx(localCtx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	// to minimize risk of modifying DB
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.Query("SELECT " + config.Query) //nolint:gosec,rowserrcheck,sqlclosecheck
	if err != nil {
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	strs := make([]*string, len(columns))
	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = &strs[i]
	}

	if config.DataJSON != nil {
		metric, err := parseRowsToJSOMMetrics(rows, config, l)
		if err != nil {
			l.Error("error: ", err)
			return nil, err
		}
		return [][]*pmmv1.ServerMetric_Metric{metric}, nil
	}

	cfgColumns := config.mapByColumn()
	return parseRowsToSimpleMetrics(rows, values, l, columns, strs, cfgColumns), nil
}

func parseRowsToSimpleMetrics(rows *sql.Rows, values []interface{}, l *logrus.Entry, columns []string, strs []*string, cfgColumns map[string][]ConfigData) [][]*pmmv1.ServerMetric_Metric {
	var metrics [][]*pmmv1.ServerMetric_Metric

	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			l.Error(err)
			continue
		}

		var metric []*pmmv1.ServerMetric_Metric
		for idx, column := range columns {
			var value string

			// skip empty values
			if strs[idx] == nil || *strs[idx] == "" {
				continue
			}

			value = *strs[idx]
			if cols, ok := cfgColumns[column]; ok {
				for _, col := range cols {
					metric = append(metric, &pmmv1.ServerMetric_Metric{
						Key:   col.MetricName,
						Value: value,
					})
				}
			}
		}
		if len(metric) != 0 {
			metrics = append(metrics, metric)
		}

	}
	return metrics
}

func parseRowsToJSOMMetrics(rows *sql.Rows, config Config, l *logrus.Entry) ([]*pmmv1.ServerMetric_Metric, error) {
	var metric []*pmmv1.ServerMetric_Metric
	jsonColumnToKeyMap := make(map[string]string)
	for _, param := range config.DataJSON.Params {
		jsonColumnToKeyMap[param.Column] = param.Key
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	strs := make([]*string, len(columns))
	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = &strs[i]
	}

	type payloadType []map[string]any
	resultObj := make(map[string]payloadType)

	for rows.Next() {
		res := make(map[string]any)
		if err := rows.Scan(values...); err != nil {
			l.Error(err)
			continue
		}

		for idx, col := range columns {
			key, ok := jsonColumnToKeyMap[col]
			if !ok {
				continue
			}
			res[key] = *strs[idx]
		}

		resultObj[config.DataJSON.MetricName] = append(resultObj[config.DataJSON.MetricName], res)
	}
	marshal, err := json.Marshal(resultObj)
	if err != nil {
		return nil, err
	}

	metric = append(metric, &pmmv1.ServerMetric_Metric{
		Key:   config.DataJSON.MetricName,
		Value: string(marshal),
	})
	return metric, nil
}
