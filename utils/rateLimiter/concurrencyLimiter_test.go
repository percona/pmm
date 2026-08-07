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
)

func TestNewConcurrencyLimiter_TryAcquireSucceedsUpToConfiguredLimit(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(3)

	if !limiter.TryAcquire() {
		t.Fatal("expected first acquire to succeed")
	}
	if !limiter.TryAcquire() {
		t.Fatal("expected second acquire to succeed")
	}
	if !limiter.TryAcquire() {
		t.Fatal("expected third acquire to succeed")
	}
	if limiter.TryAcquire() {
		t.Fatal("expected acquire to fail when limit is exhausted")
	}
}

func TestConcurrencyLimiter_ReleaseMakesSlotAvailableAgain(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(1)

	if !limiter.TryAcquire() {
		t.Fatal("expected initial acquire to succeed")
	}
	if limiter.TryAcquire() {
		t.Fatal("expected acquire to fail when slot is already taken")
	}

	limiter.Release()

	if !limiter.TryAcquire() {
		t.Fatal("expected acquire to succeed after release")
	}
}

func TestNewConcurrencyLimiter_WithZeroSlotsAlwaysRejectsAcquire(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(0)

	if limiter.TryAcquire() {
		t.Fatal("expected acquire to fail for zero-capacity limiter")
	}
}

func TestConcurrencyLimiter_ReleaseWithoutPriorAcquireIncreasesAvailableCapacity(t *testing.T) {
	t.Parallel()

	limiter := NewConcurrencyLimiter(0)
	limiter.Release()

	if !limiter.TryAcquire() {
		t.Fatal("expected acquire to succeed after release from zero capacity")
	}
	if limiter.TryAcquire() {
		t.Fatal("expected second acquire to fail after consuming released slot")
	}
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

	if got := successes.Load(); got != slots {
		t.Fatalf("unexpected number of successful acquires: got %d, want %d", got, slots)
	}
}
