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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
)

func TestAuth(t *testing.T) {
	t.Run("AuthErrors", func(t *testing.T) {
		for user, code := range map[*url.Userinfo]int{
			nil:                              401,
			url.UserPassword("bad", "wrong"): 401,
		} {
			user := user
			code := code
			t.Run(fmt.Sprintf("%s/%d", user, code), func(t *testing.T) {
				t.Parallel()

				// copy BaseURL and replace auth
				baseURL, err := url.Parse(pmmapitests.BaseURL.String())
				require.NoError(t, err)
				baseURL.User = user

				uri := baseURL.ResolveReference(&url.URL{
					Path: "v1/version",
				})
				t.Logf("URI: %s", uri)

				req, _ := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close() //nolint:gosec,errcheck

				b, err := httputil.DumpResponse(resp, true)
				require.NoError(t, err)
				assert.Equal(t, code, resp.StatusCode, "response:\n%s", b)
				require.False(t, bytes.Contains(b, []byte(`<html>`)), "response:\n%s", b)
			})
		}
	})

	t.Run("NormalErrors", func(t *testing.T) {
		for grpcCode, httpCode := range map[codes.Code]int{
			codes.Unauthenticated:  401,
			codes.PermissionDenied: 403,
		} {
			grpcCode := grpcCode
			httpCode := httpCode
			t.Run(fmt.Sprintf("%s/%d", grpcCode, httpCode), func(t *testing.T) {
				t.Parallel()

				res, err := serverClient.Default.Server.Version(&server.VersionParams{
					Dummy:   pointer.ToString(fmt.Sprintf("grpccode-%d", grpcCode)),
					Context: pmmapitests.Context,
				})
				assert.Empty(t, res)
				pmmapitests.AssertAPIErrorf(t, err, httpCode, grpcCode, "gRPC code %d (%s)", grpcCode, grpcCode)
			})
		}
	})
}

func TestSetup(t *testing.T) {
	t.Parallel()
	// make a BaseURL without authentication
	baseURL, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	baseURL.User = nil

	// make client that does not follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Run("WebPage", func(t *testing.T) {
		t.Parallel()

		uri := baseURL.ResolveReference(&url.URL{
			Path: "/setup",
		})
		t.Logf("URI: %s", uri)
		req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
		require.NoError(t, err)
		req.Header.Set("X-Test-Must-Setup", "1")

		resp, b := doRequest(t, client, req)
		defer resp.Body.Close() //nolint:gosec,errcheck

		assert.Equal(t, 200, resp.StatusCode, "response:\n%s", b)
		assert.True(t, strings.HasPrefix(string(b), `<!doctype html>`), string(b))
	})

	t.Run("Redirect", func(t *testing.T) {
		t.Parallel()
		paths := map[string]int{
			"graph":       303,
			"graph/":      303,
			"prometheus":  303,
			"prometheus/": 303,
			"swagger":     200,
			"swagger/":    301,

			"v1/readyz":           200,
			"v1/AWSInstanceCheck": 501, // only POST is expected, other request methods are seen as unimplemented
			"v1/version":          401, // Grafana authentication required
		}
		for path, code := range paths {
			path, code := path, code
			t.Run(fmt.Sprintf("%s=%d", path, code), func(t *testing.T) {
				t.Parallel()

				uri := baseURL.ResolveReference(&url.URL{
					Path: path,
				})
				t.Logf("URI: %s", uri)
				req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
				require.NoError(t, err)
				req.Header.Set("X-Test-Must-Setup", "1")

				resp, b := doRequest(t, client, req)
				defer resp.Body.Close() //nolint:gosec,errcheck

				assert.Equal(t, code, resp.StatusCode, "response:\n%s", b)
				if code == 303 {
					assert.Equal(t, "/setup", resp.Header.Get("Location"))
				}
			})
		}
	})

	t.Run("API", func(t *testing.T) {
		t.Parallel()

		uri := baseURL.ResolveReference(&url.URL{
			Path: "v1/AWSInstanceCheck",
		})
		t.Logf("URI: %s", uri)
		b, err := json.Marshal(server.AWSInstanceCheckBody{
			InstanceID: "123",
		})
		require.NoError(t, err)
		req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, uri.String(), bytes.NewReader(b))
		require.NoError(t, err)
		req.Header.Set("X-Test-Must-Setup", "1")

		resp, b := doRequest(t, client, req)
		defer resp.Body.Close() //nolint:gosec,errcheck

		assert.Equal(t, 200, resp.StatusCode, "response:\n%s", b)
		assert.Equal(t, "{}", string(b), "response:\n%s", b)
	})
}

func TestSwagger(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"swagger",
		"swagger/",
		"swagger.json",
		"swagger/swagger.json",
	} {
		path := path

		t.Run(path, func(t *testing.T) {
			t.Parallel()
			t.Run("NoAuth", func(t *testing.T) {
				t.Parallel()

				// make a BaseURL without authentication
				baseURL, err := url.Parse(pmmapitests.BaseURL.String())
				require.NoError(t, err)
				baseURL.User = nil

				uri := baseURL.ResolveReference(&url.URL{
					Path: path,
				})
				t.Logf("URI: %s", uri)
				req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
				require.NoError(t, err)

				resp, _ := doRequest(t, http.DefaultClient, req)
				defer resp.Body.Close() //nolint:gosec,errcheck

				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
			})

			t.Run("Auth", func(t *testing.T) {
				t.Parallel()

				uri := pmmapitests.BaseURL.ResolveReference(&url.URL{
					Path: path,
				})
				t.Logf("URI: %s", uri)
				req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
				require.NoError(t, err)

				resp, _ := doRequest(t, http.DefaultClient, req)
				defer resp.Body.Close() //nolint:gosec,errcheck

				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
			})
		})
	}
}

func doRequest(tb testing.TB, client *http.Client, req *http.Request) (*http.Response, []byte) {
	tb.Helper()
	resp, err := client.Do(req)
	require.NoError(tb, err)

	defer resp.Body.Close() //nolint:gosec,errcheck

	b, err := io.ReadAll(resp.Body)
	require.NoError(tb, err)

	return resp, b
}

func TestServiceAccountPermissions(t *testing.T) {
	// service account role options: viewer, editor, admin
	// service token role options: editor, admin
	// basic auth format is skipped, endpoint /auth/serviceaccount (to get info about token) requires Bearer authorization
	// service_token:token could be used in pmm-agent and pmm-admin (its transformed into Bearer authorization)

	viewerAccountID := createServiceAccountWithRole(t, "Viewer")
	defer deleteServiceAccount(t, viewerAccountID)

	editorAccountID := createServiceAccountWithRole(t, "Editor")
	editorTokenID, editorToken := createServiceToken(t, editorAccountID)
	defer deleteServiceAccount(t, editorAccountID)
	defer deleteServiceToken(t, editorAccountID, editorTokenID)

	adminAccountID := createServiceAccountWithRole(t, "Admin")
	adminTokenID, adminToken := createServiceToken(t, adminAccountID)
	defer deleteServiceAccount(t, adminAccountID)
	defer deleteServiceToken(t, adminAccountID, adminTokenID)

	type userCase struct {
		userType     string
		serviceToken string
		statusCode   int
	}

	tests := []struct {
		name     string
		url      string
		method   string
		userCase []userCase
	}{
		{name: "settings", url: "/v1/Settings/Get", method: "POST", userCase: []userCase{
			{userType: "default", statusCode: 401},
			{userType: "viewer", serviceToken: "", statusCode: 401},
			{userType: "editor", serviceToken: editorToken, statusCode: 401},
			{userType: "admin", serviceToken: adminToken, statusCode: 200},
		}},
		{name: "alerts-default", url: "/alertmanager/api/v2/alerts", method: http.MethodGet, userCase: []userCase{
			{userType: "default", statusCode: 401},
			{userType: "viewer", serviceToken: "", statusCode: 401},
			{userType: "editor", serviceToken: editorToken, statusCode: 401},
			{userType: "admin", serviceToken: adminToken, statusCode: 200},
		}},
		{name: "platform-connect", url: "/v1/Platform/Connect", method: "POST", userCase: []userCase{
			{userType: "default", statusCode: 401},
			{userType: "viewer", serviceToken: "", statusCode: 401},
			{userType: "editor", serviceToken: editorToken, statusCode: 401},
			{userType: "admin", serviceToken: adminToken, statusCode: 400}, // We send bad request, but have access to endpoint
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for _, user := range test.userCase {
				user := user
				t.Run(fmt.Sprintf("Service Token auth %s", user.userType), func(t *testing.T) {
					// make a BaseURL without authentication
					u, err := url.Parse(pmmapitests.BaseURL.String())
					require.NoError(t, err)
					u.User = nil
					u.Path = test.url

					req, err := http.NewRequestWithContext(pmmapitests.Context, test.method, u.String(), nil)
					require.NoError(t, err)

					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.serviceToken))

					resp, err := http.DefaultClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close() //nolint:gosec,errcheck

					assert.Equal(t, user.statusCode, resp.StatusCode)
				})
			}
		})
	}
}

func createServiceAccountWithRole(t *testing.T, role string) int {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/serviceaccounts"

	name := fmt.Sprintf("serviceaccount-%s-%d", role, time.Now().Nanosecond())
	data, err := json.Marshal(map[string]string{
		"name": name,
		"role": role,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, u.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck

	require.Equalf(t, http.StatusCreated, resp.StatusCode, "failed to create Service account, status code: %d, response: %s", resp.StatusCode, b)

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)

	serviceAccountID := int(m["id"].(float64))
	u.Path = fmt.Sprintf("/graph/api/serviceaccounts/%d", serviceAccountID)
	data, err = json.Marshal(map[string]string{
		"orgId": "1",
	})
	require.NoError(t, err)

	req, err = http.NewRequestWithContext(pmmapitests.Context, http.MethodPatch, u.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp1, b := doRequest(t, http.DefaultClient, req)
	defer resp1.Body.Close() //nolint:gosec,errcheck

	require.Equalf(t, http.StatusCreated, resp.StatusCode, "failed to set orgId=1 to Service account, status code: %d, response: %s", resp.StatusCode, b)

	return serviceAccountID
}

func deleteServiceAccount(t *testing.T, serviceAccountID int) {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/graph/api/serviceaccounts/%d", serviceAccountID)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodDelete, u.String(), nil)
	require.NoError(t, err)

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to delete service account, status code: %d, response: %s", resp.StatusCode, b)
}

func createServiceToken(t *testing.T, serviceAccountID int) (int, string) {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/graph/api/serviceaccounts/%d/tokens", serviceAccountID)

	name := fmt.Sprintf("servicetoken-%d-%d", serviceAccountID, time.Now().Nanosecond())
	data, err := json.Marshal(map[string]string{
		"name": name,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, u.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to create Service account, status code: %d, response: %s", resp.StatusCode, b)

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)

	return int(m["id"].(float64)), m["key"].(string)
}

func deleteServiceToken(t *testing.T, serviceAccountID, serviceTokenID int) {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/graph/api/serviceaccounts/%d/tokens/%d", serviceAccountID, serviceTokenID)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodDelete, u.String(), nil)
	require.NoError(t, err)

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to delete service token, status code: %d, response: %s", resp.StatusCode, b)
}
