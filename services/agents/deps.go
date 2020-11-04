// pmm-managed
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

package agents

import (
	"context"

	"github.com/percona/pmm/api/agentpb"
)

// prometheusService is a subset of methods of victoriametrics.Service used by this package.
// We use it instead of real type to avoid dependency cycle.
//
// FIXME Rename to victoriaMetrics.Service, update tests.
type prometheusService interface {
	RequestConfigurationUpdate()
}

// qanClient is a subset of methods of qan.Client used by this package.
// We use it instead of real type to avoid dependency cycle.
type qanClient interface {
	Collect(ctx context.Context, metricsBuckets []*agentpb.MetricsBucket) error
}
