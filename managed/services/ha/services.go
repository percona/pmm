// Copyright (C) 2023 Percona LLC
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

package ha

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type services struct {
	wg sync.WaitGroup

	rw      sync.Mutex
	all     map[string]LeaderService
	running map[string]LeaderService

	refresh chan struct{}

	l *logrus.Entry
}

// newServices creates a new services manager.
func newServices() *services {
	return &services{
		all:     make(map[string]LeaderService),
		running: make(map[string]LeaderService),
		refresh: make(chan struct{}, 1),
		l:       logrus.WithField("component", "ha-services"),
	}
}

// Add registers a new leader service.
func (s *services) Add(service LeaderService) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	id := service.ID()
	if _, ok := s.all[id]; ok {
		return fmt.Errorf("service with id %s already exists", id)
	}
	s.all[id] = service
	select {
	case s.refresh <- struct{}{}:
	default:
	}
	return nil
}

// StartAllServices starts all registered services that are not currently running.
func (s *services) StartAllServices(ctx context.Context) {
	s.rw.Lock()
	defer s.rw.Unlock()

	for id, service := range s.all {
		if _, ok := s.running[id]; ok {
			continue
		}
		s.running[id] = service

		svc, svcID := service, id
		s.wg.Go(func() {
			s.l.Infoln("Starting", svcID)
			err := svc.Start(ctx)
			if err != nil {
				s.l.Errorln(err)
			}
			// Remove the service only once it has actually stopped, so the
			// WaitGroup is balanced by the same goroutine that owns it.
			s.rw.Lock()
			delete(s.running, svcID)
			s.rw.Unlock()
		})
	}
}

// StopAllServices signals all running services to stop. Each service's goroutine
// removes itself from the running set once Start returns.
func (s *services) StopAllServices() {
	s.rw.Lock()
	defer s.rw.Unlock()

	for _, service := range s.running {
		s.l.Infoln("Stopping", service.ID())
		service.Stop()
	}
}

// Refresh returns a channel that signals when services should be refreshed.
func (s *services) Refresh() chan struct{} {
	return s.refresh
}

// Wait waits for all services to stop.
func (s *services) Wait() {
	s.wg.Wait()
}
