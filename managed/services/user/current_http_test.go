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

package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/services/grafana"
)

type fakeCurrentUserClient struct {
	user    grafana.CurrentUser
	userErr error
	orgs    []grafana.CurrentUserOrg
	orgsErr error

	getUserCalls int
	getOrgsCalls int
}

func (f *fakeCurrentUserClient) GetCurrentUser(_ context.Context, _ http.Header) (grafana.CurrentUser, error) {
	f.getUserCalls++
	return f.user, f.userErr
}

func (f *fakeCurrentUserClient) GetCurrentUserOrgs(_ context.Context, _ http.Header) ([]grafana.CurrentUserOrg, error) {
	f.getOrgsCalls++
	return f.orgs, f.orgsErr
}

func TestCurrentHTTPHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/users/current"},
		{http.MethodPut, "/v1/users/current"},
		{http.MethodDelete, "/v1/users/current/orgs"},
		{http.MethodPatch, "/v1/users/current/orgs"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			f := &fakeCurrentUserClient{user: grafana.CurrentUser{Login: "u"}}
			h := NewCurrentHTTPHandler(f)

			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			assert.Equal(t, http.MethodGet, rec.Header().Get("Allow"))

			var body map[string]string
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
			assert.Equal(t, "Method Not Allowed", body["message"])
			assert.Zero(t, f.getUserCalls)
			assert.Zero(t, f.getOrgsCalls)
		})
	}
}

func TestCurrentHTTPHandler_GET_current(t *testing.T) {
	t.Parallel()

	f := &fakeCurrentUserClient{user: grafana.CurrentUser{Login: "alice", ID: 1}}
	h := NewCurrentHTTPHandler(f)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/users/current", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, f.getUserCalls)
	assert.Zero(t, f.getOrgsCalls)

	var got grafana.CurrentUser
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "alice", got.Login)
	assert.Equal(t, 1, got.ID)
}

func TestCurrentHTTPHandler_GET_orgs(t *testing.T) {
	t.Parallel()

	f := &fakeCurrentUserClient{orgs: []grafana.CurrentUserOrg{{OrgID: 1, Name: "Main", Role: "Viewer"}}}
	h := NewCurrentHTTPHandler(f)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/users/current/orgs", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Zero(t, f.getUserCalls)
	assert.Equal(t, 1, f.getOrgsCalls)

	var got []grafana.CurrentUserOrg
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].OrgID)
	assert.Equal(t, "Viewer", got[0].Role)
}

func TestCurrentHTTPHandler_GET_errorUsesGrafanaMapping(t *testing.T) {
	t.Parallel()

	f := &fakeCurrentUserClient{userErr: errors.New("upstream unavailable")}
	h := NewCurrentHTTPHandler(f)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/users/current", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Equal(t, 1, f.getUserCalls)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "Bad Gateway", body["message"])
}

func TestCurrentHTTPHandler_notFound(t *testing.T) {
	t.Parallel()

	f := &fakeCurrentUserClient{}
	h := NewCurrentHTTPHandler(f)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/users/other", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Zero(t, f.getUserCalls)
	assert.Zero(t, f.getOrgsCalls)
}
