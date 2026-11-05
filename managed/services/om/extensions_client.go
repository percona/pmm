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

package om

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type extensionsClient struct {
	baseURL string
	token   string
	http    *http.Client

	exchangedMu sync.Mutex
	exchanged   map[[sha256.Size]byte]exchangedBearer
	exchanges   singleflight.Group
}

func (c *extensionsClient) app(module string) extensionsApp {
	return extensionsApp{client: c, path: "api/apps/" + module}
}

// settings addresses PMM Extensions' admin settings router, through which an admin changes an app's configuration.
func (c *extensionsClient) settings() extensionsApp {
	return extensionsApp{client: c, path: "api/extensions/admin/settings"}
}

func newExtensionsClient(baseURL, token string) *extensionsClient {
	return &extensionsClient{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Transport:     directTransport(),
			Timeout:       probeRequestTimeout,
			CheckRedirect: refuseRedirect,
		},
	}
}

// directTransport ignores the proxy environment, which would otherwise see the credentials and secrets sent to PMM Extensions.
func directTransport() *http.Transport {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{}
	}

	direct := defaultTransport.Clone()
	direct.Proxy = nil
	return direct
}

// net/http keeps the Authorization header on a same-host redirect even from https to http,
// which would send the bearer in clear text; PMM Extensions' API never redirects.
func refuseRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// cleartextToken reports whether a bearer sent to baseURL would leave this machine unencrypted.
func cleartextToken(baseURL string) bool {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "https" {
		return false
	}

	host := parsed.Hostname()
	if host == "" || host == "localhost" {
		return false
	}

	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

type extensionsApp struct {
	client *extensionsClient
	path   string
}

func (a extensionsApp) endpoint(path string) string {
	return strings.TrimSuffix(a.client.baseURL, "/") + "/" + a.path + "/" + strings.TrimPrefix(path, "/")
}

func (a extensionsApp) request(ctx context.Context, bearer, method, path string, query url.Values, body any, sendEmptyBody bool) (*http.Request, error) {
	endpoint := a.endpoint(path)
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var payload []byte
	switch {
	case body != nil:
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode the request: %w", err)
		}
		payload = encoded
	case sendEmptyBody:
		payload = []byte("{}")
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (a extensionsApp) setEnabled(ctx context.Context, enabled bool) error {
	return a.call(ctx, inventoryCall{method: http.MethodPut, path: "enabled", body: map[string]bool{"enabled": enabled}}, nil)
}

// triggerRun queues a full-estate probe sweep. A 409 means a sweep is already in flight, which is
// just as good.
func (a extensionsApp) triggerRun(ctx context.Context) error {
	err := a.call(ctx, inventoryCall{method: http.MethodPost, path: "runs", sendEmptyBody: true}, nil)
	if status.Code(err) == codes.Aborted {
		return nil
	}
	return err
}
