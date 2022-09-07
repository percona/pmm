// Copyright (C) 2017 Percona LLC
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

package dbaas

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
)

// Initializer handles enabling/disabling DBaaS logic.
type Initializer struct {
	db *reform.DB
	l  *logrus.Entry

	client                 dbaasClient
	dbClustersSynchronizer *DBClustersSynchronizer

	enabled bool
	cancel  func()
	m       sync.Mutex
}

// NewInitializer returns new object of Initializer type.
func NewInitializer(db *reform.DB, client dbaasClient, dbClustersSynchronizer *DBClustersSynchronizer) *Initializer {
	l := logrus.WithField("component", "dbaas_initializer")
	return &Initializer{
		db:                     db,
		l:                      l,
		client:                 client,
		dbClustersSynchronizer: dbClustersSynchronizer,
	}
}

// Enable enables DBaaS feature and runs everything needed.
func (in *Initializer) Enable(ctx context.Context) error {
	in.m.Lock()
	defer in.m.Unlock()
	if in.enabled {
		return nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	err := in.client.Connect(timeoutCtx)
	cancel()
	if err != nil {
		return err
	}
	ctx, in.cancel = context.WithCancel(ctx)
	go in.dbClustersSynchronizer.Run(ctx)
	in.enabled = true
	return nil
}

// Disable disables DBaaS feature and stops everything needed.
func (in *Initializer) Disable(ctx context.Context) error {
	in.m.Lock()
	defer in.m.Unlock()
	if !in.enabled { // Don't disable if already disabled
		return nil
	}
	if in.cancel != nil {
		in.cancel()
	}
	err := in.client.Disconnect()
	if err != nil {
		return err
	}
	in.enabled = false
	return nil
}
