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
