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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/tests"
)

func TestExtractOriginalRequest(t *testing.T) {
	t.Parallel()

	invalidUTF8URI := string([]byte{'/', 'b', 'a', 'd', 0xff})

	for _, tc := range []struct {
		name          string
		initialMethod string
		origMethod    *string
		origURI       *string
		wantMethod    string
		wantPath      string
		wantErr       string
	}{
		{
			name:          "normalizes traversal and strips query",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodPost),
			origURI:       new("/v1/server/AWSInstanceCheck/..%2f..%2fmanaged/logs.zip?foo=bar"),
			wantMethod:    http.MethodPost,
			wantPath:      "/v1/managed/logs.zip",
		},
		{
			name:          "keeps already clean path",
			initialMethod: http.MethodPost,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1/server/version"),
			wantMethod:    http.MethodGet,
			wantPath:      "/v1/server/version",
		},
		{
			name:          "collapses duplicate slashes",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1//server///logs.zip"),
			wantMethod:    http.MethodGet,
			wantPath:      "/v1/server/logs.zip",
		},
		{
			name:          "cleans plain dot segments",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodDelete),
			origURI:       new("/v1/server/../inventory/./Services/List"),
			wantMethod:    http.MethodDelete,
			wantPath:      "/v1/inventory/Services/List",
		},
		{
			name:          "cleans encoded slashes in traversal",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1/server/AWSInstanceCheck/..%2F..%2Finventory/Services/List"),
			wantMethod:    http.MethodGet,
			wantPath:      "/v1/inventory/Services/List",
		},
		{
			name:          "sanitizes encoded newline and carriage return",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1/server/logs%0A%0D.zip"),
			wantMethod:    http.MethodGet,
			wantPath:      "/v1/server/logs  .zip",
		},
		{
			name:          "sanitizes raw newline and carriage return",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1/server/logs\n\r.zip"),
			wantMethod:    http.MethodGet,
			wantPath:      "/v1/server/logs  .zip",
		},
		{
			name:          "accepts custom method",
			initialMethod: http.MethodGet,
			origMethod:    new("CUSTOM"),
			origURI:       new("/v1/management/Jobs"),
			wantMethod:    "CUSTOM",
			wantPath:      "/v1/management/Jobs",
		},
		{
			name:          "fails on missing original method",
			initialMethod: http.MethodGet,
			origURI:       new("/v1/server/version"),
			wantErr:       "empty X-Original-Method",
		},
		{
			name:          "fails on missing original uri",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			wantErr:       "empty X-Original-Uri",
		},
		{
			name:          "fails on uri without leading slash",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("v1/server/version"),
			wantErr:       "unexpected X-Original-Uri",
		},
		{
			name:          "fails on invalid utf8 uri",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new(invalidUTF8URI),
			wantErr:       "invalid X-Original-Uri",
		},
		{
			name:          "fails on invalid escape sequence",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1/server/%zz/logs.zip"),
			wantErr:       "failed to unescape path",
		},
		{
			name:          "fails on incomplete escape",
			initialMethod: http.MethodGet,
			origMethod:    new(http.MethodGet),
			origURI:       new("/v1/server/%"),
			wantErr:       "failed to unescape path",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), tc.initialMethod, "/auth_request", nil)
			if tc.origMethod != nil {
				req.Header.Set("X-Original-Method", *tc.origMethod)
			}
			if tc.origURI != nil {
				req.Header.Set("X-Original-Uri", *tc.origURI)
			}

			err := extractOriginalRequest(req)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Equal(t, tc.initialMethod, req.Method)
				assert.Equal(t, "/auth_request", req.URL.Path)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantMethod, req.Method)
			assert.Equal(t, tc.wantPath, req.URL.Path)
		})
	}
}

func TestNextPrefix(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
		name       string
		method     string
		path       string
		wantRole   role
		wantPrefix string
	}{
		{name: "returns viewer role for template listing", method: http.MethodGet, path: "/v1/alerting/templates", wantRole: viewer, wantPrefix: "/v1/alerting"},
		{name: "uses method specific rule for template creation", method: http.MethodPost, path: "/v1/alerting/templates", wantRole: editor, wantPrefix: "/v1/alerting/templates"},
		{name: "uses method specific rule for template update path", method: http.MethodPut, path: "/v1/alerting/templates/foo", wantRole: editor, wantPrefix: "/v1/alerting/templates/"},
		{name: "uses method specific rule for template delete path", method: http.MethodDelete, path: "/v1/alerting/templates/foo", wantRole: editor, wantPrefix: "/v1/alerting/templates/"},
		{name: "returns editor role for alerting rules creation", method: http.MethodPost, path: "/v1/alerting/rules", wantRole: editor, wantPrefix: "/v1/alerting/rules"},
		{name: "returns most specific matching path rule", method: http.MethodGet, path: "/v1/server/settings/readonly/details", wantRole: viewer, wantPrefix: "/v1/server/settings/readonly"},
		{name: "falls back to grafana admin when no explicit rule matches", method: http.MethodGet, path: "/v1/unknown", wantRole: grafanaAdmin, wantPrefix: "/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotPrefix := resolveRule(tc.method, tc.path, logrus.WithField("test", t.Name()))
			assert.Equal(t, tc.wantRole, got)
			assert.Equal(t, tc.wantPrefix, gotPrefix)
		})
	}
}

func TestIsLocalAgentConnection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		remoteAddr string
		path       string
		want       bool
	}{
		{name: "local connect endpoint", remoteAddr: "127.0.0.1:12345", path: connectionEndpoint, want: true},
		{name: "local rta endpoint", remoteAddr: "127.0.0.1:12345", path: rtaCollectEndpoint, want: true},
		{name: "remote endpoint", remoteAddr: "10.0.0.2:12345", path: connectionEndpoint, want: false},
		{name: "local unknown path", remoteAddr: "127.0.0.1:12345", path: "/v1/server/version", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			req.RemoteAddr = tc.remoteAddr
			assert.Equal(t, tc.want, isLocalAgentConnection(req))
		})
	}
}

func TestCleanPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path     string
		expected string
		wantErr  bool
	}{
		{
			connectionEndpointV2,
			connectionEndpointV2,
			false,
		}, {
			connectionEndpoint,
			connectionEndpoint,
			false,
		}, {
			rtaCollectEndpoint,
			rtaCollectEndpoint,
			false,
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2finventory/Services/List",
			"/v1/inventory/Services/List",
			false,
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2f..%2fmanaged/logs.zip",
			"/managed/logs.zip",
			false,
		}, {
			"/v1/server/AWSInstanceCheck/..%2f..%2f..%2f/logs.zip",
			"/logs.zip",
			false,
		}, {
			"/managed/logs.zip?download=1",
			"/managed/logs.zip",
			false,
		}, {
			"/v1/server/./logs.zip",
			"/v1/server/logs.zip",
			false,
		}, {
			"/v1/server/../logs.zip",
			"/v1/logs.zip",
			false,
		}, {
			"/graph/api/datasources/proxy/8/?query=WITH%20(%0A%20%20%20%20CASE%20%0A%20%20%20%20%20%20%20%20WHEN%20(3000%20%25%2060)%20%3D%200%20THEN%203000%0A%20%20%20%20ELSE%2060%20END%0A)%20AS%20scale%0ASELECT%0A%20%20%20%20(intDiv(toUInt32(timestamp)%2C%203000)%20*%203000)%20*%201000%20as%20t%2C%0A%20%20%20%20hostname%20h%2C%0A%20%20%20%20status%20s%2C%0A%20%20%20%20SUM(req_count)%20as%20req_count%0AFROM%20pinba.report_by_all%0AWHERE%0A%20%20%20%20timestamp%20%3E%3D%20toDateTime(1707139680)%20AND%20timestamp%20%3C%3D%20toDateTime(1707312480)%0A%20%20%20%20AND%20status%20%3E%3D%20400%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20schema%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20hostname%20%3D%20%27all%27%20ELSE%201%20END%0A%20%20%20%20AND%20CASE%20WHEN%20%27all%27%20%3C%3E%20%27all%27%20THEN%20server_name%20%3D%20%27all%27%20ELSE%201%20END%0AGROUP%20BY%20t%2C%20h%2C%20s%0AORDER%20BY%20t%20FORMAT%20JSON",
			"/graph/api/datasources/proxy/8/",
			false,
		}, {
			"/v1/server/%zz/logs.zip",
			"",
			true,
		}, {
			"/v1/server/%",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got, err := cleanPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equalf(t, tt.expected, got, "cleanPath(%v)", tt.path)
		})
	}
}

func TestExtractAuthHeaders(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		set  func(req *http.Request)
		want http.Header
	}{
		{
			name: "returns authorization and cookie headers when present",
			set: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token")
				req.Header.Set("Cookie", "session=abc")
			},
			want: http.Header{
				"Authorization": []string{"Bearer token"},
				"Cookie":        []string{"session=abc"},
			},
		},
		{
			name: "returns empty header when auth headers are missing",
			set:  func(_ *http.Request) {},
			want: http.Header{},
		},
		{
			name: "ignores unrelated headers",
			set: func(req *http.Request) {
				req.Header.Set("X-Request-ID", "req-1")
				req.Header.Set("Accept", "application/json")
			},
			want: http.Header{},
		},
		{
			name: "skips empty authorization and cookie values",
			set: func(req *http.Request) {
				req.Header["Authorization"] = []string{""}
				req.Header["Cookie"] = []string{""}
			},
			want: http.Header{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			tc.set(req)

			got := extractAuthHeaders(req)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAuthCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("empty headers", func(t *testing.T) {
		t.Parallel()

		key, err := authCacheKey(http.Header{})
		require.NoError(t, err)
		assert.Empty(t, key)
	})

	t.Run("authorization fast path", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{"Authorization": []string{"Bearer token"}}
		key, err := authCacheKey(headers)
		require.NoError(t, err)
		assert.Equal(t, "a:Bearer token", key)
	})

	t.Run("cookie fast path", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{"Cookie": []string{"grafana_session=abc"}}
		key, err := authCacheKey(headers)
		require.NoError(t, err)
		assert.Equal(t, "c:grafana_session=abc", key)
	})

	t.Run("authorization and cookie dual fast path", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{
			"Authorization": []string{"Bearer token"},
			"Cookie":        []string{"grafana_session=abc"},
		}
		key, err := authCacheKey(headers)
		require.NoError(t, err)
		assert.Equal(t, "ac:12:Bearer token|19:grafana_session=abc", key)
	})

	t.Run("fallback is deterministic for same headers", func(t *testing.T) {
		t.Parallel()

		headers1 := http.Header{"Authorization": []string{"Bearer token"}, "Cookie": []string{"grafana_session=abc"}}
		headers2 := http.Header{"Cookie": []string{"grafana_session=abc"}, "Authorization": []string{"Bearer token"}}

		key1, err := authCacheKey(headers1)
		require.NoError(t, err)
		key2, err := authCacheKey(headers2)
		require.NoError(t, err)

		assert.Equal(t, key1, key2)
	})
}
