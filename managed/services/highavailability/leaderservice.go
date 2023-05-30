package highavailability

import (
	"context"
	"sync"
)

type LeaderService interface {
	Start(ctx context.Context) error
	Stop()
	ID() string
}

type ContextService struct {
	id string

	startFunc func(context.Context) error
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewContextService(id string, startFunc func(context.Context) error) *ContextService {
	return &ContextService{
		startFunc: startFunc,
	}
}

func (s *ContextService) ID() string {
	return s.id
}

func (s *ContextService) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	return s.startFunc(ctx)
}

func (s *ContextService) Stop() {
	s.cancel()
}

type RunOnceService struct {
	id string

	startFunc func(context.Context) error
	o         sync.Once
}

func NewRunOnceService(id string, startFunc func(context.Context) error) *RunOnceService {
	return &RunOnceService{
		startFunc: startFunc,
	}
}

func (s *RunOnceService) ID() string {
	return s.id
}

func (s *RunOnceService) Start(ctx context.Context) error {
	s.o.Do(func() {
		s.startFunc(ctx)
	})
	return nil
}
func (s *RunOnceService) Stop() {}
