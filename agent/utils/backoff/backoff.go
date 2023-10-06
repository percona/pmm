// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package backoff implement the backoff strategy for reconnecting to server, or for restarting Agents.
package backoff

import (
	"math/rand"
	"time"
)

const (
	delayIncrease = 0.5  // +50%
	delayJitter   = 0.25 // ±25%
)

// Backoff encapsulates delay manipulation.
type Backoff struct {
	delayBaseMin  time.Duration
	delayBaseMax  time.Duration
	delayBaseNext time.Duration
}

// New returns new reset backoff.
func New(min, max time.Duration) *Backoff {
	b := &Backoff{
		delayBaseMin: min,
		delayBaseMax: max,
	}
	b.Reset()
	return b
}

// Reset sets next delay to the default minimum.
func (b *Backoff) Reset() {
	b.delayBaseNext = b.delayBaseMin
}

// Delay returns next delay.
func (b *Backoff) Delay() time.Duration {
	delay := b.delayBaseNext

	b.delayBaseNext += time.Duration(float64(b.delayBaseNext) * delayIncrease)
	if b.delayBaseNext > b.delayBaseMax {
		b.delayBaseNext = b.delayBaseMax
	}

	// We could use normal distribution for jitter:
	// f64 = rand.NormFloat64() / 3.0 (three sigma rule)
	// but pure random seems to be better overall.
	//nolint:gosec
	f64 := rand.Float64()*2.0 - 1.0 // [-1.0,1.0]

	delay += time.Duration(float64(delay) * f64 * delayJitter)
	if delay < 0 {
		delay = 0
	}

	return delay
}
