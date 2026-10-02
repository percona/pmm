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
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	retryMaxRetries = 6
	retryBaseDelay  = time.Second
	retryMaxDelay   = 16 * time.Second
)

// retryTransport retries requests that PMM Server rejected with 503 Service Unavailable,
// which pmm-managed returns when it cannot connect to its database (e.g. during a PostgreSQL
// failover in HA), and nginx returns during maintenance.
type retryTransport struct {
	next       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
}

func newRetryTransport(next http.RoundTripper) *retryTransport {
	return &retryTransport{
		next:       next,
		maxRetries: retryMaxRetries,
		baseDelay:  retryBaseDelay,
		maxDelay:   retryMaxDelay,
	}
}

// RoundTrip implements http.RoundTripper.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		body, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	delay := t.baseDelay
	for attempt := 0; ; attempt++ {
		r := req
		if body != nil {
			r = req.Clone(req.Context())
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		resp, err := t.next.RoundTrip(r)
		if err != nil || resp.StatusCode != http.StatusServiceUnavailable || attempt >= t.maxRetries {
			return resp, err
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		logrus.Warnf("PMM Server is unavailable for %s %s, retrying in %s (%d/%d)...",
			req.Method, req.URL.Path, delay, attempt+1, t.maxRetries)
		select {
		case <-time.After(delay):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
		delay = min(delay<<1, t.maxDelay)
	}
}

// check interfaces.
var _ http.RoundTripper = (*retryTransport)(nil)
