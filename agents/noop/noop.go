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

// Package noop runs no-op built-in Agent for testing.
package noop

import (
	"context"
	"time"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
)

// NoOp is built-in Agent for testing.
type NoOp struct {
	changes chan Change
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request *qanpb.CollectRequest
}

// New creates new NoOp.
func New() *NoOp {
	return &NoOp{
		changes: make(chan Change, 10),
	}
}

// Run is doing nothing until ctx is canceled.
func (n *NoOp) Run(ctx context.Context) {
	n.changes <- Change{Status: inventorypb.AgentStatus_STARTING}

	time.Sleep(time.Second)
	n.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}

	<-ctx.Done()

	n.changes <- Change{Status: inventorypb.AgentStatus_STOPPING}
	n.changes <- Change{Status: inventorypb.AgentStatus_DONE}
	close(n.changes)
}

// Changes returns channel that should be read until it is closed.
func (n *NoOp) Changes() <-chan Change {
	return n.changes
}
