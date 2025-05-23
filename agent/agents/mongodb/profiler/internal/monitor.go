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

package profiler

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/collector"
	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/parser"
	"github.com/percona/pmm/agent/agents/mongodb/shared/aggregator"
)

// NewMonitor creates new monitor.
func NewMonitor(client *mongo.Client, dbName string, aggregator *aggregator.Aggregator, logger *logrus.Entry) *monitor {
	return &monitor{
		client:     client,
		dbName:     dbName,
		aggregator: aggregator,
		logger:     logger,
	}
}

type monitor struct {
	// dependencies
	client     *mongo.Client
	dbName     string
	aggregator *aggregator.Aggregator
	logger     *logrus.Entry

	// internal services
	services []services

	// state
	m       sync.Mutex // Lock() to protect internal consistency of the service
	running bool       // Is this service running?
}

// Start starts monitor to collect and parse data.
func (m *monitor) Start(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()

	if m.running {
		return nil
	}

	defer func() {
		// if we failed to start
		if !m.running {
			// be sure that any started internal service is shutdown
			for _, s := range m.services {
				s.Stop()
			}
			m.services = nil
		}
	}()

	// create collector and start it
	c := collector.New(m.client, m.dbName, m.logger)
	docsChan, err := c.Start(ctx)
	if err != nil {
		return err
	}
	m.services = append(m.services, c)

	// create parser and start it
	p := parser.New(docsChan, m.aggregator, m.logger)
	err = p.Start(ctx)
	if err != nil {
		return err
	}
	m.services = append(m.services, p)

	m.running = true
	return nil
}

// Stop stops monitor.
func (m *monitor) Stop() {
	m.m.Lock()
	defer m.m.Unlock()

	if !m.running {
		return
	}

	// stop internal services
	for _, s := range m.services {
		s.Stop()
	}

	m.running = false
}

type services interface {
	Stop()
	Name() string
}
