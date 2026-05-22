// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"sync"
	"time"
)

const adreSearchMaxPerMinute = 60

// SearchRateLimiter enforces per-user search request limits (sliding 1-minute window).
type SearchRateLimiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	clock func() time.Time
}

// NewSearchRateLimiter creates a limiter with wall-clock time.
func NewSearchRateLimiter() *SearchRateLimiter {
	return &SearchRateLimiter{
		hits:  make(map[string][]time.Time),
		clock: time.Now,
	}
}

// Allow returns whether a request is allowed and suggested retry-after seconds if not.
func (r *SearchRateLimiter) Allow(user string) (ok bool, retryAfterSec int) { //nolint:nonamedreturns
	now := r.clock()
	cutoff := now.Add(-time.Minute)

	r.mu.Lock()
	defer r.mu.Unlock()

	ts := r.hits[user]
	kept := ts[:0]
	for _, t := range ts {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= adreSearchMaxPerMinute {
		// Next slot when oldest in window expires.
		oldest := kept[0]
		retry := max(int(oldest.Add(time.Minute).Sub(now)/time.Second)+1, 1)
		return false, retry
	}
	kept = append(kept, now)
	r.hits[user] = kept
	return true, 0
}
