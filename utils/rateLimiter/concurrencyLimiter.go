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

// Package rateLimiter package provides different implementations of high-performance rate limiters.
package rateLimiter

import (
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// ConcurrencyLimiter is used for limiting total active in-flight operations
// (e.g., max 50 concurrent database connections or worker threads).
// It fails fast in case there are no free slots available.
// It uses atomic operations to minimize lock contention and maximize throughput in concurrent environments.
// Useful in hot-paths.
type ConcurrencyLimiter struct {
	// Leading pad: Prevents false sharing with preceding fields if embedded in a larger struct.
	// CPU arch dependant.
	_              cpu.CacheLinePad
	availableSlots atomic.Int32
	// Trailing pad: Prevents false sharing with trailing fields or adjacent elements in a slice.
	// CPU arch dependant.
	_ cpu.CacheLinePad
}

// NewConcurrencyLimiter creates a new ConcurrencyLimiter with the specified maximum number of slots.
func NewConcurrencyLimiter(maxSlots int32) *ConcurrencyLimiter {
	cl := &ConcurrencyLimiter{}
	cl.availableSlots.Store(maxSlots)
	return cl
}

// TryAcquire claims an active slot. Returns false immediately if 0 slots remain.
func (cl *ConcurrencyLimiter) TryAcquire() bool {
	for {
		current := cl.availableSlots.Load()
		if current <= 0 {
			return false // Fail fast
		}
		if cl.availableSlots.CompareAndSwap(current, current-1) {
			return true
		}
	}
}

// Release frees an active slot back to the pool.
func (cl *ConcurrencyLimiter) Release() {
	cl.availableSlots.Add(1)
}
