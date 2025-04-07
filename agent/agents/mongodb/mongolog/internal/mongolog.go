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
	"encoding/json"
	"fmt"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/collector"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/sender"
	"github.com/percona/pmm/agent/utils/mongo_fix"
)

const (
	slowQuery               = "Slow query"
	MgoTimeoutDialInfo      = 5 * time.Second
	MgoTimeoutSessionSocket = 5 * time.Second
)

// New creates new mongolog
func New(mongoDSN string, logger *logrus.Entry, w sender.Writer, agentID string, logFilePrefix string, maxQueryLength int32) *mongolog {
	return &mongolog{
		mongoDSN:       mongoDSN,
		logFilePrefix:  logFilePrefix,
		maxQueryLength: maxQueryLength,
		logger:         logger,
		w:              w,
		agentID:        agentID,
	}
}

type mongolog struct {
	// dependencies
	mongoDSN string
	w        sender.Writer
	logger   *logrus.Entry
	agentID  string

	// internal deps
	monitor    *monitor
	client     *mongo.Client
	aggregator *aggregator.Aggregator
	sender     *sender.Sender

	// state
	m        sync.Mutex      // Lock() to protect internal consistency of the service
	running  bool            // Is this service running?
	doneChan chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg       *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown

	// others
	logFilePrefix  string
	maxQueryLength int32
}

// Start starts analyzer but doesn't wait until it exits
func (s *mongolog) Start() error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.running {
		return nil
	}

	// create new channel over which
	// we will tell goroutine it should close
	s.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	s.wg = &sync.WaitGroup{}
	s.wg.Add(1)

	ctx := context.Background()
	labels := pprof.Labels("component", "mongodb.mongolog")

	// create new session
	client, err := createSession(s.mongoDSN, s.agentID)
	if err != nil {
		return err
	}
	logsPath, err := collector.GetLogFilePath(client)
	if err != nil {
		return err
	}
	client.Disconnect(ctx)

	// create aggregator which collects documents and aggregates them into qan report
	s.aggregator = aggregator.New(time.Now(), s.agentID, s.logger, s.maxQueryLength)
	reportChan := s.aggregator.Start()

	// create sender which sends qan reports and start it
	s.sender = sender.New(reportChan, s.w, s.logger)
	err = s.sender.Start()
	if err != nil {
		return err
	}

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

	// create monitors service which we use to periodically scan server for new/removed databases
	s.monitor = NewMonitor(client, logsPath, s.aggregator, s.logger)

	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(ctx, s.monitor, s.wg, s.doneChan, ready, s.logger)
	})

	// wait until we actually fetch data from db
	ready.Wait()

	s.running = true
	return nil
}

// Stop stops running analyzer, waits until it stops
func (s *mongolog) Stop() error {
	s.m.Lock()
	defer s.m.Unlock()
	if !s.running {
		return nil
	}

	// notify goroutine to close
	close(s.doneChan)

	// wait for goroutine to exit
	s.wg.Wait()

	// set state to "not running"
	s.running = false
	return nil
}

type SlowQuery struct {
	// Ctx  string `bson:"ctx"`
	Msg  string `bson:"msg"`
	Attr json.RawMessage
}

type systemProfile struct {
	proto.SystemProfile
	// Command bson.Raw `bson:"command,omitempty"`
	Command bson.M `bson:"command"`
}

func start(ctx context.Context, monitor *monitor, wg *sync.WaitGroup, doneChan <-chan struct{}, ready *sync.Cond, logger *logrus.Entry) {
	// TODO context usage
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	// monitor log file
	err := monitor.Start(ctx)
	if err != nil {
		logger.Debugf("couldn't monitor log file (%s), reason: %v", monitor.logPath, err)
	}

	// signal we started monitoring
	signalReady(ready)

	// loop to periodically refresh
	for {
		// check if we should shutdown
		select {
		case <-doneChan:
			return
		case <-time.After(1 * time.Minute):
			// just continue after delay if not
		}

		// update monitors
		err = monitor.Start(ctx)
		if err != nil {
			logger.Debugf("couldn't monitor log file (%s), reason: %v", monitor.logPath, err)
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
		SetAppName(fmt.Sprintf("QAN-mongodb-mongolog-%s", agentID))

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}
