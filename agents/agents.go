// pmm-agent
// Copyright (C) 2018 Percona LLC
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
	"github.com/percona/pmm/api/inventorypb"
)

// Change represents built-in Agent status change and/or QAN collect request.
type Change struct {
	Status        inventorypb.AgentStatus
	MetricsBucket []*agentpb.MetricsBucket
}

// BuiltinAgent is a common interface for all built-in Agents.
type BuiltinAgent interface {
	// Run extracts stats data and sends it to the channel until ctx is canceled.
	Run(ctx context.Context)

	// Changes returns channel that should be read until it is closed.
	Changes() <-chan Change
}
