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
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestNextPrefix(t *testing.T) {
	for _, paths := range [][]string{
		{"/inventory.Nodes/ListNodes", "/inventory.Nodes/", "/inventory.Nodes", "/inventory.", "/inventory", "/", "/"},
		{"/v1/inventory/Nodes/List", "/v1/inventory/Nodes/", "/v1/inventory/Nodes", "/v1/inventory/", "/v1/inventory", "/v1/", "/v1", "/", "/"},
		{"/.x", "/.", "/", "/"},
		{".", "/", "/"},
		{"./", "/", "/"},
		{"hax0r", "/", "/"},
		{"", "/"},
		{"/v1/server/AWSInstanceCheck/..%2f..%2finventory/Services/List'"},
	} {
		t.Run(paths[0], func(t *testing.T) {
			for i, path := range paths[:len(paths)-1] {
				tests.AddToFuzzCorpus(t, "", []byte(path))

				expected := paths[i+1]
				actual := nextPrefix(path)
				assert.Equal(t, expected, actual, "path = %q", path)
			}
		})
	}
}

func TestResolveRule(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method   string
		path     string
		wantRole role
	}{
		// Alerting: only listing templates is viewable; writes need editor.
		{http.MethodGet, "/v1/alerting/templates", viewer},        // ListTemplates
		{http.MethodPost, "/v1/alerting/templates", editor},       // CreateTemplate
		{http.MethodPut, "/v1/alerting/templates/foo", editor},    // UpdateTemplate
		{http.MethodDelete, "/v1/alerting/templates/foo", editor}, // DeleteTemplate
		{http.MethodPost, "/v1/alerting/rules", editor},           // CreateRule
		// No matching rule falls back to grafanaAdmin.
		{http.MethodGet, "/v1/unknown", grafanaAdmin},
	} {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			t.Parallel()

			got, _ := resolveRule(tc.method, tc.path, logrus.WithField("test", t.Name()))
			assert.Equal(t, tc.wantRole, got)
		})
	}
}

func TestAuthServerAuthenticate(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/dummy", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "admin")
	authHeaders := req.Header

	t.Run("GrafanaAdminFallback", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/foo", nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")

		_, res := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Nil(t, res)
	})

	t.Run("NoAnonymousAccess", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/foo", nil)
		require.NoError(t, err)

		_, res := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Equal(t, &authError{code: codes.Unauthenticated, message: "Unauthorized"}, res)
	})

	for uri, minRole := range rules {
		for _, role := range []role{viewer, editor, admin} {
			t.Run(fmt.Sprintf("uri=%s,minRole=%s,role=%s", uri, minRole, role), func(t *testing.T) {
				t.Parallel()

				login := fmt.Sprintf("%s-%s-%d", minRole, role, time.Now().Nanosecond())
				userID, err := c.testCreateUser(ctx, login, role, authHeaders)
				require.NoError(t, err)
				require.NotZero(t, userID)
				if err != nil {
					defer func() {
						err = c.testDeleteUser(ctx, userID, authHeaders)
						require.NoError(t, err)
					}()
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
				require.NoError(t, err)
				req.SetBasicAuth(login, login)

				_, res := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
				if minRole <= role {
					assert.Nil(t, res)
				} else {
					assert.Equal(t, &authError{code: codes.PermissionDenied, message: "Access denied"}, res)
				}
			})
		}
	}
}

func TestServerClientConnection(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c, nil)

	t.Run("Basic auth - success", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectionEndpoint, nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")

		_, authError := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Nil(t, authError)
	})

	// Beware: Five or more wrong tries will lock user with error message: "Invalid user or password".
	t.Run("Basic auth - fail", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectionEndpoint, nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "wrong")

		_, authError := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Equal(t, codes.Unauthenticated, authError.code)
	})

	t.Run("Token auth - success", func(t *testing.T) {
		t.Parallel()

		nodeName := fmt.Sprintf("N1-%d", time.Now().UnixNano())
		headersMD := metadata.New(map[string]string{
			"Authorization": "Basic YWRtaW46YWRtaW4=",
		})
		ctx := metadata.NewIncomingContext(t.Context(), headersMD)
		_, serviceToken, err := c.CreateServiceAccount(ctx, nodeName, true)
		require.NoError(t, err)
		defer func() {
			warning, err := c.DeleteServiceAccount(ctx, nodeName, true)
			require.NoError(t, err)
			require.Empty(t, warning)
		}()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectionEndpoint, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+serviceToken)

		_, authError := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Nil(t, authError)
	})

	t.Run("Token auth - fail", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectionEndpoint, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer wrong")

		_, authError := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Equal(t, codes.Internal, authError.code)
	})
}

func TestAuthServerAddVMGatewayToken(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())
	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	defer func(t *testing.T) {
		t.Helper()

		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())
	}(t)

	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c, db)

	roleA := models.Role{
		Title:  "Role A",
		Filter: "filter A",
	}
	err := models.CreateRole(db.Querier, &roleA)
	require.NoError(t, err)

	roleB := models.Role{
		Title:  "Role B",
		Filter: "filter B",
	}
	err = models.CreateRole(db.Querier, &roleB)
	require.NoError(t, err)

	roleC := models.Role{
		Title:  "Role C",
		Filter: "",
	}
	err = models.CreateRole(db.Querier, &roleC)
	require.NoError(t, err)

	// Enable access control
	_, err = models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{
		EnableAccessControl: new(true),
	})
	require.NoError(t, err)

	for userID, roleIDs := range map[int][]int{
		1337: {int(roleA.ID)},
		1338: {int(roleA.ID), int(roleB.ID)},
		1339: {int(roleA.ID), int(roleC.ID)},
		1:    {int(roleA.ID)},
	} {
		err := db.InTransaction(func(tx *reform.TX) error {
			return models.AssignRoles(tx, userID, roleIDs)
		})
		require.NoError(t, err)
	}

	t.Run("shall properly evaluate adding filters", func(t *testing.T) {
		for uri, shallAdd := range map[string]bool{
			"/":                          false,
			"/dummy":                     false,
			"/prometheus/api/":           false,
			"/prometheus/api/v1/":        true,
			"/prometheus/api/v1/query":   true,
			"/graph/api/datasources/uid": true,
			"/graph/api/ds/query":        true,
			"/v1/qan/metrics:getFilters": true,
			"/v1/qan/query:exists":       true,
		} {
			for _, userID := range []int{0, 1337, 1338} {
				t.Run(fmt.Sprintf("uri=%s userID=%d", uri, userID), func(t *testing.T) {
					t.Parallel()
					rw := httptest.NewRecorder()
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
					require.NoError(t, err)
					if userID == 0 {
						req.SetBasicAuth("admin", "admin")
					}

					err = s.maybeAddLBACFilters(ctx, rw, req, userID, logrus.WithField("test", t.Name()))
					require.NoError(t, err)

					headerString := rw.Header().Get(lbacHeaderName)

					if shallAdd {
						require.NotEmpty(t, headerString)
					} else {
						require.Empty(t, headerString)
					}
				})
			}
		}
	})

	//nolint:paralleltest
	t.Run("shall be a valid JSON array", func(t *testing.T) {
		rw := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/prometheus/api/v1/", nil)
		require.NoError(t, err)

		err = s.maybeAddLBACFilters(ctx, rw, req, 1338, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		headerString := rw.Header().Get(lbacHeaderName)
		require.NotEmpty(t, headerString)

		filters, err := base64.StdEncoding.DecodeString(headerString)
		require.NoError(t, err)
		var parsed []string
		err = json.Unmarshal(filters, &parsed)
		require.NoError(t, err)

		require.Len(t, parsed, 2)
		require.Equal(t, "filter A", parsed[0])
		require.Equal(t, "filter B", parsed[1])
	})

	//nolint:paralleltest
	t.Run("shall not add any filters if at least one role has full access", func(t *testing.T) {
		rw := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/prometheus/api/v1/", nil)
		require.NoError(t, err)

		err = s.maybeAddLBACFilters(ctx, rw, req, 1339, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		headerString := rw.Header().Get(lbacHeaderName)
		require.Empty(t, headerString)
	})
}

func TestCleanPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path     string
		expected string
		wantErr  bool
	}{
		{
			connectionEndpointV2,
			connectionEndpointV2,
			false,
		}, {
			connectionEndpoint,
			connectionEndpoint,
			false,
		}, {
			rtaCollectEndpoint,
			rtaCollectEndpoint,
			false,
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2finventory/Services/List",
			"/v1/inventory/Services/List",
			false,
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2f..%2fmanaged/logs.zip",
			"/managed/logs.zip",
			false,
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2f..%2f/logs.zip",
			"/logs.zip",
			false,
		}, {
			"/managed/logs.zip?download=1",
			"/managed/logs.zip",
			false,
		}, {
			"/v1/server/./logs.zip",
			"/v1/server/logs.zip",
			false,
		}, {
			"/v1/server/../logs.zip",
			"/v1/logs.zip",
			false,
		}, {
			"/graph/api/datasources/proxy/8/?query=WITH%20(%0A%20%20%20%20CASE%20%0A%20%20%20%20%20%20%20%20WHEN%20(3000%20%25%2060)%20%3D%200%20THEN%203000%0A%20%20%20%20ELSE%2060%20END%0A)%20AS%20scale%0ASELECT%0A%20%20%20%20(intDiv(toUInt32(timestamp)%2C%203000)%20*%203000)%20*%201000%20as%20t%2C%0A%20%20%20%20hostname%20h%2C%0A%20%20%20%20status%20s%2C%0A%20%20%20%20SUM(req_count)%20as%20req_count%0AFROM%20pinba.report_by_all%0AWHERE%0A%20%20%20%20timestamp%20%3E%3D%20toDateTime(1707139680)%20AND%20timestamp%20%3C%3D%20toDateTime(1707312480)%0A%20%20%20%20AND%20status%20%3E%3D%20400%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20schema%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20hostname%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20server_name%20%3D%20%27all%27%20ELSE%201%20END%0AGROUP%20BY%20t%2C%20h%2C%20s%0AORDER%20BY%20t%20FORMAT%20JSON",
			"/graph/api/datasources/proxy/8/",
			false,
		}, {
			"/v1/server/%zz/logs.zip",
			"",
			true,
		}, {
			"/v1/server/%",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got, err := cleanPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equalf(t, tt.expected, got, "cleanPath(%v)", tt.path)
		})
	}
}

func TestAuthServerServeHTTPBadRequestMetricsUsesCleanedRoute(t *testing.T) {
	t.Parallel()

	s := NewAuthServer(nil, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)

	// Trigger extractOriginalRequest error (missing X-Original-Method),
	// but keep X-Original-Uri so ServeHTTP records a metric for it.
	req.Header.Set("X-Original-Uri", "/v1/server/AWSInstanceCheck/..%2f..%2f..%2f/logs.zip?foo=bar")

	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	value := testutil.ToFloat64(s.metrics.mAuthRequests.WithLabelValues(http.MethodGet, "/logs.zip", "400"))
	require.InDelta(t, 1.0, value, 0.0, "expected auth request metric with cleaned route")
}

func TestAuthServerServeHTTPBadRequestMetricsFallbackToRawRouteOnCleanError(t *testing.T) {
	t.Parallel()

	s := NewAuthServer(nil, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth_request", nil)

	// Invalid escape sequence keeps cleanPath from normalizing the path,
	// so ServeHTTP should use route value after query trimming.
	req.Header.Set("X-Original-Uri", "/bad%2?foo=bar")

	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	value := testutil.ToFloat64(s.metrics.mAuthRequests.WithLabelValues(http.MethodPost, "/bad%2", "400"))
	require.InDelta(t, 1.0, value, 0.0, "expected auth request metric with original route when cleaning fails")
}

func BenchmarkCleanPath(b *testing.B) {
	const unescapedURI = "/v1/server/AWSInstanceCheck/..%2f..%2f..%2f..%2fgraph/api/datasources/proxy/8%2f%2f.%2f..%2f8%2f%2f?query=WITH%20CASE%20WHEN%203000%20x%2060%20THEN%203000%20ELSE%2060%20END%20SELECT%20hostname%2Cstatus%20FROM%20pinba.report_by_all%20WHERE%20timestamp%3E%3D1707139680%20AND%20timestamp%3C%3D1707312480%20ORDER%20BY%20t"
	const expectedCleanPath = "/graph/api/datasources/proxy/8"

	b.ReportAllocs()

	cleanedPath, err := cleanPath(unescapedURI)
	require.NoError(b, err)
	require.Equal(b, expectedCleanPath, cleanedPath)

	b.ResetTimer()
	for b.Loop() {
		cleanedPath, err = cleanPath(unescapedURI)
		if err != nil {
			b.Fatalf("cleanPath returned error: %v", err)
		}
		if cleanedPath != expectedCleanPath {
			b.Fatalf("unexpected cleaned path: got %q, want %q", cleanedPath, expectedCleanPath)
		}
	}
}

func TestExtractOriginalRequest(t *testing.T) {
	t.Parallel()

	t.Run("valid header values are normalized", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)
		req.Header.Set("X-Original-Method", http.MethodPost)
		req.Header.Set("X-Original-Uri", "/v1/server/AWSInstanceCheck/..%2f..%2fmanaged/logs.zip?foo=bar")

		err := extractOriginalRequest(req)
		require.NoError(t, err)
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/v1/managed/logs.zip", req.URL.Path)
	})

	t.Run("invalid escaped path returns error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)
		req.Header.Set("X-Original-Method", http.MethodGet)
		req.Header.Set("X-Original-Uri", "/v1/server/%zz/logs.zip")

		err := extractOriginalRequest(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unescape path")
	})

	t.Run("sanitizes encoded newline and carriage return", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)
		req.Header.Set("X-Original-Method", http.MethodGet)
		req.Header.Set("X-Original-Uri", "/v1/server/logs%0A%0D.zip")

		err := extractOriginalRequest(req)
		require.NoError(t, err)
		assert.Equal(t, "/v1/server/logs  .zip", req.URL.Path)
	})
}

func TestIsLocalAgentConnection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		remoteAddr string
		path       string
		want       bool
	}{
		{name: "local connect endpoint", remoteAddr: "127.0.0.1:12345", path: connectionEndpoint, want: true},
		{name: "local rta endpoint", remoteAddr: "127.0.0.1:12345", path: rtaCollectEndpoint, want: true},
		{name: "remote endpoint", remoteAddr: "10.0.0.2:12345", path: connectionEndpoint, want: false},
		{name: "local unknown path", remoteAddr: "127.0.0.1:12345", path: "/v1/server/version", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			req.RemoteAddr = tc.remoteAddr
			assert.Equal(t, tc.want, isLocalAgentConnection(req))
		})
	}
}

type fakeAuthClient struct {
	user  authUser
	err   error
	calls int
}

func (c *fakeAuthClient) getAuthUser(_ context.Context, _ http.Header, _ *logrus.Entry) (authUser, error) {
	c.calls++
	if c.err != nil {
		return authUser{}, c.err
	}
	return c.user, nil
}

func TestAuthServerGetAuthUserCacheMetrics(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := &fakeAuthClient{user: authUser{role: viewer, userID: 42}}
	s := NewAuthServer(client, nil)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/management/Jobs", nil)
	req.Header.Set("Authorization", "Bearer token")

	u1, err1 := s.getAuthUser(ctx, req, logrus.WithField("test", t.Name()))
	require.Nil(t, err1)
	require.NotNil(t, u1)
	assert.Equal(t, 1, client.calls)

	u2, err2 := s.getAuthUser(ctx, req, logrus.WithField("test", t.Name()))
	require.Nil(t, err2)
	require.NotNil(t, u2)
	assert.Equal(t, 1, client.calls)
	assert.Equal(t, float64(1), testutil.ToFloat64(s.metrics.mCache.WithLabelValues("hit")))
	assert.Equal(t, float64(1), testutil.ToFloat64(s.metrics.mCache.WithLabelValues("miss")))
}

func TestAuthServerRetrieveRoleGrafanaRequestMetrics(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	authHeaders := http.Header{"Authorization": []string{"Bearer token"}}
	l := logrus.WithField("test", t.Name())

	t.Run("success increments 200", func(t *testing.T) {
		t.Parallel()

		s := NewAuthServer(&fakeAuthClient{user: authUser{role: viewer, userID: 1}}, nil)
		got, authErr := s.retrieveRole(ctx, "ok", authHeaders, l)
		require.Nil(t, authErr)
		require.NotNil(t, got)
		assert.Equal(t, float64(1), testutil.ToFloat64(s.metrics.mGrafanaAuthRequests.WithLabelValues("200")))
	})

	t.Run("401 maps to unauthenticated and increments 401", func(t *testing.T) {
		t.Parallel()

		s := NewAuthServer(&fakeAuthClient{err: &clientError{Code: http.StatusUnauthorized, ErrorMessage: "Unauthorized"}}, nil)
		got, authErr := s.retrieveRole(ctx, "unauth", authHeaders, l)
		require.Nil(t, got)
		require.NotNil(t, authErr)
		assert.Equal(t, codes.Unauthenticated, authErr.code)
		assert.Equal(t, float64(1), testutil.ToFloat64(s.metrics.mGrafanaAuthRequests.WithLabelValues("401")))
	})

	t.Run("generic error increments 500", func(t *testing.T) {
		t.Parallel()

		s := NewAuthServer(&fakeAuthClient{err: errors.New("boom")}, nil)
		got, authErr := s.retrieveRole(ctx, "internal", authHeaders, l)
		require.Nil(t, got)
		require.NotNil(t, authErr)
		assert.Equal(t, codes.Internal, authErr.code)
		assert.Equal(t, float64(1), testutil.ToFloat64(s.metrics.mGrafanaAuthRequests.WithLabelValues("500")))
	})
}

func TestAuthServerCollectMetrics(t *testing.T) {
	t.Parallel()

	s := NewAuthServer(&fakeAuthClient{user: authUser{role: viewer, userID: 1}}, nil)
	s.incAuthRequests(http.MethodGet, "/v1/server/version", http.StatusOK)
	s.incGrafanaAuthRequests(http.StatusForbidden)
	s.incCacheHit()
	s.incCacheMiss()
	s.cache["cached"] = cacheItem{u: authUser{role: viewer, userID: 1}, created: time.Now()}
	s.metrics.mDurations.WithLabelValues("total").Observe(0.01)

	const expected = `
		# HELP pmm_managed_auth_requests_total Total number of authentication requests.
		# TYPE pmm_managed_auth_requests_total counter
		pmm_managed_auth_requests_total{method="GET",route="/v1/server/version",status_code="200"} 1
		# HELP pmm_managed_auth_grafana_requests_total Total number of authentication requests to Grafana.
		# TYPE pmm_managed_auth_grafana_requests_total counter
		pmm_managed_auth_grafana_requests_total{status_code="403"} 1
		# HELP pmm_managed_auth_cache_total Total number of authentication cache requests by status (hit or miss).
		# TYPE pmm_managed_auth_cache_total counter
		pmm_managed_auth_cache_total{status="hit"} 1
		pmm_managed_auth_cache_total{status="miss"} 1
		# HELP pmm_managed_auth_cache_size Total number of items in the authentication cache.
		# TYPE pmm_managed_auth_cache_size gauge
		pmm_managed_auth_cache_size 1
	`

	err := testutil.CollectAndCompare(
		s,
		strings.NewReader(expected),
		"pmm_managed_auth_requests_total",
		"pmm_managed_auth_grafana_requests_total",
		"pmm_managed_auth_cache_total",
		"pmm_managed_auth_cache_size",
	)
	require.NoError(t, err)
}

func TestAuthServerServeHTTPBadRequestMetrics(t *testing.T) {
	t.Parallel()

	s := NewAuthServer(&fakeAuthClient{}, nil)
	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)
	req.Header.Set("X-Original-Method", http.MethodGet)
	req.Header.Set("X-Original-Uri", "/v1/server/%zz/logs.zip")

	s.ServeHTTP(rw, req)
	require.Equal(t, http.StatusBadRequest, rw.Code)
	assert.Equal(t, float64(1), testutil.ToFloat64(s.metrics.mAuthRequests.WithLabelValues(http.MethodGet, "/v1/server/%zz/logs.zip", "400")))
}
