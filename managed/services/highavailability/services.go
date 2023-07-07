package highavailability

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type services struct {
	wg      *sync.WaitGroup
	rw      sync.Mutex
	all     map[string]LeaderService
	running map[string]LeaderService

	serviceAdded chan struct{}

	l *logrus.Entry
}

func newServices() *services {
	return &services{
		wg:           new(sync.WaitGroup),
		all:          make(map[string]LeaderService),
		running:      make(map[string]LeaderService),
		serviceAdded: make(chan struct{}),
		l:            logrus.WithField("component", "ha-services"),
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
	case s.serviceAdded <- struct{}{}:
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
			s.l.Infoln("Stopping ", ls)
			ls.Stop()
			s.rw.Lock()
			defer s.rw.Unlock()
			delete(s.running, id)
		}()
	}
}

func (s *services) ServiceAdded() chan struct{} {
	return s.serviceAdded
}

func (s *services) Wait() {
	s.wg.Wait()
}
