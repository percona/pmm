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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
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

	// An agent token carries the None role, so it connects on the strength of its binding to
	// a registered node and on nothing else. This exercises the whole path: a real Grafana
	// service account, a real node row, and the binding between them.
	t.Run("Token auth - success", func(t *testing.T) {
		t.Parallel()

		nodeName := fmt.Sprintf("N1-%d", time.Now().UnixNano())
		headersMD := metadata.New(map[string]string{
			"Authorization": "Basic YWRtaW46YWRtaW4=",
		})
		ctx := metadata.NewIncomingContext(t.Context(), headersMD)
		serviceAccountID, serviceToken, err := c.CreateServiceAccount(ctx, nodeName, true)
		require.NoError(t, err)
		defer func() {
			warning, err := c.DeleteServiceAccount(ctx, nodeName, true)
			require.NoError(t, err)
			require.Empty(t, warning)
		}()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: nodeName,
			Address:  "10.30.30.30",
		})
		require.NoError(t, err)
		require.NoError(t, models.SetNodeServiceAccountID(db.Querier, node.NodeID, serviceAccountID))

		boundServer := NewAuthServer(c, db)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectionEndpoint, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+serviceToken)

		user, authError := boundServer.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		require.Nil(t, authError)
		require.NotNil(t, user)
		assert.Equal(t, none, user.role)
		assert.Equal(t, node.NodeID, user.nodeID)

		// The same token gets nowhere without the binding.
		_, authError = s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
		require.NotNil(t, authError)
		assert.Equal(t, codes.PermissionDenied, authError.code)
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
	}{
		{
			"/v1/server/AWSInstanceCheck/..%2f..%2finventory/Services/List",
			"/v1/inventory/Services/List",
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2f..%2fmanaged/logs.zip",
			"/managed/logs.zip",
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2f..%2f/logs.zip",
			"/logs.zip",
		}, {
			"/graph/api/datasources/proxy/8/?query=WITH%20(%0A%20%20%20%20CASE%20%0A%20%20%20%20%20%20%20%20WHEN%20(3000%20%25%2060)%20%3D%200%20THEN%203000%0A%20%20%20%20ELSE%2060%20END%0A)%20AS%20scale%0ASELECT%0A%20%20%20%20(intDiv(toUInt32(timestamp)%2C%203000)%20*%203000)%20*%201000%20as%20t%2C%0A%20%20%20%20hostname%20h%2C%0A%20%20%20%20status%20s%2C%0A%20%20%20%20SUM(req_count)%20as%20req_count%0AFROM%20pinba.report_by_all%0AWHERE%0A%20%20%20%20timestamp%20%3E%3D%20toDateTime(1707139680)%20AND%20timestamp%20%3C%3D%20toDateTime(1707312480)%0A%20%20%20%20AND%20status%20%3E%3D%20400%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20schema%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20hostname%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20server_name%20%3D%20%27all%27%20ELSE%201%20END%0AGROUP%20BY%20t%2C%20h%2C%20s%0AORDER%20BY%20t%20FORMAT%20JSON",
			"/graph/api/datasources/proxy/8/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			cleanedPath, err := cleanPath(tt.path)
			require.NoError(t, err)
			assert.Equalf(t, tt.expected, cleanedPath, "cleanPath(%v)", tt.path)
		})
	}
}

// fakeAuthClient returns a fixed authUser, standing in for Grafana so the binding logic
// can be exercised without minting real service accounts.
type fakeAuthClient struct {
	u authUser
}

func (f *fakeAuthClient) getAuthUser(_ context.Context, _ http.Header, _ *logrus.Entry) (authUser, error) {
	return f.u, nil
}

func TestAuthServerTokenBinding(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "bound-node",
		Address:  "10.20.30.40",
	})
	require.NoError(t, err)

	const boundServiceAccountID = 4242
	require.NoError(t, models.SetNodeServiceAccountID(db.Querier, node.NodeID, boundServiceAccountID))

	for _, tc := range []struct {
		name             string
		serviceAccountID int
		path             string
		wantAllowed      bool
	}{
		// A bound token reaches everything a pmm-agent needs, with no Grafana role at all.
		{"bound agent connect", boundServiceAccountID, connectionEndpoint, true},
		{"bound v2 agent connect", boundServiceAccountID, connectionEndpointV2, true},
		{"bound rta collect", boundServiceAccountID, rtaCollectEndpoint, true},
		{"bound vm write", boundServiceAccountID, "/victoriametrics/api/v1/write", true},

		// ...and the inventory it manages on that node.
		{"bound inventory", boundServiceAccountID, "/v1/inventory/services", true},

		// The binding grants those and nothing else.
		{"bound backups", boundServiceAccountID, "/v1/backups", false},
		{"bound settings", boundServiceAccountID, "/v1/server/settings", false},
		{"bound server logs", boundServiceAccountID, "/v1/server/logs.zip", false},
		{"bound vm read", boundServiceAccountID, "/victoriametrics/api/v1/query", false},

		// An unbound service account gets nothing from the binding path.
		{"unbound agent connect", 9999, connectionEndpoint, false},
		{"no service account", 0, connectionEndpoint, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &fakeAuthClient{u: authUser{role: none, serviceAccountID: tc.serviceAccountID}}
			s := NewAuthServer(c, db)

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, tc.path, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+tc.name)

			u, authErr := s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
			if !tc.wantAllowed {
				require.NotNil(t, authErr, "expected access to be denied")
				assert.Equal(t, codes.PermissionDenied, authErr.code)
				return
			}

			require.Nil(t, authErr)
			require.NotNil(t, u)
			assert.Equal(t, node.NodeID, u.nodeID)
		})
	}
}

func TestIsGrantedByBinding(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
		want   bool
	}{
		// The pmm-agent daemon.
		{http.MethodPost, connectionEndpoint, true},
		{http.MethodPost, connectionEndpointV2, true},
		{http.MethodPost, rtaCollectEndpoint, true},
		{http.MethodPost, "/victoriametrics/api/v1/write", true},

		// Everything pmm-admin does on behalf of its node.
		{http.MethodDelete, "/v1/management/nodes/node-1", true},   // unregister
		{http.MethodPost, "/v1/management/services", true},         // add <type>
		{http.MethodDelete, "/v1/management/services/svc-1", true}, // remove
		{http.MethodPost, "/v1/management/annotations", true},      // annotate
		{http.MethodGet, "/v1/management/nodes", true},             // list
		{http.MethodGet, "/v1/management/nodes/node-1", true},
		{http.MethodGet, "/v1/management/agents", true},
		{http.MethodGet, "/v1/management/agents/versions", true},
		{http.MethodGet, "/v1/management/services", true},
		{http.MethodGet, "/v1/inventory/services", true},
		{http.MethodGet, "/v1/inventory/nodes", true},
		{http.MethodGet, "/v1/inventory/agents", true},
		{http.MethodPost, "/v1/inventory/agents", true},
		{http.MethodPut, "/v1/inventory/agents/agent-1", true},
		{http.MethodDelete, "/v1/inventory/agents/agent-1", true},

		// Registration mints a Grafana service account with the caller's credentials,
		// which a None-role token cannot do, so it is not granted here.
		{http.MethodPost, "/v1/management/nodes", false},

		// The method matters: a bound token may delete a service, not create a node.
		{http.MethodPost, "/v1/management/nodes/node-1", false},
		{http.MethodPut, "/v1/management/services", false},

		// Server-wide surfaces stay out of reach.
		{http.MethodGet, "/v1/server/logs.zip", false},
		{http.MethodGet, "/v1/server/settings", false},
		{http.MethodPut, "/v1/server/settings", false},
		{http.MethodGet, "/v1/backups", false},
		{http.MethodGet, "/v1/dumps", false},
		{http.MethodGet, "/v1/accesscontrol", false},
		{http.MethodGet, "/v1/platform", false},
		{http.MethodGet, "/v1/users", false},
		{http.MethodGet, "/prometheus/api/v1/query", false},
		{http.MethodGet, "/victoriametrics/api/v1/query", false},
		{http.MethodGet, "/v1/qan/metrics:getFilters", false},
		{http.MethodGet, "/graph/api/serviceaccounts", false},

		// Neighbouring paths must not be swept in by prefix matching.
		{http.MethodPost, "/v1/management/services:discoverRDS", false},
		{http.MethodPost, "/v1/management/services:discoverAzure", false},
		{http.MethodGet, "/v1/inventoryX", false},
		{http.MethodGet, "/v1/inventory", false},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, isGrantedByBinding(tc.method, tc.path))
		})
	}
}

// A PMM-issued agent token is resolved against PMM's own database, so it must authenticate
// with no Grafana involvement whatsoever. NewClient points at a dead address here: if any of
// this reached Grafana the test would fail rather than silently pass.
func TestAuthServerAgentToken(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	nodeA, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "agent-token-node-a", Address: "10.2.0.1",
	})
	require.NoError(t, err)
	nodeB, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "agent-token-node-b", Address: "10.2.0.2",
	})
	require.NoError(t, err)

	_, tokenA, err := models.CreateAgentToken(db.Querier, nodeA.NodeID)
	require.NoError(t, err)
	_, tokenB, err := models.CreateAgentToken(db.Querier, nodeB.NodeID)
	require.NoError(t, err)

	// Port 1 has nothing listening: any Grafana call fails loudly.
	s := NewAuthServer(NewClient("127.0.0.1:1"), db)

	authenticate := func(t *testing.T, method, path, token string) (*authUser, *authError) {
		t.Helper()

		req, err := http.NewRequestWithContext(ctx, method, path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		return s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
	}

	t.Run("authenticates and carries its node without Grafana", func(t *testing.T) {
		for _, path := range []string{
			connectionEndpoint,
			rtaCollectEndpoint,
			"/victoriametrics/api/v1/write",
			"/v1/inventory/services",
		} {
			user, authErr := authenticate(t, http.MethodPost, path, tokenA)
			require.Nil(t, authErr, "path = %s", path)
			require.NotNil(t, user)
			assert.Equal(t, nodeA.NodeID, user.nodeID)
			assert.Equal(t, none, user.role, "an agent token carries no Grafana role at all")
		}
	})

	t.Run("reaches nothing beyond the agent grant", func(t *testing.T) {
		for _, path := range []string{
			"/v1/backups",
			"/v1/server/settings",
			"/v1/server/logs.zip",
			"/v1/accesscontrol",
			"/prometheus/api/v1/query",
			"/v1/management/nodes", // registration
		} {
			_, authErr := authenticate(t, http.MethodPost, path, tokenA)
			require.NotNil(t, authErr, "path = %s", path)
			assert.Equal(t, codes.PermissionDenied, authErr.code)
		}
	})

	t.Run("scope follows the token, and BoundNodeID agrees with authenticate", func(t *testing.T) {
		for token, wantNode := range map[string]string{tokenA: nodeA.NodeID, tokenB: nodeB.NodeID} {
			user, authErr := authenticate(t, http.MethodGet, "/v1/inventory/services", token)
			require.Nil(t, authErr)
			assert.Equal(t, wantNode, user.nodeID)

			md := metadata.New(map[string]string{"Authorization": "Bearer " + token})
			assert.Equal(t, wantNode, s.BoundNodeID(metadata.NewIncomingContext(ctx, md)),
				"the interceptor must confine the token to the same node authenticate did")
		}
	})

	t.Run("a revoked token stops working", func(t *testing.T) {
		_, token, err := models.CreateAgentToken(db.Querier, nodeA.NodeID)
		require.NoError(t, err)
		user, authErr := authenticate(t, http.MethodGet, "/v1/inventory/services", token)
		require.Nil(t, authErr)
		require.NotNil(t, user)

		require.NoError(t, models.RemoveAgentTokensForNode(db.Querier, nodeA.NodeID))

		// Falls through to Grafana, which is unreachable, so it cannot be replayed.
		_, authErr = authenticate(t, http.MethodGet, "/v1/inventory/services", token)
		require.NotNil(t, authErr)

		md := metadata.New(map[string]string{"Authorization": "Bearer " + token})
		assert.Empty(t, s.BoundNodeID(metadata.NewIncomingContext(ctx, md)))
	})
}

// The loopback exception used to admit every caller: nginx makes every auth_request
// subrequest from 127.0.0.1, so testing req.RemoteAddr matched unconditionally and left
// Connect and RTA Collect reachable without credentials from anywhere on the network.
func TestIsLocalAgentConnection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		path   string
		realIP string
		want   bool
	}{
		// PMM Server's own agent, which holds no credentials.
		{"loopback v4", connectionEndpoint, "127.0.0.1", true},
		{"loopback v6", connectionEndpoint, "::1", true},
		{"loopback with port", connectionEndpoint, "127.0.0.1:54321", true},
		{"loopback rta", rtaCollectEndpoint, "127.0.0.1", true},

		// Anything arriving over the network must authenticate.
		{"remote client", connectionEndpoint, "10.0.0.7", false},
		{"remote client rta", rtaCollectEndpoint, "192.168.1.50", false},
		{"public address", connectionEndpoint, "203.0.113.9", false},

		// Absent or unparseable means we cannot prove it is local, so it is not.
		{"header missing", connectionEndpoint, "", false},
		{"header garbage", connectionEndpoint, "not-an-ip", false},

		// The exception is limited to the two agent paths regardless of origin.
		{"loopback but other path", "/v1/inventory/services", "127.0.0.1", false},
		{"loopback but backups", "/v1/backups", "127.0.0.1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/auth_request", nil)
			require.NoError(t, err)
			// nginx always reaches pmm-managed from loopback; the original client is in X-Real-IP.
			req.RemoteAddr = "127.0.0.1:33333"
			req.Header.Set("X-Original-Uri", tc.path)
			if tc.realIP != "" {
				req.Header.Set("X-Real-IP", tc.realIP)
			}

			assert.Equal(t, tc.want, isLocalAgentConnection(req))
		})
	}
}

// An enrollment token exists so an operator can add hosts without holding Grafana Org Admin.
// It must therefore reach node registration and nothing else at all.
func TestAuthServerEnrollmentToken(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	_, token, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
		Description: "auth test",
	})
	require.NoError(t, err)

	_, exhausted, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
		Description: "used up", MaxUses: 1,
	})
	require.NoError(t, err)
	require.NoError(t, models.UseEnrollmentToken(db.Querier, exhausted))

	// Port 1 has nothing listening: any Grafana call fails rather than quietly succeeding.
	s := NewAuthServer(NewClient("127.0.0.1:1"), db)

	authenticate := func(t *testing.T, method, path, token string) (*authUser, *authError) {
		t.Helper()

		req, err := http.NewRequestWithContext(ctx, method, path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		return s.authenticate(ctx, req, logrus.WithField("test", t.Name()))
	}

	t.Run("registers a node", func(t *testing.T) {
		_, authErr := authenticate(t, http.MethodPost, "/v1/management/nodes", token)
		assert.Nil(t, authErr)
	})

	t.Run("reaches nothing else", func(t *testing.T) {
		for _, tc := range []struct{ method, path string }{
			// Not even the other half of node management.
			{http.MethodGet, "/v1/management/nodes"},
			{http.MethodDelete, "/v1/management/nodes/node-1"},
			// Nor anything an agent or pmm-admin does.
			{http.MethodPost, "/v1/management/services"},
			{http.MethodGet, "/v1/inventory/services"},
			{http.MethodPost, connectionEndpoint},
			{http.MethodPost, "/victoriametrics/api/v1/write"},
			// Nor, most importantly, minting more of itself.
			{http.MethodPost, "/v1/management/enrollmentTokens"},
			{http.MethodGet, "/v1/management/enrollmentTokens"},
			// Nor the administrative surfaces.
			{http.MethodGet, "/v1/backups"},
			{http.MethodGet, "/v1/server/settings"},
			{http.MethodGet, "/v1/server/logs.zip"},
			{http.MethodGet, "/v1/users"},
		} {
			_, authErr := authenticate(t, tc.method, tc.path, token)
			require.NotNil(t, authErr, "%s %s must be denied", tc.method, tc.path)
			assert.Equal(t, codes.PermissionDenied, authErr.code)
		}
	})

	t.Run("an exhausted token registers nothing", func(t *testing.T) {
		_, authErr := authenticate(t, http.MethodPost, "/v1/management/nodes", exhausted)
		require.NotNil(t, authErr)
	})

	t.Run("an unknown token registers nothing", func(t *testing.T) {
		_, authErr := authenticate(t, http.MethodPost, "/v1/management/nodes", models.EnrollmentTokenPrefix+"nope")
		require.NotNil(t, authErr)
	})
}
