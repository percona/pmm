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
	"os"

	telemetryv1 "github.com/percona/saas/gen/telemetry/generic"
	"github.com/sirupsen/logrus"
)

type dsEnvvars struct {
	l      *logrus.Entry
	config DSConfigEnvVars
}

// check interfaces.
var (
	_ DataSource = (*dsEnvvars)(nil)
)

// NewDataSourceEnvVars makes a new data source for collecting envvars.
func NewDataSourceEnvVars(config DSConfigEnvVars, l *logrus.Entry) DataSource {
	return &dsEnvvars{
		l:      l,
		config: config,
	}
}

// Enabled flag that determines if data source is enabled.
func (d *dsEnvvars) Enabled() bool {
	return d.config.Enabled
}

func (d *dsEnvvars) Init(_ context.Context) error {
	return nil
}

func (d *dsEnvvars) FetchMetrics(_ context.Context, config Config) ([]*telemetryv1.GenericReport_Metric, error) {
	var metrics []*telemetryv1.GenericReport_Metric

	check := make(map[string]bool, len(config.Data))

	for _, col := range config.Data {
		if col.Column == "" {
			d.l.Warnf("no column defined or empty column name in config %s", config.ID)
			continue
		}
		if value, ok := os.LookupEnv(col.Column); ok && value != "" {
			if _, alreadyHasItem := check[col.MetricName]; alreadyHasItem {
				d.l.Warnf("repeated metric key %s found in config %s, the last will win", col.MetricName, config.ID)
				continue
			}

			check[col.MetricName] = true

			metrics = append(metrics, &telemetryv1.GenericReport_Metric{
				Key:   col.MetricName,
				Value: value,
			})
		}
	}

	return metrics, nil
}

func (d *dsEnvvars) Dispose(_ context.Context) error {
	return nil
}
