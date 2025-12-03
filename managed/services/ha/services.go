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

func newServices() *services {
	return &services{
		all:     make(map[string]LeaderService),
		running: make(map[string]LeaderService),
		refresh: make(chan struct{}, 1),
		l:       logrus.WithField("component", "ha-services"),
	}
}

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

func (s *services) StartAllServices(ctx context.Context) {
	type startItem struct {
		svc LeaderService
		id  string
	}

	s.rw.Lock()
	toStart := make([]startItem, 0, len(s.all))
	for id, service := range s.all {
		if _, ok := s.running[id]; !ok {
			s.running[id] = service
			toStart = append(toStart, startItem{svc: service, id: id})
		}
	}
	s.rw.Unlock()

	for _, service := range toStart {
		s.wg.Add(1)
		go func(svc LeaderService, svcID string) {
			s.l.Infoln("Starting", svcID)
			err := svc.Start(ctx)
			if err != nil {
				s.l.Errorln(err)
				s.removeService(svcID)
			}
		}(service.svc, service.id)
	}
}

func (s *services) StopRunningServices() {
	s.rw.Lock()
	toStop := make([]LeaderService, 0, len(s.running))
	for id, service := range s.running {
		toStop = append(toStop, service)
		delete(s.running, id)
	}
	s.rw.Unlock()

	for _, service := range toStop {
		s.l.Infoln("Stopping", service.ID())
		service.Stop()
		s.wg.Done()
	}
}

func (s *services) Refresh() chan struct{} {
	return s.refresh
}

func (s *services) Wait() {
	s.wg.Wait()
}

func (s *services) removeService(id string) {
	s.rw.Lock()
	delete(s.running, id)
	s.rw.Unlock()
	s.wg.Done()
}
