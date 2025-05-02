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

// Package mongolog runs built-in QAN Agent for MongoDB Mongolog.
package mongolog

import (
	"context"
	"fmt"
	"path"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gopkg.in/mgo.v2/bson"

	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/sender"
	"github.com/percona/pmm/agent/utils/filereader"
	"github.com/percona/pmm/agent/utils/mongo_fix"
)

const (
	mgoTimeoutDialInfo      = 5 * time.Second
	mgoTimeoutSessionSocket = 5 * time.Second
)

// New creates new mongolog
func New(mongoDSN string, logger *logrus.Entry, w sender.Writer, agentID string, logFilePrefix string, maxQueryLength int32) *Mongolog {
	return &Mongolog{
		mongoDSN:       mongoDSN,
		logFilePrefix:  logFilePrefix,
		maxQueryLength: maxQueryLength,
		logger:         logger,
		w:              w,
		agentID:        agentID,
	}
}

// Mongolog represents mongolog agent helpers and properties.
type Mongolog struct {
	// dependencies
	mongoDSN string
	w        sender.Writer
	logger   *logrus.Entry
	agentID  string

	// internal deps
	monitor    *Monitor
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
func (l *Mongolog) Start() error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.running {
		return nil
	}

	// create new channel over which
	// we will tell goroutine it should close
	l.doneChan = make(chan struct{})

	ctx := context.Background()
	labels := pprof.Labels("component", "mongodb.mongolog")

	// create new session
	client, err := createSession(l.mongoDSN, l.agentID)
	if err != nil {
		return err
	}
	logsPath, err := getLogFilePath(client)
	if err != nil {
		return err
	}
	err = client.Disconnect(ctx)
	if err != nil {
		l.logger.Warningln(err)
	}

	// create aggregator which collects documents and aggregates them into qan report
	l.aggregator = aggregator.New(time.Now(), l.agentID, l.logger, l.maxQueryLength)
	reportChan := l.aggregator.Start()

	// create sender which sends qan reports and start it
	l.sender = sender.New(reportChan, l.w, l.logger)
	err = l.sender.Start()
	if err != nil {
		return err
	}

	// create new channel over which
	// we will tell goroutine it should close
	l.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	l.wg = &sync.WaitGroup{}
	l.wg.Add(2)

	// create ready sync.Cond so we could know when goroutine actually started getting data from db
	ready := sync.NewCond(&sync.Mutex{})
	ready.L.Lock()
	defer ready.L.Unlock()

	logsPathWithPrefix := path.Join(l.logFilePrefix, logsPath)
	reader, err := filereader.NewContinuousFileReader(logsPathWithPrefix, l.logger)
	if err != nil {
		return err
	}
	// create monitors service which we use to periodically scan server for new/removed databases
	l.monitor = NewMonitor(logsPathWithPrefix, reader, l.logger)

	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(ctx, l.monitor, l.aggregator, l.wg, l.doneChan, ready, l.logger)
	})

	// wait until we actually fetch data from db
	ready.Wait()

	l.running = true
	return nil
}

// Stop stops running mongolog, waits until it stops.
func (l *Mongolog) Stop() error {
	l.m.Lock()
	defer l.m.Unlock()
	if !l.running {
		return nil
	}

	// notify goroutine to close
	close(l.doneChan)

	// wait for goroutine to exit
	l.wg.Wait()

	l.monitor.reader.Close()
	// set state to "not running"
	l.running = false
	return nil
}

func start(ctx context.Context, monitor *Monitor, aggregator *aggregator.Aggregator, wg *sync.WaitGroup, doneChan <-chan struct{}, ready *sync.Cond, logger *logrus.Entry) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()
	defer monitor.Stop()

	docsChan := make(chan proto.SystemProfile, collectorChanCapacity)
	defer close(docsChan)

	// monitor log file
	err := monitor.Start(ctx, docsChan, doneChan, wg)
	if err != nil {
		logger.Debugf("couldn't monitor log file (%s), reason: %v", monitor.logPath, err)
	}

	// signal we started monitoring
	signalReady(ready)

	for {
		select {
		case <-ctx.Done():
			return
		case doc, ok := <-docsChan:
			if !ok {
				return
			}

			logger.Debugf("added to aggregator %v", doc.Query)
			err := aggregator.Add(ctx, doc)
			if err != nil {
				logger.Warnf("couldn't add document to aggregator: %s", err)
			}
		}
	}
}

func signalReady(ready *sync.Cond) {
	ready.L.Lock()
	defer ready.L.Unlock()
	ready.Broadcast()
}

func createSession(dsn string, agentID string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mgoTimeoutDialInfo)
	defer cancel()

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetSocketTimeout(mgoTimeoutSessionSocket).
		SetAppName(fmt.Sprintf("QAN-mongodb-mongolog-%s", agentID))

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getLogFilePath(client *mongo.Client) (string, error) {
	var result bson.M
	err := client.Database("admin").RunCommand(context.TODO(), bson.M{"getCmdLineOpts": 1}).Decode(&result)
	if err != nil {
		return "", errors.Wrap(err, "failed to run command getCmdLineOpts")
	}

	if parsed, ok := result["parsed"].(bson.M); ok {
		if systemLog, ok := parsed["systemLog"].(bson.M); ok {
			if logPath, ok := systemLog["path"].(string); ok {
				return logPath, nil
			}
		}
	}

	if argv, ok := result["argv"].([]interface{}); ok {
		for i := 0; i < len(argv); i++ {
			if arg, ok := argv[i].(string); ok && arg == "--logpath" && i+1 < len(argv) {
				if value, ok := argv[i+1].(string); ok {
					return value, nil
				}
			}
		}
	}

	return "", errors.New("no log path found, logs may be in Docker stdout")
}
