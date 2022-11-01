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
	"encoding/json"
	"time"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type dataSourceVictoriaMetrics struct {
	l      *logrus.Entry
	config DataSourceVictoriaMetrics
	vm     v1.API
}

// check interfaces
var (
	_ DataSource = (*dataSourceVictoriaMetrics)(nil)
)

func (d *dataSourceVictoriaMetrics) Enabled() bool {
	return d.config.Enabled
}

// NewDataSourceVictoriaMetrics makes new data source for victoria metrics.
func NewDataSourceVictoriaMetrics(config DataSourceVictoriaMetrics, l *logrus.Entry) (DataSource, error) { //nolint:ireturn
	if !config.Enabled {
		return &dataSourceVictoriaMetrics{
			l:      l,
			config: config,
			vm:     nil,
		}, nil
	}

	client, err := api.NewClient(api.Config{
		Address: config.Address,
	})
	if err != nil {
		return nil, err
	}

	return &dataSourceVictoriaMetrics{
		l:      l,
		config: config,
		vm:     v1.NewAPI(client),
	}, nil
}

func (d *dataSourceVictoriaMetrics) FetchMetrics(ctx context.Context, config Config) ([][]*pmmv1.ServerMetric_Metric, error) {
	localCtx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	r, _, err := d.vm.Query(localCtx, config.Query, time.Now())
	if err != nil {
		return nil, err
	}

	resulVector := r.(model.Vector)

	var metrics []*pmmv1.ServerMetric_Metric
	if config.DataJSON != nil {
		metrics, err = parseVMResultQueryToJSONString(config, resulVector)

		if err != nil {
			return nil, err
		}
	} else {
		metrics = parseVMQueryResultToKeyValueMetrics(config, resulVector)
	}

	return [][]*pmmv1.ServerMetric_Metric{metrics}, nil
}

func parseVMQueryResultToKeyValueMetrics(config Config, result model.Vector) []*pmmv1.ServerMetric_Metric {
	var metrics []*pmmv1.ServerMetric_Metric
	for _, v := range result {
		for _, configItem := range config.Data {
			if configItem.Label != "" {
				value, ok := v.Metric[model.LabelName(configItem.Label)]
				if ok {
					metrics = append(metrics, &pmmv1.ServerMetric_Metric{
						Key:   configItem.MetricName,
						Value: string(value),
					})
				}
			}

			if configItem.Value != "" {
				metrics = append(metrics, &pmmv1.ServerMetric_Metric{
					Key:   configItem.MetricName,
					Value: v.Value.String(),
				})
			}
		}
	}
	return metrics
}

func parseVMResultQueryToJSONString(config Config, result interface{}) ([]*pmmv1.ServerMetric_Metric, error) {
	type payloadType []map[string]any
	resultObj := make(map[string]payloadType)

	metricName := config.DataJSON.MetricName

	var metrics []*pmmv1.ServerMetric_Metric
	for _, v := range result.(model.Vector) {
		res := make(map[string]any)

		for _, param := range config.DataJSON.Params {
			if param.Label != "" {
				labelValue, ok := v.Metric[model.LabelName(param.Label)]
				if ok {
					res[param.Key] = labelValue
				}
			} else if param.Value != "" {
				res[param.Key] = v.Value.String()
			}
		}
		resultObj[metricName] = append(resultObj[metricName], res)
	}

	jsonResponse, err := json.Marshal(resultObj)
	if err != nil {
		return nil, err
	}

	metrics = append(metrics, &pmmv1.ServerMetric_Metric{
		Key:   config.DataJSON.MetricName,
		Value: string(jsonResponse),
	})

	return metrics, nil
}
