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
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errorCode is one of the tool contract's error codes. A failed tool call
// returns CallToolResult{IsError: true} whose text is "error: <code>" followed
// by a remediation message on the next line.
type errorCode string

const (
	codeNotFound               errorCode = "not_found"
	codeInvalidInput           errorCode = "invalid_input"
	codeUnauthorized           errorCode = "unauthorized"
	codeTimeout                errorCode = "timeout"
	codeNoQuerySource          errorCode = "no_query_source"
	codeInsufficientPrivileges errorCode = "insufficient_privileges"
	codeAgentUnreachable       errorCode = "agent_unreachable"
	codePMMUnavailable         errorCode = "pmm_unavailable"
	codeRateLimited            errorCode = "rate_limited"
)

// toolError is the typed error every tool returns on failure. The MCP SDK
// embeds Error() in the tool result with isError set.
type toolError struct {
	code    errorCode
	message string
	// retryAfter is how long to wait before a rate-limited call may succeed; zero when not set.
	retryAfter time.Duration
}

// retryAfterSeconds rounds a wait up to whole seconds, at least one.
func retryAfterSeconds(d time.Duration) int {
	return max(1, int(math.Ceil(d.Seconds())))
}

func (e *toolError) Error() string {
	return fmt.Sprintf("error: %s\n%s", e.code, e.message)
}

// retryable reports whether the same call may succeed if repeated later.
func (e *toolError) retryable() bool {
	switch e.code {
	case codeTimeout, codeAgentUnreachable, codePMMUnavailable, codeRateLimited:
		return true
	default:
		return false
	}
}

// errorResult is the failed tool result: text "error: <code>\n<message>" and
// the code and retryable flag in _meta, without structured content.
func errorResult(te *toolError) *mcp.CallToolResult {
	meta := map[string]any{
		"code":      string(te.code),
		"retryable": te.retryable(),
	}
	if te.retryAfter > 0 {
		meta["retry_after"] = retryAfterSeconds(te.retryAfter)
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: te.Error()}},
		Meta:    mcp.Meta{"com.percona/error": meta},
	}
}

func newToolError(code errorCode, format string, args ...any) *toolError {
	return &toolError{code: code, message: fmt.Sprintf(format, args...)}
}

// statusError is produced by the loopback transport for every non-2xx response
// so that all generated clients and the raw Grafana calls share one mapping.
type statusError struct {
	status  int
	method  string
	path    string
	message string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("PMM returned %d on %s %s: %s", e.status, e.method, e.path, e.message)
}

// mapError converts any error raised by the PMM API client into a toolError.
// HTTP 401/403 become unauthorized with PMM's own message passed through,
// 404 not_found, 400 and 422 invalid_input; everything else is pmm_unavailable except
// deadlines, which are timeout.
func mapError(err error) *toolError {
	te, ok := errors.AsType[*toolError](err)
	if ok {
		return te
	}

	if errors.Is(err, errResponseTooLarge) {
		return newToolError(codePMMUnavailable, "%s", err)
	}

	se, ok := errors.AsType[*statusError](err)
	if ok {
		switch se.status {
		case http.StatusUnauthorized, http.StatusForbidden:
			return newToolError(codeUnauthorized, "%s", se.Error())
		case http.StatusNotFound:
			return newToolError(codeNotFound, "%s", se.Error())
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return newToolError(codeInvalidInput, "%s", se.Error())
		default:
			return newToolError(codePMMUnavailable, "%s", se.Error())
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return newToolError(codeTimeout, "PMM API call did not complete in time: %s", err)
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return newToolError(codeTimeout, "PMM API call did not complete in time: %s", err)
	}

	return newToolError(codePMMUnavailable, "cannot reach the PMM API: %s", err)
}
