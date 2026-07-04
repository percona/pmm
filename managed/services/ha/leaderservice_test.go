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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardService_New(t *testing.T) {
	t.Parallel()

	startFunc := func(_ context.Context) error { return nil }
	stopFunc := func() {}

	svc := NewStandardService("test-id", startFunc, stopFunc)

	require.NotNil(t, svc)
	assert.Equal(t, "test-id", svc.id)
	assert.NotNil(t, svc.startFunc)
	assert.NotNil(t, svc.stopFunc)
}

func TestStandardService_ID(t *testing.T) {
	t.Parallel()

	svc := NewStandardService("my-service", nil, nil)

	assert.Equal(t, "my-service", svc.ID())
}

func TestStandardService_Start(t *testing.T) {
	t.Parallel()

	t.Run("calls startFunc successfully", func(t *testing.T) {
		t.Parallel()

		called := false
		startFunc := func(_ context.Context) error {
			called = true
			return nil
		}
		stopFunc := func() {}

		svc := NewStandardService("test", startFunc, stopFunc)
		err := svc.Start(t.Context())

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns error from startFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("start failed")
		startFunc := func(_ context.Context) error {
			return expectedErr
		}
		stopFunc := func() {}

		svc := NewStandardService("test", startFunc, stopFunc)
		err := svc.Start(t.Context())

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("passes context to startFunc", func(t *testing.T) {
		t.Parallel()

		var receivedCtx context.Context
		startFunc := func(ctx context.Context) error {
			receivedCtx = ctx //nolint:fatcontext
			return nil
		}
		stopFunc := func() {}

		type ctxKey string
		svc := NewStandardService("test", startFunc, stopFunc)
		ctx := context.WithValue(t.Context(), ctxKey("key"), "value")
		err := svc.Start(ctx)

		require.NoError(t, err)
		assert.Equal(t, "value", receivedCtx.Value(ctxKey("key")))
	})

	t.Run("handles concurrent start calls safely", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		var mu sync.Mutex
		startFunc := func(_ context.Context) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return nil
		}
		stopFunc := func() {}

		svc := NewStandardService("test", startFunc, stopFunc)

		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				_ = svc.Start(t.Context())
			})
		}

		wg.Wait()

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 5, callCount)
	})
}

func TestStandardService_Stop(t *testing.T) {
	t.Parallel()

	t.Run("calls stopFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		startFunc := func(_ context.Context) error { return nil }
		stopFunc := func() {
			called = true
		}

		svc := NewStandardService("test", startFunc, stopFunc)
		svc.Stop()

		assert.True(t, called)
	})

	t.Run("handles concurrent stop calls safely", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		var mu sync.Mutex
		startFunc := func(_ context.Context) error { return nil }
		stopFunc := func() {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}

		svc := NewStandardService("test", startFunc, stopFunc)

		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				svc.Stop()
			})
		}

		wg.Wait()

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 5, callCount)
	})
}

func TestStandardService_ConcurrentStartStop(t *testing.T) {
	t.Parallel()

	var startCount, stopCount int
	var mu sync.Mutex

	startFunc := func(_ context.Context) error {
		mu.Lock()
		startCount++
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		return nil
	}
	stopFunc := func() {
		mu.Lock()
		stopCount++
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}

	svc := NewStandardService("test", startFunc, stopFunc)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = svc.Start(t.Context())
		}()
		go func() {
			defer wg.Done()
			svc.Stop()
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 10, startCount)
	assert.Equal(t, 10, stopCount)
}

func TestContextService_New(t *testing.T) {
	t.Parallel()

	startFunc := func(_ context.Context) error { return nil }

	svc := NewContextService("test-id", startFunc)

	require.NotNil(t, svc)
	assert.Equal(t, "test-id", svc.id)
	assert.NotNil(t, svc.startFunc)
}

func TestContextService_ID(t *testing.T) {
	t.Parallel()

	svc := NewContextService("my-context-service", nil)

	assert.Equal(t, "my-context-service", svc.ID())
}

func TestContextService_Start(t *testing.T) {
	t.Parallel()

	t.Run("creates and stores cancel func", func(t *testing.T) {
		t.Parallel()

		startFunc := func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}

		svc := NewContextService("test", startFunc)

		go func() {
			_ = svc.Start(t.Context())
		}()

		time.Sleep(50 * time.Millisecond)

		svc.m.Lock()
		cancel := svc.cancel
		svc.m.Unlock()

		assert.NotNil(t, cancel)
	})

	t.Run("passes derived context to startFunc", func(t *testing.T) {
		t.Parallel()

		receivedDone := make(chan struct{})
		startFunc := func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				close(receivedDone)
			case <-time.After(100 * time.Millisecond):
			}
			return nil
		}

		svc := NewContextService("test", startFunc)

		go func() {
			_ = svc.Start(t.Context())
		}()

		time.Sleep(20 * time.Millisecond)
		svc.Stop()

		select {
		case <-receivedDone:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("context was not cancelled")
		}
	})

	t.Run("handles concurrent start calls", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		var mu sync.Mutex
		done := make(chan struct{})

		startFunc := func(ctx context.Context) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			select {
			case <-ctx.Done():
			case <-done:
			}
			return nil
		}

		svc := NewContextService("test", startFunc)

		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				_ = svc.Start(t.Context())
			})
		}

		time.Sleep(50 * time.Millisecond)

		// Signal all goroutines to finish
		close(done)

		wg.Wait()

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 5, callCount)
	})
}

func TestContextService_Stop(t *testing.T) {
	t.Parallel()

	t.Run("cancels context", func(t *testing.T) {
		t.Parallel()

		cancelled := make(chan struct{})
		startFunc := func(ctx context.Context) error {
			<-ctx.Done()
			close(cancelled)
			return nil
		}

		svc := NewContextService("test", startFunc)

		go func() {
			_ = svc.Start(t.Context())
		}()

		time.Sleep(50 * time.Millisecond)
		svc.Stop()

		select {
		case <-cancelled:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("context was not cancelled")
		}
	})

	t.Run("handles stop before start", func(t *testing.T) {
		t.Parallel()

		startFunc := func(_ context.Context) error { return nil }
		svc := NewContextService("test", startFunc)

		svc.m.Lock()
		svc.cancel = func() {}
		svc.m.Unlock()

		assert.NotPanics(t, func() {
			svc.Stop()
		})
	})

	t.Run("handles multiple stop calls safely", func(t *testing.T) {
		t.Parallel()

		startFunc := func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}

		svc := NewContextService("test", startFunc)

		go func() {
			_ = svc.Start(t.Context())
		}()

		time.Sleep(50 * time.Millisecond)

		assert.NotPanics(t, func() {
			var wg sync.WaitGroup
			for range 5 {
				wg.Go(func() {
					svc.Stop()
				})
			}
			wg.Wait()
		})
	})
}

func TestContextService_ConcurrentStartStop(t *testing.T) {
	t.Parallel()

	var startCount int
	var mu sync.Mutex

	startFunc := func(ctx context.Context) error {
		mu.Lock()
		startCount++
		mu.Unlock()
		<-ctx.Done()
		return nil
	}

	svc := NewContextService("test", startFunc)

	var wg sync.WaitGroup

	// Start a single service instance
	wg.Go(func() {
		_ = svc.Start(t.Context())
	})

	time.Sleep(50 * time.Millisecond)

	// Stop it
	svc.Stop()

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, startCount)
}
