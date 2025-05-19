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
	"sync"
)

// LeaderService represents a leader service in the high-availability setup.
type LeaderService interface {
	Start(ctx context.Context) error
	Stop()
	ID() string
}

// StandardService represents a standard service in the high-availability setup.
type StandardService struct {
	id string

	mu sync.Mutex

	startFunc func(context.Context) error
	stopFunc  func()
}

// NewStandardService creates a new standard service.
func NewStandardService(id string, startFunc func(context.Context) error, stopFunc func()) *StandardService {
	return &StandardService{
		id:        id,
		startFunc: startFunc,
		stopFunc:  stopFunc,
	}
}

// ID returns the ID of the standard service.
func (s *StandardService) ID() string {
	return s.id
}

// Start starts the standard service.
func (s *StandardService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.startFunc(ctx)
	return err
}

// Stop stops the standard service.
func (s *StandardService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopFunc()
}

// ContextService represents a context service.
type ContextService struct {
	id string

	startFunc func(context.Context) error

	m      sync.Mutex
	cancel context.CancelFunc
}

// NewContextService creates a new context service.
func NewContextService(id string, startFunc func(context.Context) error) *ContextService {
	return &ContextService{
		id:        id,
		startFunc: startFunc,
	}
}

// ID returns the ID of the context service.
func (s *ContextService) ID() string {
	return s.id
}

// Start starts the context service.
func (s *ContextService) Start(ctx context.Context) error {
	s.m.Lock()
	ctx, s.cancel = context.WithCancel(ctx)
	s.m.Unlock()
	return s.startFunc(ctx)
}

// Stop stops the context service.
func (s *ContextService) Stop() {
	s.m.Lock()
	defer s.m.Unlock()
	s.cancel()
}
