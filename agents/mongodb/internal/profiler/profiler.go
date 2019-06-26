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

package profiler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/aggregator"
	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/sender"
)

// New creates new Profiler
func New(mongoDSN string, logger *logrus.Entry, w sender.Writer, agentID string) *profiler {
	return &profiler{
		mongoDSN: mongoDSN,
		logger:   logger,
		w:        w,
		agentID:  agentID,
	}
}

type profiler struct {
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
	sync.RWMutex                 // Lock() to protect internal consistency of the service
	running      bool            // Is this service running?
	doneChan     chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg           *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts analyzer but doesn't wait until it exits
func (p *profiler) Start() error {
	p.Lock()
	defer p.Unlock()
	if p.running {
		return nil
	}

	// create new session
	client, err := createSession(p.mongoDSN)
	if err != nil {
		return err
	}
	p.client = client

	// create aggregator which collects documents and aggregates them into qan report
	p.aggregator = aggregator.New(time.Now(), p.agentID)
	reportChan := p.aggregator.Start()

	// create sender which sends qan reports and start it
	p.sender = sender.New(reportChan, p.w, p.logger)
	err = p.sender.Start()
	if err != nil {
		return err
	}

	f := func(client *mongo.Client, logger *logrus.Entry, dbName string) *monitor {
		return NewMonitor(client, dbName, p.aggregator, logger)
	}

	// create monitors service which we use to periodically scan server for new/removed databases
	p.monitors = NewMonitors(client, f, p.logger)

	// create new channel over which
	// we will tell goroutine it should close
	p.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	p.wg = &sync.WaitGroup{}
	p.wg.Add(1)

	// create ready sync.Cond so we could know when goroutine actually started getting data from db
	ready := sync.NewCond(&sync.Mutex{})
	ready.L.Lock()
	defer ready.L.Unlock()

	go start(p.monitors, p.wg, p.doneChan, ready)

	// wait until we actually fetch data from db
	ready.Wait()

	p.running = true
	return nil
}

// Status returns list of statuses
func (p *profiler) Status() map[string]string {
	p.RLock()
	defer p.RUnlock()
	if !p.running {
		return nil
	}

	statuses := &sync.Map{}
	monitors := p.monitors.GetAll()

	wg := &sync.WaitGroup{}
	wg.Add(len(monitors))
	for dbName, m := range monitors {
		go func(dbName string, m *monitor) {
			defer wg.Done()
			for k, v := range m.Status() {
				key := fmt.Sprintf("%s-%s", k, dbName)
				statuses.Store(key, v)
			}
		}(dbName, m)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for k, v := range p.aggregator.Status() {
			key := fmt.Sprintf("%s-%s", "aggregator", k)
			statuses.Store(key, v)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for k, v := range p.sender.Status() {
			key := fmt.Sprintf("%s-%s", "sender", k)
			statuses.Store(key, v)
		}
	}()

	wg.Wait()

	statusesMap := map[string]string{}
	statuses.Range(func(key, value interface{}) bool {
		statusesMap[key.(string)] = value.(string)
		return true
	})
	return statusesMap
}

// Stop stops running analyzer, waits until it stops
func (p *profiler) Stop() error {
	p.Lock()
	defer p.Unlock()
	if !p.running {
		return nil
	}

	// notify goroutine to close
	close(p.doneChan)

	// wait for goroutine to exit
	p.wg.Wait()

	// stop aggregator; do it after goroutine is closed
	p.aggregator.Stop()

	// stop sender; do it after goroutine is closed
	p.sender.Stop()

	// close the session; do it after goroutine is closed
	p.client.Disconnect(context.TODO()) //nolint:errcheck

	// set state to "not running"
	p.running = false
	return nil
}

func start(monitors *monitors, wg *sync.WaitGroup, doneChan <-chan struct{}, ready *sync.Cond) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	// stop all monitors
	defer monitors.StopAll()

	// monitor all databases
	monitors.MonitorAll()

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
		monitors.MonitorAll()
	}
}

func signalReady(ready *sync.Cond) {
	ready.L.Lock()
	defer ready.L.Unlock()
	ready.Broadcast()
}

func createSession(dsn string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), MgoTimeoutDialInfo)
	defer cancel()
	opts := options.Client().
		ApplyURI(dsn).
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetSocketTimeout(MgoTimeoutSessionSocket)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}
