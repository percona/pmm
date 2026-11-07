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

package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/percona/pmm/managed/utils/auth"
)

const (
	// Live-inspection calls one caller may have in flight.
	maxLiveCallsPerCaller = 2
	// Live-inspection calls one pmm-managed replica may have in flight.
	maxLiveCallsPerReplica = 8
	// Live-inspection calls one service may have in flight on a replica.
	maxLiveCallsPerService = 1
	// Metric and QAN calls one caller may have in flight.
	maxMetricCallsPerCaller = 4
	// Metric and QAN calls one pmm-managed replica may have in flight.
	maxMetricCallsPerReplica = 8

	// Live-inspection calls per minute one caller may start, and how many it may start at once.
	liveCallsPerMinute = 10
	liveCallsBurst     = 4
	// Metric calls per minute one caller may start, and how many it may start at once.
	metricCallsPerMinute = 60
	metricCallsBurst     = 10

	sessionCookieName = "pmm_session"
)

// limitHit says which cap refused a call.
type limitHit int

const (
	limitNone limitHit = iota
	limitPerCaller
	limitPerReplica
)

// limiter bounds the calls in flight per key and in total.
type limiter struct {
	mu        sync.Mutex
	perCaller int
	perTotal  int
	total     int
	byKey     map[string]int
}

func newLimiter(perCaller, total int) *limiter {
	return &limiter{perCaller: perCaller, perTotal: total, byKey: map[string]int{}}
}

// acquire takes a slot for key. It returns a release function and limitNone, or nil
// and the cap that refused the call; the per-caller cap is checked first.
func (l *limiter) acquire(key string) (func(), limitHit) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.byKey[key] >= l.perCaller {
		return nil, limitPerCaller
	}
	if l.total >= l.perTotal {
		return nil, limitPerReplica
	}
	l.byKey[key]++
	l.total++

	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()

			l.total--
			l.byKey[key]--
			if l.byKey[key] <= 0 {
				delete(l.byKey, key)
			}
		})
	}, limitNone
}

// rateLimiter is a token bucket per key that drops a bucket once it has refilled to its burst,
// so the map holds only recent callers.
type rateLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(perMinute, burst int) *rateLimiter {
	return &rateLimiter{
		rate:    float64(perMinute) / time.Minute.Seconds(),
		burst:   float64(burst),
		buckets: map[string]*bucket{},
	}
}

// take takes one token for key and returns true, or returns the wait until the next token and false.
func (r *rateLimiter) take(key string, now time.Time) (time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, b := range r.buckets {
		if r.refilled(b, now) >= r.burst {
			delete(r.buckets, k)
		}
	}

	b, ok := r.buckets[key]
	if !ok {
		b = &bucket{tokens: r.burst, last: now}
		r.buckets[key] = b
	}
	b.tokens = r.refilled(b, now)
	b.last = now

	if b.tokens < 1 {
		return time.Duration((1 - b.tokens) / r.rate * float64(time.Second)), false
	}
	b.tokens--
	return 0, true
}

func (r *rateLimiter) refilled(b *bucket, now time.Time) float64 {
	return min(r.burst, b.tokens+max(0, now.Sub(b.last).Seconds())*r.rate)
}

// takeToken takes one token from r for the caller, or answers rate_limited with retry_after.
func (s *Service) takeToken(r *rateLimiter, a callerAuth) error {
	wait, ok := r.take(callerKey(a), s.now())
	if ok {
		return nil
	}
	te := newToolError(codeRateLimited, "too many calls for these credentials; retry in %d s", retryAfterSeconds(wait))
	te.retryAfter = wait
	return te
}

// callerKey identifies the caller by a hash of its credential, so that the same
// token sent as Bearer or Basic, with or without a cookie, counts as one caller.
func callerKey(a callerAuth) string {
	h := http.Header{}
	h.Set("Authorization", a.authorization)
	if token := auth.GetTokenFromHeaders(h); token != "" {
		return hashKey("token\x00" + token)
	}

	req := &http.Request{Header: h}
	user, _, isBasic := req.BasicAuth()

	cookieReq := &http.Request{Header: http.Header{"Cookie": {a.cookie}}}
	c, cookieErr := cookieReq.Cookie(sessionCookieName)

	if isBasic && cookieErr == nil {
		// Grafana falls back from Basic to the session, so neither value alone identifies the caller.
		return hashKey("basic+session\x00")
	}
	if isBasic {
		// Grafana logins are case-insensitive.
		return hashKey("user\x00" + strings.ToLower(user))
	}
	if cookieErr == nil {
		// Grafana unescapes the session cookie value before looking it up.
		v, err := url.QueryUnescape(c.Value)
		if err != nil {
			v = c.Value
		}
		return hashKey("session\x00" + v)
	}
	return hashKey("")
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// startLive takes a token and a live-inspection slot for the caller, or answers rate_limited.
func (s *Service) startLive(a callerAuth) (func(), error) {
	err := s.takeToken(s.liveRate, a)
	if err != nil {
		return nil, err
	}
	release, hit := s.live.acquire(callerKey(a))
	switch hit {
	case limitPerCaller:
		return nil, newToolError(codeRateLimited,
			"at most %d live-inspection calls can be in flight for these credentials; retry when one finishes", maxLiveCallsPerCaller)
	case limitPerReplica:
		return nil, newToolError(codeRateLimited,
			"this replica already has %d live-inspection calls in flight; retry when one finishes", maxLiveCallsPerReplica)
	case limitNone:
	}
	return release, nil
}

// startLiveOn takes the live-inspection slot of a service, or answers rate_limited.
func (s *Service) startLiveOn(serviceID string) (func(), error) {
	release, hit := s.liveByService.acquire(serviceID)
	switch hit {
	case limitPerCaller:
		return nil, newToolError(codeRateLimited,
			"a live-inspection call on this service is already in flight; retry when it finishes")
	case limitPerReplica:
		return nil, newToolError(codeRateLimited,
			"this replica already has %d live-inspection calls in flight; retry when one finishes", maxLiveCallsPerReplica)
	case limitNone:
	}
	return release, nil
}

// startMetricCall takes a metric token and then a metric slot for the caller, or answers rate_limited.
func (s *Service) startMetricCall(a callerAuth) (func(), error) {
	err := s.takeToken(s.metricRate, a)
	if err != nil {
		return nil, err
	}
	return s.startMetrics(a)
}

// startMetrics takes a metric-call slot for the caller, or answers rate_limited.
func (s *Service) startMetrics(a callerAuth) (func(), error) {
	release, hit := s.metrics.acquire(callerKey(a))
	switch hit {
	case limitPerCaller:
		return nil, newToolError(codeRateLimited,
			"at most %d metric calls can be in flight for these credentials; retry when one finishes", maxMetricCallsPerCaller)
	case limitPerReplica:
		return nil, newToolError(codeRateLimited,
			"this replica already has %d metric calls in flight; retry when one finishes", maxMetricCallsPerReplica)
	case limitNone:
	}
	return release, nil
}
