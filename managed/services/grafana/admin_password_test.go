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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGrafana keeps the password of the admin user, ID 1. The session cookie names the user: admin, the server
// admin, or user, ID 2.
type fakeGrafana struct {
	m        sync.Mutex
	password string
	logins   int
}

func (g *fakeGrafana) adminPassword() string {
	g.m.Lock()
	defer g.m.Unlock()
	return g.password
}

func (g *fakeGrafana) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.m.Lock()
	defer g.m.Unlock()

	userID := 0
	if login, password, ok := r.BasicAuth(); ok {
		g.logins++
		if login == "admin" && password == g.password {
			userID = 1
		}
	}
	c, err := r.Cookie("grafana_session")
	if err == nil {
		userID = map[string]int{"admin": 1, "user": 2}[c.Value]
	}
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/user":
		_, _ = fmt.Fprintf(w, `{"id":%d}`, userID)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/users/"):
		if userID != 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"id":` + strings.TrimPrefix(r.URL.Path, "/api/users/") + `}`))
	case r.Method == http.MethodPut && r.URL.Path == "/graph/api/user/password":
		if userID == 1 {
			if body["oldPassword"] != g.password {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"Invalid old password"}`))
				return
			}
			g.password = body["newPassword"]
		}
		_, _ = w.Write([]byte(`{"message":"User password changed"}`))
	case r.Method == http.MethodPut && r.URL.Path == "/graph/api/admin/users/1/password":
		g.password = body["password"]
		_, _ = w.Write([]byte(`{"message":"User password updated"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// fakeAdminVault is the secret in Vault that the admin password reference points to.
type fakeAdminVault struct {
	m        sync.Mutex
	password string
	readErr  error
	writeErr error
	resets   []string
}

func newTestAdminPassword(t *testing.T, ref, grafanaPassword, vaultPassword string) (*AdminPassword, *fakeGrafana, *fakeAdminVault) {
	t.Helper()

	g := &fakeGrafana{password: grafanaPassword}
	srv := httptest.NewServer(g)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	v := &fakeAdminVault{password: vaultPassword}
	a := NewAdminPassword(ref, "admin", u.Host)
	a.read = func(context.Context) (string, error) {
		v.m.Lock()
		defer v.m.Unlock()
		return v.password, v.readErr
	}
	a.write = func(_ context.Context, password string) error {
		v.m.Lock()
		defer v.m.Unlock()
		if v.writeErr != nil {
			return v.writeErr
		}
		v.password = password
		return nil
	}
	a.reset = func(_ context.Context, password string) error {
		v.m.Lock()
		v.resets = append(v.resets, password)
		v.m.Unlock()

		g.m.Lock()
		defer g.m.Unlock()
		g.password = password
		return nil
	}
	return a, g, v
}

func changePassword(t *testing.T, a *AdminPassword, session, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, path, strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "grafana_session", Value: session})
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	return rec
}

const testAdminRef = "vault:secret/data/pmm/server#admin_password"

func TestAdminPasswordSync(t *testing.T) {
	t.Parallel()

	t.Run("in step, then changed in Vault", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		require.NoError(t, a.sync(t.Context()))
		assert.Empty(t, v.resets)

		v.password = "p2"
		require.NoError(t, a.sync(t.Context()))
		assert.Equal(t, []string{"p2"}, v.resets)
		assert.Equal(t, "p2", g.adminPassword())

		// Known since, so Grafana is not asked: a failed login counts toward Grafana's lockout.
		require.NoError(t, a.sync(t.Context()))
		assert.Len(t, v.resets, 1)
		assert.Equal(t, 1, g.logins)
	})

	t.Run("not in step at first", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "changed-before", "p1")
		require.NoError(t, a.sync(t.Context()))
		assert.Equal(t, []string{"p1"}, v.resets)
		assert.Equal(t, "p1", g.adminPassword())
	})

	t.Run("not from Vault", func(t *testing.T) {
		t.Parallel()

		a, _, v := newTestAdminPassword(t, "", "p1", "p2")
		a.Run(t.Context(), 0)
		assert.Empty(t, v.resets)
	})
}

func TestAdminPasswordChange(t *testing.T) {
	t.Parallel()

	t.Run("the admin's own", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		rec := changePassword(t, a, "admin", "/graph/api/user/password", `{"oldPassword":"p1","newPassword":"p2","confirmNew":"p2"}`)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"message":"User password changed"}`, rec.Body.String())
		assert.Equal(t, "p2", g.adminPassword())
		assert.Equal(t, "p2", v.password)

		// Not taken for a change in Vault, to be applied again.
		require.NoError(t, a.sync(t.Context()))
		assert.Empty(t, v.resets)
	})

	t.Run("by a server admin", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		rec := changePassword(t, a, "admin", "/graph/api/admin/users/1/password", `{"password":"p3"}`)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "p3", g.adminPassword())
		assert.Equal(t, "p3", v.password)
	})

	t.Run("refused by Grafana", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		rec := changePassword(t, a, "admin", "/graph/api/user/password", `{"oldPassword":"wrong","newPassword":"p2"}`)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.JSONEq(t, `{"message":"Invalid old password"}`, rec.Body.String())
		assert.Equal(t, "p1", g.adminPassword())
		assert.Equal(t, "p1", v.password)
	})

	t.Run("of another user", func(t *testing.T) {
		t.Parallel()

		a, _, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		rec := changePassword(t, a, "user", "/graph/api/user/password", `{"oldPassword":"x","newPassword":"y"}`)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "p1", v.password)
	})

	t.Run("Vault cannot store it", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		v.writeErr = errors.New("permission denied")
		rec := changePassword(t, a, "admin", "/graph/api/user/password", `{"oldPassword":"p1","newPassword":"p2"}`)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "The password could not be stored in Vault, so it was not changed: permission denied")
		assert.Equal(t, "p1", g.adminPassword(), "the password it had is given back")
		assert.Equal(t, []string{"p1"}, v.resets)
	})

	t.Run("Vault cannot be read", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, testAdminRef, "p1", "p1")
		v.readErr = errors.New("connection refused")
		rec := changePassword(t, a, "admin", "/graph/api/user/password", `{"oldPassword":"p1","newPassword":"p2"}`)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), "The password cannot be changed while the one in Vault cannot be read")
		assert.Equal(t, "p1", g.adminPassword(), "Grafana is not asked")
		assert.Empty(t, v.resets)
	})

	t.Run("not from Vault", func(t *testing.T) {
		t.Parallel()

		a, g, v := newTestAdminPassword(t, "", "p1", "p1")
		rec := changePassword(t, a, "admin", "/graph/api/user/password", `{"oldPassword":"p1","newPassword":"p2"}`)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "p2", g.adminPassword())
		assert.Equal(t, "p1", v.password)
	})
}
