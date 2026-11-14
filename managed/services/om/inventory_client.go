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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PMM proxies the inventory app so the browser never holds a PMM Extensions bearer, and an unreachable
// app shows up as an error inside a page that still renders.

const inventoryRequestTimeout = 15 * time.Second

type extensionsFreshness struct {
	FirstSeenAt         *time.Time `json:"first_seen_at"`
	LastAttemptAt       *time.Time `json:"last_attempt_at"`
	LastSuccessAt       *time.Time `json:"last_success_at"`
	FailingSince        *time.Time `json:"failing_since"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastError           *string    `json:"last_error"`
	LastErrorCode       *string    `json:"last_error_code"`
	LastRunID           *string    `json:"last_run_id"`
}

type extensionsService struct {
	extensionsFreshness

	ServiceID string         `json:"service_id"`
	NodeID    string         `json:"node_id"`
	Name      string         `json:"name"`
	Port      *int32         `json:"port"`
	Role      *string        `json:"role"`
	Observed  map[string]any `json:"observed"`
}

type extensionsHost struct {
	extensionsFreshness

	NodeID       string              `json:"node_id"`
	Name         string              `json:"name"`
	Address      *string             `json:"address"`
	ExecutorHost *string             `json:"executor_host"`
	Observed     map[string]any      `json:"observed"`
	Services     []extensionsService `json:"services"`
}

type extensionsRunCounts struct {
	ServicesTotal    int32 `json:"services_total"`
	ServicesResolved int32 `json:"services_resolved"`
	ServicesOrphaned int32 `json:"services_orphaned"`
	ServicesAnswered int32 `json:"services_answered"`
	HostsTotal       int32 `json:"hosts_total"`
	HostsProbeable   int32 `json:"hosts_probeable"`
	HostsAnswered    int32 `json:"hosts_answered"`
	HostsFinished    int32 `json:"hosts_finished"`
}

type extensionsRun struct {
	RunID        string                     `json:"run_id"`
	Status       string                     `json:"status"`
	StartedAt    *time.Time                 `json:"started_at"`
	FinishedAt   *time.Time                 `json:"finished_at"`
	Counts       extensionsRunCounts        `json:"counts"`
	Scope        []string                   `json:"scope"`
	Error        *string                    `json:"error"`
	FailingNodes []extensionsRunFailingNode `json:"failing_nodes"`
	// Nodes is filled only by the detail endpoint, not by the list.
	Nodes []extensionsRunNode `json:"nodes"`
}

type extensionsRunFailingNode struct {
	NodeID string  `json:"node_id"`
	Name   *string `json:"name"`
}

type extensionsRunNode struct {
	NodeID        string                     `json:"node_id"`
	HostName      *string                    `json:"host_name"`
	ExecutorHost  *string                    `json:"executor_host"`
	Resolution    string                     `json:"resolution"`
	Answered      bool                       `json:"answered"`
	Duration      *float64                   `json:"duration_seconds"`
	TaskHistoryID *int64                     `json:"task_history_id"`
	Error         *string                    `json:"error"`
	Services      []extensionsRunNodeService `json:"services"`
}

type extensionsRunNodeService struct {
	ServiceID   *string `json:"service_id"`
	ServiceName *string `json:"service_name"`
	Answered    bool    `json:"answered"`
	Error       *string `json:"error"`
}

type extensionsSetting struct {
	Key          string  `json:"key"`
	Value        any     `json:"value"`
	DefaultValue any     `json:"default_value"`
	Type         string  `json:"type"`
	Reload       string  `json:"reload"`
	HasOverride  bool    `json:"has_override"`
	IsAdvanced   bool    `json:"is_advanced"`
	Description  *string `json:"description"`
}

// Detail is a string for the app's own errors and a list of per-field objects for a validation failure.
type extensionsError struct {
	Detail any `json:"detail"`
}

type extensionsPage[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// extensionsPageSize is PMM Extensions' MAX_PAGINATION_LIMIT.
const extensionsPageSize = 200

func fetchAllPages[T any](fetchPage func(offset, limit int) (extensionsPage[T], error)) ([]T, error) {
	all := make([]T, 0)
	offset := 0
	for {
		page, err := fetchPage(offset, extensionsPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			return all, nil
		}
	}
}

type inventoryCall struct {
	method string
	path   string
	query  url.Values
	body   any
}

func (a extensionsApp) call(ctx context.Context, c inventoryCall, out any) error {
	ctx, cancel := context.WithTimeout(ctx, inventoryRequestTimeout)
	defer cancel()

	cred, err := a.client.credential(ctx)
	if err != nil {
		return err
	}
	req, err := a.request(ctx, cred.bearer, c.method, c.path, c.query, c.body)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to build the request: %s", err)
	}

	resp, err := a.client.http.Do(req)
	if err != nil {
		// *url.Error leads with the verb and URL, which pushes the cause past the UI's 120-character snackbar.
		cause := err
		for next := errors.Unwrap(cause); next != nil; next = errors.Unwrap(cause) {
			cause = next
		}
		return status.Errorf(codes.Unavailable, "PMM Extensions is unreachable: %s", cause)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// A refusal may come from a role the cached bearer predates, so the next call exchanges again.
		refused := resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
		if refused && cred.session != "" {
			a.client.forget(cred.session)
		}
		return extensionsStatusError(resp, a.client.token, cred.caller)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to decode the inventory app's answer: %s", err)
	}
	return nil
}

// The gateway has no code it renders as 422, so a validation failure reaches the browser as 400;
// the app's per-field detail is kept in the message.
func extensionsStatusError(resp *http.Response, token string, callers bool) error {
	detail := extensionsErrorDetail(resp)
	switch resp.StatusCode {
	case http.StatusNotFound:
		return status.Error(codes.NotFound, detail)
	case http.StatusConflict:
		return status.Error(codes.Aborted, detail)
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return status.Error(codes.InvalidArgument, detail)
	case http.StatusServiceUnavailable:
		return status.Error(codes.Unavailable, detail)
	// A caller's own 401 or 403 passes through; a rejected shared token is PMM's own fault,
	// so it must not tell the browser to sign in again.
	case http.StatusUnauthorized:
		if callers {
			return status.Error(codes.Unauthenticated, detail)
		}
		if token == "" {
			return status.Errorf(codes.Internal, "PMM has no token for PMM Extensions, set PMM_EXTENSIONS_TOKEN (%s: %s)", resp.Status, detail)
		}
		return status.Errorf(codes.Internal, "PMM's token does not match EXTENSIONS_INTERNAL_TOKEN on PMM Extensions (%s: %s)", resp.Status, detail)
	case http.StatusForbidden:
		if callers {
			return status.Error(codes.PermissionDenied, detail)
		}
		return status.Errorf(codes.Internal, "PMM Extensions refused PMM's request (%s): %s", resp.Status, detail)
	default:
		return status.Errorf(codes.Internal, "the inventory app answered %s: %s", resp.Status, detail)
	}
}

func extensionsErrorDetail(resp *http.Response) string {
	var envelope extensionsError
	err := json.NewDecoder(resp.Body).Decode(&envelope)
	if err != nil || envelope.Detail == nil {
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

// IDs reaching here are PMM's and may contain characters that would change the path they address.
func inventoryPath(segments ...string) string {
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		escaped = append(escaped, url.PathEscape(segment))
	}
	return strings.Join(escaped, "/")
}
