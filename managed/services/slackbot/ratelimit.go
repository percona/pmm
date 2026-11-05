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

package slackbot

import (
	"sync"
	"time"
)

const (
	// slackRatePerMinute / slackRateBurst bound human chat turns per Slack user.
	slackRatePerMinute = 5
	slackRateBurst     = 3
	// denyCooldownWindow throttles the "ask an admin" / "too fast" replies per (channel,user).
	denyCooldownWindow = 10 * time.Minute
	// rateLimiterMaxEntries / cooldownMaxEntries bound the maps so churn of distinct IDs can't grow
	// them without limit.
	rateLimiterMaxEntries = 4096
	cooldownMaxEntries    = 4096
)

// tokenBucket is a simple refilling token bucket.
type tokenBucket struct {
	tokens   float64
	lastSeen time.Time
}

// userRateLimiter is a per-key (Slack user) token-bucket limiter, safe for concurrent use and bounded
// in size (oldest entries are evicted once the cap is exceeded).
type userRateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*tokenBucket
	perMinute  float64
	burst      float64
	maxEntries int
}

func newUserRateLimiter(perMinute, burst, maxEntries int) *userRateLimiter {
	return &userRateLimiter{
		buckets:    make(map[string]*tokenBucket),
		perMinute:  float64(perMinute),
		burst:      float64(burst),
		maxEntries: maxEntries,
	}
}

// allow consumes one token for key, refilling based on elapsed time. It returns false when the bucket
// is empty (the caller should throttle).
func (r *userRateLimiter) allow(key string) bool {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()

	b := r.buckets[key]
	if b == nil {
		r.evictIfFullLocked()
		b = &tokenBucket{tokens: r.burst, lastSeen: now}
		r.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Minutes()
		b.tokens += elapsed * r.perMinute
		if b.tokens > r.burst {
			b.tokens = r.burst
		}
		b.lastSeen = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// evictIfFullLocked drops the least-recently-seen entry when the map is at capacity. Caller holds mu.
func (r *userRateLimiter) evictIfFullLocked() {
	if len(r.buckets) < r.maxEntries {
		return
	}
	var oldestKey string
	var oldest time.Time
	for k, b := range r.buckets {
		if oldestKey == "" || b.lastSeen.Before(oldest) {
			oldestKey, oldest = k, b.lastSeen
		}
	}
	if oldestKey != "" {
		delete(r.buckets, oldestKey)
	}
}

// cooldown rate-limits a notice (e.g. a denial reply) per key over a fixed window, so it can't be
// weaponized into a reply-spam amplifier. Bounded in size.
type cooldown struct {
	mu         sync.Mutex
	lastSent   map[string]time.Time
	window     time.Duration
	maxEntries int
}

func newCooldown(window time.Duration, maxEntries int) *cooldown {
	return &cooldown{
		lastSent:   make(map[string]time.Time),
		window:     window,
		maxEntries: maxEntries,
	}
}

// allow reports whether a notice for key may be sent now (and records the send).
func (c *cooldown) allow(key string) bool {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if last, ok := c.lastSent[key]; ok && now.Sub(last) < c.window {
		return false
	}
	if len(c.lastSent) >= c.maxEntries {
		var oldestKey string
		var oldest time.Time
		for k, t := range c.lastSent {
			if oldestKey == "" || t.Before(oldest) {
				oldestKey, oldest = k, t
			}
		}
		if oldestKey != "" {
			delete(c.lastSent, oldestKey)
		}
	}
	c.lastSent[key] = now
	return true
}

// slackUserLimiter and denyCooldown are the package-level instances used by the event router.
var (
	slackUserLimiter = newUserRateLimiter(slackRatePerMinute, slackRateBurst, rateLimiterMaxEntries)
	denyCooldown     = newCooldown(denyCooldownWindow, cooldownMaxEntries)
)
