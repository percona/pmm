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

package om

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// victoriaMetricsClient is a subset of methods of prometheus' API used by this package.
type victoriaMetricsClient interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error)
}

// haChecker reports HA leadership, so a write-triggering RPC can refuse on a follower
// rather than race the leader for the same collection.
//
// Optional like probe: a Service built without one (every test in this package but the
// ones that care) treats every node as the leader, which is the single-node default too.
type haChecker interface {
	IsLeader() bool
	LeaderID() string
}

// agentConnectionChecker reports whether a pmm-agent is currently connected. A subset
// of *agents.Registry's exported surface -- the "PMM-Client installed and healthy"
// signal ListInventoryHosts needs, which om_inventory has no way to answer for itself:
// a node existing in its estate only means PMM registered it once, not that the agent
// on it is alive now. See inventory.go's automationEligibility.
//
// Optional like probe and ha: a Service built without one treats every host as
// disconnected, which is the fail-closed default for something that gates automation
// eligibility.
type agentConnectionChecker interface {
	IsConnected(pmmAgentID string) bool
}
