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
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func BenchmarkAuthServerAuthenticateUser(b *testing.B) {
	l := logrus.WithField("benchmark", b.Name())

	b.Run("localhost static endpoint", func(b *testing.B) {
		s := NewAuthServer(newMockGrafanaAuthUserGetter(b), nil)
		req := httptest.NewRequestWithContext(b.Context(), http.MethodPost, connectionEndpoint, nil)
		req.RemoteAddr = "127.0.0.1:12345"

		b.ReportAllocs()
		for b.Loop() {
			got, authErr := s.authenticateUser(b.Context(), req, l)
			if authErr != nil {
				b.Fatalf("authenticateUser returned error: %v", authErr)
			}
			if got == nil {
				b.Fatal("authenticateUser returned nil user")
			}
		}
	})

	b.Run("remote cache hit", func(b *testing.B) {
		grafanaMock := newMockGrafanaAuthUserGetter(b)
		s := NewAuthServer(grafanaMock, nil)
		req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/v1/server/settings", nil)
		req.RemoteAddr = "10.0.0.1:443"
		req.Header.Set("Authorization", "Bearer hit")

		grafanaMock.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{role: admin, userID: 42}, nil)

		_, authErr := s.authenticateUser(b.Context(), req, l)
		if authErr != nil {
			b.Fatalf("warmup authenticateUser returned error: %v", authErr)
		}

		b.ReportAllocs()
		for b.Loop() {
			got, err := s.authenticateUser(b.Context(), req, l)
			if err != nil {
				b.Fatalf("authenticateUser returned error: %v", err)
			}
			if got == nil {
				b.Fatal("authenticateUser returned nil user")
			}
		}
	})

	b.Run("remote cache miss", func(b *testing.B) {
		grafanaMock := newMockGrafanaAuthUserGetter(b)
		s := NewAuthServer(grafanaMock, nil)

		grafanaMock.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
			Return(authUser{role: admin, userID: 42}, nil)

		seq := 0
		b.ReportAllocs()
		for b.Loop() {
			req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/v1/server/settings", nil)
			req.RemoteAddr = "10.0.0.1:443"
			req.Header.Set("Authorization", "Bearer "+strconv.Itoa(seq))
			seq++

			got, err := s.authenticateUser(b.Context(), req, l)
			if err != nil {
				b.Fatalf("authenticateUser returned error: %v", err)
			}
			if got == nil {
				b.Fatal("authenticateUser returned nil user")
			}
		}
	})
}

func BenchmarkAuthServerServeHTTP(b *testing.B) {
	grafanaMock := newMockGrafanaAuthUserGetter(b)
	accessControlMock := newMockAccessControl(b)
	accessControlMock.On("isEnabled").Return(false).Maybe()
	b.Cleanup(func() {
		grafanaMock.AssertExpectations(b)
		accessControlMock.AssertExpectations(b)
	})

	s := NewAuthServer(grafanaMock, nil)
	s.accessControl = accessControlMock

	grafanaMock.On("getAuthUser", mock.Anything, mock.Anything, mock.Anything).
		Return(authUser{role: admin, userID: 1001}, nil)

	b.ReportAllocs()

	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "method specific alerting write", method: http.MethodPut, path: "/v1/alerting/templates/template-id"},
		{name: "metrics write path", method: http.MethodGet, path: "/victoriametrics/api/v1/write"},
		{name: "query metrics path", method: http.MethodGet, path: "/graph/api/ds/query"},
		{name: "server readyz path", method: http.MethodGet, path: "/v1/server/readyz"},
		{name: "pmm agent connect path", method: http.MethodGet, path: "/agent.v1.AgentService/Connect"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			tokenSeq := 0
			for b.Loop() {
				req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/auth_request", nil)
				req.Header.Set("X-Original-Method", tc.method)
				req.Header.Set("X-Original-Uri", tc.path)
				req.Header.Set("Authorization", "Bearer "+strconv.Itoa(tokenSeq))
				tokenSeq++

				rw := httptest.NewRecorder()
				s.ServeHTTP(rw, req)
				if rw.Code != http.StatusOK {
					b.Fatalf("unexpected status code: got %d, want %d", rw.Code, http.StatusOK)
				}
			}
		})
	}
}
