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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/managed/utils/tests"
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

func TestAuthServerMustSetup(t *testing.T) {
	t.Run("MustCheck", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/graph", nil)
		require.NoError(t, err)

		checker := &mockAwsInstanceChecker{}
		checker.Test(t)
		defer checker.AssertExpectations(t)

		s := NewAuthServer(nil, checker)

		t.Run("Subrequest", func(t *testing.T) {
			checker.On("MustCheck").Return(true)
			rw := httptest.NewRecorder()
			assert.True(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:errcheck
			assert.Equal(t, 401, resp.StatusCode)
			assert.Equal(t, "1", resp.Header.Get("X-Must-Setup"))
			assert.Equal(t, "", resp.Header.Get("Location"))
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Empty(t, b)
		})

		t.Run("Request", func(t *testing.T) {
			req.Header.Set("X-Must-Setup", "1")

			checker.On("MustCheck").Return(true)
			rw := httptest.NewRecorder()
			assert.True(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:errcheck
			assert.Equal(t, 303, resp.StatusCode)
			assert.Equal(t, "", resp.Header.Get("X-Must-Setup"))
			assert.Equal(t, "/setup", resp.Header.Get("Location"))
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Empty(t, b)
		})
	})

	t.Run("MustNotCheck", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/graph", nil)
		require.NoError(t, err)

		checker := &mockAwsInstanceChecker{}
		checker.Test(t)
		defer checker.AssertExpectations(t)

		s := NewAuthServer(nil, checker)

		t.Run("Subrequest", func(t *testing.T) {
			checker.On("MustCheck").Return(false)
			rw := httptest.NewRecorder()
			assert.False(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:errcheck
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, "", resp.Header.Get("X-Must-Setup"))
			assert.Equal(t, "", resp.Header.Get("Location"))
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Empty(t, b)
		})
	})

	t.Run("SkipNonUI", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/dummy", nil)
		require.NoError(t, err)

		checker := &mockAwsInstanceChecker{}
		checker.Test(t)
		defer checker.AssertExpectations(t)

		s := NewAuthServer(nil, checker)

		t.Run("Subrequest", func(t *testing.T) {
			rw := httptest.NewRecorder()
			assert.False(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:errcheck
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, "", resp.Header.Get("X-Must-Setup"))
			assert.Equal(t, "", resp.Header.Get("Location"))
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Empty(t, b)
		})
	})
}

func TestAuthServerAuthenticate(t *testing.T) {
	// logrus.SetLevel(logrus.TraceLevel)

	checker := &mockAwsInstanceChecker{}
	checker.Test(t)
	defer checker.AssertExpectations(t)

	ctx := context.Background()
	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c, checker)

	req, err := http.NewRequest("GET", "/dummy", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "admin")
	authHeaders := req.Header

	t.Run("GrafanaAdminFallback", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", "/foo", nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")

		res := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Nil(t, res)
	})

	t.Run("NoAnonymousAccess", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", "/foo", nil)
		require.NoError(t, err)

		res := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		assert.Equal(t, &authError{code: codes.Unauthenticated, message: "Unauthorized"}, res)
	})

	for uri, minRole := range map[string]role{
		"/agent.Agent/Connect": none,

		"/inventory.Nodes/ListNodes":                          admin,
		"/management.Actions/StartMySQLShowTableStatusAction": viewer,
		"/management.Service/RemoveService":                   admin,
		"/management.Annotation/AddAnnotation":                admin,
		"/server.Server/CheckUpdates":                         viewer,
		"/server.Server/StartUpdate":                          admin,
		"/server.Server/UpdateStatus":                         none,
		"/server.Server/AWSInstanceCheck":                     none,

		"/v1/inventory/Nodes/List":                         admin,
		"/v1/management/Actions/StartMySQLShowTableStatus": viewer,
		"/v1/management/Service/Remove":                    admin,
		"/v1/Updates/Check":                                viewer,
		"/v1/Updates/Start":                                admin,
		"/v1/Updates/Status":                               none,
		"/v1/Settings/Get":                                 admin,
		"/v1/AWSInstanceCheck":                             none,
		"/v1/Platform/Connect":                             admin,

		"/v1/readyz": none,
		"/ping":      none,

		"/v1/version":         viewer,
		"/managed/v1/version": viewer,

		"/v0/qan/ObjectDetails/GetQueryExample": viewer,

		"/prometheus/":   admin,
		"/alertmanager/": admin,
		"/logs.zip":      admin,
	} {
		for _, role := range []role{viewer, editor, admin} {
			uri := uri
			minRole := minRole
			role := role

			t.Run(fmt.Sprintf("uri=%s,minRole=%s,role=%s", uri, minRole, role), func(t *testing.T) {
				// do not run this test in parallel - they lock Grafana's sqlite3 database
				// t.Parallel()

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

				req, err := http.NewRequest("GET", uri, nil)
				require.NoError(t, err)
				req.SetBasicAuth(login, login)

				res := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
				if minRole <= role {
					assert.Nil(t, res)
				} else {
					assert.Equal(t, &authError{code: codes.PermissionDenied, message: "Access denied."}, res)
				}
			})
		}
	}
}
