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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

// rules maps original URL prefix to minimal required role.
var rules = map[string]role{
	// TODO https://jira.percona.com/browse/PMM-4420
	"/agent.Agent/Connect": none,

	"/inventory.":                     admin,
	"/management.":                    admin,
	"/management.Actions/":            viewer,
	"/server.Server/CheckUpdates":     viewer,
	"/server.Server/UpdateStatus":     none, // special token-based auth
	"/server.Server/AWSInstanceCheck": none, // special case - used before Grafana can be accessed
	"/server.":                        admin,

	"/v1/inventory/":                              admin,
	"/v1/inventory/Services/ListTypes":            viewer,
	"/v1/management/":                             admin,
	"/v1/management/Actions/":                     viewer,
	"/v1/management/Jobs":                         viewer,
	"/v1/Updates/Check":                           viewer,
	"/v1/Updates/Status":                          none, // special token-based auth
	"/v1/AWSInstanceCheck":                        none, // special case - used before Grafana can be accessed
	"/v1/Updates/":                                admin,
	"/v1/Settings/":                               admin,
	"/v1/Platform/Connect":                        admin,
	"/v1/Platform/Disconnect":                     admin,
	"/v1/Platform/SearchOrganizationTickets":      viewer,
	"/v1/Platform/SearchOrganizationEntitlements": viewer,
	"/v1/Platform/GetContactInformation":          viewer,
	"/v1/Platform/ServerInfo":                     viewer,
	"/v1/Platform/UserStatus":                     viewer,

	"/v1/user": viewer,

	// must be available without authentication for health checking
	"/v1/readyz": none,
	"/ping":      none, // PMM 1.x variant

	// must not be available without authentication as it can leak data
	"/v1/version":         viewer,
	"/managed/v1/version": viewer, // PMM 1.x variant

	"/v0/qan/": viewer,

	// mustSetupRules group
	"/prometheus":      admin,
	"/victoriametrics": admin,
	"/alertmanager":    admin,
	"/graph":           none,
	"/swagger":         none,

	"/logs.zip": admin,
	// "/auth_request" and "/setup" have auth_request disabled in nginx config

	// "/" is a special case in this code
}

// Only UI is blocked by setup wizard; APIs can be used.
// Critically, AWSInstanceCheck must be available for the setup wizard itself to work;
// and /agent.Agent/Connect and Management APIs should be available for pmm-agent on PMM Server registration.
var mustSetupRules = []string{
	"/prometheus",
	"/victoriametrics",
	"/alertmanager",
	"/graph",
	"/swagger",
}

// nginx auth_request directive supports only 401 and 403 - every other code results in 500.
// Our APIs can return codes.PermissionDenied which maps to 403 / http.StatusForbidden.
// Our APIs MUST NOT return codes.Unauthenticated which maps to 401 / http.StatusUnauthorized
// as this code is reserved for auth_request.
const authenticationErrorCode = 401

// cacheInvalidationPeriod is and period when cache for grafana response should be invalidated.
const cacheInvalidationPeriod = 3 * time.Second

// Base64 encoded jwt token providing full access via vmgateway with long expiration.
const jwtFullAccess = "eyJ0eXAiOiJKV1QiLCJhbGciOiJQUzI1NiIsImtpZCI6IjI1OTMzN2RiLTc0MTItNDVkYS1hZDg2LWI2M2M5Nzc5NjU4OCJ9.eyJpYXQiOjE2NjE3NjM3MzUsIm5iZiI6MTY2MTc2MzczNSwiZXhwIjo0ODE1MzYzNzQwLCJqdGkiOiJ2eTU5VzRFMXJQVjk0OGVyazNCOTYiLCJ2bV9hY2Nlc3MiOnsidGVuYW5kX2lkIjp7fX19.yckDKjbFrnnBNlxN5Cjlk8fSbgKc2KzToJVbfQw_rOqSgdl_lNe7TDEtQ7NG2BITJW9rxbN4vdAYY-7gQ2loEz5Ev9YIl1QnzM553Eyw6HcES0JGjU2b9Pn7LXDlAigORoeX_3BryG7LXMUv7Cz2zfjrfaYuStBX1Jcv13BayGDeVpUAJ4s1IPlDwMpHvxqvGH4OjsxugdVIyfumhEvNpAx6kdy8SUABSsmh5Hb2yuIhyW0LKiSMgoCxegjShUFwiark7WkUb7APrO4yYRUV6DYnaYvP8HmV1ItD-wvl_2NDHe9xnUJXx0zMrszl36V5H15_5IDJdAccCAO7CaZr7g" //nolint:lll

// clientError contains authentication error response details.
type authError struct {
	code    codes.Code // error code for API client; not mapped to HTTP status code
	message string
}

type cacheItem struct {
	r       role
	created time.Time
}

// clientInterface exist only to make fuzzing simpler.
type clientInterface interface {
	getRole(context.Context, http.Header) (role, error)
}

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	c       clientInterface
	checker awsInstanceChecker
	l       *logrus.Entry

	cache map[string]cacheItem
	rw    sync.RWMutex

	// TODO server metrics should be provided by middleware https://jira.percona.com/browse/PMM-4326
}

// NewAuthServer creates new AuthServer.
func NewAuthServer(c clientInterface, checker awsInstanceChecker) *AuthServer {
	return &AuthServer{
		c:       c,
		checker: checker,
		l:       logrus.WithField("component", "grafana/auth"),
		cache:   make(map[string]cacheItem),
	}
}

// Run runs cache invalidator which removes expired cache items.
func (s *AuthServer) Run(ctx context.Context) {
	t := time.NewTicker(cacheInvalidationPeriod)

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			now := time.Now()
			s.rw.Lock()
			for key, item := range s.cache {
				if now.Add(-cacheInvalidationPeriod).After(item.created) {
					delete(s.cache, key)
				}
			}
			s.rw.Unlock()
		}
	}
}

// ServeHTTP serves internal location /auth_request for both authentication subrequests
// and subsequent normal requests.
func (s *AuthServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if s.l.Logger.GetLevel() >= logrus.DebugLevel {
		b, err := httputil.DumpRequest(req, true)
		if err != nil {
			s.l.Errorf("Failed to dump request: %v.", err)
		}
		s.l.Debugf("Request:\n%s", b)
	}

	if err := extractOriginalRequest(req); err != nil {
		s.l.Warnf("Failed to parse request: %s.", err)
		rw.WriteHeader(400)
		return
	}

	l := s.l.WithField("req", fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	// TODO l := logger.Get(ctx) once we have it after https://jira.percona.com/browse/PMM-4326

	if s.mustSetup(rw, req, l) {
		return
	}

	// fail-safe
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	if err := s.authenticate(ctx, req, l); err != nil {
		// nginx completely ignores auth_request subrequest response body.
		// We respond with 401 (authenticationErrorCode); our nginx configuration then sends
		// the same request as a normal request to the same location and returns response body to the client.

		// copy grpc-gateway behavior: set correct codes, set both "error" and "message"
		m := map[string]interface{}{
			"code":    int(err.code),
			"error":   err.message,
			"message": err.message,
		}
		rw.Header().Set("Content-Type", "application/json")

		rw.WriteHeader(authenticationErrorCode)
		if err := json.NewEncoder(rw).Encode(m); err != nil {
			l.Warnf("%s", err)
		}
	}

	// Adds a header indicating full access to all metrics when used with vmgateway.
	rw.Header().Set("x-percona-vmgateway-token", "Bearer "+jwtFullAccess)
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
		return errors.Errorf("unexpected X-Original-Uri: %q", origURI)
	}
	if !utf8.ValidString(origURI) {
		return errors.Errorf("invalid X-Original-Uri: %q", origURI)
	}

	req.Method = origMethod
	req.URL.Path = origURI
	return nil
}

// mustSetup returns true if AWS instance ID must be checked.
func (s *AuthServer) mustSetup(rw http.ResponseWriter, req *http.Request, l *logrus.Entry) bool {
	// Only UI is blocked by setup wizard; APIs can be used.
	var found bool
	for _, r := range mustSetupRules {
		if strings.HasPrefix(req.URL.Path, r) {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// This header is used to pass information that setup is required from auth_request subrequest
	// to normal request to return redirect with location - something that auth_request can't do.
	const mustSetupHeader = "X-Must-Setup"

	// Redirect to /setup page.
	if req.Header.Get(mustSetupHeader) != "" {
		const redirectCode = 303 // temporary, not cacheable, always GET
		l.Warnf("AWS instance ID must be checked, returning %d with Location.", redirectCode)
		rw.Header().Set("Location", "/setup")
		rw.WriteHeader(redirectCode)
		return true
	}

	// Use X-Test-Must-Setup header for testing.
	// There is no way to skip check, only to enforce it.
	mustCheck := s.checker.MustCheck()
	if req.Header.Get("X-Test-Must-Setup") != "" {
		l.Debug("X-Test-Must-Setup is present, enforcing AWS instance ID check.")
		mustCheck = true
	}

	if mustCheck {
		l.Warnf("AWS instance ID must be checked, returning %d with %s.", authenticationErrorCode, mustSetupHeader)
		rw.Header().Set(mustSetupHeader, "1") // any non-empty value is ok
		rw.WriteHeader(authenticationErrorCode)
		return true
	}

	return false
}

// nextPrefix returns path's prefix, stopping on slashes and dots:
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

	i := strings.LastIndexAny(path, "/.")
	return path[:i+1]
}

func (s *AuthServer) authenticate(ctx context.Context, req *http.Request, l *logrus.Entry) *authError {
	// find the longest prefix present in rules
	prefix := req.URL.Path
	for prefix != "/" {
		if _, ok := rules[prefix]; ok {
			break
		}
		prefix = nextPrefix(prefix)
	}

	// fallback to Grafana admin if there is no explicit rule
	minRole, ok := rules[prefix]
	if ok {
		l = l.WithField("prefix", prefix)
	} else {
		l.Warn("No explicit rule, falling back to Grafana admin.")
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
	j, err := json.Marshal(authHeaders)
	if err != nil {
		l.Warnf("%s", err)
		return &authError{code: codes.Internal, message: "Internal server error."}
	}
	hash := base64.StdEncoding.EncodeToString(j)
	var role role
	s.rw.RLock()
	item, ok := s.cache[hash]
	s.rw.RUnlock()
	if ok {
		role = item.r
	} else {
		role, err = s.c.getRole(ctx, authHeaders)
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
		s.rw.Lock()
		s.cache[hash] = cacheItem{
			r:       role,
			created: time.Now(),
		}
		s.rw.Unlock()
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
