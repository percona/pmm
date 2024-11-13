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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
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
		{"/v1/AWSInstanceCheck/..%2finventory/Services/List'"},
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
		req, err := http.NewRequest(http.MethodGet, "/graph", nil)
		require.NoError(t, err)

		checker := &mockAwsInstanceChecker{}
		checker.Test(t)
		defer checker.AssertExpectations(t)

		s := NewAuthServer(nil, checker, nil)

		t.Run("Subrequest", func(t *testing.T) {
			checker.On("MustCheck").Return(true)
			rw := httptest.NewRecorder()
			assert.True(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint
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
			defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint
			assert.Equal(t, 303, resp.StatusCode)
			assert.Equal(t, "", resp.Header.Get("X-Must-Setup"))
			assert.Equal(t, "/setup", resp.Header.Get("Location"))
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Empty(t, b)
		})
	})

	t.Run("MustNotCheck", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/graph", nil)
		require.NoError(t, err)

		checker := &mockAwsInstanceChecker{}
		checker.Test(t)
		defer checker.AssertExpectations(t)

		s := NewAuthServer(nil, checker, nil)

		t.Run("Subrequest", func(t *testing.T) {
			checker.On("MustCheck").Return(false)
			rw := httptest.NewRecorder()
			assert.False(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, "", resp.Header.Get("X-Must-Setup"))
			assert.Equal(t, "", resp.Header.Get("Location"))
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Empty(t, b)
		})
	})

	t.Run("SkipNonUI", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/dummy", nil)
		require.NoError(t, err)

		checker := &mockAwsInstanceChecker{}
		checker.Test(t)
		defer checker.AssertExpectations(t)

		s := NewAuthServer(nil, checker, nil)

		t.Run("Subrequest", func(t *testing.T) {
			rw := httptest.NewRecorder()
			assert.False(t, s.mustSetup(rw, req, logrus.WithField("test", t.Name())))

			resp := rw.Result()
			defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint
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
	t.Parallel()
	// logrus.SetLevel(logrus.TraceLevel)

	checker := &mockAwsInstanceChecker{}
	checker.Test(t)
	t.Cleanup(func() { checker.AssertExpectations(t) })

	ctx := context.Background()
	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c, checker, nil)

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

	for uri, minRole := range map[string]role{
		"/agent.Agent/Connect": none,

		"/inventory.Nodes/ListNodes":                          admin,
		"/management.Actions/StartMySQLShowTableStatusAction": viewer,
		"/management.Service/RemoveService":                   admin,
		"/management.Service/ListServices":                    admin,
		"/management.Annotation/AddAnnotation":                admin,
		"/server.Server/CheckUpdates":                         viewer,
		"/server.Server/StartUpdate":                          admin,
		"/server.Server/UpdateStatus":                         none,
		"/server.Server/AWSInstanceCheck":                     none,

		"/v1/inventory/Nodes/List":                         admin,
		"/v1/management/Actions/StartMySQLShowTableStatus": viewer,
		"/v1/management/Service/Remove":                    admin,
		"/v1/management/Service/List":                      admin,
		"/v1/management/Agent/List":                        admin,
		"/v1/Updates/Check":                                viewer,
		"/v1/Updates/Start":                                admin,
		"/v1/Updates/Status":                               none,
		"/v1/Settings/Get":                                 admin,
		"/v1/AWSInstanceCheck":                             none,
		"/v1/Platform/Connect":                             admin,

		"/v1/AWSInstanceCheck/..%2finventory/Services/List": admin,
		"/v1/AWSInstanceCheck/..%2f..%2flogs.zip":           admin,

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
				// This test couldn't run in parallel on sqlite3 - they locked Grafana's sqlite3 database
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
					assert.Equal(t, &authError{code: codes.PermissionDenied, message: "Access denied."}, res)
				}
			})
		}
	}
}

func TestAuthServerAddVMGatewayToken(t *testing.T) {
	ctx := logger.Set(context.Background(), t.Name())
	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	defer func(t *testing.T) {
		t.Helper()

		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())
	}(t)

	var checker mockAwsInstanceChecker
	checker.Test(t)
	defer checker.AssertExpectations(t)

	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c, &checker, db)

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
		EnableAccessControl: true,
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
			"/":                        false,
			"/dummy":                   false,
			"/prometheus/api/":         false,
			"/prometheus/api/v1/":      true,
			"/prometheus/api/v1/query": true,
		} {
			uri := uri
			shallAdd := shallAdd

			for _, userID := range []int{0, 1337, 1338} {
				userID := userID
				t.Run(fmt.Sprintf("uri=%s userID=%d", uri, userID), func(t *testing.T) {
					t.Parallel()
					rw := httptest.NewRecorder()
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
					require.NoError(t, err)
					if userID == 0 {
						req.SetBasicAuth("admin", "admin")
					}

					err = s.maybeAddVMProxyFilters(ctx, rw, req, userID, logrus.WithField("test", t.Name()))
					require.NoError(t, err)

					headerString := rw.Header().Get(vmProxyHeaderName)

					if shallAdd {
						require.True(t, len(headerString) > 0)
					} else {
						require.Equal(t, headerString, "")
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

		err = s.maybeAddVMProxyFilters(ctx, rw, req, 1338, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		headerString := rw.Header().Get(vmProxyHeaderName)
		require.True(t, len(headerString) > 0)

		filters, err := base64.StdEncoding.DecodeString(headerString)
		require.NoError(t, err)
		var parsed []string
		err = json.Unmarshal(filters, &parsed)
		require.NoError(t, err)

		require.Equal(t, len(parsed), 2)
		require.Equal(t, parsed[0], "filter A")
		require.Equal(t, parsed[1], "filter B")
	})

	//nolint:paralleltest
	t.Run("shall not add any filters if at least one role has full access", func(t *testing.T) {
		rw := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/prometheus/api/v1/", nil)
		require.NoError(t, err)

		err = s.maybeAddVMProxyFilters(ctx, rw, req, 1339, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		headerString := rw.Header().Get(vmProxyHeaderName)
		require.Equal(t, len(headerString), 0)
	})
}

func Test_cleanPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path     string
		expected string
	}{
		{
			"/v1/AWSInstanceCheck/..%2finventory/Services/List",
			"/v1/inventory/Services/List",
		}, {
			"/v1/AWSInstanceCheck/..%2f..%2fmanaged/logs.zip",
			"/managed/logs.zip",
		}, {
			"/v1/AWSInstanceCheck/..%2f..%2f/logs.zip",
			"/logs.zip",
		}, {
			"/graph/api/datasources/proxy/8/?query=WITH%20(%0A%20%20%20%20CASE%20%0A%20%20%20%20%20%20%20%20WHEN%20(3000%20%25%2060)%20%3D%200%20THEN%203000%0A%20%20%20%20ELSE%2060%20END%0A)%20AS%20scale%0ASELECT%0A%20%20%20%20(intDiv(toUInt32(timestamp)%2C%203000)%20*%203000)%20*%201000%20as%20t%2C%0A%20%20%20%20hostname%20h%2C%0A%20%20%20%20status%20s%2C%0A%20%20%20%20SUM(req_count)%20as%20req_count%0AFROM%20pinba.report_by_all%0AWHERE%0A%20%20%20%20timestamp%20%3E%3D%20toDateTime(1707139680)%20AND%20timestamp%20%3C%3D%20toDateTime(1707312480)%0A%20%20%20%20AND%20status%20%3E%3D%20400%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20schema%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20hostname%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20server_name%20%3D%20%27all%27%20ELSE%201%20END%0AGROUP%20BY%20t%2C%20h%2C%20s%0AORDER%20BY%20t%20FORMAT%20JSON",
			"/graph/api/datasources/proxy/8/",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			cleanedPath, err := cleanPath(tt.path)
			require.NoError(t, err)
			assert.Equalf(t, tt.expected, cleanedPath, "cleanPath(%v)", tt.path)
		})
	}
}
