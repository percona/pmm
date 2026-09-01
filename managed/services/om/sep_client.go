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
	"net/http"
	"net/url"
	"strings"
)

// sepClient is where SEP is and how PMM authenticates to it: the base URL, the bearer
// token, the http.Client every SEP-backed source and proxy handler shares.
//
// It knows nothing about a specific app -- that is app()'s job. Splitting the two is what
// lets a second SEP-backed source (restart, upgrade, whatever om_actions turns out to
// need) get a client of its own by calling app() again on the one sepClient the Service
// already holds, rather than by copying the transport. Before this split, probeSource was
// three things at once: the on-host fact source, the transport to SEP, and (through
// inventory.go's use of it) half of the inventory REST surface. Only the first of those
// is a probeSource's business; the other two are this file's.
type sepClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// app returns a handle addressed relative to /api/apps/<module>, the mount every SEP app
// follows. Which app to ask is the caller's business, not the operator's: `--sep-url`
// points at SEP, not at an app, so the same setting serves every source this package
// gains.
func (c *sepClient) app(module string) sepApp {
	return sepApp{client: c, path: "api/apps/" + module}
}

// sepApp addresses one SEP app through a shared sepClient. Cheap to copy: it is a
// pointer and a path.
type sepApp struct {
	client *sepClient
	path   string
}

// endpoint builds an absolute URL for a path relative to this app.
func (a sepApp) endpoint(path string) string {
	return strings.TrimSuffix(a.client.baseURL, "/") + "/" + a.path + "/" + strings.TrimPrefix(path, "/")
}

// request builds one HTTP request against this app.
//
// Body is marshalled as JSON when non-nil, and when sendEmptyBody is set instead sends
// "{}" -- some proxied endpoints need that: the app's trigger endpoint takes an optional
// object and FastAPI rejects a bodyless POST against a typed body.
//
// The caller sets its own deadline on ctx and performs the request itself -- callers want
// different timeouts (a probe's own read of its estate is budgeted tighter than a proxied
// request the browser is waiting on) and different error mappings (a gRPC status for the
// inventory proxy, a plain error for a factSource's run receipt), and only the request
// itself -- the URL, the query, the body encoding, the bearer header -- is common to both.
func (a sepApp) request(ctx context.Context, method, path string, query url.Values, body any, sendEmptyBody bool) (*http.Request, error) {
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

// patchConfig sends fields to the app's PATCH /config endpoint and discards the
// response body -- callers that only want to write settings, not read the
// SettingResponse rows PATCH returns, use this instead of building the request
// themselves.
func (a sepApp) patchConfig(ctx context.Context, fields map[string]any) error {
	req, err := a.request(ctx, http.MethodPatch, "config", nil, fields, false)
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}

	resp, err := a.client.http.Do(req)
	if err != nil {
		return fmt.Errorf("PATCH %s: %w", a.endpoint("config"), err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PATCH %s: unexpected status %s", a.endpoint("config"), resp.Status)
	}
	return nil
}

// triggerRun queues a full-estate probe sweep via the app's POST /runs endpoint and
// discards the accepted run's body. Body is sent empty rather than omitted -- the
// endpoint takes an optional scope object, and FastAPI rejects a bodyless POST
// against a typed body.
//
// A 409 (a sweep already in flight) is not treated as a failure to log: the estate
// is about to be swept either way, which is exactly the outcome a caller here wants.
func (a sepApp) triggerRun(ctx context.Context) error {
	req, err := a.request(ctx, http.MethodPost, "runs", nil, nil, true)
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}

	resp, err := a.client.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", a.endpoint("runs"), err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("POST %s: unexpected status %s", a.endpoint("runs"), resp.Status)
	}
	return nil
}
