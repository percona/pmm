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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func BenchmarkCleanPath(b *testing.B) {
	const unescapedURI = "/v1/server/AWSInstanceCheck/..%2f..%2f..%2f..%2fgraph/api/datasources/proxy/8%2f%2f.%2f..%2f8%2f%2f?query=WITH%20CASE%20WHEN%203000%20x%2060%20THEN%203000%20ELSE%2060%20END%20SELECT%20hostname%2Cstatus%20FROM%20pinba.report_by_all%20WHERE%20timestamp%3E%3D1707139680%20AND%20timestamp%3C%3D1707312480%20ORDER%20BY%20t"
	const expectedCleanPath = "/graph/api/datasources/proxy/8"

	b.ReportAllocs()

	// cleanedPath, err := cleanPath(unescapedURI)
	// require.NoError(b, err)
	// require.Equal(b, expectedCleanPath, cleanedPath)

	b.ResetTimer()
	for b.Loop() {
		cleanedPath, err := cleanPath(unescapedURI)
		if err != nil {
			b.Fatalf("cleanPath returned error: %v", err)
		}
		if cleanedPath != expectedCleanPath {
			b.Fatalf("unexpected cleaned path: got %q, want %q", cleanedPath, expectedCleanPath)
		}
	}
}

func BenchmarkAuthCacheKey(b *testing.B) {
	b.ReportAllocs()

	for _, tc := range []struct {
		name string
		set  func(*http.Request)
	}{
		{
			name: "authorization-only",
			set: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token")
			},
		},
		{
			name: "cookie-only",
			set: func(r *http.Request) {
				r.Header.Set("Cookie", "grafana_session=abc")
			},
		},
		{
			name: "authorization-and-cookie",
			set: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token")
				r.Header.Set("Cookie", "grafana_session=abc")
			},
		},
		{
			name: "fallback-with-extra-header",
			set: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token")
				r.Header.Set("Cookie", "grafana_session=abc")
				r.Header.Set("X-Extra", "1")
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/", nil)
			tc.set(req)
			for b.Loop() {
				key := getAuthCacheKey(req)
				if key == ":" {
					b.Fatalf("authCacheKey returned empty key")
				}
			}
		})
	}
}

func BenchmarkResolveRule(b *testing.B) {
	b.ReportAllocs()

	logrus.SetOutput(io.Discard)
	l := logrus.NewEntry(logrus.StandardLogger())
	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "method specific alerting write", method: http.MethodPut, path: "/v1/alerting/templates/template-id"},
		{name: "unknown path fallback", method: http.MethodGet, path: "/v1/not-found-endpoint"},
		{name: "metrics write path", method: http.MethodPost, path: "/victoriametrics/api/v1/write"},
		{name: "query metrics path", method: http.MethodGet, path: "/graph/api/ds/query"},
		{name: "server readyz path", method: http.MethodGet, path: "/v1/server/readyz"},
		{name: "pmm agent connect path", method: http.MethodPost, path: "/agent.v1.AgentService/Connect"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = resolveRule(tc.method, tc.path, l)
			}
		})
	}
}

func BenchmarkIsLocalAgentConnection(b *testing.B) {
	for _, tc := range []struct {
		name       string
		remoteAddr string
		path       string
	}{
		// local IPv4
		{name: "local connect endpoint IPv4", remoteAddr: "127.0.0.1:12345", path: connectionEndpoint},
		{name: "local connectV2 endpoint IPv4", remoteAddr: "127.0.0.1:12345", path: connectionEndpointV2},
		{name: "local rta endpoint IPv4", remoteAddr: "127.0.0.1:12345", path: rtaCollectEndpoint},
		{name: "local unknown endpoint IPv4", remoteAddr: "127.0.0.1:12345", path: "/v1/server/version"},
		// local IPv6
		{name: "local connect endpoint IPv6", remoteAddr: "[::1]:12345", path: connectionEndpoint},
		{name: "local connectV2 endpoint IPv6", remoteAddr: "[::1]:12345", path: connectionEndpointV2},
		{name: "local rta endpoint IPv6", remoteAddr: "[::1]:12345", path: rtaCollectEndpoint},
		{name: "local unknown endpoint IPv6", remoteAddr: "[::1]:12345", path: "/v1/server/version"},
		// remote
		{name: "remote connect endpoint IPv4", remoteAddr: "10.0.0.2:12345", path: connectionEndpoint},
		{name: "remote connectV2 endpoint IPv4", remoteAddr: "10.0.0.2:12345", path: connectionEndpointV2},
		{name: "remote rta endpoint IPv4", remoteAddr: "10.0.0.2:12345", path: rtaCollectEndpoint},
		{name: "remote unknown endpoint IPv4", remoteAddr: "10.0.0.2:12345", path: "/v1/server/version"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, tc.path, nil)
			req.RemoteAddr = tc.remoteAddr

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = isLocalAgentConnection(req)
			}
		})
	}
}
