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
	"strings"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"
)

type dsEnvVars struct {
	l      *logrus.Entry
	config DSConfigEnvVars
}

// check interfaces.
var (
	_ DataSource = (*dsEnvVars)(nil)
)

// NewDataSourceEnvVars makes a new data source for collecting envvars.
func NewDataSourceEnvVars(config DSConfigEnvVars, l *logrus.Entry) DataSource {
	return &dsEnvVars{
		l:      l,
		config: config,
	}
}

// Enabled flag that determines if data source is enabled.
func (d *dsEnvVars) Enabled() bool {
	return d.config.Enabled
}

func (d *dsEnvVars) Init(_ context.Context) error {
	return nil
}

func (d *dsEnvVars) FetchMetrics(_ context.Context, config Config) ([]*pmmv1.ServerMetric_Metric, error) {
	var metrics []*pmmv1.ServerMetric_Metric
	var envVars []string

	for _, envVar := range strings.Split(config.Query, ",") {
		if v := strings.TrimSpace(envVar); v != "" {
			envVars = append(envVars, v)
		}
	}

	for _, envVar := range envVars {
		if value, ok := os.LookupEnv(envVar); ok && value != "" {
			metrics = append(metrics, &pmmv1.ServerMetric_Metric{
				Key:   envVar,
				Value: value,
			})
		}
	}

	return metrics, nil
}

func (d *dsEnvVars) Dispose(_ context.Context) error {
	return nil
}
