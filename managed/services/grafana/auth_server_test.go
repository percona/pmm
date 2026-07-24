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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

// newTestAuthServer creates an AuthServer with access-control cache primed,
// so tests never trigger DB reload unless they explicitly need it.
func newTestAuthServer(c clientInterface, db *reform.DB) *AuthServer {
	s := NewAuthServer(c, db)
	s.accessControl.mu.Lock()
	s.accessControl.enabled = false
	s.accessControl.lastUpdated = time.Now()
	s.accessControl.mu.Unlock()
	return s
}

func setAccessControlCachedValue(s *AuthServer, enabled bool) {
	s.accessControl.mu.Lock()
	s.accessControl.enabled = enabled
	s.accessControl.lastUpdated = time.Now()
	s.accessControl.mu.Unlock()
}

func newOriginalReq(t *testing.T, method, path string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)
	req.Header.Set("X-Original-Method", method)
	req.Header.Set("X-Original-Uri", path)
	return req
}

func setupLBACServer(t *testing.T) (*AuthServer, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlMock.ExpectationsWereMet())
		_ = sqlDB.Close()
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	s := newTestAuthServer(newMockClientInterface(t), db)

	setAccessControlCachedValue(s, true)

	return s, sqlMock
}

func roleRows(rows ...struct {
	id     uint32
	title  string
	filter string
},
) *sqlmock.Rows {
	r := sqlmock.NewRows([]string{"id", "title", "description", "filter", "created_at", "updated_at"})
	now := time.Now().UTC()
	for _, row := range rows {
		r = r.AddRow(row.id, row.title, "", row.filter, now, now)
	}
	return r
}

func TestStatusCodeToString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		code int
		want string
	}{
		{name: "ok", code: http.StatusOK, want: "200"},
		{name: "bad request", code: http.StatusBadRequest, want: "400"},
		{name: "unauthorized", code: http.StatusUnauthorized, want: "401"},
		{name: "forbidden", code: http.StatusForbidden, want: "403"},
		{name: "not found", code: http.StatusNotFound, want: "404"},
		{name: "method not allowed", code: http.StatusMethodNotAllowed, want: "405"},
		{name: "request timeout", code: http.StatusRequestTimeout, want: "408"},
		{name: "too many requests", code: http.StatusTooManyRequests, want: "429"},
		{name: "internal", code: http.StatusInternalServerError, want: "500"},
		{name: "service unavailable", code: http.StatusServiceUnavailable, want: "503"},
		{name: "unknown", code: http.StatusTeapot, want: strconv.Itoa(http.StatusTeapot)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, statusCodeToString(tc.code))
		})
	}
}

func TestAuthErrorError(t *testing.T) {
	t.Parallel()

	err := authError{code: codes.PermissionDenied, message: errStaticAuthErrorPermissionDenied.message}
	assert.Equal(t, fmt.Sprintf("%s: %s", errStaticAuthErrorPermissionDenied.message, codes.PermissionDenied), err.Error())
}

func TestHTTPStatusForAuthError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		code codes.Code
		want int
	}{
		{name: "permission denied", code: codes.PermissionDenied, want: http.StatusForbidden},
		{name: "unauthenticated", code: codes.Unauthenticated, want: authenticationErrorCode},
		{name: "internal", code: codes.Internal, want: authenticationErrorCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, httpStatusForAuthError(tc.code))
		})
	}
}

func TestAuthServerShallAddLBACFilters(t *testing.T) {
	t.Parallel()

	t.Run("disabled LBAC - lbacPrefixes", func(t *testing.T) {
		t.Parallel()
		s := newTestAuthServer(newMockClientInterface(t), nil)
		setAccessControlCachedValue(s, false)
		for _, prefix := range lbacPrefixes {
			t.Run(prefix, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, prefix, nil)
				assert.False(t, s.needAddLBACFilters(req))
			})
		}
	})

	t.Run("disabled LBAC - other prefixes", func(t *testing.T) {
		t.Parallel()
		s := newTestAuthServer(newMockClientInterface(t), nil)
		setAccessControlCachedValue(s, false)
		for _, prefix := range []string{
			"/v1/server/settings",
			"/inventory.",
			"/v1/advisors/checks:",
			"/v1/alerting",
			"/v1/users/current",
			"/v1/realtimeanalytics/sessions:start",
		} {
			t.Run(prefix, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, prefix, nil)
				assert.False(t, s.needAddLBACFilters(req))
			})
		}
	})

	t.Run("enabled LBAC - lbacPrefixes", func(t *testing.T) {
		t.Parallel()
		s := newTestAuthServer(newMockClientInterface(t), nil)
		setAccessControlCachedValue(s, true)
		for _, prefix := range lbacPrefixes {
			t.Run(prefix, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, prefix, nil)
				assert.True(t, s.needAddLBACFilters(req))
			})
		}
	})

	t.Run("enabled LBAC - other prefixes", func(t *testing.T) {
		t.Parallel()
		s := newTestAuthServer(newMockClientInterface(t), nil)
		setAccessControlCachedValue(s, true)
		for _, prefix := range []string{
			"/v1/server/settings",
			"/inventory.",
			"/v1/advisors/checks:",
			"/v1/alerting",
			"/v1/users/current",
			"/v1/realtimeanalytics/sessions:start",
		} {
			t.Run(prefix, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, prefix, nil)
				assert.False(t, s.needAddLBACFilters(req))
			})
		}
	})
}

func TestAuthServerGetLBACFilters(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		userID    int
		setupMock func(sqlmock.Sqlmock, int)
		want      []string
		wantErr   error
	}{
		{
			name:   "returns all filters when user has restricted roles",
			userID: 1001,
			setupMock: func(m sqlmock.Sqlmock, userID int) {
				m.ExpectQuery("SELECT").WithArgs(userID).WillReturnRows(roleRows(
					struct {
						id     uint32
						title  string
						filter string
					}{id: 1, title: "Role A", filter: "filter-a"},
					struct {
						id     uint32
						title  string
						filter string
					}{id: 2, title: "Role B", filter: "filter-b"},
				))
			},
			want: []string{"filter-a", "filter-b"},
		},
		{
			name:   "returns empty slice when any role has empty filter",
			userID: 1002,
			setupMock: func(m sqlmock.Sqlmock, userID int) {
				m.ExpectQuery("SELECT").WithArgs(userID).WillReturnRows(roleRows(
					struct {
						id     uint32
						title  string
						filter string
					}{id: 3, title: "Role Full Access", filter: ""},
					struct {
						id     uint32
						title  string
						filter string
					}{id: 4, title: "Role Ignored", filter: "filter-b"},
				))
			},
			want: []string{},
		},
		{
			name:   "returns error when roles query fails",
			userID: 1003,
			setupMock: func(m sqlmock.Sqlmock, userID int) {
				m.ExpectQuery("SELECT").WithArgs(userID).WillReturnError(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, sqlMock := setupLBACServer(t)
			tc.setupMock(sqlMock, tc.userID)

			got, err := s.getLBACFilters(t.Context(), tc.userID)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAuthServerAddLBACFilters(t *testing.T) {
	t.Parallel()

	log := logrus.WithField("test", t.Name())

	t.Run("non proxied request skips", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/settings", nil)
		filters, err := s.addLBACFilters(t.Context(), req, 1001, log)
		require.NoError(t, err)
		assert.Empty(t, filters)
		c.AssertNotCalled(t, "getAuthUser", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get user id when grafana auth fails", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		setAccessControlCachedValue(s, true)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/prometheus/api/v1/query", nil)
		req.Header.Set("Authorization", "Bearer broken")

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{}, &clientError{Code: http.StatusUnauthorized, ErrorMessage: http.StatusText(http.StatusUnauthorized)}).
			Once()

		filters, err := s.addLBACFilters(t.Context(), req, 0, log)
		require.ErrorIs(t, err, errFailGetUserID)
		assert.Empty(t, filters)
		c.AssertExpectations(t)
	})

	t.Run("anonymous user is allowed without filters", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		setAccessControlCachedValue(s, true)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/prometheus/api/v1/query", nil)
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{role: viewer, userID: 0}, nil).
			Once()

		filters, err := s.addLBACFilters(t.Context(), req, 0, log)
		require.NoError(t, err)
		assert.Empty(t, filters)
		c.AssertExpectations(t)
	})

	t.Run("encodes filters for proxied request", func(t *testing.T) {
		t.Parallel()
		s, sqlMock := setupLBACServer(t)
		sqlMock.ExpectQuery("SELECT").WithArgs(1001).WillReturnRows(roleRows(
			struct {
				id     uint32
				title  string
				filter string
			}{id: 1, title: "Role A", filter: "filter-a"},
			struct {
				id     uint32
				title  string
				filter string
			}{id: 2, title: "Role B", filter: "filter-b"},
		))
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/prometheus/api/v1/query", nil)

		encoded, err := s.addLBACFilters(t.Context(), req, 1001, log)
		require.NoError(t, err)
		require.NotEmpty(t, encoded)

		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)

		var parsed []string
		require.NoError(t, json.Unmarshal(decoded, &parsed))
		assert.Equal(t, []string{"filter-a", "filter-b"}, parsed)
	})

	t.Run("shall not add any filters if at least one role has full access", func(t *testing.T) {
		t.Parallel()
		s, sqlMock := setupLBACServer(t)
		sqlMock.ExpectQuery("SELECT").WithArgs(1002).WillReturnRows(roleRows(
			struct {
				id     uint32
				title  string
				filter string
			}{id: 1, title: "Role A", filter: "filter-a"},
			struct {
				id     uint32
				title  string
				filter string
			}{id: 2, title: "Role B", filter: ""},
		))
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/prometheus/api/v1/query", nil)

		encoded, err := s.addLBACFilters(t.Context(), req, 1002, log)
		require.NoError(t, err)
		require.Empty(t, encoded)
	})

	t.Run("returns error when user has no assigned roles and default role assignment fails", func(t *testing.T) {
		t.Parallel()
		s, sqlMock := setupLBACServer(t)
		sqlMock.ExpectQuery("SELECT").WithArgs(1003).WillReturnRows(roleRows())
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery("SELECT").WillReturnError(assert.AnError)
		sqlMock.ExpectRollback()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/prometheus/api/v1/query", nil)

		encoded, err := s.addLBACFilters(t.Context(), req, 1003, log)
		require.ErrorIs(t, err, assert.AnError)
		require.Empty(t, encoded)
	})
}

func TestAuthorizeUserAuthServer(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())

	for _, tc := range []struct {
		name    string
		minRole role
		user    *authUser
		wantErr *authError
	}{
		{name: "nil user", minRole: viewer, user: nil, wantErr: errStaticAuthErrorInternalError},
		{name: "grafana admin", minRole: admin, user: &authUser{role: grafanaAdmin}, wantErr: nil},
		{name: "role allowed", minRole: viewer, user: &authUser{role: editor}, wantErr: nil},
		{name: "role denied", minRole: admin, user: &authUser{role: viewer}, wantErr: errStaticAuthErrorPermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := authorizeUser(tc.minRole, tc.user, l)
			assert.Equal(t, tc.wantErr, got)
		})
	}
}

func TestAuthServerAuthenticateUser(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())

	t.Run("local static endpoint is authorized", func(t *testing.T) {
		t.Parallel()

		s := newTestAuthServer(newMockClientInterface(t), nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, connectionEndpoint, nil)
		req.RemoteAddr = "127.0.0.1:12345"

		got, err := s.authenticateUser(t.Context(), req, l)
		require.Nil(t, err)
		require.NotNil(t, got)
		assert.Equal(t, staticAuthUsers[connectionEndpoint], got)
	})

	t.Run("localhost non-whitelisted path falls back to grafana auth", func(t *testing.T) {
		t.Parallel()

		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/qan/", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{}, &clientError{Code: http.StatusUnauthorized, ErrorMessage: http.StatusText(http.StatusUnauthorized)}).
			Once()

		got, err := s.authenticateUser(t.Context(), req, l)
		assert.Nil(t, got)
		require.NotNil(t, err)
		assert.Equal(t, codes.Unauthenticated, err.code)
		c.AssertExpectations(t)
	})

	t.Run("remote request goes through grafana auth", func(t *testing.T) {
		t.Parallel()

		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/settings", nil)
		req.RemoteAddr = "10.0.0.1:443"
		req.Header.Set("Authorization", "Bearer ok")

		want := authUser{role: admin, userID: 42}
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).Return(want, nil).Once()

		got, err := s.authenticateUser(t.Context(), req, l)
		require.Nil(t, err)
		require.NotNil(t, got)
		assert.Equal(t, want, *got)
		c.AssertExpectations(t)
	})

	t.Run("remote request with grafana failed auth returns unauthenticated", func(t *testing.T) {
		t.Parallel()

		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/settings", nil)
		req.RemoteAddr = "10.0.0.1:443"
		req.Header.Set("Authorization", "Bearer broken")

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{}, &clientError{Code: http.StatusUnauthorized, ErrorMessage: http.StatusText(http.StatusUnauthorized)}).
			Once()

		got, err := s.authenticateUser(t.Context(), req, l)
		assert.Nil(t, got)
		require.NotNil(t, err)
		assert.Equal(t, codes.Unauthenticated, err.code)
		c.AssertExpectations(t)
	})

	t.Run("get empty user info for anonymous user", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/qan/query", nil)

		userInfo := authUser{role: none, userID: 0}
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(userInfo, nil).
			Once()

		got, err := s.authenticateUser(t.Context(), req, l)
		require.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, userInfo, *got)
		// assert.True(t, len(s.cache) == 0, "cache should be empty on anonymous user")
		c.AssertExpectations(t)
	})
}

func TestAuthServerGetGrafanaAuthUser(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())
	headers := http.Header{"Authorization": []string{"Bearer token"}}

	for _, tc := range []struct {
		name        string
		retUser     authUser
		retErr      error
		wantUser    *authUser
		wantErr     *authError
		mustBeError func(*testing.T, *authError)
	}{
		{
			name:     "success",
			retUser:  authUser{role: editor, userID: 7},
			wantUser: &authUser{role: editor, userID: 7},
		},
		{
			name:    "unauthorized from grafana",
			retErr:  &clientError{Code: http.StatusUnauthorized, ErrorMessage: http.StatusText(http.StatusUnauthorized)},
			wantErr: &authError{code: codes.Unauthenticated, message: http.StatusText(http.StatusUnauthorized)},
		},
		{
			name:    "forbidden from grafana",
			retErr:  &clientError{Code: http.StatusForbidden, ErrorMessage: http.StatusText(http.StatusForbidden)},
			wantErr: &authError{code: codes.Unauthenticated, message: http.StatusText(http.StatusForbidden)},
		},
		{
			name:    "upstream internal keeps internal code",
			retErr:  &clientError{Code: http.StatusInternalServerError, ErrorMessage: http.StatusText(http.StatusInternalServerError)},
			wantErr: &authError{code: codes.Internal, message: http.StatusText(http.StatusInternalServerError)},
		},
		{
			name:    "generic error maps to static internal",
			retErr:  errors.New("boom"),
			wantErr: errStaticAuthErrorInternalError,
			mustBeError: func(t *testing.T, got *authError) {
				t.Helper()
				assert.Equal(t, errStaticAuthErrorInternalError, got)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := newMockClientInterface(t)
			s := newTestAuthServer(c, nil)
			c.On("getAuthUser", mock.Anything, headers, mock.Anything).Return(tc.retUser, tc.retErr).Once()

			got, err := s.getGrafanaAuthUser(t.Context(), headers, l)

			if tc.wantErr == nil {
				require.Nil(t, err)
				require.NotNil(t, got)
				assert.Equal(t, *tc.wantUser, *got)
			} else {
				require.Nil(t, got)
				require.Error(t, err)
				assert.Equal(t, tc.wantErr, err)
				if tc.mustBeError != nil {
					tc.mustBeError(t, err)
				}
			}

			c.AssertExpectations(t)
		})
	}
}

func TestAuthServerGetAuthUser(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())

	mkReq := func(t *testing.T, auth string) *http.Request {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/settings", nil)
		req.Header.Set("Authorization", auth)
		return req
	}

	t.Run("cache hit uses cached user", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := mkReq(t, "Bearer cached")

		headers := extractAuthHeaders(req)
		j, err := json.Marshal(headers)
		require.NoError(t, err)
		hash := base64.StdEncoding.EncodeToString(j)
		s.cache[hash] = cacheItem{u: authUser{role: viewer, userID: 11}, created: time.Now()}

		got, authErr := s.getAuthUser(t.Context(), req, l)
		require.Nil(t, authErr)
		require.NotNil(t, got)
		assert.Equal(t, authUser{role: viewer, userID: 11}, *got)
		c.AssertNotCalled(t, "getAuthUser", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("stale cache refreshes via grafana", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := mkReq(t, "Bearer stale")

		headers := extractAuthHeaders(req)
		j, err := json.Marshal(headers)
		require.NoError(t, err)
		hash := base64.StdEncoding.EncodeToString(j)
		s.cache[hash] = cacheItem{u: authUser{role: viewer, userID: 1}, created: time.Now().Add(-cacheInvalidationInterval - time.Second)}

		want := authUser{role: admin, userID: 99}
		c.On("getAuthUser", mock.Anything, headers, mock.Anything).Return(want, nil).Once()

		got, authErr := s.getAuthUser(t.Context(), req, l)
		require.Nil(t, authErr)
		require.NotNil(t, got)
		assert.Equal(t, want, *got)
		assert.Equal(t, want, s.cache[hash].u)
		c.AssertExpectations(t)
	})

	t.Run("cache miss calls grafana and caches response", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		token := "miss"
		req := mkReq(t, "Bearer "+token)
		headers := extractAuthHeaders(req)

		want := authUser{role: editor, userID: 8}
		c.On("getAuthUser", mock.Anything, headers, mock.Anything).Return(want, nil).Once()

		got, authErr := s.getAuthUser(t.Context(), req, l)
		require.Nil(t, authErr)
		require.NotNil(t, got)
		assert.Equal(t, want, *got)

		j, err := json.Marshal(headers)
		require.NoError(t, err)
		hash := base64.StdEncoding.EncodeToString(j)
		item, ok := s.cache[hash]
		require.True(t, ok)
		assert.Equal(t, want, item.u)
		c.AssertExpectations(t)
	})

	t.Run("grafana auth failure is returned", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := mkReq(t, "Bearer fail")
		headers := extractAuthHeaders(req)

		c.On("getAuthUser", mock.Anything, headers, mock.Anything).
			Return(authUser{}, &clientError{Code: http.StatusUnauthorized, ErrorMessage: http.StatusText(http.StatusUnauthorized)}).
			Once()

		got, authErr := s.getAuthUser(t.Context(), req, l)
		assert.Nil(t, got)
		require.NotNil(t, authErr)
		assert.Equal(t, codes.Unauthenticated, authErr.code)
		assert.Empty(t, s.cache, "cache should be empty on auth failure")
		c.AssertExpectations(t)
	})
}

func TestAuthServerProcessRequest(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())

	t.Run("none role path bypasses auth", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/readyz", nil)

		res, authErr := s.processRequest(t.Context(), req, l)
		assert.Nil(t, authErr)
		assert.Nil(t, res)
		assert.Empty(t, s.cache, "cache should be empty on none role path")
		c.AssertNotCalled(t, "getAuthUser", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("authentication failure is propagated", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/settings", nil)
		req.Header.Set("Authorization", "Bearer bad")

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{}, &clientError{Code: http.StatusUnauthorized, ErrorMessage: http.StatusText(http.StatusUnauthorized)}).
			Once()

		res, authErr := s.processRequest(t.Context(), req, l)
		assert.Nil(t, res)
		require.NotNil(t, authErr)
		assert.Equal(t, codes.Unauthenticated, authErr.code)
		assert.Empty(t, s.cache, "cache should be empty on auth failure")
		c.AssertExpectations(t)
	})

	t.Run("authorization failure is propagated", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/server/settings", nil)
		token := "viewer"
		req.Header.Set("Authorization", "Bearer "+token)
		headers := extractAuthHeaders(req)

		userInfo := authUser{role: viewer, userID: 11}
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(userInfo, nil).
			Once()

		res, authErr := s.processRequest(t.Context(), req, l)
		assert.Nil(t, res)

		j, err := json.Marshal(headers)
		require.NoError(t, err)
		hash := base64.StdEncoding.EncodeToString(j)
		item, ok := s.cache[hash]
		require.True(t, ok)
		assert.Equal(t, userInfo, item.u)

		assert.Equal(t, errStaticAuthErrorPermissionDenied, authErr)
		c.AssertExpectations(t)
	})

	t.Run("success returns auth result", func(t *testing.T) {
		t.Parallel()

		for uri, minRole := range rules {
			for _, role := range []role{viewer, editor, admin} {
				t.Run(fmt.Sprintf("uri=%s,minRole=%s,role=%s", uri, minRole, role), func(t *testing.T) {
					t.Parallel()

					c := newMockClientInterface(t)
					s := newTestAuthServer(c, nil)

					token := fmt.Sprintf("%s-%s-%d", minRole, role, time.Now().Nanosecond())

					req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, uri, nil)
					req.Header.Set("Authorization", "Bearer "+token)

					if minRole > none {
						userInfo := authUser{role: role, userID: 99}
						c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
							Return(userInfo, nil).
							Once()
					}

					res, authErr := s.processRequest(t.Context(), req, l)
					if minRole <= role {
						require.Nil(t, authErr)
						// require.NotNil(t, res)
						// assert.Empty(t, res.vmProxyFilters)
					} else {
						assert.Equal(t, errStaticAuthErrorPermissionDenied, authErr)
						require.Nil(t, res)
					}

					c.AssertExpectations(t)
				})
			}
		}
	})

	t.Run("access forbidden for anonymous user", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/qan/query", nil)

		userInfo := authUser{role: none, userID: 0}
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(userInfo, nil).
			Once()

		res, authErr := s.processRequest(t.Context(), req, l)
		assert.Equal(t, errStaticAuthErrorPermissionDenied, authErr)
		require.Nil(t, res)
		// assert.True(t, len(s.cache) == 0, "cache should be empty on anonymous user")
		c.AssertExpectations(t)
	})

	t.Run("access granted for anonymous user", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/qan", nil)

		userInfo := authUser{role: viewer, userID: 0}
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(userInfo, nil).
			Once()

		res, authErr := s.processRequest(t.Context(), req, l)
		assert.Nil(t, authErr)
		assert.NotNil(t, res)
		assert.Empty(t, res.vmProxyFilters)
		// assert.True(t, len(s.cache) == 0, "cache should be empty on anonymous user")
		c.AssertExpectations(t)
	})

	t.Run("access granted for anonymous user with LBAC enabled", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)

		sqlDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, sqlMock.ExpectationsWereMet())
			_ = sqlDB.Close()
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		s := newTestAuthServer(c, db)

		setAccessControlCachedValue(s, true)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/qan", nil)

		userInfo := authUser{role: viewer, userID: 0}
		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(userInfo, nil).
			Once()

		res, authErr := s.processRequest(t.Context(), req, l)
		assert.Nil(t, authErr)
		assert.NotNil(t, res)
		assert.Empty(t, res.vmProxyFilters)
		// assert.True(t, len(s.cache) == 0, "cache should be empty on anonymous user")
		c.AssertExpectations(t)
	})
}

func TestAuthServerServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("bad original request headers returns 400", func(t *testing.T) {
		t.Parallel()
		s := newTestAuthServer(newMockClientInterface(t), nil)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)

		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, req)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("permission denied returns 403 json payload", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)

		req := newOriginalReq(t, http.MethodGet, "/v1/server/settings")
		req.Header.Set("Authorization", "Bearer viewer")

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{role: viewer, userID: 17}, nil).
			Once()

		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusForbidden, rw.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &body))
		code, ok := body["code"].(float64)
		require.True(t, ok)
		assert.InDelta(t, float64(codes.PermissionDenied), code, 0)
		assert.Equal(t, errStaticAuthErrorPermissionDenied.message, body["error"])
		assert.Equal(t, errStaticAuthErrorPermissionDenied.message, body["message"])
		c.AssertExpectations(t)
	})

	t.Run("success with disabled LBAC", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s := newTestAuthServer(c, nil)

		req := newOriginalReq(t, http.MethodGet, "/prometheus/api/v1/query")
		req.Header.Set("Authorization", "Bearer admin")

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{role: admin, userID: 1001}, nil).
			Once()

		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		header := rw.Header().Get(lbacHeaderName)
		require.Empty(t, header)
		c.AssertExpectations(t)
	})

	t.Run("success with enabled LBAC", func(t *testing.T) {
		t.Parallel()
		c := newMockClientInterface(t)
		s, sqlMock := setupLBACServer(t)

		s.c = c
		sqlMock.ExpectQuery("SELECT").WithArgs(1001).WillReturnRows(roleRows(
			struct {
				id     uint32
				title  string
				filter string
			}{id: 1, title: "Role A", filter: "filter-a"},
			struct {
				id     uint32
				title  string
				filter string
			}{id: 2, title: "Role B", filter: "filter-b"},
		))

		req := newOriginalReq(t, http.MethodGet, "/prometheus/api/v1/query")
		req.Header.Set("Authorization", "Bearer admin")

		c.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{role: admin, userID: 1001}, nil).
			Once()

		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		header := rw.Header().Get(lbacHeaderName)
		require.NotEmpty(t, header)

		decoded, err := base64.StdEncoding.DecodeString(header)
		require.NoError(t, err)
		var filters []string
		require.NoError(t, json.Unmarshal(decoded, &filters))
		assert.Equal(t, []string{"filter-a", "filter-b"}, filters)
		c.AssertExpectations(t)
	})
}
