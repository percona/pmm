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
	"log"
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
		monitors:   map[string]*monitor{},
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
	sync.RWMutex // Lock() to protect internal consistency of the service
}

func (ms *monitors) MonitorAll() error {
	databases := map[string]struct{}{}
	databasesSlice, err := ms.listDatabases()
	if err != nil {
		return err
	}
	for _, dbName := range databasesSlice {
		// Skip admin and local databases to avoid collecting queries from replication and mongodb_exporter
		//switch dbName {
		//case "admin", "local":
		//	continue
		//default:
		//}

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
		err := m.Start()
		if err != nil {
			log.Println(err)
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

	ms.Lock()
	defer ms.Unlock()
	delete(ms.monitors, dbName)
}

func (ms *monitors) Get(dbName string) *monitor {
	ms.RLock()
	defer ms.RUnlock()

	return ms.monitors[dbName]
}

func (ms *monitors) GetAll() map[string]*monitor {
	ms.RLock()
	defer ms.RUnlock()

	list := map[string]*monitor{}
	for dbName, m := range ms.monitors {
		list[dbName] = m
	}

	return list
}

func (ms *monitors) listDatabases() ([]string, error) {
	return ms.client.ListDatabaseNames(context.TODO(), bson.M{})
}
