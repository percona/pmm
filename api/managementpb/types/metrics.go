package types

import (
	"errors"
	"strings"
)

// this list should be in sync with managementpb/metrics.go
const (
	MetricsModeAuto = "AUTO"
	MetricsModePull = "PULL"
	MetricsModePush = "PUSH"
)

var (
	validMetricModes   = []string{MetricsModeAuto, MetricsModePull, MetricsModePush}
	InvalidMetricsMode = errors.New("invalid metrics mode")
)

// FormatMetricsMode - formats given input metrics mode with
// valid enum value.
func FormatMetricsMode(input string) (string, error) {
	for _, mode := range validMetricModes {
		if strings.EqualFold(input, mode) {
			return mode, nil
		}
	}
	return "", InvalidMetricsMode
}
