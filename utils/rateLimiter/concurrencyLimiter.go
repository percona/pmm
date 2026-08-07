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

// Package rateLimiter package provides different implementations of high-performance rate limiters.
package rateLimiter

import (
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// ConcurrencyLimiter is used limiting total active in-flight operations
// (e.g., max 50 concurrent database connections or worker threads).
type ConcurrencyLimiter struct {
	// Leading pad: Prevents false sharing with preceding fields if embedded in a larger struct.
	_              cpu.CacheLinePad
	availableSlots atomic.Int32
	// Trailing pad: Prevents false sharing with trailing fields or adjacent elements in a slice.
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
