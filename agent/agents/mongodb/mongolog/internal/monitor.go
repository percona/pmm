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

	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/collector"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/parser"
)

// NewMonitor creates new monitor.
func NewMonitor(logPath string, aggregator *aggregator.Aggregator, logger *logrus.Entry) *Monitor {
	return &Monitor{
		logPath:    logPath,
		aggregator: aggregator,
		logger:     logger,
	}
}

// Monitor represents mongolog aggregator and helpers.
type Monitor struct {
	// dependencies
	logPath    string
	aggregator *aggregator.Aggregator
	logger     *logrus.Entry

	// state
	m       sync.Mutex
	running bool
}

func (m *Monitor) Start(ctx context.Context) error {
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

func (m *Monitor) Stop() {
	m.m.Lock()
	defer m.m.Unlock()

	if !m.running {
		return
	}

	m.running = false
}
