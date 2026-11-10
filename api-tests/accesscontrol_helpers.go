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

package apitests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	accesscontrolClient "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client"
	accesscontrol "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client/access_control_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// SetAccessControl turns Access Control on or off and restores the previous setting when the test ends.
func SetAccessControl(t *testing.T, enabled bool) {
	t.Helper()

	was := readSettings(t.Context(), t).EnableAccessControl
	require.NoError(t, changeAccessControl(t.Context(), enabled))
	t.Cleanup(func() {
		err := changeAccessControl(Context, was)
		if err != nil {
			t.Errorf("failed to restore Access Control to %t: %s", was, err)
		}
	})
}

// CreateRole creates an Access Role with the filter and deletes it when the test ends.
func CreateRole(t *testing.T, title, filter string) int64 {
	t.Helper()

	res, err := accessControlService().CreateRole(&accesscontrol.CreateRoleParams{
		Body:    accesscontrol.CreateRoleBody{Title: title, Filter: filter, Description: "PMM-15531 MCP api-tests"},
		Context: t.Context(),
	})
	require.NoError(t, err)
	roleID := res.Payload.RoleID

	t.Cleanup(func() {
		_, err := accessControlService().DeleteRole(&accesscontrol.DeleteRoleParams{RoleID: roleID, Context: Context})
		if err != nil {
			t.Errorf("failed to delete Access Role %d: %s", roleID, err)
		}
	})

	return roleID
}

// AssignRoles replaces the user's Access Roles with roleIDs and gives the user the default Access Role when the test ends.
func AssignRoles(t *testing.T, userID int64, roleIDs ...int64) {
	t.Helper()

	_, err := accessControlService().AssignRoles(&accesscontrol.AssignRolesParams{
		Body:    accesscontrol.AssignRolesBody{UserID: userID, RoleIds: roleIDs},
		Context: t.Context(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		defaultRoleID := readSettings(Context, t).DefaultRoleID
		if defaultRoleID == 0 {
			return
		}

		_, err := accessControlService().AssignRoles(&accesscontrol.AssignRolesParams{
			Body:    accesscontrol.AssignRolesBody{UserID: userID, RoleIds: []int64{defaultRoleID}},
			Context: Context,
		})
		if err != nil {
			t.Errorf("failed to give user %d the default Access Role %d: %s", userID, defaultRoleID, err)
		}
	})
}

// SetDefaultRole makes the Access Role the default one, restores the previous default when the test ends, and must be called after CreateRole.
func SetDefaultRole(t *testing.T, roleID int64) {
	t.Helper()

	previous := readSettings(t.Context(), t).DefaultRoleID
	require.NotZero(t, previous, "the server has no default Access Role")
	_, err := accessControlService().SetDefaultRole(&accesscontrol.SetDefaultRoleParams{
		Body:    accesscontrol.SetDefaultRoleBody{RoleID: roleID},
		Context: t.Context(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := accessControlService().SetDefaultRole(&accesscontrol.SetDefaultRoleParams{
			Body:    accesscontrol.SetDefaultRoleBody{RoleID: previous},
			Context: Context,
		})
		if err != nil {
			t.Errorf("failed to restore the default Access Role %d: %s", previous, err)
		}
	})
}

// CreateUser creates a Grafana user with the organization role (Viewer, Editor or Admin) and deletes it when the test ends.
func CreateUser(t *testing.T, login, password, grafanaRole string) int64 {
	t.Helper()

	code, body := grafanaRequest(t, http.MethodPost, "/graph/api/admin/users", map[string]string{
		"name":     login,
		"email":    login + "@percona.invalid",
		"login":    login,
		"password": password,
	})
	require.Equal(t, http.StatusOK, code, "failed to create Grafana user: %s", body)

	var created struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))
	require.NotZero(t, created.ID)

	t.Cleanup(func() {
		code, body := grafanaRequest(t, http.MethodDelete, fmt.Sprintf("/graph/api/admin/users/%d", created.ID), nil)
		if code != http.StatusOK {
			t.Errorf("failed to delete Grafana user %d: %d %s", created.ID, code, body)
		}
	})

	code, body = grafanaRequest(t, http.MethodPatch, fmt.Sprintf("/graph/api/org/users/%d", created.ID), map[string]string{"role": grafanaRole})
	require.Equal(t, http.StatusOK, code, "failed to set the role of Grafana user %d: %s", created.ID, body)

	return created.ID
}

//nolint:ireturn
func accessControlService() accesscontrol.ClientService {
	return accesscontrolClient.Default.AccessControlService
}

func readSettings(ctx context.Context, t *testing.T) *server.GetSettingsOKBodySettings {
	t.Helper()

	res, err := serverClient.Default.ServerService.GetSettings(&server.GetSettingsParams{Context: ctx})
	require.NoError(t, err)

	return res.Payload.Settings
}

func changeAccessControl(ctx context.Context, enabled bool) error {
	_, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
		Body:    server.ChangeSettingsBody{EnableAccessControl: new(enabled)},
		Context: ctx,
	})
	if err != nil {
		return fmt.Errorf("failed to set Access Control to %t: %w", enabled, err)
	}

	return nil
}

// grafanaRequest sends a JSON request to Grafana as the user of the server URL and returns the status and body.
func grafanaRequest(t *testing.T, method, path string, payload any) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}

	u := BaseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(Context, method, u.String(), reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	res, err := HTTPClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return res.StatusCode, body
}
