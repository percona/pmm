package highavailability

import (
	"context"
	"fmt"
	"sync"
)

type services struct {
	wg      *sync.WaitGroup
	rw      sync.Mutex
	all     map[string]LeaderService
	running map[string]LeaderService

	serviceAdded chan struct{}
}

func newServices() *services {
	return &services{
		wg:           new(sync.WaitGroup),
		all:          make(map[string]LeaderService),
		running:      make(map[string]LeaderService),
		serviceAdded: make(chan struct{}),
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
			go func(ls LeaderService) {
				defer s.wg.Done()
				ls.Start(ctx)
				s.rw.Lock()
				defer s.rw.Unlock()
				delete(s.running, id)
			}(service)
			s.running[id] = service
		}
	}
}

func (s *services) StartService(ctx context.Context, id string) error {
	s.rw.Lock()
	defer s.rw.Unlock()
	service, ok := s.all[id]
	if !ok {
		return fmt.Errorf("service with ID %s not found", id)
	}
	if _, ok := s.running[id]; !ok {
		go service.Start(ctx)
		s.running[id] = service
	}
	return nil
}

func (s *services) StopRunningServices() {
	s.rw.Lock()
	defer s.rw.Unlock()

	for _, service := range s.running {
		go service.Stop()
	}
}

func (s *services) ServiceAdded() chan struct{} {
	return s.serviceAdded
}

func (s *services) Wait() {
	s.wg.Wait()
}
