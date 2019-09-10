// pmm-agent
// Copyright 2019 Percona LLC
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
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/aggregator"
	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/collector"
	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/parser"
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
	sync.RWMutex      // Lock() to protect internal consistency of the service
	running      bool // Is this service running?
}

func (m *monitor) Start() error {
	m.Lock()
	defer m.Unlock()

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
	docsChan, err := c.Start()
	if err != nil {
		return err
	}
	m.services = append(m.services, c)

	// create parser and start it
	p := parser.New(docsChan, m.aggregator)
	err = p.Start()
	if err != nil {
		return err
	}
	m.services = append(m.services, p)

	m.running = true
	return nil
}

func (m *monitor) Stop() {
	m.Lock()
	defer m.Unlock()

	if !m.running {
		return
	}

	// stop internal services
	for _, s := range m.services {
		s.Stop()
	}

	m.running = false
}

// Status returns list of statuses
func (m *monitor) Status() map[string]string {
	m.RLock()
	defer m.RUnlock()

	statuses := &sync.Map{}

	wg := &sync.WaitGroup{}
	wg.Add(len(m.services))
	for _, s := range m.services {
		go func(s services) {
			defer wg.Done()
			for k, v := range s.Status() {
				key := fmt.Sprintf("%s-%s", s.Name(), k)
				statuses.Store(key, v)
			}
		}(s)
	}
	wg.Wait()

	statusesMap := map[string]string{}
	statuses.Range(func(key, value interface{}) bool {
		statusesMap[key.(string)] = value.(string)
		return true
	})

	return statusesMap
}

type services interface {
	Status() map[string]string
	Stop()
	Name() string
}
