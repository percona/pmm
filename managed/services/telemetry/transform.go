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
	"encoding/json"

	telemetryv1 "github.com/percona/saas/gen/telemetry/generic"
	"github.com/pkg/errors"
)

type itemsType []map[string]any

func transformToJSON(config *Config, metrics []*telemetryv1.GenericReport_Metric) ([]*telemetryv1.GenericReport_Metric, error) {
	if len(metrics) == 0 {
		return metrics, nil
	}

	if config.Transform == nil {
		return nil, errors.Errorf("no transformation config is set")
	}

	if config.Transform.Type != JSONTransform {
		return nil, errors.Errorf("unsupported transformation type [%s], it must be [%s]", config.Transform.Type, JSONTransform)
	}

	if len(config.Data) == 0 || config.Data[0].MetricName == "" {
		return nil, errors.Errorf("invalid metrics config")
	}

	// consider first metric is the beginning of the object window
	firstMetric := metrics[0].Key

	result := make(map[string]itemsType)

	var items itemsType
	var next map[string]any
	for _, metric := range metrics {
		windowStarts := metric.Key == firstMetric // marker to know that the object begins
		if windowStarts {
			if next != nil {
				items = append(items, next)
			}
			next = make(map[string]any)
		}
		if _, alreadyHasItem := next[metric.Key]; alreadyHasItem {
			return nil, errors.Errorf("invalid metrics sequence")
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

	return []*telemetryv1.GenericReport_Metric{
		{
			Key:   config.Transform.Metric,
			Value: string(resultAsJSON),
		},
	}, nil
}

func transformExportValues(config *Config, metrics []*telemetryv1.GenericReport_Metric) ([]*telemetryv1.GenericReport_Metric, error) {
	if len(metrics) == 0 {
		return metrics, nil
	}

	if config.Transform.Type != StripValuesTransform {
		return nil, errors.Errorf("unspported transformation type [%s], it must be [%s]", config.Transform.Type, StripValuesTransform)
	}

	if config.Source != string(dsEnvVars) {
		return nil, errors.Errorf("this transform can only be used for %s data source", dsEnvVars)
	}

	for _, metric := range metrics {
		// Here we replace the metric value with "1", which stands for "present".
		metric.Value = "1"
	}

	return metrics, nil
}

func removeEmpty(metrics []*telemetryv1.GenericReport_Metric) []*telemetryv1.GenericReport_Metric {
	result := make([]*telemetryv1.GenericReport_Metric, 0, len(metrics))

	for _, metric := range metrics {
		if metric.Value != "" {
			result = append(result, metric)
		}
	}

	return result
}
