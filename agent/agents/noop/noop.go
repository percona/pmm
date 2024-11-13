// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package noop runs no-op built-in Agent for testing.
package noop

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/api/inventorypb"
)

// NoOp is built-in Agent for testing.
type NoOp struct {
	changes chan agents.Change
}

// New creates new NoOp.
func New() *NoOp {
	return &NoOp{
		changes: make(chan agents.Change, 10),
	}
}

// Run is doing nothing until ctx is canceled.
func (n *NoOp) Run(ctx context.Context) {
	n.changes <- agents.Change{Status: inventorypb.AgentStatus_STARTING}

	time.Sleep(time.Second)
	n.changes <- agents.Change{Status: inventorypb.AgentStatus_RUNNING}

	<-ctx.Done()

	n.changes <- agents.Change{Status: inventorypb.AgentStatus_STOPPING}
	n.changes <- agents.Change{Status: inventorypb.AgentStatus_DONE}
	close(n.changes)
}

// Changes returns channel that should be read until it is closed.
func (n *NoOp) Changes() <-chan agents.Change {
	return n.changes
}

// Describe implements prometheus.Collector.
func (n *NoOp) Describe(ch chan<- *prometheus.Desc) { //nolint:revive
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (n *NoOp) Collect(ch chan<- prometheus.Metric) { //nolint:revive
	// This method is needed to satisfy interface.
}

// check interfaces.
var (
	_ prometheus.Collector = (*NoOp)(nil)
)
