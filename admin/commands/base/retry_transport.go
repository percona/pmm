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

package base

import (
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	retryUnavailableDelay = time.Second
	// Covers a PostgreSQL failover in PMM HA, which can take up to Patroni's default 30s TTL.
	retryUnavailableFor = 30 * time.Second
)

// retryTransport resends a request PMM Server answered with 503 Service Unavailable, which
// it returns while a backend such as its database is briefly unreachable.
type retryTransport struct {
	next     http.RoundTripper
	delay    time.Duration
	duration time.Duration
}

// RoundTrip implements http.RoundTripper.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline := time.Now().Add(t.duration)
	for {
		resp, err := t.next.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusServiceUnavailable {
			return resp, err
		}

		replayable := req.Body == nil || req.GetBody != nil
		if !replayable || time.Now().Add(t.delay).After(deadline) {
			return resp, nil
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		logrus.Warnf("PMM Server is temporarily unavailable, retrying %s %s in %s...", req.Method, req.URL.Path, t.delay)

		timer := time.NewTimer(t.delay)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.Body = body
		}
	}
}

// Unwrap returns the transport requests are sent with.
func (t *retryTransport) Unwrap() http.RoundTripper {
	return t.next
}
