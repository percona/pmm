// pmm-managed
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

// This file contains implementation for concurrent safe RNG.
package pmmapitests

import (
	"math/rand"
	"sync"
)

// ConcurrentRand wraps rand.Rand with mutex.
type ConcurrentRand struct {
	m    sync.Mutex
	rand *rand.Rand
}

// NewConcurrentRand constructs new ConcurrentRand with provided seed.
func NewConcurrentRand(seed int64) *ConcurrentRand {
	r := &ConcurrentRand{
		rand: rand.New(rand.NewSource(seed)),
	}
	return r
}

// Seed uses the provided seed value to initialize the generator to a deterministic state.
func (r *ConcurrentRand) Seed(seed int64) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rand.Seed(seed)
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func (r *ConcurrentRand) Int63() int64 {
	r.m.Lock()
	defer r.m.Unlock()
	return r.rand.Int63()
}

// Uint64 returns a pseudo-random 64-bit value as a uint64.
func (r *ConcurrentRand) Uint64() uint64 {
	r.m.Lock()
	defer r.m.Unlock()
	return r.rand.Uint64()
}
