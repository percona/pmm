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

	result := map[string]itemsType{}

	var items []map[string]any
	var next map[string]any
	for _, metric := range metrics {
		isFirstMetric := metric.Key == firstMetric
		if isFirstMetric {
			if next != nil {
				items = append(items, next)
			}
			next = map[string]any{}
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
