package telemetry

import (
	"context"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
)

// Extension provides dynamic extension point for Telemetry.
type Extension interface {
	FetchMetrics(ctx context.Context, report *Config) ([]*pmmv1.ServerMetric_Metric, error)
}
