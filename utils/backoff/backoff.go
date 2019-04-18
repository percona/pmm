// pmm-agent
// Copyright (C) 2018 Percona LLC
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
	f64 := rand.Float64()*2.0 - 1.0 // [-1.0,1.0]

	delay += time.Duration(float64(delay) * f64 * delayJitter)
	if delay < 0 {
		delay = 0
	}

	return delay
}
