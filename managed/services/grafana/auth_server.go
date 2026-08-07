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
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/cache"
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
	// Handled in NGINX config.
	"/v1/server/readyz":            none,
	"/v1/server/leaderHealthCheck": none,

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

const (
	// Nginx auth_request directive supports only 401 and 403 - every other code results in 500.
	// Our APIs can return codes.PermissionDenied which maps to 403 / http.StatusForbidden.
	// Our APIs MUST NOT return codes.Unauthenticated which maps to 401 / http.StatusUnauthorized
	// as this code is reserved for auth_request.
	authenticationErrorCode = 401
	// HTTP headers used to pass auth error details back to NGINX.
	// The same headers are parsed in NGINX configuration file.
	authResponseCodeHeader    = "X-Auth-Code"
	authResponseErrorHeader   = "X-Auth-Error"
	authResponseMessageHeader = "X-Auth-Message"
)

const (
	authenticationTimeout = 15 * time.Second
	prometheusNamespace   = "pmm_managed"
	prometheusSubsystem   = "auth"
)

const (
	// Ttl for auth response validiness in auth cache.
	cacheItemTTL = 60 * time.Second
	// Auth response cache cleanup interval.
	cacheInvalidationInterval = 2 * cacheItemTTL
)

// authResult contains authentication response details that is a result of all
// authentication and authorization (including LBAC) checks.
type authResult struct {
	// encoded filers to be added as proxy headers.
	vmProxyFilters string
}

// clientError contains authentication error response details.
type authError struct {
	code    codes.Code // error code for API client; not mapped to HTTP status code
	message string
}

func (a *authError) Error() string {
	return fmt.Sprintf("%s: %s", a.message, a.code)
}

type authMetrics struct {
	// mAuthRequests tracks total auth requests by method, route, and response code.
	mAuthRequests *prom.CounterVec
	// mGrafanaAuthRequests tracks auth requests made to Grafana by response code.
	mGrafanaAuthRequests *prom.CounterVec
	// mCache tracks total authentication cache requests by status (hit or miss).
	mCache *prom.CounterVec
	// mCacheSizeDesc is the descriptor for the number of items in the auth cache.
	mCacheSizeDesc *prom.Desc
	// mDurations tracks latency of auth operations (labels: total, grafana, db).
	mDurations *prom.HistogramVec
}

type cachedAuthUser struct {
	user          authUser
	authorization string
	cookie        string
}

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	// c is the client used to interact with the Grafana API.
	c grafanaAuthUserGetter
	// db is the PostgreSQL database handle using reform ORM.
	db *reform.DB
	// l is the structured logger for the auth component.
	l *logrus.Entry

	// cache stores authentication responses to reduce Grafana API calls.
	// Stores positive responses only.
	// TODO: cache negative response as well.
	cache *cache.Cache[uint64, cachedAuthUser]
	// authUserGroup deduplicates concurrent Grafana auth lookups for the same auth header set.
	authUserGroup singleflight.Group

	// accessControl manages RBAC and LBAC filtering logic.
	accessControl accessControl

	// TODO server metrics should be provided by middleware https://jira.percona.com/browse/PMM-4326
	// Prometheus metrics for the AuthServer.
	metrics authMetrics
}

// NewAuthServer creates new AuthServer.
func NewAuthServer(ctx context.Context, c grafanaAuthUserGetter, db *reform.DB) *AuthServer {
	cache, err := cache.New[uint64, cachedAuthUser](ctx, cacheItemTTL, cacheInvalidationInterval)
	if err != nil {
		panic(err)
	}
	s := &AuthServer{
		c:  c,
		db: db,
		l:  logrus.WithField("component", "grafana/auth"),
		accessControl: &accessControlCache{
			db: db,
		},
		cache: cache,
		metrics: authMetrics{
			mAuthRequests: prom.NewCounterVec(
				prom.CounterOpts{
					Name: prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "requests_total"),
					Help: "Total number of authentication requests.",
				},
				[]string{"method", "route", "status_code"},
			),
			mGrafanaAuthRequests: prom.NewCounterVec(
				prom.CounterOpts{
					Name: prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "grafana_requests_total"),
					Help: "Total number of authentication requests to Grafana.",
				},
				[]string{"status_code"},
			),
			mCache: prom.NewCounterVec(
				prom.CounterOpts{
					Name: prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "cache_total"),
					Help: "Total number of authentication cache requests by status (hit or miss).",
				},
				[]string{"status"},
			),
			mCacheSizeDesc: prom.NewDesc(
				prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "cache_size"),
				"Total number of items in the authentication cache.",
				nil,
				nil,
			),
			mDurations: prom.NewHistogramVec(prom.HistogramOpts{ // labels: total, grafana, db
				Name:    prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "duration_seconds"),
				Help:    "Latency of authentication operations in seconds.",
				Buckets: prom.DefBuckets,
			}, []string{"type"}),
		},
	}
	return s
}

// Describe implements prom.Collector interface.
func (s *AuthServer) Describe(ch chan<- *prom.Desc) {
	s.metrics.mAuthRequests.Describe(ch)
	s.metrics.mGrafanaAuthRequests.Describe(ch)
	s.metrics.mCache.Describe(ch)
	ch <- s.metrics.mCacheSizeDesc
	s.metrics.mDurations.Describe(ch)
}

// Collect implements prom.Collector interface.
func (s *AuthServer) Collect(ch chan<- prom.Metric) {
	s.metrics.mAuthRequests.Collect(ch)
	s.metrics.mGrafanaAuthRequests.Collect(ch)
	s.metrics.mCache.Collect(ch)

	ch <- prom.MustNewConstMetric(s.metrics.mCacheSizeDesc, prom.GaugeValue, float64(s.cache.Size()))

	s.metrics.mDurations.Collect(ch)
}

func (s *AuthServer) incAuthRequests(method, route string, code int) {
	s.metrics.mAuthRequests.WithLabelValues(method, route, statusCodeToString(code)).Inc()
}

func (s *AuthServer) incGrafanaAuthRequests(code int) {
	s.metrics.mGrafanaAuthRequests.WithLabelValues(statusCodeToString(code)).Inc()
}

func (s *AuthServer) incCacheHit() {
	s.metrics.mCache.WithLabelValues("hit").Inc()
}

func (s *AuthServer) incCacheMiss() {
	s.metrics.mCache.WithLabelValues("miss").Inc()
}

// ServeHTTP serves internal location /auth_request for both authentication subrequests
// and subsequent normal requests.
func (s *AuthServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()
	defer func() {
		s.metrics.mDurations.WithLabelValues("total").Observe(time.Since(start).Seconds())
	}()

	if s.l.Logger.IsLevelEnabled(logrus.DebugLevel) {
		b, err := httputil.DumpRequest(req, true)
		if err != nil {
			s.l.Errorf("Failed to dump request: %v.", err)
		}
		s.l.Debugf("Request:\n%s", b)
	}

	err := extractOriginalRequest(req)
	if err != nil {
		s.l.WithError(err).Warn("Failed to parse original request headers.")
		rw.WriteHeader(http.StatusBadRequest)

		s.incAuthRequests(req.Method, req.URL.Path, http.StatusBadRequest)
		return
	}

	// NOTE: now req.Method and req.URL.Path contain original request values
	// that NGINX received from the client. The original request values are used for
	// logging and authentication.

	l := s.l.WithFields(logrus.Fields{"method": req.Method, "path": req.URL.Path})
	// TODO l := logger.Get(ctx) once we have it after https://jira.percona.com/browse/PMM-4326

	// Limit the total time spent on authentication to avoid long delays
	// in case of Grafana being slow or unavailable.
	ctx, cancel := context.WithTimeout(req.Context(), authenticationTimeout)
	defer cancel()

	authRes, err := s.processRequest(ctx, req, l)
	if err != nil {
		authErr, ok := errors.AsType[*authError](err)
		if ok {
			status := convertAuthErrorToHTTPStatus(authErr.code)
			s.incAuthRequests(req.Method, req.URL.Path, status)
			writeResponseErrorStatus(rw, status, int(authErr.code), authErr.message, authErr.message)
			return
		}
		s.incAuthRequests(req.Method, req.URL.Path, http.StatusInternalServerError)
		writeResponseErrorStatus(rw, http.StatusInternalServerError, http.StatusInternalServerError,
			statusCodeToString(http.StatusInternalServerError), statusCodeToString(http.StatusInternalServerError))
		return
	}

	if authRes.vmProxyFilters != "" {
		// Add HTTP headers to response based on filled fields in authResult.
		rw.Header().Set(lbacHeaderName, authRes.vmProxyFilters)
	}
	s.incAuthRequests(req.Method, req.URL.Path, http.StatusOK)
}

// addLBACFilters adds extra filters to requests proxied through VMProxy.
func (s *AuthServer) addLBACFilters(ctx context.Context, userID int, l *logrus.Entry) (string, error) {
	if userID <= 0 {
		// Anonymous users don't have a numeric user ID and cannot have LBAC roles.
		// Skip adding filters and allow the request to proceed.
		l.Debugf("Skipping LBAC filters for anonymous user.")
		return "", nil
	}

	filters, err := s.getLBACFilters(ctx, userID)
	if err != nil {
		return "", err
	}

	if len(filters) == 0 {
		return "", nil
	}

	jsonFilters, err := json.Marshal(filters)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LBAC filters: %w", err)
	}

	return base64.StdEncoding.EncodeToString(jsonFilters), nil
}

// needAddLBACFilters decides if LBAC filters must be added to the outgoing request.
func (s *AuthServer) needAddLBACFilters(urlPath string) bool {
	if !s.accessControl.isEnabled() {
		return false
	}

	for _, p := range lbacPrefixes {
		if strings.HasPrefix(urlPath, p) {
			return true
		}
	}

	return false
}

// getLBACFilters retrieves LBAC filters for the user.
func (s *AuthServer) getLBACFilters(ctx context.Context, userID int) ([]string, error) {
	start := time.Now()
	defer func() {
		s.metrics.mDurations.WithLabelValues("db").Observe(time.Since(start).Seconds())
	}()

	roles, err := models.GetUserRoles(s.db.WithContext(ctx), userID)
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
		roles, err = models.GetUserRoles(s.db.WithContext(ctx), userID)
		if err != nil {
			return nil, err
		}
	}

	if len(roles) == 0 {
		logrus.Errorf("User %d has no roles", userID)
		return nil, fmt.Errorf("user %d has no roles", userID)
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

// processRequest checks if user has access to a specific path.
// It returns user information retrieved during authentication.
// Paths which require no Grafana role return zero value for
// some user fields such as authUser.userID.
// This func expects that req.Method and req.URL.Path are already replaced
// with original request values - extractOriginalRequest(req) has been called beforehand.
func (s *AuthServer) processRequest(ctx context.Context, req *http.Request, l *logrus.Entry) (authResult, error) {
	// Determine the minimal required role for the (already cleaned) original request path.
	minRole, prefix := resolveRule(req.Method, req.URL.Path, l)
	l = l.WithField("prefix", prefix)

	needLbacFilter := s.needAddLBACFilters(req.URL.Path)
	if minRole == none && !needLbacFilter {
		l.WithField("path", req.URL.Path).Debugf("Minimum required role is %s, granting access without authentication.", minRole)
		return authResult{}, nil
	}

	user, err := s.authenticateUser(ctx, req, l)
	if err != nil {
		l.WithError(err).Error("Failed to authenticate user.")
		var zero authResult
		return zero, err
	}

	l = l.WithField("role", user.role.String())
	err = authorizeUser(minRole, user, l)
	if err != nil {
		l.WithError(err).Error("Failed to authorize user.")
		var zero authResult
		return zero, err
	}

	var lbacFilters string
	if needLbacFilter {
		lbacFilters, err = s.addLBACFilters(ctx, user.userID, l)
		if err != nil {
			l.WithError(err).Error("Failed to add VMProxy LBAC filters.")
			var zero authResult
			return zero, errStaticAuthErrorInternalError
		}
	}

	return authResult{vmProxyFilters: lbacFilters}, nil
}

var (
	// Holds mapping of local agent connect endpoints to *authUser.
	staticAuthUsers = map[string]authUser{
		connectionEndpoint:   {role: rules[connectionEndpoint], userID: 0},
		connectionEndpointV2: {role: rules[connectionEndpointV2], userID: 0},
		rtaCollectEndpoint:   {role: rules[rtaCollectEndpoint], userID: 0},
	}
	errStaticAuthErrorPermissionDenied = &authError{code: codes.PermissionDenied, message: "Access denied."}
	errStaticAuthErrorInternalError    = &authError{code: codes.Internal, message: "Internal server error."}
)

// authenticateUser performs identity/authentication only.
func (s *AuthServer) authenticateUser(ctx context.Context, req *http.Request, l *logrus.Entry) (authUser, error) {
	if isLocalAgentConnection(req) {
		user, ok := staticAuthUsers[req.URL.Path]
		if ok {
			return user, nil
		}
		var zero authUser
		return zero, errStaticAuthErrorPermissionDenied
	}
	// Non-local requests require user info retrieval from Grafana.
	return s.getAuthUser(ctx, req, l)
}

// authorizeUser performs role check only.
func authorizeUser(minRole role, user authUser, l *logrus.Entry) error {
	if user.role == grafanaAdmin {
		l.Debugf("Grafana admin, granting access.")
		return nil
	}

	if minRole <= user.role {
		l.Debugf("Minimal required role is %s, granting access.", minRole)
		return nil
	}

	l.Warnf("Minimal required role is %s, denying access.", minRole)
	return errStaticAuthErrorPermissionDenied
}

func (s *AuthServer) getAuthUser(ctx context.Context, req *http.Request, l *logrus.Entry) (authUser, error) {
	authorization := req.Header.Get("Authorization")
	cookie := req.Header.Get("Cookie")

	hash := authCacheKey(req)
	// Hot-path: lookup user in cache first.
	if cached, ok := s.cache.Get(hash); ok {
		// Verify auth headers for this hash to prevent serving wrong user on rare hash collisions.
		if cached.authorization == authorization && cached.cookie == cookie {
			s.incCacheHit()
			return cached.user, nil
		}
	}
	s.incCacheMiss()

	// Cold-path: Cache miss / Stale data.
	// Use single-flight to avoid calling Grafana for the same user's authHeaders.
	// It appears when after restart the same vm-agent starts to send buffered metrics
	// to server in parallel and all such requests have to be authenticated.
	hashStr := strconv.FormatUint(hash, 10)
	res, err, _ := s.authUserGroup.Do(hashStr, func() (any, error) {
		// Recheck inside singleflight to avoid duplicate upstream calls when
		// another goroutine already populated cache while we were waiting.
		if cached, ok := s.cache.Get(hash); ok {
			if cached.authorization == authorization && cached.cookie == cookie {
				return cached.user, nil
			}
		}

		// NOTE 1: The first request for a singlefligh.Do() becomes the leader and runs the closure with its ctx.
		// If this ctx is canceled - it will have no effect on the closure, so that all waiting
		// requests will be able to get the result.

		// NOTE 2: leader's context is not used here directly in order to allow to finish
		// the request to Grafana, so that even if leader's request is already terminated -
		// the rest of waiters in singleflight group will receive the response from Grafana.
		deadLine, ok := ctx.Deadline()
		if !ok {
			deadLine = time.Now().Add(authenticationTimeout)
		}
		grafanaCtx, cancel := context.WithDeadline(context.Background(), deadLine)
		defer cancel()

		userAuthInfo, authErr := s.getGrafanaAuthUser(grafanaCtx, extractAuthHeaders(req), l) //nolint:contextcheck
		if authErr != nil {
			// IMPORTANT: On error, we call Forget(hash) IMMEDIATELY.
			// This prevents the error from getting stuck in the internal singleflight map
			// and allows the next request to retry immediately (e.g. if the Grafana is being restored).
			s.authUserGroup.Forget(hashStr)
			return nil, authErr
		}

		// Store the retrieved user info in cache for future requests.
		s.cache.Set(hash, cachedAuthUser{
			user:          userAuthInfo,
			authorization: authorization,
			cookie:        cookie,
		})
		return userAuthInfo, nil
	})
	if err != nil {
		l.WithError(err).Error("Grafana user lookup failed.")
		var zero authUser
		return zero, err
	}

	user, ok := res.(authUser)
	if !ok {
		l.WithField("type", fmt.Sprintf("%T", res)).Error("Unexpected Grafana user result type.")
		var zero authUser
		return zero, errStaticAuthErrorInternalError
	}

	return user, nil
}

// getGrafanaAuthUser calls Grafana to retrieve user's info. Passed authHeaders are used for authentication.
func (s *AuthServer) getGrafanaAuthUser(ctx context.Context, authHeaders http.Header, l *logrus.Entry) (authUser, error) {
	start := time.Now()
	defer func() {
		s.metrics.mDurations.WithLabelValues("grafana").Observe(time.Since(start).Seconds())
	}()

	authUserInfo, err := s.c.getAuthUser(ctx, authHeaders, l)
	if err != nil {
		var zero authUser
		l.WithError(err).Error("Failed to retrieve user info from Grafana.")
		cErr, ok := errors.AsType[*clientError](err)
		if ok {
			s.incGrafanaAuthRequests(cErr.Code)

			code := codes.Internal
			if cErr.Code == http.StatusUnauthorized || cErr.Code == http.StatusForbidden {
				code = codes.Unauthenticated
			}
			return zero, &authError{code: code, message: cErr.ErrorMessage}
		}
		s.incGrafanaAuthRequests(http.StatusInternalServerError)
		return zero, errStaticAuthErrorInternalError
	}

	s.incGrafanaAuthRequests(http.StatusOK)
	return authUserInfo, nil
}

var _ prom.Collector = (*AuthServer)(nil)
