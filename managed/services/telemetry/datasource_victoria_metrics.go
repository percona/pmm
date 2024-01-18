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
	"time"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type dataSourceVictoriaMetrics struct {
	l      *logrus.Entry
	config DSConfigVM
	vm     v1.API
}

// check interfaces.
var (
	_ DataSource = (*dataSourceVictoriaMetrics)(nil)
)

func (d *dataSourceVictoriaMetrics) Enabled() bool {
	return d.config.Enabled
}

// NewDataSourceVictoriaMetrics makes new data source for victoria metrics.
func NewDataSourceVictoriaMetrics(config DSConfigVM, l *logrus.Entry) (DataSource, error) { //nolint:ireturn,nolintlint
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

func (d *dataSourceVictoriaMetrics) FetchMetrics(ctx context.Context, config Config) ([]*pmmv1.ServerMetric_Metric, error) {
	localCtx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	result, _, err := d.vm.Query(localCtx, config.Query, time.Now())
	if err != nil {
		return nil, err
	}

	var metrics []*pmmv1.ServerMetric_Metric

	for _, v := range result.(model.Vector) { //nolint:forcetypeassert
		for _, configItem := range config.Data {
			if configItem.Label != "" {
				value := v.Metric[model.LabelName(configItem.Label)]
				metrics = append(metrics, &pmmv1.ServerMetric_Metric{
					Key:   configItem.MetricName,
					Value: string(value),
				})
			}

			if configItem.Value != "" {
				metrics = append(metrics, &pmmv1.ServerMetric_Metric{
					Key:   configItem.MetricName,
					Value: v.Value.String(),
				})
			}
		}
	}

	return metrics, nil
}

func (d *dataSourceVictoriaMetrics) Init(ctx context.Context) error { //nolint:revive
	return nil
}

func (d *dataSourceVictoriaMetrics) Dispose(ctx context.Context) error { //nolint:revive
	return nil
}
