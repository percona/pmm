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

package slowlog

import (
	"context"
	"fmt"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm/agent/agents/mongodb/slowlog/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/slowlog/internal/sender"
	"github.com/percona/pmm/agent/utils/mongo_fix"
)

// New creates new slowlog
func New(mongoDSN string, logger *logrus.Entry, w sender.Writer, agentID string, maxQueryLength int32) *slowlog {
	return &slowlog{
		mongoDSN:       mongoDSN,
		maxQueryLength: maxQueryLength,
		logger:         logger,
		w:              w,
		agentID:        agentID,
	}
}

type slowlog struct {
	// dependencies
	mongoDSN string
	w        sender.Writer
	logger   *logrus.Entry
	agentID  string

	// internal deps
	monitors   *monitors
	client     *mongo.Client
	aggregator *aggregator.Aggregator
	sender     *sender.Sender

	// state
	m        sync.Mutex      // Lock() to protect internal consistency of the service
	running  bool            // Is this service running?
	doneChan chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg       *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown

	// others
	maxQueryLength int32
}

// Start starts analyzer but doesn't wait until it exits
func (s *slowlog) Start() error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.running {
		return nil
	}

	// create new session
	client, err := createSession(s.mongoDSN, s.agentID)
	if err != nil {
		return err
	}
	s.client = client

	// create aggregator which collects documents and aggregates them into qan report
	s.aggregator = aggregator.New(time.Now(), s.agentID, s.logger, s.maxQueryLength)
	reportChan := s.aggregator.Start()

	// create sender which sends qan reports and start it
	s.sender = sender.New(reportChan, s.w, s.logger)
	err = s.sender.Start()
	if err != nil {
		return err
	}

	f := func(client *mongo.Client, logger *logrus.Entry, dbName string) *monitor {
		return NewMonitor(client, dbName, s.aggregator, logger)
	}

	// create monitors service which we use to periodically scan server for new/removed databases
	s.monitors = NewMonitors(client, f, s.logger)

	// create new channel over which
	// we will tell goroutine it should close
	s.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	s.wg = &sync.WaitGroup{}
	s.wg.Add(1)

	// create ready sync.Cond so we could know when goroutine actually started getting data from db
	ready := sync.NewCond(&sync.Mutex{})
	ready.L.Lock()
	defer ready.L.Unlock()

	ctx := context.Background()
	labels := pprof.Labels("component", "mongodb.slowlog")
	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(ctx, s.monitors, s.wg, s.doneChan, ready, s.logger)
	})

	// wait until we actually fetch data from db
	ready.Wait()

	s.running = true
	return nil
}

// Stop stops running analyzer, waits until it stops
func (s *slowlog) Stop() error {
	s.m.Lock()
	defer s.m.Unlock()
	if !s.running {
		return nil
	}

	// notify goroutine to close
	close(s.doneChan)

	// wait for goroutine to exit
	s.wg.Wait()

	// stop aggregator; do it after goroutine is closed
	s.aggregator.Stop()

	// stop sender; do it after goroutine is closed
	s.sender.Stop()

	// close the session; do it after goroutine is closed
	s.client.Disconnect(context.TODO()) //nolint:errcheck

	// set state to "not running"
	s.running = false
	return nil
}

func start(ctx context.Context, monitors *monitors, wg *sync.WaitGroup, doneChan <-chan struct{}, ready *sync.Cond, logger *logrus.Entry) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	// stop all monitors
	defer monitors.StopAll()

	// monitor all databases
	err := monitors.MonitorAll(ctx)
	if err != nil {
		logger.Debugf("couldn't monitor all databases, reason: %v", err)
	}

	// signal we started monitoring
	signalReady(ready)

	// loop to periodically refresh monitors
	for {
		// check if we should shutdown
		select {
		case <-doneChan:
			return
		case <-time.After(1 * time.Minute):
			// just continue after delay if not
		}

		// update monitors
		err = monitors.MonitorAll(ctx)
		if err != nil {
			logger.Debugf("couldn't monitor all databases, reason: %v", err)
		}
	}
}

func signalReady(ready *sync.Cond) {
	ready.L.Lock()
	defer ready.L.Unlock()
	ready.Broadcast()
}

func createSession(dsn string, agentID string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), MgoTimeoutDialInfo)
	defer cancel()

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetSocketTimeout(MgoTimeoutSessionSocket).
		SetAppName(fmt.Sprintf("QAN-mongodb-slowlog-%s", agentID))

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}
