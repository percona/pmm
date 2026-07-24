// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grafana

import (
	"io"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func BenchmarkCleanPath(b *testing.B) {
	const unescapedURI = "/v1/server/AWSInstanceCheck/..%2f..%2f..%2f..%2fgraph/api/datasources/proxy/8%2f%2f.%2f..%2f8%2f%2f?query=WITH%20CASE%20WHEN%203000%20x%2060%20THEN%203000%20ELSE%2060%20END%20SELECT%20hostname%2Cstatus%20FROM%20pinba.report_by_all%20WHERE%20timestamp%3E%3D1707139680%20AND%20timestamp%3C%3D1707312480%20ORDER%20BY%20t"
	const expectedCleanPath = "/graph/api/datasources/proxy/8"

	b.ReportAllocs()

	cleanedPath, err := cleanPath(unescapedURI)
	require.NoError(b, err)
	require.Equal(b, expectedCleanPath, cleanedPath)

	b.ResetTimer()
	for b.Loop() {
		cleanedPath, err = cleanPath(unescapedURI)
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
		name    string
		headers http.Header
	}{
		{
			name:    "authorization-only",
			headers: http.Header{"Authorization": []string{"Bearer token"}},
		},
		{
			name:    "cookie-only",
			headers: http.Header{"Cookie": []string{"grafana_session=abc"}},
		},
		{
			name: "authorization-and-cookie",
			headers: http.Header{
				"Authorization": []string{"Bearer token"},
				"Cookie":        []string{"grafana_session=abc"},
			},
		},
		{
			name: "fallback-with-extra-header",
			headers: http.Header{
				"Authorization": []string{"Bearer token"},
				"Cookie":        []string{"grafana_session=abc"},
				"X-Extra":       []string{"1"},
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				key, err := authCacheKey(tc.headers)
				if err != nil {
					b.Fatalf("authCacheKey returned error: %v", err)
				}
				if key == "" {
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
