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

// Package server implements pmm-managed Server API.
package server

import (
	"sync"
	"time"
)

var sideContainerUpdateStatus = &sideContainerUpdate{}

type sideContainerUpdate struct {
	// Last time pmm-updater checked for status.
	lastUpdaterCheck time.Time
	checkMu          sync.RWMutex

	// Time when last update was requested.
	requestedOn time.Time
	reqMu       sync.RWMutex
}

// updateLastUpdaterCheck updates last time pmm-updater checked for status.
func (s *sideContainerUpdate) updateLastUpdaterCheck() {
	s.checkMu.Lock()
	defer s.checkMu.Unlock()

	s.lastUpdaterCheck = time.Now()
}

func (s *sideContainerUpdate) getLastUpdaterCheck() time.Time {
	s.checkMu.RLock()
	defer s.checkMu.RUnlock()

	return s.lastUpdaterCheck
}

func (s *sideContainerUpdate) requestUpdate() {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()

	s.requestedOn = time.Now()
}

func (s *sideContainerUpdate) isRequested() bool {
	s.reqMu.RLock()
	defer s.reqMu.RUnlock()

	return time.Since(s.requestedOn) < 2*time.Hour
}
