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
	"strconv"
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
				defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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
		defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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
				defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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
		defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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
				defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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
				defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
			})
		})
	}
}

func TestPermissions(t *testing.T) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	none := "none-" + ts
	viewer := "viewer-" + ts
	editor := "editor-" + ts
	admin := "admin-" + ts

	noneID := createUser(t, none)
	defer deleteUser(t, noneID)

	viewerID := createUserWithRole(t, viewer, "Viewer")
	defer deleteUser(t, viewerID)

	editorID := createUserWithRole(t, editor, "Editor")
	defer deleteUser(t, editorID)

	adminID := createUserWithRole(t, admin, "Admin")
	defer deleteUser(t, adminID)

	viewerAPIKeyID, viewerAPIKey := createAPIKeyWithRole(t, "api-"+viewer, "Viewer")
	defer deleteAPIKey(t, viewerAPIKeyID)

	editorAPIKeyID, editorAPIKey := createAPIKeyWithRole(t, "api-"+editor, "Editor")
	defer deleteAPIKey(t, editorAPIKeyID)

	adminAPIKeyID, adminAPIKey := createAPIKeyWithRole(t, "api-"+admin, "Admin")
	defer deleteAPIKey(t, adminAPIKeyID)

	type userCase struct {
		userType   string
		login      string
		apiKey     string
		statusCode int
	}

	tests := []struct {
		name     string
		url      string
		method   string
		userCase []userCase
	}{
		{name: "settings", url: "/v1/Settings/Get", method: "POST", userCase: []userCase{
			{userType: "default", login: none, statusCode: 401},
			{userType: "viewer", login: viewer, apiKey: viewerAPIKey, statusCode: 401},
			{userType: "editor", login: editor, apiKey: editorAPIKey, statusCode: 401},
			{userType: "admin", login: admin, apiKey: adminAPIKey, statusCode: 200},
		}},
		{name: "alerts-default", url: "/alertmanager/api/v2/alerts", method: http.MethodGet, userCase: []userCase{
			{userType: "default", login: none, statusCode: 401},
			{userType: "viewer", login: viewer, apiKey: viewerAPIKey, statusCode: 401},
			{userType: "editor", login: editor, apiKey: editorAPIKey, statusCode: 401},
			{userType: "admin", login: admin, apiKey: adminAPIKey, statusCode: 200},
		}},
		{name: "platform-connect", url: "/v1/Platform/Connect", method: "POST", userCase: []userCase{
			{userType: "default", login: none, statusCode: 401},
			{userType: "viewer", login: viewer, apiKey: viewerAPIKey, statusCode: 401},
			{userType: "editor", login: editor, apiKey: editorAPIKey, statusCode: 401},
			{userType: "admin", login: admin, apiKey: adminAPIKey, statusCode: 400}, // We send bad request, but have access to endpoint
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for _, user := range test.userCase {
				user := user
				t.Run(fmt.Sprintf("Basic auth %s", user.userType), func(t *testing.T) {
					// make a BaseURL without authentication
					u, err := url.Parse(pmmapitests.BaseURL.String())
					require.NoError(t, err)
					u.User = url.UserPassword(user.login, user.login)
					u.Path = test.url

					req, err := http.NewRequestWithContext(pmmapitests.Context, test.method, u.String(), nil)
					require.NoError(t, err)

					resp, err := http.DefaultClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

					assert.Equal(t, user.statusCode, resp.StatusCode)
				})

				t.Run(fmt.Sprintf("API Key auth %s", user.userType), func(t *testing.T) {
					if user.apiKey == "" {
						t.Skip("API Key is not exist")
					}
					// make a BaseURL without authentication
					u, err := url.Parse(pmmapitests.BaseURL.String())
					require.NoError(t, err)
					u.User = nil
					u.Path = test.url

					req, err := http.NewRequestWithContext(pmmapitests.Context, test.method, u.String(), nil)
					require.NoError(t, err)

					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.apiKey))

					resp, err := http.DefaultClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

					assert.Equal(t, user.statusCode, resp.StatusCode)
				})

				t.Run(fmt.Sprintf("API Key Basic auth %s", user.userType), func(t *testing.T) {
					if user.apiKey == "" {
						t.Skip("API Key is not exist")
					}
					// make a BaseURL without authentication
					u, err := url.Parse(pmmapitests.BaseURL.String())
					require.NoError(t, err)
					u.User = url.UserPassword("api_key", user.apiKey)
					u.Path = test.url

					req, err := http.NewRequestWithContext(pmmapitests.Context, test.method, u.String(), nil)
					require.NoError(t, err)

					resp, err := http.DefaultClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

					assert.Equal(t, user.statusCode, resp.StatusCode)
				})
			}
		})
	}
}

func doRequest(tb testing.TB, client *http.Client, req *http.Request) (*http.Response, []byte) {
	tb.Helper()
	resp, err := client.Do(req)
	require.NoError(tb, err)

	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	require.NoError(tb, err)

	return resp, b
}

func createUserWithRole(t *testing.T, login, role string) int {
	t.Helper()
	userID := createUser(t, login)
	setRole(t, userID, role)

	return userID
}

func deleteUser(t *testing.T, userID int) {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/admin/users/" + strconv.Itoa(userID)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodDelete, u.String(), nil)
	require.NoError(t, err)

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to delete user, status code: %d, response: %s", resp.StatusCode, b)
}

func createUser(t *testing.T, login string) int {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/admin/users"

	// https://grafana.com/docs/http_api/admin/#global-users
	data, err := json.Marshal(map[string]string{
		"name":     login,
		"email":    login + "@percona.invalid",
		"login":    login,
		"password": login,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, u.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint
	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to create user, status code: %d, response: %s", resp.StatusCode, b)

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)

	return int(m["id"].(float64))
}

func setRole(t *testing.T, userID int, role string) {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/org/users/" + strconv.Itoa(userID)

	// https://grafana.com/docs/http_api/org/#updates-the-given-user
	data, err := json.Marshal(map[string]string{
		"role": role,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPatch, u.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to set role for user, response: %s", b)
}

func deleteAPIKey(t *testing.T, apiKeyID int) {
	t.Helper()
	// https://grafana.com/docs/grafana/latest/http_api/auth/#delete-api-key
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/auth/keys/" + strconv.Itoa(apiKeyID)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodDelete, u.String(), nil)
	require.NoError(t, err)

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to delete API Key, status code: %d, response: %s", resp.StatusCode, b)
}

func createAPIKeyWithRole(t *testing.T, name, role string) (int, string) {
	t.Helper()
	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/auth/keys"

	// https://grafana.com/docs/grafana/latest/http_api/auth/#create-api-key
	data, err := json.Marshal(map[string]string{
		"name": name,
		"role": role,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, u.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, b := doRequest(t, http.DefaultClient, req)
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to create API key, status code: %d, response: %s", resp.StatusCode, b)

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)
	apiKey := m["key"].(string)

	u.User = nil
	u.Path = "/graph/api/auth/key"
	req, err = http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, u.String(), bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp1, b := doRequest(t, http.DefaultClient, req)
	defer resp1.Body.Close() //nolint:gosec,errcheck,nolintlint

	require.Equalf(t, http.StatusOK, resp1.StatusCode, "failed to get API key, status code: %d, response: %s", resp1.StatusCode, b)

	var k map[string]interface{}
	err = json.Unmarshal(b, &k)
	require.NoError(t, err)

	apiKeyID := int(k["id"].(float64))

	return apiKeyID, apiKey
}
