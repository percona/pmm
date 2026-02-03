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

// Package realtimeanalytics runs built-in Real-Time Analytics Agent for MongoDB.
package realtimeanalytics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

// MongoDBRTA extracts Real-Time Analytics data (currently running DB queries) from MongoDB.
type MongoDBRTA struct {
	agentID string
	l       *logrus.Entry

	// Channel to obtain data from this agent.
	changes chan agents.Change

	// DSN to connect to MongoDB.
	mongoDSN string
	// CollectInterval is how often to collect data from MongoDB.
	CollectInterval time.Duration
}

// Params represent Agent parameters.
type Params struct {
	AgentID         string
	DSN             string        // DSN to connect to MongoDB.
	CollectInterval time.Duration // CollectInterval is how often to collect data from MongoDB.
}

// New creates new MongoDBRTA service.
func New(params *Params, l *logrus.Entry) (*MongoDBRTA, error) {
	// if params.DSN is incorrect we should exit immediately as this is not gonna correct itself
	_, err := connstring.Parse(params.DSN)
	if err != nil {
		return nil, err
	}

	return &MongoDBRTA{
		agentID:         params.AgentID,
		mongoDSN:        params.DSN,
		CollectInterval: params.CollectInterval,
		l:               l,
		changes:         make(chan agents.Change, 10),
	}, nil
}

// Run extracts currently running DB queries from MongoDB
// and sends it to the channel until ctx is canceled.
func (m *MongoDBRTA) Run(ctx context.Context) {
	m.l.Info("Starting MongoDB RTA agent")
	defer func() {
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(m.changes)
	}()

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}
	// TODO: run actual RTA data collection

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
	<-ctx.Done()
	m.l.Info("Stopping MongoDB RTA agent")
	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
}

// Changes returns channel that should be read until it is closed.
func (m *MongoDBRTA) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector.
func (m *MongoDBRTA) Describe(_ chan<- *prometheus.Desc) {
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (m *MongoDBRTA) Collect(_ chan<- prometheus.Metric) {
	// This method is needed to satisfy interface.
}

// check interfaces.
var (
	_ prometheus.Collector = (*MongoDBRTA)(nil)
)
