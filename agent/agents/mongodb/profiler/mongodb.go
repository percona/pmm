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

// Package profiler runs built-in QAN Agent for MongoDB profiler.
package profiler

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"

	"github.com/percona/pmm/agent/agents"
	profiler "github.com/percona/pmm/agent/agents/mongodb/profiler/internal"
	"github.com/percona/pmm/agent/agents/mongodb/shared/report"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

// MongoDB extracts performance data from Mongo op log.
type MongoDB struct {
	agentID string
	l       *logrus.Entry
	changes chan agents.Change

	mongoDSN       string
	maxQueryLength int32
}

// Params represent Agent parameters.
type Params struct {
	DSN            string
	AgentID        string
	MaxQueryLength int32
}

// New creates new MongoDB QAN service.
func New(params *Params, l *logrus.Entry) (*MongoDB, error) {
	// if dsn is incorrect we should exit immediately as this is not gonna correct itself
	_, err := connstring.Parse(params.DSN)
	if err != nil {
		return nil, err
	}

	return newMongo(params.DSN, l, params), nil
}

func newMongo(mongoDSN string, l *logrus.Entry, params *Params) *MongoDB {
	return &MongoDB{
		agentID:        params.AgentID,
		mongoDSN:       mongoDSN,
		maxQueryLength: params.MaxQueryLength,

		l:       l,
		changes: make(chan agents.Change, 10),
	}
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *MongoDB) Run(ctx context.Context) {
	var prof Profiler

	defer func() {
		prof.Stop() //nolint:errcheck
		prof = nil
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(m.changes)
	}()

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	prof = profiler.New(m.mongoDSN, m.l, m, m.agentID, m.maxQueryLength)
	if err := prof.Start(); err != nil {
		m.l.Errorf("can't run profiler, reason: %v", err)
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
		return
	}

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}

	<-ctx.Done()
	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
}

// Changes returns channel that should be read until it is closed.
func (m *MongoDB) Changes() <-chan agents.Change {
	return m.changes
}

// Write writes MetricsBuckets to pmm-managed
func (m *MongoDB) Write(r *report.Report) error {
	m.changes <- agents.Change{MetricsBucket: r.Buckets}
	return nil
}

type Profiler interface { //nolint:revive
	Start() error
	Stop() error
}

// Describe implements prometheus.Collector.
func (m *MongoDB) Describe(ch chan<- *prometheus.Desc) { //nolint:revive
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (m *MongoDB) Collect(ch chan<- prometheus.Metric) { //nolint:revive
	// This method is needed to satisfy interface.
}

// check interfaces.
var (
	_ prometheus.Collector = (*MongoDB)(nil)
)
