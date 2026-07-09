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
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	connectionEndpointV2 = "/agent.Agent/Connect"
	connectionEndpoint   = "/agent.v1.AgentService/Connect"
	rtaCollectEndpoint   = "/realtimeanalytics.v1.CollectorService/Collect"
)

// rules maps original URL prefix to minimal required role.
// In case of multiple matches, the longest prefix wins. The prefix is matched
// against the original URL path, not the cleaned one. The prefix must end with a slash,
// dot, or colon, so that "/v1/inventory" does not match "/v1/inventoryX".
// If several methods share the same path, they must be distinguished by methodRules.
var rules = map[string]role{
	// TODO https://jira.percona.com/browse/PMM-4420
	connectionEndpointV2: admin, // compatibility for v2 agents
	connectionEndpoint:   admin,

	"/inventory.":                           admin,
	"/management.":                          admin,
	"/actions.":                             viewer,
	"/advisors.v1.":                         editor,
	"/server.v1.ServerService/CheckUpdates": viewer,
	"/server.v1.ServerService/AWSInstanceCheck": none, // special case - used before Grafana can be accessed
	"/server.":                  admin, // TODO: do we need it for older agents?
	"/server.v1.":               admin,
	"/qan.v1.CollectorService.": viewer,
	"/qan.v1.QANService.":       viewer,

	"/v1/alerting":                    viewer,
	"/v1/alerting/rules":              editor,
	"/v1/advisors":                    editor,
	"/v1/advisors/checks:":            editor,
	"/v1/advisors/failedServices":     editor,
	"/v1/actions":                     viewer,
	"/v1/actions:":                    viewer,
	"/v1/backups":                     admin,
	"/v1/dumps":                       admin,
	"/v1/accesscontrol":               admin,
	"/v1/ha":                          viewer,
	"/v1/inventory":                   admin,
	"/v1/inventory/services:getTypes": viewer,
	"/v1/management":                  admin,
	"/v1/management/Jobs":             viewer,
	"/v1/server/AWSInstance":          none, // special case - used before Grafana can be accessed
	"/v1/server/updates":              viewer,
	"/v1/server/settings":             admin,
	"/v1/server/settings/readonly":    viewer,
	"/v1/platform:":                   admin,
	"/v1/platform":                    viewer,
	"/v1/users":                       viewer,
	"/v1/users/current":               none,
	"/v1/users/current/orgs":          none,

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

	"/v1/server/logs.zip": admin,

	// kept for backwards compatibility with PMM v2
	"/v1/readyz":  none,   // redirects to /v1/server/readyz
	"/v1/version": viewer, // redirects to /v1/server/version
	"/logs.zip":   admin,  // redirects to /v1/server/logs.zip

	// Real-Time Analytics endpoints.
	rtaCollectEndpoint:                     admin,
	"/v1/realtimeanalytics/sessions:start": admin,
	"/v1/realtimeanalytics/sessions:stop":  admin,
	"/v1/realtimeanalytics/sessions":       viewer,
	"/v1/realtimeanalytics/services":       viewer,
	"/v1/realtimeanalytics/queries:search": viewer,

	// "/auth_request"  has auth_request disabled in nginx config

	// "/" is a special case in this code
}

// methodRules maps "METHOD url-prefix" to the minimal role. Entries take precedence
// over rules, letting operations on a shared path differ by HTTP method.
var methodRules = map[string]role{
	// Template writes need editor; they share paths with the viewer-readable
	// list (POST) or sit under it (PUT/DELETE), so they're qualified by method.
	http.MethodPost + " /v1/alerting/templates":    editor,
	http.MethodPut + " /v1/alerting/templates/":    editor,
	http.MethodDelete + " /v1/alerting/templates/": editor,
}

var lbacPrefixes = []string{
	"/graph/api/datasources/uid",
	"/graph/api/ds/query",
	// "/graph/api/v1/labels", // Note: this path appears not to be used in Grafana
	"/prometheus/api/v1/",
	"/v1/qan/",
	// https://github.com/grafana/grafana/blob/146c3120a79e71e9a4836ddf1e1dc104854c7851/public/app/core/utils/query.ts#L35
	"/graph/api/datasources/proxy/1/api/v1/",
}

const lbacHeaderName = "X-Proxy-Filter"

// nginx auth_request directive supports only 401 and 403 - every other code results in 500.
// Our APIs can return codes.PermissionDenied which maps to 403 / http.StatusForbidden.
// Our APIs MUST NOT return codes.Unauthenticated which maps to 401 / http.StatusUnauthorized
// as this code is reserved for auth_request.
const authenticationErrorCode = 401

const (
	// Note: cacheInvalidationInterval is used to invalidate cache for grafana responses.
	cacheInvalidationInterval = 3 * time.Second
	authenticationTimeout     = 3 * time.Second
)

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
	t := time.NewTicker(cacheInvalidationInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			now := time.Now()
			s.rw.Lock()
			for key, item := range s.cache {
				if now.Add(-cacheInvalidationInterval).After(item.created) {
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

	err := extractOriginalRequest(req)
	if err != nil {
		s.l.Warnf("Failed to parse request: %s.", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	l := s.l.WithField("req", fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	// TODO l := logger.Get(ctx) once we have it after https://jira.percona.com/browse/PMM-4326

	ctx, cancel := context.WithTimeout(req.Context(), authenticationTimeout)
	defer cancel()

	authUser, authErr := s.authenticate(ctx, req, l)
	if authErr != nil {
		// copy grpc-gateway behavior: set correct codes, set both "error" and "message"
		m := map[string]any{
			"code":    int(authErr.code),
			"error":   authErr.message,
			"message": authErr.message, //nolint:goconst
		}
		s.returnError(rw, httpStatusForAuthError(authErr.code), m, l)
		return
	}

	var userID int
	if authUser != nil {
		userID = authUser.userID
	}

	errF := s.maybeAddLBACFilters(ctx, rw, req, userID, l)
	if errF != nil {
		// copy grpc-gateway behavior: set correct codes, set both "error" and "message"
		m := map[string]any{
			"code":    int(codes.Internal),
			"error":   "Internal server error.",
			"message": "Internal server error.",
		}
		l.Errorf("Failed to add VMProxy filters: %s", errF)

		s.returnError(rw, authenticationErrorCode, m, l)
		return
	}
}

// httpStatusForAuthError maps an authError code to the HTTP status nginx receives.
// PermissionDenied uses 403 so nginx denies outright; the 401 re-run is a GET and would
// wrongly pass method-specific rules. Authentication and internal errors stay 401.
func httpStatusForAuthError(code codes.Code) int {
	if code == codes.PermissionDenied {
		return http.StatusForbidden
	}
	return authenticationErrorCode
}

func (s *AuthServer) returnError(rw http.ResponseWriter, status int, msg map[string]any, l *logrus.Entry) {
	// nginx ignores the auth_request subrequest body: on 401 it re-runs the request to
	// /auth_request to fetch this body; on 403 it serves a static body via error_page 403.
	rw.Header().Set("Content-Type", "application/json")

	rw.WriteHeader(status)
	err := json.NewEncoder(rw).Encode(msg)
	if err != nil {
		l.Warnf("%s", err)
	}
}

// maybeAddLBACFilters adds extra filters to requests proxied through VMProxy.
// In case the request is not proxied through VMProxy, this is a no-op.
func (s *AuthServer) maybeAddLBACFilters(ctx context.Context, rw http.ResponseWriter, req *http.Request, userID int, l *logrus.Entry) error {
	if !s.shallAddLBACFilters(req) {
		l.Debugf("Skipping LBAC filters for non-proxied request.")
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
		// Anonymous users don't have a numeric user ID and cannot have LBAC roles.
		// Skip adding filters and allow the request to proceed.
		l.Debugf("Skipping LBAC filters for anonymous user.")
		return nil
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
		return fmt.Errorf("failed to marshal LBAC filters: %w", err)
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
			// Handle race condition: if another concurrent request already assigned the default role,
			// we'll get a duplicate key error. In this case, just go fetch the roles.
			var pgErr *pq.Error
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.Constraint == "user_roles_pkey" {
				s.l.Debugf("Default role already assigned to user ID %d by another request", userID)
			} else {
				return nil, err
			}
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
		return fmt.Errorf("unexpected X-Original-Uri: %s", origURI)
	}
	if !utf8.ValidString(origURI) {
		return fmt.Errorf("invalid X-Original-Uri: %s", origURI)
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

func isLocalAgentConnection(req *http.Request) bool {
	ip := strings.Split(req.RemoteAddr, ":")[0]
	// pmmAgent := req.Header.Get("Pmm-Agent-Id")
	path := req.Header.Get("X-Original-Uri")
	if ip == "127.0.0.1" &&
		(path == connectionEndpoint || path == rtaCollectEndpoint) {
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

	minRole, prefix := resolveRule(req.Method, cleanedPath, l)
	l = l.WithField("prefix", prefix)

	if minRole == none {
		l.Debugf("Minimal required role is %s, granting access without checking Grafana.", minRole)
		return nil, nil
	}

	var user *authUser
	if isLocalAgentConnection(req) {
		if req.Header.Get("X-Original-Uri") == connectionEndpoint {
			user = &authUser{
				role:   rules[connectionEndpoint],
				userID: 0,
			}
		} else {
			user = &authUser{
				role:   rules[rtaCollectEndpoint],
				userID: 0,
			}
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
		l.Debugf("Grafana admin, granting access.")
		return user, nil
	}

	if minRole <= user.role {
		l.Debugf("Minimal required role is %s, granting access.", minRole)
		return user, nil
	}

	l.Warnf("Minimal required role is %s, denying access.", minRole)
	return nil, &authError{code: codes.PermissionDenied, message: "Access denied"}
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
		cErr, ok := errors.AsType[*clientError](err)
		if ok {
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
