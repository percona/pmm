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

package parser

import (
	"sync"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	mstats "github.com/percona/percona-toolkit/src/go/mongolib/stats"

	"github.com/percona/pmm-agent/agents/builtin/mongodb/internal/profiler/aggregator"
	"github.com/percona/pmm-agent/agents/builtin/mongodb/internal/status"
)

func New(docsChan <-chan proto.SystemProfile, aggregator *aggregator.Aggregator) *Parser {
	return &Parser{
		docsChan:   docsChan,
		aggregator: aggregator,
	}
}

type Parser struct {
	// dependencies
	docsChan   <-chan proto.SystemProfile
	aggregator *aggregator.Aggregator

	// status
	status *status.Status

	// state
	sync.RWMutex                 // Lock() to protect internal consistency of the service
	running      bool            // Is this service running?
	doneChan     chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg           *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts but doesn't wait until it exits
func (self *Parser) Start() error {
	self.Lock()
	defer self.Unlock()
	if self.running {
		return nil
	}

	// create new channels over which we will communicate to...
	// ... inside goroutine to close it
	self.doneChan = make(chan struct{})

	// set status
	stats := &stats{}
	self.status = status.New(stats)

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	self.wg = &sync.WaitGroup{}
	self.wg.Add(1)
	go start(
		self.wg,
		self.docsChan,
		self.aggregator,
		self.doneChan,
		stats,
	)

	self.running = true
	return nil
}

// Stop stops running
func (self *Parser) Stop() {
	self.Lock()
	defer self.Unlock()
	if !self.running {
		return
	}
	self.running = false

	// notify goroutine to close
	close(self.doneChan)

	// wait for goroutines to exit
	self.wg.Wait()
	return
}

func (self *Parser) Status() map[string]string {
	self.RLock()
	defer self.RUnlock()
	if !self.running {
		return nil
	}

	return self.status.Map()
}

func (self *Parser) Name() string {
	return "parser"
}

func start(wg *sync.WaitGroup, docsChan <-chan proto.SystemProfile, aggregator *aggregator.Aggregator, doneChan <-chan struct{}, stats *stats) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	// update stats
	for {
		// check if we should shutdown
		select {
		case <-doneChan:
			return
		default:
			// just continue if not
		}

		// aggregate documents and create report
		select {
		case doc, ok := <-docsChan:
			// if channel got closed we should exit as there is nothing we can listen to
			if !ok {
				return
			}

			// we got new doc, increase stats
			stats.InDocs += 1

			// aggregate the doc
			var err error
			err = aggregator.Add(doc)
			switch err.(type) {
			case nil:
				stats.OkDocs += 1
			case *mstats.StatsFingerprintError:
				stats.ErrFingerprint += 1
			default:
				stats.ErrParse += 1
			}
		case <-doneChan:
			// doneChan needs to be repeated in this select as docsChan can block
			// doneChan needs to be also in separate select statement
			// as docsChan could be always picked since select picks channels pseudo randomly
			return
		}
	}
}
