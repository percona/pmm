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

package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/percona/pmm-agent/agents/mongodb/internal/status"
)

const (
	MgoTimeoutTail = 1 * time.Second
)

// New creates new Collector.
func New(client *mongo.Client, dbName string, logger *logrus.Entry) *Collector {
	return &Collector{
		client: client,
		dbName: dbName,
		logger: logger,
	}
}

type Collector struct {
	// dependencies
	client *mongo.Client
	dbName string
	logger *logrus.Entry

	// provides
	docsChan chan proto.SystemProfile

	// status
	status *status.Status

	// state
	sync.RWMutex                 // Lock() to protect internal consistency of the service
	running      bool            // Is this service running?
	doneChan     chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg           *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts but doesn't wait until it exits
func (self *Collector) Start() (<-chan proto.SystemProfile, error) {
	self.Lock()
	defer self.Unlock()
	if self.running {
		return nil, nil
	}

	// create new channels over which we will communicate to...
	// ... outside world by sending collected docs
	self.docsChan = make(chan proto.SystemProfile, 100)
	// ... inside goroutine to close it
	self.doneChan = make(chan struct{})

	// set status
	stats := &stats{}
	self.status = status.New(stats)

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	self.wg = &sync.WaitGroup{}
	self.wg.Add(1)

	// create ready sync.Cond so we could know when goroutine actually started getting data from db
	ready := sync.NewCond(&sync.Mutex{})
	ready.L.Lock()
	defer ready.L.Unlock()

	go start(
		self.wg,
		self.client,
		self.dbName,
		self.docsChan,
		self.doneChan,
		stats,
		ready,
		self.logger,
	)

	// wait until we actually fetch data from db
	ready.Wait()

	self.running = true
	return self.docsChan, nil
}

// Stop stops running
func (self *Collector) Stop() {
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

	// we can now safely close channels goroutines write to as goroutine is stopped
	close(self.docsChan)
	return
}

func (self *Collector) Status() map[string]string {
	self.RLock()
	defer self.RUnlock()
	if !self.running {
		return nil
	}

	s := self.status.Map()
	s["profile"] = getProfile(context.TODO(), self.client, self.dbName)

	return s
}

func getProfile(ctx context.Context, client *mongo.Client, dbName string) string {
	result := struct {
		Was       int
		Slowms    int
		Ratelimit int
	}{}
	err := client.Database(dbName).RunCommand(ctx, bson.M{"profile": -1}).Decode(&result)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}

	if result.Was == 0 {
		return "Profiling disabled. Please enable profiling for this database or whole MongoDB server (https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/)."
	}

	if result.Was == 1 {
		return fmt.Sprintf("Profiling enabled for slow queries only (slowms: %d)", result.Slowms)
	}

	if result.Was == 2 {
		// if result.Ratelimit == 0 we assume ratelimit is not supported
		// so all queries have ratelimit = 1 (log all queries)
		if result.Ratelimit == 0 {
			result.Ratelimit = 1
		}
		return fmt.Sprintf("Profiling enabled for all queries (ratelimit: %d)", result.Ratelimit)
	}
	return fmt.Sprintf("Unknown profiling state: %d", result.Was)
}

func (self *Collector) Name() string {
	return "collector"
}

func start(wg *sync.WaitGroup, client *mongo.Client, dbName string, docsChan chan<- proto.SystemProfile, doneChan <-chan struct{}, stats *stats, ready *sync.Cond, logger *logrus.Entry) { //nolint: lll
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	firstTry := true
	for {
		// make a connection and collect data
		connectAndCollect(
			client,
			dbName,
			docsChan,
			doneChan,
			stats,
			ready,
			logger,
		)

		select {
		// check if we should shutdown
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

func connectAndCollect(client *mongo.Client, dbName string, docsChan chan<- proto.SystemProfile, doneChan <-chan struct{}, stats *stats, ready *sync.Cond, logger *logrus.Entry) { //nolint: lll
	collection := client.Database(dbName).Collection("system.profile")
	query := createQuery(dbName)

	timeoutCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	cursor, err := createIterator(timeoutCtx, collection, query)
	if err != nil {
		logger.Errorf("couldn't create system.profile iterator, reason: %v", err)
		return
	}
	// do not cancel cursor closing when ctx is canceled
	defer cursor.Close(context.Background()) //nolint:errcheck

	stats.IteratorCreated = time.Now().UTC().Format("2006-01-02 15:04:05")
	stats.IteratorCounter += 1

	// we got iterator, we are ready
	signalReady(ready)

	for {
		// check if we should shutdown
		select {
		case <-doneChan:
			return
		default:
			// just continue if not
		}
		for cursor.Next(timeoutCtx) {
			doc := proto.SystemProfile{}
			e := cursor.Decode(&doc)
			if e != nil {
				logger.Error(e)
				stats.IteratorErrCounter++
				stats.IteratorErrLast = e.Error()
				continue
			}

			stats.In += 1

			// check if we should shutdown
			select {
			case <-doneChan:
				return
			default:
				// just continue if not
			}

			// try to push doc
			select {
			case docsChan <- doc:
				stats.Out += 1
			// or exit if we can't push the doc and we should shutdown
			// note that if we can push the doc then exiting is not guaranteed
			// that's why we have separate `select <-doneChan` above
			case <-doneChan:
				return
			}
		}
		if err := cursor.Err(); err != nil {
			stats.IteratorErrCounter++
			stats.IteratorErrLast = err.Error()
			return
		}

		// If Next() is false it means iterator is no longer valid
		// and the query needs to be restarted.
		stats.IteratorRestartCounter++
		return
	}
}

func createQuery(dbName string) bson.M {
	return bson.M{
		"ns": bson.M{"$ne": dbName + ".system.profile"},
		"ts": bson.M{"$gt": time.Now()},
	}
}

func createIterator(ctx context.Context, collection *mongo.Collection, query bson.M) (*mongo.Cursor, error) {
	opts := options.Find().SetSort(bson.M{"$natural": 1}).SetCursorType(options.Tailable)
	return collection.Find(ctx, query, opts)
}

func signalReady(ready *sync.Cond) {
	ready.L.Lock()
	defer ready.L.Unlock()
	ready.Broadcast()
}
