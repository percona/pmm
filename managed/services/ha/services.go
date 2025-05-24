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
			ls := service
			go func() {
				s.l.Infoln("Starting ", ls.ID())
				err := ls.Start(ctx)
				if err != nil {
					s.l.Errorln(err)
				}
			}()
		}
	}
}

func (s *services) StopRunningServices() {
	s.rw.Lock()
	defer s.rw.Unlock()

	for id, service := range s.running {
		id := id
		ls := service
		go func() {
			defer s.wg.Done()
			s.l.Infoln("Stopping", ls.ID())
			ls.Stop()
			s.rw.Lock()
			defer s.rw.Unlock()
			delete(s.running, id)
		}()
	}
}

func (s *services) Refresh() chan struct{} {
	return s.refresh
}

func (s *services) Wait() {
	s.wg.Wait()
}
