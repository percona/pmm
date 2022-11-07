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

package telemetry

import (
	"encoding/json"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/pkg/errors"
)

type itemsType []map[string]any

func transformToJSON(config *Config, metrics []*pmmv1.ServerMetric_Metric) ([]*pmmv1.ServerMetric_Metric, error) {
	if metrics == nil {
		return nil, nil
	}

	if config.Transform == nil {
		return nil, errors.Errorf("no transformation config is set")
	}

	if config.Transform.Type != JSONTransformType {
		return nil, errors.Errorf("not supported transformation type [%s], it must be [%s]", config.Transform.Type, JSONTransformType)
	}

	if len(config.Data) == 0 || config.Data[0].MetricName == "" {
		return nil, errors.Errorf("invalid metrics config")
	}

	firstMetric := config.Data[0].MetricName

	result := make(map[string]itemsType)

	var items itemsType
	var next map[string]any
	for _, metric := range metrics {
		isFirstMetric := metric.Key == firstMetric
		if isFirstMetric {
			if next != nil {
				items = append(items, next)
			}
			next = make(map[string]any)
		}
		if next == nil {
			return nil, errors.Errorf("invalid metrics sequence: no match with first metric")
		}
		next[metric.Key] = metric.Value
	}
	if next != nil {
		items = append(items, next)
	}

	// nothing to process
	if items == nil {
		return metrics, nil
	}

	result["v"] = items

	resultAsJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return []*pmmv1.ServerMetric_Metric{
		{
			Key:   config.Transform.Metric,
			Value: string(resultAsJSON),
		},
	}, nil
}

func removeEmpty(metrics []*pmmv1.ServerMetric_Metric) []*pmmv1.ServerMetric_Metric {
	result := make([]*pmmv1.ServerMetric_Metric, 0, len(metrics))

	for _, metric := range metrics {
		if metric.Value != "" {
			result = append(result, metric)
		}
	}

	return result
}
