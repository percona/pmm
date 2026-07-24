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
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
)

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
	ip := strings.Split(req.RemoteAddr, ":")[0]
	// pmmAgent := req.Header.Get("Pmm-Agent-Id")
	path := req.URL.Path
	if ip == "127.0.0.1" &&
		(path == connectionEndpoint ||
			path == connectionEndpointV2 ||
			path == rtaCollectEndpoint) {
		return true
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
	authHeaders := make(http.Header)
	for _, k := range []string{
		"Authorization",
		"Cookie",
	} {
		if v := req.Header.Get(k); v != "" {
			authHeaders.Set(k, v)
		}
	}
	return authHeaders
}
