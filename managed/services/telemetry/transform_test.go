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
	"fmt"
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
				config:  config(),
				metrics: nil,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "empty metrics",
			args: args{
				config:  config(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Transform in config",
			args: args{
				config:  config().noTransform(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Metrics config",
			args: args{
				config:  config().noFirstMetricConfig(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "no Metric Name config",
			args: args{
				config:  config().noFirstMetricNameConfig(),
				metrics: noMetrics,
			},
			want:    noMetrics,
			wantErr: assert.NoError,
		},
		{
			name: "invalid seq",
			args: args{
				config: config(),
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
				config: config(),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: "my-metric", Value: "v1"},
					{Key: "b", Value: "v1"},
					{Key: "my-metric", Value: "v1"},
					{Key: "b", Value: "v1"},
				},
			},
			want: []*pmmv1.ServerMetric_Metric{
				{Key: config().Transform.Metric, Value: `{"v":[{"b":"v1","my-metric":"v1"},{"b":"v1","my-metric":"v1"}]}`},
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy path",
			args: args{
				config: config(),
				metrics: []*pmmv1.ServerMetric_Metric{
					{Key: config().Data[0].MetricName, Value: "v1"},
					{Key: config().Data[0].MetricName, Value: "v2"},
				},
			},
			want: []*pmmv1.ServerMetric_Metric{
				{Key: config().Transform.Metric, Value: `{"v":[{"my-metric":"v1"},{"my-metric":"v2"}]}`},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformToJSON(tt.args.config, tt.args.metrics)
			if !tt.wantErr(t, err, fmt.Sprintf("transformToJSON(%v, %v)", tt.args.config, tt.args.metrics)) {
				return
			}
			assert.Equalf(t, tt.want, got, "transformToJSON(%v, %v)", tt.args.config, tt.args.metrics)
		})
	}
}

func config() *Config {
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
