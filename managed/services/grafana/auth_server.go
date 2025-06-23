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

const (
	connectionEndpoint = "/agent.v1.AgentService/Connect"
)

// rules maps original URL prefix to minimal required role.
var rules = map[string]role{
	// TODO https://jira.percona.com/browse/PMM-4420
	"/agent.Agent/Connect": admin, // compatibility for v2 agents
	connectionEndpoint:     admin,

	"/inventory.":                               admin,
	"/management.":                              admin,
	"/actions.":                                 viewer,
	"/advisors.v1.":                             editor,
	"/server.v1.ServerService/CheckUpdates":     viewer,
	"/server.v1.ServerService/UpdateStatus":     none,  // special token-based auth
	"/server.v1.ServerService/AWSInstanceCheck": none,  // special case - used before Grafana can be accessed
	"/server.":                                  admin, // TODO: do we need it for older agents?
	"/server.v1.":                               admin,
	"/qan.v1.CollectorService.":                 viewer,
	"/qan.v1.QANService.":                       viewer,

	"/v1/alerting":                    viewer,
	"/v1/advisors":                    editor,
	"/v1/advisors/checks:":            editor,
	"/v1/advisors/failedServices":     editor,
	"/v1/actions/":                    viewer,
	"/v1/actions:":                    viewer,
	"/v1/backups":                     admin,
	"/v1/dumps":                       admin,
	"/v1/accesscontrol":               admin,
	"/v1/inventory/":                  admin,
	"/v1/inventory/services:getTypes": viewer,
	"/v1/management/":                 admin,
	"/v1/management/Jobs":             viewer,
	"/v1/server/AWSInstance":          none, // special case - used before Grafana can be accessed
	"/v1/server/updates":              viewer,
	"/v1/server/updates:start":        admin,
	"/v1/server/updates:getStatus":    none, // special token-based auth
	"/v1/server/settings":             admin,
	"/v1/server/settings/readonly":    viewer,
	"/v1/platform:":                   admin,
	"/v1/platform/":                   viewer,
	"/v1/users":                       viewer,

	// must be available without authentication for health checking
	"/v1/server/readyz":            none,
	"/v1/server/leaderHealthCheck": none,
	"/ping":                        none, // PMM 1.x variant

	// must not be available without authentication as it can leak data
	"/v1/server/version": viewer,

	"/v1/qan":  viewer,
	"/v1/qan:": viewer,

	"/prometheus":      admin,
	"/victoriametrics": admin,
	"/nomad":           admin,
	"/graph":           none,
	"/swagger":         viewer,

	"/v1/mcp": none, // TODO: remove this once we have a proper auth for mcp

	// AI Chat API - requires viewer role for basic access
	"/v1/chat/":       viewer,
	"/v1/chat/health": none, // health check doesn't require auth

	"/v1/server/logs.zip": admin,

	// kept for backwards compatibility with PMM v2
	"/v1/readyz":  none,   // redirects to /v1/server/readyz
	"/v1/version": viewer, // redirects to /v1/server/version
	"/logs.zip":   admin,  // redirects to /v1/server/logs.zip

	// "/auth_request"  has auth_request disabled in nginx config

	// "/" is a special case in this code
}

var lbacPrefixes = []string{
	"/graph/api/datasources/uid",
	"/graph/api/ds/query",
	// "/graph/api/v1/labels", // Note: this path appears not to be used in Grafana
	"/prometheus/api/v1/",
	"/v1/qan/",
	"/graph/api/datasources/proxy/1/api/v1/", // https://github.com/grafana/grafana/blob/146c3120a79e71e9a4836ddf1e1dc104854c7851/public/app/core/utils/query.ts#L35
}

const lbacHeaderName = "X-Proxy-Filter"

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
	getAuthUser(ctx context.Context, authHeaders http.Header, l *logrus.Entry) (authUser, error)
}

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	c  clientInterface
	db *reform.DB
	l  *logrus.Entry

	cache map[string]cacheItem
	rw    sync.RWMutex

	accessControl *accessControl

	// TODO server metrics should be provided by middleware https://jira.percona.com/browse/PMM-4326
}

// NewAuthServer creates new AuthServer.
func NewAuthServer(c clientInterface, db *reform.DB) *AuthServer {
	return &AuthServer{
		c:     c,
		db:    db,
		l:     logrus.WithField("component", "grafana/auth"),
		cache: make(map[string]cacheItem),
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

	// Set X-User-ID header for AI Chat endpoints
	if s.isAIChatEndpoint(req) && userID > 0 {
		rw.Header().Set("X-User-ID", fmt.Sprintf("%d", userID))
		l.Infof("Set X-User-ID header: %d for AI Chat endpoint", userID)
	}

	if err := s.maybeAddLBACFilters(ctx, rw, req, userID, l); err != nil {
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

// maybeAddLBACFilters adds extra filters to requests proxied through VMProxy.
// In case the request is not proxied through VMProxy, this is a no-op.
func (s *AuthServer) maybeAddLBACFilters(ctx context.Context, rw http.ResponseWriter, req *http.Request, userID int, l *logrus.Entry) error {
	if !s.shallAddLBACFilters(req) {
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

	filters, err := s.getLBACFilters(ctx, userID)
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

	rw.Header().Set(lbacHeaderName, base64.StdEncoding.EncodeToString(jsonFilters))

	return nil
}

// shallAddLBACFilters decides if LBAC filters must be added to the outgoing request.
func (s *AuthServer) shallAddLBACFilters(req *http.Request) bool {
	if !s.accessControl.isEnabled() {
		return false
	}

	for _, p := range lbacPrefixes {
		if strings.HasPrefix(req.URL.Path, p) {
			return true
		}
	}

	return false
}

// getLBACFilters retrieves LBAC filters for the user.
func (s *AuthServer) getLBACFilters(ctx context.Context, userID int) ([]string, error) {
	roles, err := models.GetUserRoles(s.db.Querier, userID)
	if err != nil {
		return nil, err
	}

	// We may see this user for the first time.
	// If the role is not defined, we automatically assign a default role.
	if len(roles) == 0 {
		err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

func isLocalAgentConnection(req *http.Request) bool {
	ip := strings.Split(req.RemoteAddr, ":")[0]
	pmmAgent := req.Header.Get("Pmm-Agent-Id")
	path := req.Header.Get("X-Original-Uri")
	if ip == "127.0.0.1" && pmmAgent == "pmm-server" && path == connectionEndpoint {
		return true
	}

	return false
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

	var user *authUser
	if isLocalAgentConnection(req) {
		user = &authUser{
			role:   rules[connectionEndpoint],
			userID: 0,
		}
	} else {
		var authErr *authError
		// Get authenticated user from Grafana
		user, authErr = s.getAuthUser(ctx, req, l)
		if authErr != nil {
			return nil, authErr
		}
	}
	l = l.WithField("role", user.role.String())

	if user.role == grafanaAdmin {
		l.Debugf("Grafana admin, allowing access.")
		return user, nil
	}

	if minRole <= user.role {
		l.Debugf("Minimal required role is %q, granting access.", minRole)
		return user, nil
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
	authUser, err := s.c.getAuthUser(ctx, authHeaders, l)
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

// isAIChatEndpoint checks if the request is for an AI Chat endpoint
func (s *AuthServer) isAIChatEndpoint(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, "/v1/chat/")
}
