// pmm-managed
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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthServer(t *testing.T) {
	// logrus.SetLevel(logrus.TraceLevel)

	ctx := context.Background()
	c := NewClient("127.0.0.1:3000")
	s := NewAuthServer(c)

	req, err := http.NewRequest("GET", "/dummy", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "admin")
	authHeaders := req.Header

	t.Run("GrafanaAdminFallback", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", "/auth_request", nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")
		req.Header.Set("X-Original-Uri", "/foo")

		code := s.authenticate(ctx, req)
		assert.Equal(t, 200, code)
	})

	t.Run("EmptyOriginalUri", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", "/auth_request", nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")

		code := s.authenticate(ctx, req)
		assert.Equal(t, 500, code)
	})

	for uri, minRole := range map[string]role{
		"/v0/inventory/Nodes/List": editor,
		"/v0/inventory/Nodes/":     admin,
		"/v0/inventory/Nodes":      admin,
		"/v0/inventory/":           admin,
		"/agent.Agent/Connect":     none,
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

				req, err := http.NewRequest("GET", "/auth_request", nil)
				require.NoError(t, err)
				req.SetBasicAuth(login, login)
				req.Header.Set("X-Original-Uri", uri)

				code := s.authenticate(ctx, req)
				if minRole <= role {
					assert.Equal(t, 200, code)
				} else {
					assert.Equal(t, 403, code)
				}
			})
		}
	}
}
