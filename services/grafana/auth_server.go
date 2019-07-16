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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// rules maps original URL prefix to minimal required role.
// TODO https://jira.percona.com/browse/PMM-4338
var rules = map[string]role{
	"/v0/inventory/Nodes/List":    editor,
	"/v0/inventory/Nodes/Get":     editor,
	"/v0/inventory/Services/List": editor,
	"/v0/inventory/Services/Get":  editor,
	"/v0/inventory/Agents/List":   editor,
	"/v0/inventory/Agents/Get":    editor,

	"/v0/inventory/": admin,

	"/": admin, // fail-safe
}

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	c             *Client
	l             *logrus.Entry
	skipAuthCheck bool // FIXME Remove after https://jira.percona.com/browse/PMM-4338

	// TODO server metrics should be provided by middleware https://jira.percona.com/browse/PMM-4326
}

// NewAuthServer creates new AuthServer.
func NewAuthServer(c *Client) *AuthServer {
	return &AuthServer{
		c:             c,
		l:             logrus.WithField("component", "grafana/auth"),
		skipAuthCheck: true, // FIXME https://jira.percona.com/browse/PMM-4338
	}
}

// ServeHTTP serves internal location /auth_request.
func (s *AuthServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// fail-safe
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	if err := s.authenticate(ctx, req); err != nil {
		switch e := err.(type) {
		case *apiError:
			s.l.Warnf("%+v", err)
			rw.WriteHeader(e.code)
		default:
			s.l.Errorf("%+v", err)
			rw.WriteHeader(500)
		}
	}
}

func (s *AuthServer) authenticate(ctx context.Context, req *http.Request) error {
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
		return errors.Errorf("Unexpected path %s.", req.URL.Path)
	}

	uri := req.Header.Get("X-Original-Uri")
	if uri == "" {
		return errors.Errorf("Empty X-Original-Uri.")
	}
	l = l.WithField("req", fmt.Sprintf("%s %s", req.Header.Get("X-Original-Method"), uri))

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
	l = l.WithField("role", role.String())
	if err != nil {
		if s.skipAuthCheck {
			l.Warnf("Not authenticated, but authenticating anyway: %v", err)
			err = nil
		}
		return err
	}

	if role == grafanaAdmin {
		l.Debugf("Grafana admin, allowing access.")
		return nil
	}

	// find the longest prefix present in rules
	for {
		if _, ok := rules[uri]; ok {
			break
		}
		uri = path.Dir(uri)
	}
	minRole := rules[uri]
	if minRole <= role {
		l.Debugf("Minimal required role is %q, granting access.", minRole)
		return nil
	}

	if s.skipAuthCheck {
		l.Warnf("Minimal required role is %q, but authenticating anyway.", minRole)
		return nil
	}
	return &apiError{
		code: 403,
		body: fmt.Sprintf("Minimal required role is %q, actual role is %q.", minRole, role),
	}
}
