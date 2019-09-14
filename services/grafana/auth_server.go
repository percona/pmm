// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

// rules maps original URL prefix to minimal required role.
var rules = map[string]role{
	// TODO https://jira.percona.com/browse/PMM-4420
	"/agent.Agent/Connect": none,

	"/inventory.":                 admin,
	"/management.":                admin,
	"/management.Actions/":        viewer,
	"/server.Server/CheckUpdates": viewer,
	"/server.Server/UpdateStatus": none, // special token-based auth
	"/server.":                    admin,

	"/v1/inventory/":          admin,
	"/v1/management/":         admin,
	"/v1/management/Actions/": viewer,
	"/v1/Updates/Check":       viewer,
	"/v1/Updates/Status":      none, // special token-based auth
	"/v1/Updates/":            admin,
	"/v1/Settings/":           admin,

	// must be available without authentication for health checking
	"/v1/readyz": none,
	"/ping":      none, // PMM 1.x variant

	// must not be available without authentication as it can leak data
	"/v1/version":         viewer,
	"/managed/v1/version": viewer, // PMM 1.x variant

	"/v0/qan/": viewer,

	// not rules for /qan and /swagger UIs as there are no auth_request for them in nginx configuration

	"/prometheus/": admin,

	// "/" is a special case
}

// clientError contains authentication error response details.
type authError struct {
	code    codes.Code
	message string
}

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	c *Client
	l *logrus.Entry

	// TODO server metrics should be provided by middleware https://jira.percona.com/browse/PMM-4326
}

// NewAuthServer creates new AuthServer.
func NewAuthServer(c *Client) *AuthServer {
	return &AuthServer{
		c: c,
		l: logrus.WithField("component", "grafana/auth"),
	}
}

// ServeHTTP serves internal location /auth_request.
func (s *AuthServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// fail-safe
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	if err := s.authenticate(ctx, req); err != nil {
		// nginx completely ignores auth_request subrequest response body;
		// out nginx configuration then sends the same request as a normal request
		// and returns response body to the client

		// copy grpc-gateway behavior: set correct codes, set both "error" and "message"
		m := map[string]interface{}{
			"code":    int(err.code),
			"error":   err.message,
			"message": err.message,
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(runtime.HTTPStatusFromCode(err.code))
		if err := json.NewEncoder(rw).Encode(m); err != nil {
			s.l.Warnf("%s", err)
		}
	}
}

// nextPrefix returns path's prefix, stopping on slashes and dots:
// /foo.Bar/Baz -> /foo.Bar/ -> /foo. -> /
// That works for both gRPC and JSON URLs.
func nextPrefix(path string) string {
	path = strings.TrimRight(path, "/.")
	if i := strings.LastIndexAny(path, "/."); i != -1 {
		return path[:i+1]
	}
	return path
}

func (s *AuthServer) authenticate(ctx context.Context, req *http.Request) *authError {
	// TODO l := logger.Get(ctx) once we have it after https://jira.percona.com/browse/PMM-4326
	l := s.l

	if l.Logger.GetLevel() >= logrus.DebugLevel {
		b, err := httputil.DumpRequest(req, true)
		if err != nil {
			l.Errorf("Failed to dump request: %v.", err)
		}
		l.Debugf("Request:\n%s", b)
	}

	if req.URL.Path != "/auth_request" {
		l.Errorf("Unexpected path %s.", req.URL.Path)
		return &authError{code: codes.Internal, message: "Internal server error."}
	}

	origURI := req.Header.Get("X-Original-Uri")
	if origURI == "" {
		l.Errorf("Empty X-Original-Uri.")
		return &authError{code: codes.Internal, message: "Internal server error."}
	}
	l = l.WithField("req", fmt.Sprintf("%s %s", req.Header.Get("X-Original-Method"), origURI))

	// find the longest prefix present in rules, stopping on slashes and dots:
	// /foo.Bar/Baz -> /foo.Bar/ -> /foo. -> /
	prefix := origURI
	for prefix != "/" {
		if _, ok := rules[prefix]; ok {
			break
		}
		prefix = nextPrefix(prefix)
	}

	// fallback to Grafana admin if there is no explicit rule
	// TODO https://jira.percona.com/browse/PMM-4338
	minRole, ok := rules[prefix]
	if ok {
		l = l.WithField("prefix", prefix)
	} else {
		l.Warnf("No explicit rule for %q, falling back to Grafana admin.", origURI)
		minRole = grafanaAdmin
	}

	if minRole == none {
		l.Debugf("Minimal required role is %q, granting access without checking Grafana.", minRole)
		return nil
	}

	// check Grafana with some headers from request
	authHeaders := make(http.Header)
	for _, k := range []string{
		"Authorization",
		"Cookie",
	} {
		if v := req.Header.Get(k); v != "" {
			authHeaders.Set(k, v)
		}
	}
	role, err := s.c.getRole(ctx, authHeaders)
	if err != nil {
		l.Warnf("%s", err)
		if cErr, ok := errors.Cause(err).(*clientError); ok {
			code := codes.Internal
			if cErr.Code == 401 || cErr.Code == 403 {
				code = codes.Unauthenticated
			}
			return &authError{code: code, message: cErr.ErrorMessage}
		}
		return &authError{code: codes.Internal, message: "Internal server error."}
	}
	l = l.WithField("role", role.String())

	if role == grafanaAdmin {
		l.Debugf("Grafana admin, allowing access.")
		return nil
	}

	if minRole <= role {
		l.Debugf("Minimal required role is %q, granting access.", minRole)
		return nil
	}

	l.Warnf("Minimal required role is %q.", minRole)
	return &authError{code: codes.PermissionDenied, message: "Access denied."}
}
