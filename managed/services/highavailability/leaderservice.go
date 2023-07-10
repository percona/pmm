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

package highavailability

import (
	"context"
	"sync"
)

type LeaderService interface {
	Start(ctx context.Context) error
	Stop() error
	ID() string
}

type StandardService struct {
	id string

	startFunc func(context.Context) error
	stopFunc  func() error
}

func NewStandardService(id string, startFunc func(context.Context) error, stopFunc func() error) *StandardService {
	return &StandardService{
		id:        id,
		startFunc: startFunc,
		stopFunc:  stopFunc,
	}
}

func (s *StandardService) ID() string {
	return s.id
}

func (s *StandardService) Start(ctx context.Context) error {
	return s.startFunc(ctx)
}

func (s *StandardService) Stop() error {
	return s.stopFunc()
}

type ContextService struct {
	id string

	startFunc func(context.Context) error
	ctx       context.Context
	cancel    context.CancelFunc

	m sync.Mutex
}

func NewContextService(id string, startFunc func(context.Context) error) *ContextService {
	return &ContextService{
		id:        id,
		startFunc: startFunc,
	}
}

func (s *ContextService) ID() string {
	return s.id
}

func (s *ContextService) Start(ctx context.Context) error {
	s.m.Lock()
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.m.Unlock()
	return s.startFunc(ctx)
}

func (s *ContextService) Stop() error {
	s.m.Lock()
	defer s.m.Unlock()
	s.cancel()
	return nil
}

type RunOnceService struct {
	id string

	startFunc func(context.Context) error
	o         sync.Once
}

func NewRunOnceService(id string, startFunc func(context.Context) error) *RunOnceService {
	return &RunOnceService{
		id:        id,
		startFunc: startFunc,
	}
}

func (s *RunOnceService) ID() string {
	return s.id
}

func (s *RunOnceService) Start(ctx context.Context) error {
	var err error
	s.o.Do(func() {
		err = s.startFunc(ctx)
	})
	return err
}

func (s *RunOnceService) Stop() error {
	return nil
}
