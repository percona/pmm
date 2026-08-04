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

package grafana

import (
	"errors"
	"fmt"
	"hash/maphash"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

// statusCodeToString returns HTTP status code string presentation.
func statusCodeToString(code int) string {
	switch code {
	case http.StatusOK:
		return "200"
	case http.StatusBadRequest:
		return "400"
	case http.StatusUnauthorized:
		return "401"
	case http.StatusForbidden:
		return "403"
	case http.StatusNotFound:
		return "404"
	case http.StatusMethodNotAllowed:
		return "405"
	case http.StatusRequestTimeout:
		return "408"
	case http.StatusTooManyRequests:
		return "429"
	case http.StatusInternalServerError:
		return "500"
	case http.StatusServiceUnavailable:
		return "503"
	default:
		return strconv.Itoa(code)
	}
}

// convertAuthErrorToHTTPStatus maps an authError code to the HTTP status nginx receives.
// PermissionDenied uses 403 so nginx denies outright; the 401 re-run is a GET and would
// wrongly pass method-specific rules. Authentication and internal errors stay 401.
func convertAuthErrorToHTTPStatus(code codes.Code) int {
	if code == codes.PermissionDenied {
		return http.StatusForbidden
	}
	return authenticationErrorCode
}

// writeResponseErrorStatus sends an HTTP response header with the provided
// status code and writes custom HTTP headers with auth error details.
func writeResponseErrorStatus(rw http.ResponseWriter, status, authCode int, authError, authMessage string) {
	// nginx ignores the auth_request subrequest body: we use custom HTTP headers
	// to pass auth response error details to nginx.
	rw.Header().Set(authResponseCodeHeader, strconv.Itoa(authCode))
	rw.Header().Set(authResponseErrorHeader, authError)
	rw.Header().Set(authResponseMessageHeader, authMessage)
	rw.WriteHeader(status)
}

// extractOriginalRequest replaces req.Method and req.URL.Path with values from original request.
// Error is returned if original request information is missing or invalid.
func extractOriginalRequest(req *http.Request) error {
	origMethod, origURI := req.Header.Get("X-Original-Method"), req.Header.Get("X-Original-Uri")

	if origMethod == "" {
		return errors.New("empty X-Original-Method")
	}

	if origURI == "" {
		return errors.New("empty X-Original-Uri")
	}

	if origURI[0] != '/' {
		return fmt.Errorf("unexpected X-Original-Uri: %q", origURI)
	}

	if !utf8.ValidString(origURI) {
		return fmt.Errorf("invalid X-Original-Uri: %q", origURI)
	}

	cleanedOrigURI, err := cleanPath(origURI)
	if err != nil {
		return fmt.Errorf("failed to unescape path %q: %w", origURI, err)
	}

	req.Method = origMethod
	req.URL.Path = cleanedOrigURI
	return nil
}

// nextPrefix returns path's prefix, stopping on slashes, dots, and colons, e.g.:
// /inventory.Nodes/ListNodes -> /inventory.Nodes/ -> /inventory.Nodes -> /inventory. -> /inventory -> /
// /v1/inventory/Nodes/List -> /v1/inventory/Nodes/ -> /v1/inventory/Nodes -> /v1/inventory/ -> /v1/inventory -> /v1/ -> /v1 -> /
// That works for both gRPC and JSON URLs.
// The chain ends with "/" no matter what.
func nextPrefix(path string) string {
	if len(path) == 0 || path[0] != '/' || path == "/" {
		return "/"
	}

	if t := strings.TrimRight(path, "."); t != path {
		return t
	}

	if t := strings.TrimRight(path, "/"); t != path {
		return t
	}

	if t := strings.TrimRight(path, ":"); t != path {
		return t
	}

	i := strings.LastIndexAny(path, "/.:")
	return path[:i+1]
}

// resolveRule returns the minimal role for the given method and path, plus the matched
// prefix. It walks prefixes longest-to-shortest; a method-specific rule ("METHOD prefix")
// beats a path-only rule at the same prefix, so read and write on a shared path can differ.
// With no match it logs a warning and falls back to grafanaAdmin.
func resolveRule(method, cleanedPath string, l *logrus.Entry) (role, string) {
	prefix := cleanedPath
	for {
		if r, ok := methodRules[method+" "+prefix]; ok {
			return r, prefix
		}
		if r, ok := rules[prefix]; ok {
			return r, prefix
		}
		if prefix == "/" {
			l.Warn("No explicit rule, falling back to Grafana admin.")
			return grafanaAdmin, prefix
		}
		prefix = nextPrefix(prefix)
	}
}

// isLocalAgentConnection reports whether the request is a local PMM agent
// connection for endpoints that are allowed from localhost.
// This func expects that req.Method and req.URL.Path are already replaced
// with original request values - extractOriginalRequest(req) has been called beforehand.
func isLocalAgentConnection(req *http.Request) bool {
	path := req.URL.Path
	if isLocalhostRemoteAddr(req.RemoteAddr) &&
		(path == connectionEndpoint ||
			path == connectionEndpointV2 ||
			path == rtaCollectEndpoint) {
		return true
	}

	return false
}

// isLocalhostRemoteAddr validates if an HTTP req.RemoteAddr originates from loopback.
// Execution time: ~1-2ns (fast path) / ~15ns (fallback). Heap Allocations: 0.
func isLocalhostRemoteAddr(remoteAddr string) bool {
	// 1. Optimistic Fast Path (Branch Predictor friendly)
	// Go's HTTP server canonically formats IPv4/IPv6 loopbacks as exactly these strings.
	// The trailing colon (':') is critical to prevent matching IPs like "127.0.0.10:80".
	// strings.HasPrefix is highly optimized, bypassing parsing overhead entirely.
	if strings.HasPrefix(remoteAddr, "127.0.0.1:") || strings.HasPrefix(remoteAddr, "[::1]:") {
		return true
	}

	// 2. Strict Semantic Parsing Fallback
	// Catches edge cases like 127.0.0.2:port or IPv4-mapped IPv6 (::ffff:127.0.0.1:port).
	// netip.ParseAddrPort is zero-allocation. It returns a stack-allocated struct (netip.AddrPort)
	// without pointers, completely bypassing the Green Tea GC mark/sweep phases.
	// Loopback traffic can originate from 127.0.0.2 (common in Kubernetes/mesh proxies),
	// IPv6 ::1, or IPv4-mapped IPv6 addresses like ::ffff:127.0.0.1
	ap, err := netip.ParseAddrPort(remoteAddr)
	if err == nil {
		return ap.Addr().IsLoopback()
	}

	return false
}

// cleanPath returns a clean, unescaped path from a raw URI.
// It achieves 0 allocations if the path requires no modifications.
func cleanPath(uri string) (string, error) {
	// 1. Strip query parameters
	if i := strings.IndexByte(uri, '?'); i >= 0 {
		uri = uri[:i]
	}

	// 2. Fast-path check: scan for characters that require processing
	needsWork := false
	for i := range len(uri) {
		c := uri[i]
		// Check for URL encoding (%), dot\-segments (/\. or /\.\.), double slashes (//), or CR/LF.
		if c == '%' || c == '\n' || c == '\r' || (c == '/' && i > 0 && uri[i-1] == '/') {
			needsWork = true
			break
		}

		if c == '.' && i > 0 && uri[i-1] == '/' {
			// Match only dot\-segments, not dots inside normal segments (e.g. logs.zip).
			if i+1 == len(uri) || uri[i+1] == '/' ||
				(uri[i+1] == '.' && (i+2 == len(uri) || uri[i+2] == '/')) {
				needsWork = true
				break
			}
		}
	}

	// 3. Return zero-allocation slice if clean
	if !needsWork {
		return uri, nil
	}

	// 4. Slow-path: Allocate and process
	unescaped, err := url.PathUnescape(uri)
	if err != nil {
		return "", err
	}
	unescaped = strings.ReplaceAll(unescaped, "\n", " ")
	unescaped = strings.ReplaceAll(unescaped, "\r", " ")
	return path.Clean(unescaped), nil
}

// extractAuthHeaders extracts auth info from request.
func extractAuthHeaders(req *http.Request) http.Header {
	authorization := req.Header.Get("Authorization")
	cookie := req.Header.Get("Cookie")

	// Fast path: no auth headers -> no map allocation.
	if authorization == "" && cookie == "" {
		return nil
	}

	h := make(http.Header, 2) //nolint:mnd
	if authorization != "" {
		h.Set("Authorization", authorization)
	}
	if cookie != "" {
		h.Set("Cookie", cookie)
	}
	return h
}

var seed = maphash.MakeSeed()

// authCacheKey builds a deterministic cache key directly from request auth headers.
func authCacheKey(req *http.Request) uint64 {
	var h maphash.Hash // Stack allocated, does not escape

	// Seed ensures deterministic hashing across the lifetime of the process.
	// If you need stable hashes across process restarts (e.g., for persistent Redis keys),
	// use a fixed byte array with a custom implementation like xxHash instead.
	h.SetSeed(seed)

	authorization := req.Header.Get("Authorization")
	cookie := req.Header.Get("Cookie")

	if authorization == "" && cookie == "" {
		return 0
	}

	if authorization != "" && cookie == "" {
		_, _ = h.WriteString(authorization)
		_ = h.WriteByte(':') // Delimiter prevents concatenation collisions

		return h.Sum64()
	}

	if cookie != "" && authorization == "" {
		_ = h.WriteByte(':') // Delimiter prevents concatenation collisions
		_, _ = h.WriteString(cookie)

		return h.Sum64()
	}

	_, _ = h.WriteString(authorization)
	_ = h.WriteByte(':') // Delimiter prevents concatenation collisions
	_, _ = h.WriteString(cookie)

	return h.Sum64()
}
