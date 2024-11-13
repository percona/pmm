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

// Package grafana contains Grafana related functionality.
package grafana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
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
	"/v1/management/Role":                         admin,
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
	"/v1/readyz":            none,
	"/v1/leaderHealthCheck": none,
	"/ping":                 none, // PMM 1.x variant

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

var vmProxyPrefixes = []string{
	"/graph/api/datasources/proxy/1/api/v1/",
	"/graph/api/ds/query",
	"/graph/api/v1/labels",
	"/prometheus/api/v1/",
}

const vmProxyHeaderName = "X-Proxy-Filter"

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

// clientError contains authentication error response details.
type authError struct {
	code    codes.Code // error code for API client; not mapped to HTTP status code
	message string
}

// ErrInvalidUserID is returned when user ID is not valid.
var ErrInvalidUserID = errors.New("InvalidUserID")

// ErrCannotGetUserID is returned when we cannot retrieve user ID.
var ErrCannotGetUserID = errors.New("CannotGetUserID")

type cacheItem struct {
	u       authUser
	created time.Time
}

// clientInterface exist only to make fuzzing simpler.
type clientInterface interface {
	getAuthUser(context.Context, http.Header) (authUser, error)
}

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	c       clientInterface
	checker awsInstanceChecker
	db      *reform.DB
	l       *logrus.Entry

	cache map[string]cacheItem
	rw    sync.RWMutex

	accessControl *accessControl

	// TODO server metrics should be provided by middleware https://jira.percona.com/browse/PMM-4326
}

// NewAuthServer creates new AuthServer.
func NewAuthServer(c clientInterface, checker awsInstanceChecker, db *reform.DB) *AuthServer {
	return &AuthServer{
		c:       c,
		checker: checker,
		db:      db,
		l:       logrus.WithField("component", "grafana/auth"),
		cache:   make(map[string]cacheItem),
		accessControl: &accessControl{
			db: db,
		},
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
		rw.WriteHeader(http.StatusBadRequest)
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

	authUser, err := s.authenticate(ctx, req, l)
	if err != nil {
		// copy grpc-gateway behavior: set correct codes, set both "error" and "message"
		m := map[string]any{
			"code":    int(err.code),
			"error":   err.message,
			"message": err.message,
		}
		s.returnError(rw, m, l)
		return
	}

	var userID int
	if authUser != nil {
		userID = authUser.userID
	}

	if err := s.maybeAddVMProxyFilters(ctx, rw, req, userID, l); err != nil {
		// copy grpc-gateway behavior: set correct codes, set both "error" and "message"
		m := map[string]any{
			"code":    int(codes.Internal),
			"error":   "Internal server error.",
			"message": "Internal server error.",
		}
		l.Errorf("Failed to add VMProxy filters: %s", err)

		s.returnError(rw, m, l)
		return
	}
}

func (s *AuthServer) returnError(rw http.ResponseWriter, msg map[string]any, l *logrus.Entry) {
	// nginx completely ignores auth_request subrequest response body.
	// We respond with 401 (authenticationErrorCode); our nginx configuration then sends
	// the same request as a normal request to the same location and returns response body to the client.
	rw.Header().Set("Content-Type", "application/json")

	rw.WriteHeader(authenticationErrorCode)
	if err := json.NewEncoder(rw).Encode(msg); err != nil {
		l.Warnf("%s", err)
	}
}

// maybeAddVMProxyFilters adds extra filters to requests proxied through VMProxy.
// In case the request is not proxied through VMProxy, this is a no-op.
func (s *AuthServer) maybeAddVMProxyFilters(ctx context.Context, rw http.ResponseWriter, req *http.Request, userID int, l *logrus.Entry) error {
	if !s.shallAddVMProxyFilters(req) {
		return nil
	}

	if userID == 0 {
		l.Debugf("Getting authenticated user info")
		authUser, err := s.getAuthUser(ctx, req, l)
		if err != nil {
			return ErrCannotGetUserID
		}

		if authUser == nil {
			return fmt.Errorf("%w: user is empty", ErrCannotGetUserID)
		}

		userID = authUser.userID
	}

	if userID <= 0 {
		return ErrInvalidUserID
	}

	filters, err := s.getFiltersForVMProxy(userID)
	if err != nil {
		return err
	}

	if len(filters) == 0 {
		return nil
	}

	jsonFilters, err := json.Marshal(filters)
	if err != nil {
		return errors.WithStack(err)
	}

	rw.Header().Set(vmProxyHeaderName, base64.StdEncoding.EncodeToString(jsonFilters))

	return nil
}

func (s *AuthServer) shallAddVMProxyFilters(req *http.Request) bool {
	addFilters := false
	for _, p := range vmProxyPrefixes {
		if strings.HasPrefix(req.URL.Path, p) {
			addFilters = true
			break
		}
	}

	if !addFilters {
		return false
	}

	return s.accessControl.isEnabled()
}

func (s *AuthServer) getFiltersForVMProxy(userID int) ([]string, error) {
	roles, err := models.GetUserRoles(s.db.Querier, userID)
	if err != nil {
		return nil, err
	}

	// We may see this user for the first time.
	// If the role is not defined, we automatically assign a default role.
	if len(roles) == 0 {
		err := s.db.InTransaction(func(tx *reform.TX) error {
			s.l.Infof("Assigning default role to user ID %d", userID)
			return models.AssignDefaultRole(tx, userID)
		})
		if err != nil {
			return nil, err
		}

		// Reload roles
		roles, err = models.GetUserRoles(s.db.Querier, userID)
		if err != nil {
			return nil, err
		}
	}

	if len(roles) == 0 {
		logrus.Panicf("User %d has no roles", userID)
	}

	filters := make([]string, 0, len(roles))
	for _, r := range roles {
		if r.Filter == "" {
			// Special case when a user has assigned a role with no filters.
			// In this case it's irrelevant what other roles are assigned to the user.
			// The user shall have full access.
			return []string{}, nil
		}

		filters = append(filters, r.Filter)
	}
	return filters, nil
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

// authenticate checks if user has access to a specific path.
// It returns user information retrieved during authentication.
// Paths which require no Grafana role return zero value for
// some user fields such as authUser.userID.
func (s *AuthServer) authenticate(ctx context.Context, req *http.Request, l *logrus.Entry) (*authUser, *authError) {
	// Unescape the URL-encoded parts of the path.
	p := req.URL.Path
	cleanedPath, err := cleanPath(p)
	if err != nil {
		l.Warnf("Error while unescaping path %s: %q", p, err)
		return nil, &authError{
			code:    codes.Internal,
			message: "Internal server error.",
		}
	}

	// find the longest prefix present in rules
	prefix := cleanedPath
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
		return nil, nil
	}

	// Get authenticated user from Grafana
	authUser, authErr := s.getAuthUser(ctx, req, l)
	if authErr != nil {
		return nil, authErr
	}

	l = l.WithField("role", authUser.role.String())

	if authUser.role == grafanaAdmin {
		l.Debugf("Grafana admin, allowing access.")
		return authUser, nil
	}

	if minRole <= authUser.role {
		l.Debugf("Minimal required role is %q, granting access.", minRole)
		return authUser, nil
	}

	l.Warnf("Minimal required role is %q.", minRole)
	return nil, &authError{code: codes.PermissionDenied, message: "Access denied."}
}

func cleanPath(p string) (string, error) {
	unescaped, err := url.PathUnescape(p)
	if err != nil {
		return "", err
	}

	cleanedPath := path.Clean(unescaped)

	cleanedPath = strings.ReplaceAll(cleanedPath, "\n", " ")

	u, err := url.Parse(cleanedPath)
	if err != nil {
		return "", err
	}
	u.RawQuery = ""
	return u.String(), nil
}

func (s *AuthServer) getAuthUser(ctx context.Context, req *http.Request, l *logrus.Entry) (*authUser, *authError) {
	// check Grafana with some headers from request
	authHeaders := s.authHeaders(req)
	j, err := json.Marshal(authHeaders)
	if err != nil {
		l.Warnf("%s", err)
		return nil, &authError{code: codes.Internal, message: "Internal server error."}
	}
	hash := base64.StdEncoding.EncodeToString(j)
	s.rw.RLock()
	item, ok := s.cache[hash]
	s.rw.RUnlock()
	if ok {
		return &item.u, nil
	}

	return s.retrieveRole(ctx, hash, authHeaders, l)
}

func (s *AuthServer) authHeaders(req *http.Request) http.Header {
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

func (s *AuthServer) retrieveRole(ctx context.Context, hash string, authHeaders http.Header, l *logrus.Entry) (*authUser, *authError) {
	authUser, err := s.c.getAuthUser(ctx, authHeaders)
	if err != nil {
		l.Warnf("%s", err)
		if cErr, ok := errors.Cause(err).(*clientError); ok { //nolint:errorlint
			code := codes.Internal
			if cErr.Code == 401 || cErr.Code == 403 {
				code = codes.Unauthenticated
			}
			return nil, &authError{code: code, message: cErr.ErrorMessage}
		}
		return nil, &authError{code: codes.Internal, message: "Internal server error."}
	}
	s.rw.Lock()
	s.cache[hash] = cacheItem{
		u:       authUser,
		created: time.Now(),
	}
	s.rw.Unlock()

	return &authUser, nil
}
