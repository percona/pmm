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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	// How long a live-inspection call waits for its service's slot before it answers rate_limited.
	liveServiceWait = 10 * time.Second
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
	// kind names the calls in the answer to a refused one, such as "metric".
	kind      string
	mu        sync.Mutex
	perCaller int
	perTotal  int
	total     int
	byKey     map[string]int
	// released is closed, and replaced, whenever a slot is released.
	released chan struct{}
}

func newLimiter(kind string, perCaller, total int) *limiter {
	return &limiter{kind: kind, perCaller: perCaller, perTotal: total, byKey: map[string]int{}, released: make(chan struct{})}
}

// refusal is the rate_limited answer to a call that hit refused, or nil for limitNone.
func (l *limiter) refusal(hit limitHit) error {
	switch hit {
	case limitPerCaller:
		return newToolError(codeRateLimited,
			"at most %d %s calls can be in flight for these credentials; retry when one finishes", l.perCaller, l.kind)
	case limitPerReplica:
		return newToolError(codeRateLimited, "this replica already has %d %s calls in flight; retry when one finishes", l.perTotal, l.kind)
	case limitNone:
	}
	return nil
}

// acquire takes a slot for key. It returns a release function and limitNone, or nil
// and the cap that refused the call; the per-caller cap is checked first.
func (l *limiter) acquire(key string) (func(), limitHit) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.acquireLocked(key)
}

// acquireWait is acquire, but while the per-caller cap refuses the call it waits for a release,
// for at most wait or until ctx is done; the per-replica cap still refuses at once.
func (l *limiter) acquireWait(ctx context.Context, key string, wait time.Duration) (func(), limitHit, error) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		l.mu.Lock()
		release, hit := l.acquireLocked(key)
		released := l.released
		l.mu.Unlock()
		if hit != limitPerCaller {
			return release, hit, nil
		}

		select {
		case <-released:
		case <-timer.C:
			return nil, limitPerCaller, nil
		case <-ctx.Done():
			return nil, limitPerCaller, ctx.Err()
		}
	}
}

func (l *limiter) acquireLocked(key string) (func(), limitHit) {
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
			close(l.released)
			l.released = make(chan struct{})
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
	// A call that read the time before a later one took the lock must not move last back, or the gap is refilled twice.
	if now.After(b.last) {
		b.last = now
	}

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
	wait, ok := r.take(a.key, s.now())
	if ok {
		return nil
	}
	te := newToolError(codeRateLimited, "too many calls for these credentials; retry in %d s", retryAfterSeconds(wait))
	te.retryAfter = wait
	return te
}

// callerKey identifies the caller by a hash of its credential: its token when it sent one, so that the same token sent
// as Bearer or Basic, with or without a cookie, counts as one caller, else its Basic-auth user or session cookie.
func callerKey(token, authorization, cookie string) string {
	if token != "" {
		return hashKey("token\x00" + token)
	}

	req := &http.Request{Header: http.Header{"Authorization": {authorization}, "Cookie": {cookie}}}
	user, _, isBasic := req.BasicAuth()
	c, cookieErr := req.Cookie(auth.SessionCookieName)

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

// startLive starts a live-inspection call on a service: it waits up to liveServiceWait for the service's slot, then
// takes a caller and a replica slot and a token without waiting. When one refuses, it releases the others and answers
// rate_limited.
func (s *Service) startLive(ctx context.Context, a callerAuth, serviceID string) (func(), error) {
	releaseService, err := s.startLiveOn(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	release, hit := s.live.acquire(a.key)
	if hit != limitNone {
		releaseService()
		return nil, s.live.refusal(hit)
	}
	err = s.takeToken(s.liveRate, a)
	if err != nil {
		release()
		releaseService()
		return nil, err
	}
	return func() {
		release()
		releaseService()
	}, nil
}

// startLiveOn takes the live-inspection slot of a service, waiting up to liveServiceWait for it, or answers rate_limited;
// a call whose deadline passes while it waits answers rate_limited too.
func (s *Service) startLiveOn(ctx context.Context, serviceID string) (func(), error) {
	release, hit, err := s.liveByService.acquireWait(ctx, serviceID, s.liveServiceWait)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, newToolError(codeRateLimited,
			"a live-inspection call on this service is still in flight, and this call ran out of time waiting for it; retry when it finishes")
	}
	if err != nil {
		return nil, err
	}
	if hit == limitPerCaller {
		return nil, newToolError(codeRateLimited,
			"a live-inspection call on this service is still in flight after %s; retry when it finishes", s.liveServiceWait)
	}
	if hit != limitNone {
		return nil, s.liveByService.refusal(hit)
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
	release, hit := s.metrics.acquire(a.key)
	if hit != limitNone {
		return nil, s.metrics.refusal(hit)
	}
	return release, nil
}
