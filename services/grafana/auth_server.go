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
	"fmt"
	"net/http"
	"net/http/httputil"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// rules maps original URL prefix to minimal required role.
var rules = map[string]role{
	"/agent.Agent/Connect": none,

	"/inventory.Agents/Get":    editor,
	"/inventory.Agents/List":   editor,
	"/inventory.Nodes/Get":     editor,
	"/inventory.Nodes/List":    editor,
	"/inventory.Services/Get":  editor,
	"/inventory.Services/List": editor,
	"/inventory.":              admin,

	"/management.": admin,

	"/server.": admin,

	"/v0/inventory/Agents/Get":    editor,
	"/v0/inventory/Agents/List":   editor,
	"/v0/inventory/Nodes/Get":     editor,
	"/v0/inventory/Nodes/List":    editor,
	"/v0/inventory/Services/Get":  editor,
	"/v0/inventory/Services/List": editor,
	"/v0/inventory/":              admin,

	"/v0/management/": admin,

	"/v1/ChangeSettings": admin,
	"/v1/GetSettings":    admin,

	"/v0/qan/": editor,

	"/qan/":        viewer,
	"/prometheus/": admin,

	// FIXME should be viewer, would leak info without any authentication
	"/v1/version":         none,
	"/v1/readyz":          none,
	"/managed/v1/version": none, // PMM 1.x variant
	"/ping":               none, // PMM 1.x variant

	// "/" is a special case
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

	// response body is ignored by nginx
	code := s.authenticate(ctx, req)
	rw.WriteHeader(code)
}

func (s *AuthServer) authenticate(ctx context.Context, req *http.Request) int {
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
		return 500
	}

	origURI := req.Header.Get("X-Original-Uri")
	if origURI == "" {
		l.Errorf("Empty X-Original-Uri.")
		return 500
	}
	l = l.WithField("req", fmt.Sprintf("%s %s", req.Header.Get("X-Original-Method"), origURI))

	// find the longest prefix present in rules:
	// /foo/bar -> /foo/ -> /foo -> /
	prefix := origURI
	for prefix != "/" {
		if _, ok := rules[prefix]; ok {
			break
		}

		if strings.HasSuffix(prefix, "/") {
			prefix = strings.TrimSuffix(prefix, "/")
		} else {
			prefix = path.Dir(prefix) + "/"
		}
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
		return 200
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
			return cErr.code
		}
		return 500
	}
	l = l.WithField("role", role.String())

	if role == grafanaAdmin {
		l.Debugf("Grafana admin, allowing access.")
		return 200
	}

	if minRole <= role {
		l.Debugf("Minimal required role is %q, granting access.", minRole)
		return 200
	}

	l.Warnf("Minimal required role is %q.", minRole)
	return 403
}
