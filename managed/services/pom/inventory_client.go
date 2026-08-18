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

package pom

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// This file is the client half of the inventory proxy: the shapes SEP's pom_discovery
// app serves, and one place that speaks HTTP to it.
//
// It exists because the browser must not hold a SEP bearer. A page that talks to SEP
// directly needs one minted from the PMM session, which means the page is gated on SEP
// being up, configured and willing to exchange the token -- and it fails closed, so a
// sick SEP blanks the page rather than showing an estate with an error on it. Proxying
// here costs a second hop and buys "SEP is unreachable" as an error *inside* a page
// that still renders.

// inventoryRequestTimeout bounds a proxied request.
//
// Every path here is a database read on the other side; the one that starts work,
// POST /runs, returns as soon as the run row is written and does not wait for the Nomad
// jobs. So nothing behind this endpoint should take seconds, and one that does is a
// sign the app is unwell rather than busy.
const inventoryRequestTimeout = 15 * time.Second

// sepFreshness is the freshness block both estate rows carry.
type sepFreshness struct {
	FirstSeenAt         *time.Time `json:"first_seen_at"`
	LastAttemptAt       *time.Time `json:"last_attempt_at"`
	LastSuccessAt       *time.Time `json:"last_success_at"`
	FailingSince        *time.Time `json:"failing_since"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastError           *string    `json:"last_error"`
}

// sepService is one row of GET /services, and of the `services` list on a host.
type sepService struct {
	sepFreshness

	ServiceID string         `json:"service_id"`
	NodeID    string         `json:"node_id"`
	Name      string         `json:"name"`
	Port      *int32         `json:"port"`
	Role      *string        `json:"role"`
	Observed  map[string]any `json:"observed"`
}

// sepHost is one row of GET /hosts.
type sepHost struct {
	sepFreshness

	NodeID       string         `json:"node_id"`
	Name         string         `json:"name"`
	Address      *string        `json:"address"`
	ExecutorHost *string        `json:"executor_host"`
	Observed     map[string]any `json:"observed"`
	Services     []sepService   `json:"services"`
}

// sepRunCounts is what one refresh saw.
type sepRunCounts struct {
	ServicesTotal    int32 `json:"services_total"`
	ServicesResolved int32 `json:"services_resolved"`
	ServicesOrphaned int32 `json:"services_orphaned"`
	ServicesAnswered int32 `json:"services_answered"`
}

// sepRun is one row of GET /runs.
type sepRun struct {
	RunID      string       `json:"run_id"`
	Status     string       `json:"status"`
	StartedAt  *time.Time   `json:"started_at"`
	FinishedAt *time.Time   `json:"finished_at"`
	Counts     sepRunCounts `json:"counts"`
	Scope      []string     `json:"scope"`
	Error      *string      `json:"error"`
}

// sepSetting is one row of GET /config.
type sepSetting struct {
	Key          string  `json:"key"`
	Value        any     `json:"value"`
	DefaultValue any     `json:"default_value"`
	Type         string  `json:"type"`
	Reload       string  `json:"reload"`
	HasOverride  bool    `json:"has_override"`
	IsAdvanced   bool    `json:"is_advanced"`
	Description  *string `json:"description"`
}

// sepError is FastAPI's error envelope.
//
// `detail` is a string for the app's own errors and a list of per-field objects for a
// validation failure, so it is decoded as `any` and rendered rather than typed.
type sepError struct {
	Detail any `json:"detail"`
}

// inventoryCall describes one proxied request.
type inventoryCall struct {
	method string
	path   string
	query  url.Values
	// body is marshalled as JSON when non-nil. A nil body on a POST still sends `{}`
	// when sendEmptyBody is set, because the app's trigger endpoint takes an optional
	// object and FastAPI rejects a bodyless POST against a typed body.
	body          any
	sendEmptyBody bool
}

// call performs one proxied request and decodes the answer into out.
//
// The out parameter may be nil for a DELETE, whose 204 carries none. Errors come back as gRPC status
// errors with the code the gateway will turn back into the status SEP gave, so a 404
// from the app reaches the browser as a 404 rather than as a 500 about a 404.
func (s probeSource) call(ctx context.Context, c inventoryCall, out any) error {
	ctx, cancel := context.WithTimeout(ctx, inventoryRequestTimeout)
	defer cancel()

	endpoint := s.endpoint(c.path)
	if len(c.query) > 0 {
		endpoint += "?" + c.query.Encode()
	}

	var payload []byte
	switch {
	case c.body != nil:
		encoded, err := json.Marshal(c.body)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to encode the request: %s", err)
		}
		payload = encoded
	case c.sendEmptyBody:
		payload = []byte("{}")
	}

	req, err := http.NewRequestWithContext(ctx, c.method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to build the request: %s", err)
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return status.Errorf(codes.Unavailable, "the discovery app is unreachable: %s", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= http.StatusBadRequest {
		return sepStatusError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil { //nolint:noinlineerr
		return status.Errorf(codes.Internal, "failed to decode the discovery app's answer: %s", err)
	}
	return nil
}

// sepStatusError turns the app's error response into a gRPC status error.
//
// The mapping is chosen so the gateway reproduces the app's own status where it can:
// 404 stays 404 and 409 stays 409, because "no such host" and "another refresh already
// holds this host" are both answers a caller acts on differently from a failure.
//
// The one exception is 422: gRPC has no code the gateway renders as 422, so a validation
// failure arrives as InvalidArgument and the browser sees 400. The app's per-field
// detail is carried through in the message rather than dropped, which is what the UI
// needs to render inline errors; only the status number is lost.
func sepStatusError(resp *http.Response) error {
	detail := sepErrorDetail(resp)
	switch resp.StatusCode {
	case http.StatusNotFound:
		return status.Error(codes.NotFound, detail)
	case http.StatusConflict:
		return status.Error(codes.Aborted, detail)
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return status.Error(codes.InvalidArgument, detail)
	case http.StatusUnauthorized, http.StatusForbidden:
		// Deliberately not passed through as-is: a 401 from PMM's own gateway means the
		// *caller* is unauthenticated, and reflecting the app's 401 would tell the
		// browser to re-authenticate against PMM when what actually failed is PMM's
		// credential for SEP. That is an operator's problem, so it reads as one.
		return status.Errorf(codes.Internal,
			"PMM's credential for the discovery app was rejected (%s): %s", resp.Status, detail)
	default:
		return status.Errorf(codes.Internal, "the discovery app answered %s: %s", resp.Status, detail)
	}
}

// sepErrorDetail extracts something readable from the app's error body.
func sepErrorDetail(resp *http.Response) string {
	var envelope sepError
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil || envelope.Detail == nil { //nolint:noinlineerr
		return resp.Status
	}
	if text, ok := envelope.Detail.(string); ok {
		return text
	}
	encoded, err := json.Marshal(envelope.Detail)
	if err != nil {
		return resp.Status
	}
	return string(encoded)
}

// inventoryPath joins path segments under the app, escaping each one.
//
// Escaping matters: an ID reaching here is PMM's, not the app's, and nothing guarantees
// it is free of characters that would change which path it addresses.
func inventoryPath(segments ...string) string {
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		escaped = append(escaped, url.PathEscape(segment))
	}
	return strings.Join(escaped, "/")
}
