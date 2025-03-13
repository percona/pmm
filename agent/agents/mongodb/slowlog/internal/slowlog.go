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
	"log"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/agents/mongodb/slowlog/internal/reader"
	"github.com/percona/pmm/agent/utils/mongo_fix"
)

const (
	MgoTimeoutDialInfo      = 5 * time.Second
	MgoTimeoutSessionSocket = 5 * time.Second
)

// New creates new slowlog
func New(mongoDSN string, logger *logrus.Entry, changes chan agents.Change, agentID string, maxQueryLength int32) *slowlog {
	return &slowlog{
		mongoDSN:       mongoDSN,
		maxQueryLength: maxQueryLength,
		logger:         logger,
		changes:        changes,
		agentID:        agentID,
	}
}

type slowlog struct {
	// dependencies
	mongoDSN string
	changes  chan agents.Change
	logger   *logrus.Entry
	agentID  string

	// internal deps
	client *mongo.Client

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
	logsPath, err := reader.GetLogFilePath(client)
	if err != nil {
		return err
	}
	client.Disconnect(context.TODO())

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
		start(ctx, s.wg, s.changes, s.doneChan, ready, logsPath, s.logger)
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

	// set state to "not running"
	s.running = false
	return nil
}

func start(ctx context.Context, wg *sync.WaitGroup, changes chan agents.Change, doneChan <-chan struct{}, ready *sync.Cond, logsPath string, logger *logrus.Entry) {
	// TODO context usage
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	// signal we started monitoring
	signalReady(ready)

	fr := reader.NewFileReader(logsPath)
	lineChannel := make(chan string)
	ticker := time.NewTicker(1 * time.Minute)
	var s []string

	go func() {
		err := fr.ReadFile(lineChannel)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}()

	for {
		select {
		case line := <-lineChannel:
			s = append(s, line)
		case <-doneChan:
			ticker.Stop()
			return
		case <-ticker.C:
			// TODO convert to metrics buckets
			fmt.Println(s)
			changes <- agents.Change{MetricsBucket: nil}
			s = nil
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
