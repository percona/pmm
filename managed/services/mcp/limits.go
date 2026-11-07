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

	"github.com/percona/pmm/managed/utils/auth"
)

const (
	// Live-inspection calls one caller may have in flight.
	maxLiveCallsPerCaller = 2
	// Live-inspection calls one pmm-managed replica may have in flight.
	maxLiveCallsPerReplica = 8
	// Live-inspection calls one service may have in flight on a replica.
	maxLiveCallsPerService = 1

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

// startLive takes a live-inspection slot for the caller, or answers rate_limited.
func (s *Service) startLive(a callerAuth) (func(), error) {
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
