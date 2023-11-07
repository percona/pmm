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

package telemetry

import (
	"testing"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/stretchr/testify/assert"
)

func TestTransformToJSON(t *testing.T) {
	type args struct {
		config  *Config
		metrics []*pmmv1.ServerMetric_Metric
	}

	noMetrics := []*pmmv1.ServerMetric_Metric{}

	tests := []struct {
		name    string
		args    args
		want    []*pmmv1.ServerMetric_Metric
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nil metrics",
			args: args{
				config:  configJSON(),
				metrics: nil,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "empty metrics",
			args: args{
				config:  configJSON(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Transform in config",
			args: args{
				config:  configJSON().noTransform(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Metrics config",
			args: args{
				config:  configJSON().noFirstMetricConfig(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Metric Name config",
			args: args{
				config:  configJSON().noFirstMetricNameConfig(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "invalid seq",
			args: args{
				config: configJSON(),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: "my-metric", Value: "v1"},
					{Key: "b", Value: "v1"},
					{Key: "b", Value: "v1"}, // <--- will override second metric
					{Key: "my-metric", Value: "v1"},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "correct seq",
			args: args{
				config: configJSON(),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: "my-metric", Value: "v1"},
					{Key: "b", Value: "v1"},
					{Key: "my-metric", Value: "v1"},
					{Key: "b", Value: "v1"},
				},
			},
			want: []*pmmv1.ServerMetric_Metric{
				{Key: configJSON().Transform.Metric, Value: `{"v":[{"b":"v1","my-metric":"v1"},{"b":"v1","my-metric":"v1"}]}`},
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy path",
			args: args{
				config: configJSON(),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: configJSON().Data[0].MetricName, Value: "v1"},
					{Key: configJSON().Data[0].MetricName, Value: "v2"},
				},
			},
			want: []*pmmv1.ServerMetric_Metric{
				{Key: configJSON().Transform.Metric, Value: `{"v":[{"my-metric":"v1"},{"my-metric":"v2"}]}`},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformToJSON(tt.args.config, tt.args.metrics)
			if !tt.wantErr(t, err) {
				t.Logf("config: %v", tt.args.config)
				return
			}
			assert.Equalf(t, tt.want, got, "transformToJSON(%v, %v)", tt.args.config, tt.args.metrics)
		})
	}
}

func TestTransformExportValues(t *testing.T) {
	type args struct {
		config  *Config
		metrics []*pmmv1.ServerMetric_Metric
	}

	noMetrics := []*pmmv1.ServerMetric_Metric{}

	tests := []struct {
		name    string
		args    args
		want    []*pmmv1.ServerMetric_Metric
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nil metrics",
			args: args{
				config:  configEnvVars(),
				metrics: nil,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "empty metrics",
			args: args{
				config:  configEnvVars(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Transform in config",
			args: args{
				config:  configEnvVars().noTransform(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Metrics config",
			args: args{
				config:  configEnvVars().noFirstMetricConfig(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "invalid data source",
			args: args{
				config: configEnvVars().changeDataSource(dsPMMDBSelect),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: "metric-a", Value: "v1"},
					{Key: "metric-b", Value: "v2"},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "happy path",
			args: args{
				config: configEnvVars(),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: "metric-a", Value: "v1"},
					{Key: "metric-b", Value: "v2"},
				},
			},
			want: []*pmmv1.ServerMetric_Metric{
				{Key: "metric-a", Value: "1"},
				{Key: "metric-b", Value: "1"},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformExportValues(tt.args.config, tt.args.metrics)
			if !tt.wantErr(t, err) {
				t.Logf("config: %v", tt.args.config)
				return
			}
			assert.Equalf(t, tt.want, got, "transformExportValues(%v, %v)", tt.args.config, tt.args.metrics)
		})
	}
}

func configJSON() *Config {
	return &Config{
		Transform: &ConfigTransform{
			Metric: "metric",
			Type:   JSONTransform,
		},
		Data: []ConfigData{
			{MetricName: "my-metric", Label: "label"},
		},
	}
}

func configEnvVars() *Config {
	return &Config{
		Source: "ENV_VARS",
		Transform: &ConfigTransform{
			Type: StripValuesTransform,
		},
	}
}

func (c *Config) noTransform() *Config {
	c.Transform = nil
	return c
}

func (c *Config) noFirstMetricConfig() *Config {
	c.Data = nil
	return c
}

func (c *Config) noFirstMetricNameConfig() *Config {
	c.Data[0].MetricName = ""
	return c
}

func (c *Config) changeDataSource(s DataSourceName) *Config {
	c.Source = string(s)
	return c
}

func TestRemoveEmpty(t *testing.T) {
	type args struct {
		metrics []*pmmv1.ServerMetric_Metric
	}
	tests := []struct {
		name string
		args args
		want []*pmmv1.ServerMetric_Metric
	}{
		{
			name: "should remove metrics with empty values",
			args: args{metrics: []*pmmv1.ServerMetric_Metric{
				{
					Key:   "empty_value",
					Value: "",
				},
				{
					Key:   "not_empty",
					Value: "not_empty",
				},
				{
					Key:   "empty_value",
					Value: "",
				},
			}},
			want: []*pmmv1.ServerMetric_Metric{
				{
					Key:   "not_empty",
					Value: "not_empty",
				},
			},
		},
		{
			name: "should not remove anything if metrics are not empty",
			args: args{metrics: []*pmmv1.ServerMetric_Metric{
				{
					Key:   "not_empty",
					Value: "not_empty",
				},
				{
					Key:   "not_empty_2",
					Value: "not_empty",
				},
			}},
			want: []*pmmv1.ServerMetric_Metric{
				{
					Key:   "not_empty",
					Value: "not_empty",
				},
				{
					Key:   "not_empty_2",
					Value: "not_empty",
				},
			},
		},
		{
			name: "should not remove anything if metrics are not empty",
			args: args{metrics: nil},
			want: []*pmmv1.ServerMetric_Metric{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, removeEmpty(tt.args.metrics), "removeEmpty(%v)", tt.args.metrics)
		})
	}
}
