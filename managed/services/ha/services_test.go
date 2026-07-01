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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServices(t *testing.T) {
	t.Parallel()

	s := newServices()

	require.NotNil(t, s)
	assert.NotNil(t, s.all)
	assert.NotNil(t, s.running)
	assert.NotNil(t, s.refresh)
	assert.NotNil(t, s.l)
	assert.Empty(t, s.all)
	assert.Empty(t, s.running)
}

func TestServices_Add(t *testing.T) {
	t.Parallel()

	t.Run("add single service succeeds", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("test-service-1")

		err := s.Add(svc)

		require.NoError(t, err)
		assert.Len(t, s.all, 1)
		assert.Equal(t, svc, s.all["test-service-1"])
	})

	t.Run("add duplicate service returns error", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc1 := newMockLeaderService("duplicate-id")
		svc2 := newMockLeaderService("duplicate-id")

		err := s.Add(svc1)
		require.NoError(t, err)

		err = s.Add(svc2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
		assert.Len(t, s.all, 1)
	})

	t.Run("add triggers refresh signal", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("test-service")

		err := s.Add(svc)
		require.NoError(t, err)

		select {
		case <-s.refresh:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("refresh signal not received")
		}
	})

	t.Run("concurrent add operations", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		const numServices = 10
		var wg sync.WaitGroup

		for i := range numServices {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				svc := newMockLeaderService(fmt.Sprintf("svc-%d", id))
				_ = s.Add(svc)
			}(i)
		}

		wg.Wait()
		assert.Len(t, s.all, numServices)
	})
}

func TestServices_StartAllServices(t *testing.T) {
	t.Parallel()

	t.Run("starts only non-running services", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc1 := newMockLeaderService("service-1")
		svc2 := newMockLeaderService("service-2")

		require.NoError(t, s.Add(svc1))
		require.NoError(t, s.Add(svc2))

		ctx := t.Context()
		s.StartAllServices(ctx)

		time.Sleep(50 * time.Millisecond)

		assert.True(t, svc1.isStarted())
		assert.True(t, svc2.isStarted())

		s.rw.Lock()
		runningCount := len(s.running)
		s.rw.Unlock()
		assert.Equal(t, 2, runningCount)
	})

	t.Run("marks services as running", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("test-service")

		require.NoError(t, s.Add(svc))

		ctx := t.Context()
		s.StartAllServices(ctx)

		time.Sleep(50 * time.Millisecond)

		s.rw.Lock()
		runningCount := len(s.running)
		_, exists := s.running["test-service"]
		s.rw.Unlock()

		assert.Equal(t, 1, runningCount)
		assert.True(t, exists)
	})

	t.Run("handles service start errors", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("failing-service")
		svc.startErr = errors.New("start failed")
		svc.startDone = make(chan struct{})

		require.NoError(t, s.Add(svc))

		ctx := t.Context()
		s.StartAllServices(ctx)

		select {
		case <-svc.startDone:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("service start did not complete")
		}

		assert.True(t, svc.isStarted())

		// The failing service removes itself from the running map.
		assert.Eventually(t, func() bool {
			s.rw.Lock()
			defer s.rw.Unlock()
			return len(s.running) == 0
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("does not restart already running services", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("test-service")

		require.NoError(t, s.Add(svc))

		ctx := t.Context()
		s.StartAllServices(ctx)

		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 1, svc.getStartCount())

		s.StartAllServices(ctx)
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 1, svc.getStartCount())
	})
}

func TestServices_StopAllServices(t *testing.T) {
	t.Parallel()

	t.Run("stops all running services", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc1 := newMockLeaderService("service-1")
		svc2 := newMockLeaderService("service-2")

		require.NoError(t, s.Add(svc1))
		require.NoError(t, s.Add(svc2))

		ctx := t.Context()
		s.StartAllServices(ctx)

		time.Sleep(50 * time.Millisecond)

		s.StopAllServices()

		assert.True(t, svc1.isStopped())
		assert.True(t, svc2.isStopped())
	})

	t.Run("removes services from running map", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("test-service")

		require.NoError(t, s.Add(svc))

		ctx := t.Context()
		s.StartAllServices(ctx)

		time.Sleep(50 * time.Millisecond)

		s.rw.Lock()
		runningCount := len(s.running)
		s.rw.Unlock()
		assert.Equal(t, 1, runningCount)

		s.StopAllServices()

		// Each service's goroutine removes itself once Start returns.
		assert.Eventually(t, func() bool {
			s.rw.Lock()
			defer s.rw.Unlock()
			return len(s.running) == 0
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("handles stopping with no running services", func(t *testing.T) {
		t.Parallel()

		s := newServices()

		assert.NotPanics(t, func() {
			s.StopAllServices()
		})

		s.rw.Lock()
		isEmpty := len(s.running) == 0
		s.rw.Unlock()
		assert.True(t, isEmpty)
	})
}

func TestServices_Refresh(t *testing.T) {
	t.Parallel()

	t.Run("returns valid channel", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		ch := s.Refresh()

		require.NotNil(t, ch)
	})

	t.Run("channel receives signals on add", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		ch := s.Refresh()

		svc := newMockLeaderService("test-service")
		err := s.Add(svc)
		require.NoError(t, err)

		select {
		case <-ch:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("refresh signal not received")
		}
	})
}

func TestServices_Wait(t *testing.T) {
	t.Parallel()

	t.Run("waits for all services to complete", func(t *testing.T) {
		t.Parallel()

		s := newServices()
		svc := newMockLeaderService("blocking-service")

		require.NoError(t, s.Add(svc))

		ctx := t.Context()
		s.StartAllServices(ctx)

		time.Sleep(50 * time.Millisecond)

		done := make(chan struct{})
		go func() {
			s.Wait()
			close(done)
		}()

		select {
		case <-done:
			t.Fatal("Wait returned before service completed")
		case <-time.After(50 * time.Millisecond):
		}

		s.StopAllServices()

		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Wait did not return after service completed")
		}
	})
}

// mockLeaderService models a leader service whose Start blocks until the service
// is stopped (or its context is cancelled), mirroring the real ContextService.
type mockLeaderService struct {
	id        string
	startErr  error
	startDone chan struct{}

	mu         sync.Mutex
	started    bool
	stopped    bool
	startCount int
	stopCh     chan struct{}
}

func newMockLeaderService(id string) *mockLeaderService {
	return &mockLeaderService{
		id:     id,
		stopCh: make(chan struct{}),
	}
}

func (m *mockLeaderService) ID() string {
	return m.id
}

func (m *mockLeaderService) Start(ctx context.Context) error {
	m.mu.Lock()
	m.started = true
	m.startCount++
	err := m.startErr
	startDone := m.startDone
	m.mu.Unlock()

	if startDone != nil {
		close(startDone)
	}

	if err != nil {
		return err
	}

	select {
	case <-m.stopCh:
	case <-ctx.Done():
	}

	return nil
}

func (m *mockLeaderService) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return
	}
	m.stopped = true
	close(m.stopCh)
}

func (m *mockLeaderService) isStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *mockLeaderService) isStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func (m *mockLeaderService) getStartCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCount
}
