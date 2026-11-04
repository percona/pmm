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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type extensionsClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func (c *extensionsClient) app(module string) extensionsApp {
	return extensionsApp{client: c, path: "api/apps/" + module}
}

func newExtensionsClient(baseURL, token string) *extensionsClient {
	return &extensionsClient{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout:       probeRequestTimeout,
			CheckRedirect: refuseRedirect,
		},
	}
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

func (a extensionsApp) request(ctx context.Context, method, path string, query url.Values, body any, sendEmptyBody bool) (*http.Request, error) {
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
	if a.client.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.client.token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (a extensionsApp) patchConfig(ctx context.Context, fields map[string]any) error {
	req, err := a.request(ctx, http.MethodPatch, "config", nil, fields, false)
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}

	resp, err := a.client.http.Do(req)
	if err != nil {
		// The *url.Error already names the verb and the URL.
		return err //nolint:wrapcheck
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PATCH %s: unexpected status %s", a.endpoint("config"), resp.Status)
	}
	return nil
}

// triggerRun queues a full-estate probe sweep. A 409 means a sweep is already in flight, which is
// just as good.
func (a extensionsApp) triggerRun(ctx context.Context) error {
	req, err := a.request(ctx, http.MethodPost, "runs", nil, nil, true)
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}

	resp, err := a.client.http.Do(req)
	if err != nil {
		// The *url.Error already names the verb and the URL.
		return err //nolint:wrapcheck
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("POST %s: unexpected status %s", a.endpoint("runs"), resp.Status)
	}
	return nil
}
