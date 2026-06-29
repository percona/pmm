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

package profiler

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	mgoTimeoutDialInfo      = 5 * time.Second
	mgoTimeoutSessionSync   = 5 * time.Second
	mgoTimeoutSessionSocket = 5 * time.Second
)

type newMonitor func(client *mongo.Client, logger *logrus.Entry, dbName string) *Monitor

// NewMonitors creates a new Monitors instance that manages the lifecycle of individual
// database profilers for a MongoDB instance.
func NewMonitors(client *mongo.Client, newMonitor newMonitor, logger *logrus.Entry) *Monitors {
	return &Monitors{
		client:     client,
		newMonitor: newMonitor,
		monitors:   make(map[string]*Monitor),
		logger:     logger,
	}
}

// Monitors manages the lifecycle of individual database monitors for a MongoDB instance.
// It tracks which databases should be monitored by periodically reconciling the list
// of existing databases with active monitor instances. It ensures thread-safe access
// to the collection of monitors.
type Monitors struct {
	// dependencies
	client     *mongo.Client
	newMonitor newMonitor
	logger     *logrus.Entry

	// monitors
	monitors map[string]*Monitor

	// state
	rw sync.RWMutex // Lock() to protect internal consistency of the service
}

// MonitorAll reconciles the list of currently active database monitors with the databases
// existing on the MongoDB server. It starts new monitors for newly discovered databases
// and stops monitors for databases that have been dropped.
func (ms *Monitors) MonitorAll(ctx context.Context) error {
	databases := make(map[string]struct{})
	databasesSlice, err := ms.listDatabases(ctx)
	if err != nil {
		return err
	}
	for _, dbName := range databasesSlice {
		// Skip admin and local databases to avoid collecting queries from replication and mongodb_exporter
		// switch dbName {
		// case "admin", "local":
		//	continue
		// default:
		// }

		// change slice to map for easier lookup
		databases[dbName] = struct{}{}

		// if database is already monitored then nothing to do, skip it
		if _, ok := ms.monitors[dbName]; ok {
			continue
		}

		// if database is not monitored yet then we need to create new monitor
		m := ms.newMonitor(
			ms.client,
			ms.logger,
			dbName,
		)
		// ... and start it
		err := m.Start(ctx)
		if err != nil {
			ms.logger.Debugf("couldn't start monitor, reason: %v", err)
			return err
		}

		// add new monitor to list of monitored databases
		ms.monitors[dbName] = m
	}

	// if database is no longer present then stop monitoring it
	for dbName := range ms.monitors {
		if _, ok := databases[dbName]; !ok {
			ms.monitors[dbName].Stop()
			delete(ms.monitors, dbName)
		}
	}

	return nil
}

// StopAll gracefully stops all currently active database monitors and clears the registry.
func (ms *Monitors) StopAll() {
	monitors := ms.GetAll()

	for dbName := range monitors {
		ms.Stop(dbName)
	}
}

// Stop stops the monitor for a specific database and removes it from the internal registry.
func (ms *Monitors) Stop(dbName string) {
	m := ms.Get(dbName)
	m.Stop()

	ms.rw.Lock()
	defer ms.rw.Unlock()
	delete(ms.monitors, dbName)
}

// Get returns the monitor instance for the specified database name.
// Returns nil if the database is not being monitored.
func (ms *Monitors) Get(dbName string) *Monitor {
	ms.rw.RLock()
	defer ms.rw.RUnlock()

	return ms.monitors[dbName]
}

// GetAll returns a thread-safe copy of the map containing all active database monitors,
// where keys are database names.
func (ms *Monitors) GetAll() map[string]*Monitor {
	ms.rw.RLock()
	defer ms.rw.RUnlock()

	list := make(map[string]*Monitor)
	maps.Copy(list, ms.monitors)

	return list
}

func (ms *Monitors) listDatabases(ctx context.Context) ([]string, error) {
	return ms.client.ListDatabaseNames(ctx, bson.M{})
}
