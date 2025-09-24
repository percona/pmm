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

package agents

import (
	"context"
	"net/url"

	"github.com/sirupsen/logrus"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	qanv1 "github.com/percona/pmm/api/qan/v1"
)

// prometheusService is a subset of methods of victoriametrics.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
//
// FIXME Rename to victoriaMetrics.Service, update tests.
type prometheusService interface {
	RequestConfigurationUpdate()
	BuildScrapeConfigForVMAgent(pmmAgentID string) ([]byte, error)
}

// qanClient is a subset of methods of qan.Client used by this package.
// We use it instead of real type to avoid dependency cycle.
type qanClient interface {
	Collect(ctx context.Context, metricsBuckets []*agentv1.MetricsBucket) error
	QueryExists(ctx context.Context, serviceID, query string) error
	ExplainFingerprintByQueryID(ctx context.Context, serviceID, queryID string) (*qanv1.ExplainFingerprintByQueryIDResponse, error)
	SchemaByQueryID(ctx context.Context, serviceID, queryID string) (*qanv1.SchemaByQueryIDResponse, error)
}

// retentionService is a subset of methods of backup.Client used by this package.
// We use it instead of real type to avoid dependency cycle.
type retentionService interface {
	EnforceRetention(scheduleID string) error
}

// jobsService is a subset of methods of agents.JobsService used by this package.
// We use it instead of real type to avoid dependency cycle.
type jobsService interface {
	handleJobResult(ctx context.Context, l *logrus.Entry, result *agentv1.JobResult)
	handleJobProgress(ctx context.Context, progress *agentv1.JobProgress)
}

// victoriaMetricsParams is a subset of methods of models.VMParams used by this package.
// We use it instead of real type to avoid dependency cycle.
type victoriaMetricsParams interface {
	ExternalVM() bool
	URLFor(path string) (*url.URL, error)
	URL() string
	VMAgentArgs() []string
}

type nomad interface {
	GetCACert() (string, error)
	GetClientCert() (string, error)
	GetClientKey() (string, error)
}
