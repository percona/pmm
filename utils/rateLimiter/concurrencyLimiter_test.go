// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rateLimiter

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConcurrencyLimiter_TryAcquireSucceedsUpToConfiguredLimit(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(3)

	require.True(t, limiter.TryAcquire(), "expected first acquire to succeed")
	require.True(t, limiter.TryAcquire(), "expected second acquire to succeed")
	require.True(t, limiter.TryAcquire(), "expected third acquire to succeed")
	require.False(t, limiter.TryAcquire(), "expected acquire to fail when limit is exhausted")
}

func TestConcurrencyLimiter_ReleaseMakesSlotAvailableAgain(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(1)

	require.True(t, limiter.TryAcquire(), "expected initial acquire to succeed")
	require.False(t, limiter.TryAcquire(), "expected acquire to fail when slot is already taken")

	limiter.Release()

	require.True(t, limiter.TryAcquire(), "expected acquire to succeed after release")
}

func TestNewConcurrencyLimiter_WithZeroSlotsAlwaysRejectsAcquire(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(0)

	require.False(t, limiter.TryAcquire(), "expected acquire to fail for zero-capacity limiter")
}

func TestConcurrencyLimiter_ExtraReleaseDoesNotIncreaseCapacity(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(1)
	require.True(t, limiter.TryAcquire(), "expected initial acquire to succeed")

	limiter.Release()
	limiter.Release() // unmatched release must not increase capacity beyond configured max

	require.True(t, limiter.TryAcquire(), "expected acquire to succeed after matched release")
	require.False(t, limiter.TryAcquire(), "expected acquire to fail after configured capacity is consumed")
}

func TestConcurrencyLimiter_TryAcquireConcurrentCallersNeverExceedsLimit(t *testing.T) {
	t.Parallel()

	const (
		slots   int32 = 8
		workers int   = 128
	)

	limiter := NewConcurrencyLimiter(slots)
	var wg sync.WaitGroup
	var successes atomic.Int32

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			if limiter.TryAcquire() {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, slots, successes.Load(), "unexpected number of successful acquires")
}
