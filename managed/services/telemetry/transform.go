package telemetry

import (
	"encoding/json"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/pkg/errors"
)

type itemsType []map[string]any

func transformToJSON(config *Config, metrics []*pmmv1.ServerMetric_Metric) ([]*pmmv1.ServerMetric_Metric, error) {
	if config.Transform.Type != JSONTransformType {
		return nil, errors.Errorf("not supported transformation type [%s], it must be [%s]", config.Transform.Type, JSONTransformType)
	}

	result := map[string]itemsType{}

	var items []map[string]any
	var next map[string]any
	for _, metric := range metrics {
		isFirstMetric := metric.Key == config.Data[0].MetricName
		if isFirstMetric {
			if next != nil {
				items = append(items, next)
			}
			next = map[string]any{}
		}
		next[metric.Key] = metric.Value
	}
	if next != nil {
		items = append(items, next)
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
	result := make([]*pmmv1.ServerMetric_Metric, len(metrics))

	for _, metric := range metrics {
		if metric.Value != "" {
			result = append(result, metric)
		}
	}

	return result
}
