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
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// rules maps original URL prefix to minimal required role.
var rules = map[string]role{
	"/agent.Agent/Connect": none,

	"/v0/inventory/Agents/Get":    editor,
	"/v0/inventory/Agents/List":   editor,
	"/v0/inventory/Nodes/Get":     editor,
	"/v0/inventory/Nodes/List":    editor,
	"/v0/inventory/Services/Get":  editor,
	"/v0/inventory/Services/List": editor,
	"/v0/inventory/":              admin,

	"/v0/management/": admin,

	"/v0/qan/": editor,

	"/v1/ChangeSettings": admin,
	"/v1/GetSettings":    admin,

	"/v1/version":         viewer,
	"/managed/v1/version": viewer, // PMM 1.x variant
	"/ping":               viewer, // would leak info without any authentication

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

	if err := s.authenticate(ctx, req); err != nil {
		switch e := err.(type) {
		case *apiError:
			// Add both error and message to mirror gRPC/grpc-gateway errors:
			// https://github.com/grpc-ecosystem/grpc-gateway/blob/bebc7374a79e1105d786ef3468b474e47d652511/runtime/errors.go#L67-L75
			m := map[string]interface{}{
				"code":    e.code,
				"error":   e.body,
				"message": e.body,
			}
			rw.WriteHeader(e.code)
			if err = json.NewEncoder(rw).Encode(m); err != nil {
				s.l.Warnf("Failed to encode apiError: %s.", err)
			}
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
		return err
	}

	if role == grafanaAdmin {
		l.Debugf("Grafana admin, allowing access.")
		return nil
	}

	// find the longest prefix present in rules:
	// /foo/bar -> /foo/ -> /foo -> /
	for uri != "/" {
		if _, ok := rules[uri]; ok {
			break
		}

		if strings.HasSuffix(uri, "/") {
			uri = strings.TrimSuffix(uri, "/")
		} else {
			uri = path.Dir(uri) + "/"
		}
	}
	l = l.WithField("uri", uri)

	if uri == "/" {
		l.Error("Unhandled URI.")
		return &apiError{
			code: 403,
			body: "Forbidden.",
		}
	}

	minRole := rules[uri]
	if minRole <= role {
		l.Debugf("Minimal required role is %q, granting access.", minRole)
		return nil
	}

	l.Warnf("Minimal required role is %q.", minRole)
	return &apiError{
		code: 403,
		body: "Forbidden.",
	}
}
