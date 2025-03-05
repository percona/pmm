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
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	MgoTimeoutDialInfo      = 5 * time.Second
	MgoTimeoutSessionSync   = 5 * time.Second
	MgoTimeoutSessionSocket = 5 * time.Second
)

type newMonitor func(client *mongo.Client, logger *logrus.Entry, dbName string) *monitor

func NewMonitors(client *mongo.Client, newMonitor newMonitor, logger *logrus.Entry) *monitors {
	return &monitors{
		client:     client,
		newMonitor: newMonitor,
		monitors:   make(map[string]*monitor),
		logger:     logger,
	}
}

type monitors struct {
	// dependencies
	client     *mongo.Client
	newMonitor newMonitor
	logger     *logrus.Entry

	// monitors
	monitors map[string]*monitor

	// state
	rw sync.RWMutex // Lock() to protect internal consistency of the service
}

func (ms *monitors) MonitorAll(ctx context.Context) error {
	databases := make(map[string]struct{})
	databasesSlice, err := ms.listDatabases()
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
			dbName)
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

func (ms *monitors) StopAll() {
	monitors := ms.GetAll()

	for dbName := range monitors {
		ms.Stop(dbName)
	}
}

func (ms *monitors) Stop(dbName string) {
	m := ms.Get(dbName)
	m.Stop()

	ms.rw.Lock()
	defer ms.rw.Unlock()
	delete(ms.monitors, dbName)
}

func (ms *monitors) Get(dbName string) *monitor {
	ms.rw.RLock()
	defer ms.rw.RUnlock()

	return ms.monitors[dbName]
}

func (ms *monitors) GetAll() map[string]*monitor {
	ms.rw.RLock()
	defer ms.rw.RUnlock()

	list := make(map[string]*monitor)
	for dbName, m := range ms.monitors {
		list[dbName] = m
	}

	return list
}

func (ms *monitors) listDatabases() ([]string, error) {
	return ms.client.ListDatabaseNames(context.TODO(), bson.M{})
}
