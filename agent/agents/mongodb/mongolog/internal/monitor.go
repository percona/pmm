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

package mongolog

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/collector"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/parser"
)

// NewMonitor creates new monitor.
func NewMonitor(client *mongo.Client, logPath string, aggregator *aggregator.Aggregator, logger *logrus.Entry) *monitor {
	return &monitor{
		client:     client,
		logPath:    logPath,
		aggregator: aggregator,
		logger:     logger,
	}
}

type monitor struct {
	// dependencies
	client     *mongo.Client // TODO REMOVE???
	logPath    string
	aggregator *aggregator.Aggregator
	logger     *logrus.Entry

	// state
	m       sync.Mutex // Lock() to protect internal consistency of the service
	running bool       // Is this service running?
}

func (m *monitor) Start(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()

	if m.running {
		return nil
	}

	// create collector and start it
	c := collector.New(m.logPath, m.logger)
	docsChan, err := c.Start(ctx)
	if err != nil {
		return err
	}

	// create parser and start it
	p := parser.New(docsChan, m.aggregator, m.logger)
	err = p.Start(ctx)
	if err != nil {
		return err
	}

	m.running = true
	return nil
}

func (m *monitor) Stop() {
	m.m.Lock()
	defer m.m.Unlock()

	if !m.running {
		return
	}

	m.running = false
}

type services interface {
	Stop()
	Name() string
}
