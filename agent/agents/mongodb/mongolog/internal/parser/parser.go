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

package parser

import (
	"context"
	"runtime/pprof"
	"sync"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/aggregator"
)

// New creates new parser.
func New(docsChan <-chan proto.SystemProfile, aggregator *aggregator.Aggregator, logger *logrus.Entry) *Parser {
	return &Parser{
		docsChan:   docsChan,
		aggregator: aggregator,
		logger:     logger,
	}
}

// Parser represents docs channel, aggregator and helpers.
type Parser struct {
	// dependencies
	docsChan   <-chan proto.SystemProfile
	aggregator *aggregator.Aggregator

	logger *logrus.Entry

	// state
	m        sync.Mutex      // Lock() to protect internal consistency of the service
	running  bool            // Is this service running?
	doneChan chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg       *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts but doesn't wait until it exits.
func (p *Parser) Start(ctx context.Context) error {
	p.m.Lock()
	defer p.m.Unlock()
	if p.running {
		return nil
	}

	// create new channels over which we will communicate to...
	// ... inside goroutine to close it
	p.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	p.wg = &sync.WaitGroup{}
	p.wg.Add(1)

	labels := pprof.Labels("component", "mongodb.monitor")
	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(
			ctx,
			p.wg,
			p.docsChan,
			p.aggregator,
			p.doneChan,
			p.logger)
	})

	p.running = true
	return nil
}

// Stop stops running parser.
func (p *Parser) Stop() {
	p.m.Lock()
	defer p.m.Unlock()
	if !p.running {
		return
	}
	p.running = false

	// notify goroutine to close
	close(p.doneChan)

	// wait for goroutines to exit
	p.wg.Wait()
}

func start(ctx context.Context, wg *sync.WaitGroup, docsChan <-chan proto.SystemProfile, aggregator *aggregator.Aggregator,
	doneChan <-chan struct{}, logger *logrus.Entry,
) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	// update stats
	for {
		// aggregate documents and create report
		select {
		case doc, ok := <-docsChan:
			// if channel got closed we should exit as there is nothing we can listen to
			if !ok {
				return
			}

			logger.Debugf("added to aggregator %v", doc.Query)
			// aggregate the doc
			err := aggregator.Add(ctx, doc)
			if err != nil {
				logger.Warnf("couldn't add document to aggregator: %s", err)
			}
		case <-doneChan:
			// doneChan needs to be repeated in this select as docsChan can block
			// doneChan needs to be also in separate select statement
			// as docsChan could be always picked since select picks channels pseudo randomly
			return
		case <-ctx.Done():
			return
		}
	}
}
