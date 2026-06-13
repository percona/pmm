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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	stringsgen "github.com/percona/pmm/utils/strings"
)

func TestResolveAnonymousOrgRole(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		anonOrg string
		want    string
	}{
		{"Viewer", grafanaOrgRoleViewer},
		{"Editor", grafanaOrgRoleViewer},
		{"Admin", grafanaOrgRoleViewer},
		{"GrafanaAdmin", grafanaOrgRoleViewer},
		{"", grafanaOrgRoleNone},
		{"  ", grafanaOrgRoleNone},
		{"None", grafanaOrgRoleNone},
	} {
		assert.Equal(t, tc.want, resolveAnonymousOrgRole(tc.anonOrg), "%q", tc.anonOrg)
	}
}

func TestGetAuthUserAnonymousFallback(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())
	ctx := context.Background()

	t.Run("returns viewer role when anonymous role is non-viewer", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Editor"}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))

		u, err := c.getAuthUser(ctx, http.Header{}, l)
		require.NoError(t, err)
		assert.Equal(t, viewer, u.role)
		assert.Equal(t, 0, u.userID)
	})

	t.Run("no anonymous fallback when credentials are present", func(t *testing.T) {
		t.Parallel()

		settingsCalled := false
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				settingsCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Admin"}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Invalid username or password"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		headers := http.Header{}
		headers.Set("Authorization", "Basic YmFkOnBhc3M=")

		u, err := c.getAuthUser(ctx, headers, l)
		require.Error(t, err)
		assert.Equal(t, none, u.role)
		assert.False(t, settingsCalled)
	})

	t.Run("cookie-only request still falls back to anonymous role", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer"}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		headers := http.Header{}
		headers.Set("Cookie", "some-non-auth-cookie=value")

		u, err := c.getAuthUser(ctx, headers, l)
		require.NoError(t, err)
		assert.Equal(t, viewer, u.role)
	})

	t.Run("no frontend settings when cookie session succeeds on api user", func(t *testing.T) {
		t.Parallel()

		var settingsCalls int
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				settingsCalls++
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer"}`)
			case "/api/user":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"id":42,"isGrafanaAdmin":false}`)
			case "/api/user/orgs":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `[{"orgId":1,"role":"Viewer"}]`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		headers := http.Header{}
		headers.Set("Cookie", "grafana_session=ok")

		u, err := c.getAuthUser(ctx, headers, l)
		require.NoError(t, err)
		assert.Equal(t, viewer, u.role)
		assert.Equal(t, 42, u.userID)
		assert.Zero(t, settingsCalls, "/api/frontend/settings must not be called when /api/user succeeds")
	})

	t.Run("no fallback when anonymous is disabled", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":false}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))

		u, err := c.getAuthUser(ctx, http.Header{}, l)
		require.Error(t, err)
		assert.Equal(t, none, u.role)
	})

	t.Run("non-viewer anonymous role is clamped to viewer", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Admin"}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		u, err := c.getAuthUser(ctx, http.Header{}, l)
		require.NoError(t, err)
		assert.Equal(t, viewer, u.role)
	})

	t.Run("anonymous fallback uses anonymousOrgRole viewer when user orgRole omitted", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer","user":{"orgId":1}}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		u, err := c.getAuthUser(ctx, http.Header{}, l)
		require.NoError(t, err)
		assert.Equal(t, viewer, u.role)
	})
}

func TestCurrentUserAnonymousFallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("GetCurrentUser uses anonymous fallback", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer","user":{"orgId":1,"orgName":"Main Org."}}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		user, err := c.GetCurrentUser(ctx, http.Header{})
		require.NoError(t, err)
		assert.Equal(t, "anonymous", user.Login)
		assert.Equal(t, 1, user.OrgID)
		assert.True(t, user.IsAnonymous)
	})

	t.Run("GetCurrentUserOrgs returns viewer role from frontend settings", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Editor","user":{"orgId":1,"orgName":"Main Org."}}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			case "/api/user/orgs":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		orgs, err := c.GetCurrentUserOrgs(ctx, http.Header{})
		require.NoError(t, err)
		user, err := c.GetCurrentUser(ctx, http.Header{})
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		assert.Equal(t, "Viewer", orgs[0].Role)
		assert.Equal(t, 1, orgs[0].OrgID)
		assert.True(t, user.IsAnonymous)
	})

	t.Run("GetCurrentUser returns unauthorized when anonymous role missing", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		_, err := c.GetCurrentUser(ctx, http.Header{})
		require.Error(t, err)
	})

	t.Run("GetCurrentUser allows anonymousOrgRole viewer without user orgRole", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer","user":{"orgId":1}}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		user, err := c.GetCurrentUser(ctx, http.Header{})
		require.NoError(t, err)
		assert.True(t, user.IsAnonymous)
		assert.Equal(t, "anonymous", user.Login)
		assert.Equal(t, 1, user.OrgID)
	})

	t.Run("GetCurrentUserOrgs returns unauthorized when anonymous role missing", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true}`)
			case "/api/user/orgs":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		orgs, err := c.GetCurrentUserOrgs(ctx, http.Header{})
		require.Error(t, err)
		assert.Nil(t, orgs)
	})

	t.Run("GetCurrentUser does not fallback when credentials are present", func(t *testing.T) {
		t.Parallel()

		settingsCalled := false
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				settingsCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer"}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Invalid username or password"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		headers := http.Header{}
		headers.Set("Authorization", "Basic YmFkOnBhc3M=")
		_, err := c.GetCurrentUser(ctx, headers)
		require.Error(t, err)
		assert.False(t, settingsCalled)
	})

	t.Run("GetCurrentUser falls back with cookie-only request", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Viewer","user":{"orgId":1,"orgName":"Main Org."}}`)
			case "/api/user":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		headers := http.Header{}
		headers.Set("Cookie", "some-non-auth-cookie=value")

		user, err := c.GetCurrentUser(ctx, headers)
		require.NoError(t, err)
		assert.True(t, user.IsAnonymous)
		assert.Equal(t, "anonymous", user.Login)
	})

	t.Run("GetCurrentUserOrgs clamps non-viewer role to viewer", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/frontend/settings":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"anonymousEnabled":true,"anonymousOrgRole":"Admin","user":{"orgId":1,"orgName":"Main Org."}}`)
			case "/api/user/orgs":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"Unauthorized"}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		orgs, err := c.GetCurrentUserOrgs(ctx, http.Header{})
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		assert.Equal(t, viewer.String(), orgs[0].Role)
	})
}

func TestCurrentUserHTTPResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"generic", errors.New("boom"), http.StatusBadGateway, "Bad Gateway"},
		{"401 with message", errors.WithStack(&clientError{Code: http.StatusUnauthorized, ErrorMessage: "Invalid"}), http.StatusUnauthorized, "Invalid"},
		{"401 empty message", errors.WithStack(&clientError{Code: http.StatusUnauthorized}), http.StatusUnauthorized, "Unauthorized"},
		{"403", errors.WithStack(&clientError{Code: http.StatusForbidden}), http.StatusForbidden, "Forbidden"},
		{"404", errors.WithStack(&clientError{Code: http.StatusNotFound, ErrorMessage: "nf"}), http.StatusBadGateway, "Bad Gateway"},
		{"500 upstream", errors.WithStack(&clientError{Code: http.StatusInternalServerError}), http.StatusBadGateway, "Bad Gateway"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, body := CurrentUserHTTPResponse(tc.err)
			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantMsg, body["message"])
		})
	}
}

func TestClient(t *testing.T) {
	l := logrus.WithField("test", t.Name())

	ctx := t.Context()
	c := NewClient("127.0.0.1:3000")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/dummy", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "admin")
	authHeaders := req.Header

	t.Run("getRole", func(t *testing.T) {
		t.Run("GrafanaAdmin", func(t *testing.T) {
			u, err := c.getAuthUser(ctx, authHeaders, l)
			role := u.role
			require.NoError(t, err)
			assert.Equal(t, grafanaAdmin, role)
			assert.Equal(t, "GrafanaAdmin", role.String())
		})

		t.Run("NoAnonymousAccess", func(t *testing.T) {
			// See [auth.anonymous] in grafana.ini.
			// Even if anonymous access is enabled, returned role is None, not org_role.

			u, err := c.getAuthUser(ctx, nil, l)
			role := u.role
			clientError, _ := errors.Cause(err).(*clientError) //nolint:errorlint
			require.NotNil(t, clientError, "got role %s", role)
			assert.Equal(t, 401, clientError.Code)

			body := clientError.Body
			body = strings.ReplaceAll(body, "\n", "") // different grafana versions format response differently
			body = strings.ReplaceAll(body, " ", "")  // so we cleanup response from spaces and newlines to get unified result
			assert.JSONEq(t, `{"extra":null,"message":"Unauthorized","messageId":"auth.unauthorized","statusCode":401,"traceID":""}`, body)
			assert.Equal(t, `Unauthorized`, clientError.ErrorMessage)
			assert.Equal(t, none, role)
			assert.Equal(t, "None", role.String())
		})

		t.Run("NewUserViewerByDefault", func(t *testing.T) {
			// See [users] in grafana.ini.

			login := fmt.Sprintf("%s-%d", none, time.Now().Nanosecond())
			userID, err := c.testCreateUser(ctx, login, none, authHeaders)
			require.NoError(t, err)
			require.NotZero(t, userID)
			if err != nil {
				defer func() {
					err = c.testDeleteUser(ctx, userID, authHeaders)
					require.NoError(t, err)
				}()
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/dummy", nil)
			require.NoError(t, err)
			req.SetBasicAuth(login, login)
			userAuthHeaders := req.Header

			u, err := c.getAuthUser(ctx, userAuthHeaders, l)
			actualRole := u.role
			require.NoError(t, err)
			assert.Equal(t, viewer, actualRole)
			assert.Equal(t, viewer.String(), actualRole.String())
		})

		for _, role := range []role{viewer, editor, admin} {
			t.Run("Basic auth "+role.String(), func(t *testing.T) {
				login := fmt.Sprintf("basic-%s-%d", role, time.Now().Nanosecond())
				userID, err := c.testCreateUser(ctx, login, role, authHeaders)
				require.NoError(t, err)
				require.NotZero(t, userID)
				if err != nil {
					defer func() {
						err = c.testDeleteUser(ctx, userID, authHeaders)
						require.NoError(t, err)
					}()
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/dummy", nil)
				require.NoError(t, err)
				req.SetBasicAuth(login, login)
				userAuthHeaders := req.Header

				u, err := c.getAuthUser(ctx, userAuthHeaders, l)
				actualRole := u.role
				require.NoError(t, err)
				assert.Equal(t, role, actualRole)
				assert.Equal(t, role.String(), actualRole.String())
			})

			t.Run("Service token auth "+role.String(), func(t *testing.T) {
				name, err := stringsgen.GenerateRandomString(256)
				require.NoError(t, err)
				nodeName := fmt.Sprintf("%s-%s", name, role)
				serviceAccountID, err := c.createServiceAccount(ctx, role, nodeName, true, authHeaders)
				require.NoError(t, err)
				defer func() {
					err := c.deleteServiceAccount(ctx, serviceAccountID, authHeaders)
					require.NoError(t, err)
				}()

				serviceTokenID, serviceToken, err := c.createServiceToken(ctx, serviceAccountID, nodeName, true, authHeaders)
				require.NoError(t, err)
				require.NotZero(t, serviceTokenID)
				require.NotEmpty(t, serviceToken)
				defer func() {
					err := c.deletePMMAgentServiceToken(ctx, serviceAccountID, nodeName, authHeaders)
					require.NoError(t, err)
				}()

				serviceTokenAuthHeaders := http.Header{}
				serviceTokenAuthHeaders.Set("Authorization", "Bearer "+serviceToken)
				u, err := c.getAuthUser(ctx, serviceTokenAuthHeaders, l)
				require.NoError(t, err)
				actualRole := u.role
				assert.Equal(t, role, actualRole)
				assert.Equal(t, role.String(), actualRole.String())
			})
		}
	})

	t.Run("CreateAnnotation", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/dummy", nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")
		authorization := req.Header.Get("Authorization")

		t.Run("Normal", func(t *testing.T) {
			from := time.Now()
			msg, err := c.CreateAnnotation(ctx, []string{"tag1", "tag2"}, from, "Normal", authorization)
			require.NoError(t, err)
			assert.Equal(t, "Annotation added", msg)

			annotations, err := c.findAnnotations(ctx, from, from.Add(time.Second), authorization)
			require.NoError(t, err)
			for _, a := range annotations {
				if a.Text == "Normal" {
					assert.Equal(t, []string{"tag1", "tag2"}, a.Tags)
					assert.InDelta(t, from.Unix(), a.Time.Unix(), 1)
					return
				}
			}
			assert.Fail(t, "annotation not found", "%s", annotations)
		})

		t.Run("Empty", func(t *testing.T) {
			_, err := c.CreateAnnotation(ctx, nil, time.Now(), "", authorization)
			require.Error(t, err)
		})

		t.Run("No tags", func(t *testing.T) {
			from := time.Now()
			msg, err := c.CreateAnnotation(ctx, nil, from, "No tags", authorization)
			require.NoError(t, err)
			assert.Equal(t, "Annotation added", msg)

			annotations, err := c.findAnnotations(ctx, from, from.Add(time.Second), authorization)
			require.NoError(t, err)
			for _, a := range annotations {
				if a.Text == "No tags" {
					assert.Empty(t, a.Tags)
					assert.InDelta(t, from.Unix(), a.Time.Unix(), 1)
					return
				}
			}
			assert.Fail(t, "annotation not found", "%s", annotations)
		})

		t.Run("Auth error", func(t *testing.T) {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/dummy", nil)
			req.SetBasicAuth("nouser", "wrongpassword")
			authorization := req.Header.Get("Authorization")
			_, err = c.CreateAnnotation(ctx, nil, time.Now(), "", authorization)
			require.ErrorContains(t, err, "failed to create annotation: clientError: POST http://127.0.0.1:3000/api/annotations -> 401")
			require.ErrorContains(t, err, "Invalid username or password")
		})
	})

	t.Run("IsReady", func(t *testing.T) {
		err := c.IsReady(ctx)
		require.NoError(t, err)
	})
}

func TestAlertAnnotations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const wantAuth = "Basic YWRtaW46YWRtaW4=" // admin:admin

	t.Run("CreateAlertAnnotation", func(t *testing.T) {
		t.Parallel()

		var gotMethod, gotPath, gotAuth string
		var gotBodyBytes []byte
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
			gotBodyBytes, _ = io.ReadAll(r.Body)
			_, _ = fmt.Fprint(w, `{"id":42,"message":"Annotation added"}`)
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		start := time.Unix(1700000000, 0)
		id, err := c.CreateAlertAnnotation(ctx, []string{"mysql", "fingerprint:abc"}, start, "MySQL down")
		require.NoError(t, err)
		assert.Equal(t, 42, id)
		assert.Equal(t, http.MethodPost, gotMethod)
		assert.Equal(t, "/api/annotations", gotPath)
		assert.Equal(t, wantAuth, gotAuth)

		var gotBody annotation
		require.NoError(t, json.Unmarshal(gotBodyBytes, &gotBody))
		assert.Equal(t, []string{"mysql", "fingerprint:abc"}, gotBody.Tags)
		assert.Equal(t, "MySQL down", gotBody.Text)
		assert.Equal(t, start.UnixMilli(), gotBody.TimeInt)
		assert.Zero(t, gotBody.TimeEndInt)
	})

	t.Run("SetAlertAnnotationEnd", func(t *testing.T) {
		t.Parallel()

		var gotMethod, gotPath, gotAuth string
		var gotBodyBytes []byte
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
			gotBodyBytes, _ = io.ReadAll(r.Body)
			_, _ = fmt.Fprint(w, `{"message":"Annotation patched"}`)
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		end := time.Unix(1700000600, 0)
		err := c.SetAlertAnnotationEnd(ctx, 42, end)
		require.NoError(t, err)
		assert.Equal(t, http.MethodPatch, gotMethod)
		assert.Equal(t, "/api/annotations/42", gotPath)
		assert.Equal(t, wantAuth, gotAuth)

		var gotBody annotation
		require.NoError(t, json.Unmarshal(gotBodyBytes, &gotBody))
		assert.Equal(t, end.UnixMilli(), gotBody.TimeEndInt)
		assert.Zero(t, gotBody.TimeInt)
	})

	t.Run("FindAlertAnnotationID", func(t *testing.T) {
		t.Parallel()

		var gotMethod, gotPath, gotType string
		var gotTags []string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath = r.Method, r.URL.Path
			gotTags = r.URL.Query()["tags"]
			gotType = r.URL.Query().Get("type")
			_, _ = fmt.Fprint(w, `[{"id":7},{"id":8}]`)
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		id, err := c.FindAlertAnnotationID(ctx, []string{"fingerprint:abc"}, time.Unix(1700000000, 0), time.Unix(1700000600, 0))
		require.NoError(t, err)
		assert.Equal(t, 7, id)
		assert.Equal(t, http.MethodGet, gotMethod)
		assert.Equal(t, "/api/annotations", gotPath)
		assert.Equal(t, []string{"fingerprint:abc"}, gotTags)
		assert.Equal(t, "annotation", gotType)
	})

	t.Run("FindAlertAnnotationID returns 0 when none found", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `[]`)
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		id, err := c.FindAlertAnnotationID(ctx, []string{"x"}, time.Unix(1, 0), time.Unix(2, 0))
		require.NoError(t, err)
		assert.Zero(t, id)
	})
}
