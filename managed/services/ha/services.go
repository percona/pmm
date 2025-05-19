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
		return fmt.Errorf("service with id %s is already exist", id)
	}
	s.all[id] = service
	select {
	case s.refresh <- struct{}{}:
	default:
	}
	return nil
}

func (s *services) StartAllServices(ctx context.Context) {
	s.rw.Lock()
	defer s.rw.Unlock()

	for id, service := range s.all {
		if _, ok := s.running[id]; !ok {
			s.wg.Add(1)
			s.running[id] = service
			s.l.Infoln("Starting ", service.ID())
			err := service.Start(ctx)
			if err != nil {
				s.l.Errorln(err)
			}
		}
	}
}

func (s *services) StopRunningServices() {
	s.rw.Lock()
	defer s.rw.Unlock()

	for id, service := range s.running {
		s.l.Infoln("Stopping", service.ID())
		service.Stop()
		delete(s.running, id)
		s.wg.Done()
	}
}

func (s *services) Refresh() chan struct{} {
	return s.refresh
}

func (s *services) Wait() {
	s.wg.Wait()
}
