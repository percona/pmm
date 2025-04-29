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

// Package collector implements collecting mongo logs from file.
package collector

import (
	"context"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
)

const collectorChanCapacity = 100

// New creates new Collector.
func New(logsPath string, logger *logrus.Entry) *Collector {
	return &Collector{
		logsPath: logsPath,
		logger:   logger.WithField("log", logsPath),
	}
}

// Collector is used by Mongolog agent.
type Collector struct {
	// dependencies
	logsPath string
	logger   *logrus.Entry

	// provides
	docsChan chan proto.SystemProfile

	// state
	m        sync.Mutex      // Lock() to protect internal consistency of the service
	running  bool            // Is this service running?
	doneChan chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg       *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts but doesn't wait until it exits.
func (c *Collector) Start(ctx context.Context) (<-chan proto.SystemProfile, error) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.running {
		return nil, nil
	}

	// create new channels over which we will communicate to...
	// ... outside world by sending collected docs
	c.docsChan = make(chan proto.SystemProfile, collectorChanCapacity)
	// ... inside goroutine to close it
	c.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	c.wg = &sync.WaitGroup{}
	c.wg.Add(1)

	// create ready sync.Cond so we could know when goroutine actually started getting data from db
	ready := sync.NewCond(&sync.Mutex{})
	ready.L.Lock()
	defer ready.L.Unlock()

	labels := pprof.Labels("component", "mongodb.aggregator")
	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(
			ctx,
			c.wg,
			c.logsPath,
			c.docsChan,
			c.doneChan,
			ready,
			c.logger)
	})

	// wait until we actually fetch data from db
	ready.Wait()

	c.running = true

	return c.docsChan, nil
}

// Stop stops running collector.
func (c *Collector) Stop() {
	c.m.Lock()
	defer c.m.Unlock()

	if !c.running {
		return
	}

	c.running = false
	close(c.doneChan) // notify goroutine to close
	c.wg.Wait()       // wait for goroutines to exit
	close(c.docsChan) // we can now safely close channels goroutines write to as goroutine is stopped
}

func start(ctx context.Context, wg *sync.WaitGroup, logsPath string,
	docsChan chan<- proto.SystemProfile, doneChan <-chan struct{}, ready *sync.Cond, logger *logrus.Entry,
) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	fr, err := NewReader(docsChan, doneChan, logsPath, logger)
	if err != nil {
		logger.Error(err)
		return
	}
	go func() {
		fr.ReadFile(ctx)
		logger.Debugln("reading routine quit")
	}()

	firstTry := true

	for {
		select {
		// check if we should shutdown
		case <-ctx.Done():
			return
		case <-doneChan:
			return
		// wait some time before reconnecting
		case <-time.After(1 * time.Second):
		}

		// After first failure in connection we signal that we are ready anyway
		// this way service starts, and will automatically connect when db is available.
		if firstTry {
			signalReady(ready)
			firstTry = false
		}
	}
}

func signalReady(ready *sync.Cond) {
	ready.L.Lock()
	defer ready.L.Unlock()
	ready.Broadcast()
}
